package cos

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"pkg.package-operator.run/boxcutter"
	"pkg.package-operator.run/boxcutter/machinery"
	"pkg.package-operator.run/boxcutter/machinery/types"
	"pkg.package-operator.run/boxcutter/managedcache"
	"pkg.package-operator.run/boxcutter/ownerhandling"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
	cosac "github.com/joelanford/orb-operator/applyconfigurations/api/v1alpha1"
	"github.com/joelanford/orb-operator/internal/cosutil"
	orberrors "github.com/joelanford/orb-operator/internal/errors"
	"github.com/joelanford/orb-operator/internal/object"
	"github.com/joelanford/orb-operator/internal/revision"
	cosstatus "github.com/joelanford/orb-operator/internal/status/cos"
)

const (
	fieldOwner   = "cos-controller"
	managedBy    = "orb-operator"
	systemPrefix = "orb.operatorframework.io"
	finalizerKey = "orb.operatorframework.io/cos-finalizer"
	groupIndex   = ".spec.group"
)

type Reconciler struct {
	client          client.Client
	scheme          *runtime.Scheme
	restMapper      meta.RESTMapper
	discoveryClient discovery.OpenAPIV3SchemaInterface
	accessManager   managedcache.ObjectBoundAccessManager[*orbv1alpha1.ClusterObjectSet]
	ownerStrategy   boxcutter.OwnerStrategy
	resolver        *object.Resolver
}

func NewReconciler(
	c client.Client,
	scheme *runtime.Scheme,
	restMapper meta.RESTMapper,
	discoveryClient discovery.OpenAPIV3SchemaInterface,
	accessManager managedcache.ObjectBoundAccessManager[*orbv1alpha1.ClusterObjectSet],
) *Reconciler {
	return &Reconciler{
		client:          c,
		scheme:          scheme,
		restMapper:      restMapper,
		discoveryClient: discoveryClient,
		accessManager:   accessManager,
		ownerStrategy:   ownerhandling.NewNative(scheme),
		resolver:        object.NewResolver(c),
	}
}

func SetupIndexes(mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&orbv1alpha1.ClusterObjectSet{},
		groupIndex,
		func(obj client.Object) []string {
			cos := obj.(*orbv1alpha1.ClusterObjectSet)
			return []string{cos.Spec.Group}
		},
	)
}

// +kubebuilder:rbac:groups=orb.operatorframework.io,resources=clusterobjectslices,verbs=get;list;watch

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("cos").
		For(&orbv1alpha1.ClusterObjectSet{}).
		Watches(&orbv1alpha1.ClusterObjectSlice{}, handler.EnqueueRequestsFromMapFunc(r.cosesForSlice)).
		WatchesRawSource(
			r.accessManager.Source(
				handler.EnqueueRequestForOwner(r.scheme, mgr.GetRESTMapper(), &orbv1alpha1.ClusterObjectSet{}, handler.OnlyControllerOwner()),
			),
		).
		WithOptions(controller.Options{MaxConcurrentReconciles: 4}).
		Complete(r)
}

func (r *Reconciler) cosesForSlice(ctx context.Context, obj client.Object) []ctrl.Request {
	slice := obj.(*orbv1alpha1.ClusterObjectSlice)
	var cosList orbv1alpha1.ClusterObjectSetList
	if err := r.client.List(ctx, &cosList); err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "listing COSs for slice watch")
		return nil
	}

	var requests []ctrl.Request
	for _, cos := range cosList.Items {
		if cosReferencesSlice(&cos, slice.Name) {
			requests = append(requests, ctrl.Request{
				NamespacedName: client.ObjectKeyFromObject(&cos),
			})
		}
	}
	return requests
}

func cosReferencesSlice(cos *orbv1alpha1.ClusterObjectSet, sliceName string) bool {
	for _, p := range cos.Spec.Phases {
		for _, o := range p.Objects {
			if o.ObjectRef != nil && o.ObjectRef.SliceName == sliceName {
				return true
			}
		}
	}
	return false
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	existing := &orbv1alpha1.ClusterObjectSet{}
	if err := r.client.Get(ctx, req.NamespacedName, existing); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return r.reconcile(ctx, log, existing)
}

func (r *Reconciler) reconcile(ctx context.Context, log logr.Logger, cos *orbv1alpha1.ClusterObjectSet) (ctrl.Result, error) {
	if !cos.DeletionTimestamp.IsZero() || cos.Spec.LifecycleState == orbv1alpha1.LifecycleStateArchived {
		return r.teardown(ctx, log, cos)
	}
	return ctrl.Result{}, r.reconcileActive(ctx, log, cos)
}

func (r *Reconciler) reconcileActive(ctx context.Context, log logr.Logger, cos *orbv1alpha1.ClusterObjectSet) error {
	log.Info("reconciling active COS")

	siblings, err := r.resolveSiblings(ctx, cos)
	if err != nil {
		return err
	}

	if applied, err := r.ensureFinalizer(ctx, cos); applied || err != nil {
		return err
	}

	existing := cos.DeepCopy()
	result, reconcileErr := r.doReconcile(ctx, cos, siblings)
	cosstatus.Apply(cos, cosstatus.FromReconcile(cos, result, reconcileErr, time.Now()))

	if !equality.Semantic.DeepEqual(existing.Status, cos.Status) {
		if err := r.client.Status().Update(ctx, cos); err != nil {
			return errors.Join(reconcileErr, fmt.Errorf("updating status for %s: %w", cos.Name, err))
		}
	}
	return reconcileErr
}

func (r *Reconciler) doReconcile(ctx context.Context, cos *orbv1alpha1.ClusterObjectSet, siblings []*orbv1alpha1.ClusterObjectSet) (machinery.RevisionResult, error) {
	engine, rev, err := r.resolveAndPrepare(ctx, cos, siblings)
	if err != nil {
		return nil, err
	}
	return engine.Reconcile(ctx, rev, types.WithAggregatePhaseReconcileErrors())
}

func (r *Reconciler) teardown(ctx context.Context, log logr.Logger, cos *orbv1alpha1.ClusterObjectSet) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(cos, finalizerKey) {
		return ctrl.Result{}, nil
	}

	if !cos.DeletionTimestamp.IsZero() && controllerutil.ContainsFinalizer(cos, "orphan") {
		log.Info("orphan finalizer present, skipping teardown")
		return ctrl.Result{}, r.release(ctx, cos)
	}

	existing := cos.DeepCopy()
	result, teardownErr := r.doTeardown(ctx, cos)
	cosstatus.Apply(cos, cosstatus.FromTeardown(cos, result, teardownErr, time.Now()))

	if !equality.Semantic.DeepEqual(existing.Status, cos.Status) {
		if err := r.client.Status().Update(ctx, cos); err != nil {
			return ctrl.Result{}, errors.Join(teardownErr, fmt.Errorf("updating status for %s: %w", cos.Name, err))
		}
	}
	if teardownErr != nil {
		return ctrl.Result{}, teardownErr
	}
	if result != nil && !result.IsComplete() {
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, r.release(ctx, cos)
}

func (r *Reconciler) doTeardown(ctx context.Context, cos *orbv1alpha1.ClusterObjectSet) (machinery.RevisionTeardownResult, error) {
	engine, rev, err := r.resolveAndPrepare(ctx, cos, nil)
	if err != nil {
		return nil, err
	}
	return engine.Teardown(ctx, rev, types.WithAggregatePhaseTeardownErrors())
}

func (r *Reconciler) resolveAndPrepare(ctx context.Context, cos *orbv1alpha1.ClusterObjectSet, siblings []*orbv1alpha1.ClusterObjectSet) (*revision.Engine, boxcutter.Revision, error) {
	resolved, err := r.resolver.Resolve(ctx, cos.Spec.Phases)
	if err != nil {
		return nil, nil, &orberrors.ObjectResolutionError{Err: err}
	}
	if err := resolved.VerifyHash(cos.Status.ResolvedContentHash); err != nil {
		return nil, nil, &orberrors.ObjectResolutionError{Err: err}
	}
	if cos.Status.ResolvedContentHash == "" {
		cos.Status.ResolvedContentHash = resolved.Hash
	}

	engine, err := r.newEngine(ctx, cos, resolved)
	if err != nil {
		return nil, nil, &orberrors.InternalError{Err: err}
	}

	rev := revision.Build(cos, resolved, siblings, r.ownerStrategy)
	return engine, rev, nil
}

func (r *Reconciler) newEngine(ctx context.Context, cos *orbv1alpha1.ClusterObjectSet, resolved *object.Result) (*revision.Engine, error) {
	usedFor := resolved.ManagedObjects()
	accessor, err := r.accessManager.GetWithUser(ctx, cos, cos, usedFor)
	if err != nil {
		return nil, fmt.Errorf("getting accessor: %w", err)
	}
	return revision.NewEngine(boxcutter.RevisionEngineOptions{
		Scheme:           r.scheme,
		FieldOwner:       "cos-group/" + cos.Spec.Group,
		SystemPrefix:     systemPrefix,
		ManagedBy:        managedBy,
		DiscoveryClient:  r.discoveryClient,
		RestMapper:       r.restMapper,
		Writer:           accessor,
		Reader:           accessor,
		UnfilteredReader: accessor.UnfilteredReader(),
	}, cos.Status.ObservedPhases)
}

func (r *Reconciler) resolveSiblings(ctx context.Context, cos *orbv1alpha1.ClusterObjectSet) ([]*orbv1alpha1.ClusterObjectSet, error) {
	groupMembers, err := r.listGroupMembers(ctx, cos.Spec.Group)
	if err != nil {
		return nil, err
	}
	members := revision.FilterByOwner(groupMembers, cos)
	chain := revision.BuildChain(members)
	return chain.SiblingsOf(cos), nil
}

func (r *Reconciler) listGroupMembers(ctx context.Context, group string) ([]orbv1alpha1.ClusterObjectSet, error) {
	var list orbv1alpha1.ClusterObjectSetList
	if err := r.client.List(ctx, &list, client.MatchingFields{groupIndex: group}); err != nil {
		return nil, fmt.Errorf("listing group members: %w", err)
	}
	slices.SortFunc(list.Items, func(a, b orbv1alpha1.ClusterObjectSet) int {
		return cmp.Compare(a.Spec.Revision, b.Spec.Revision)
	})
	return list.Items, nil
}

func (r *Reconciler) ensureFinalizer(ctx context.Context, cos *orbv1alpha1.ClusterObjectSet) (bool, error) {
	applied, err := cosutil.Apply(ctx, r.client, cos, fieldOwner,
		func(cos *orbv1alpha1.ClusterObjectSet) bool {
			return !controllerutil.ContainsFinalizer(cos, finalizerKey)
		},
		func(ac *cosac.ClusterObjectSetApplyConfiguration) {
			ac.WithFinalizers(finalizerKey)
		},
	)
	if err != nil {
		return false, fmt.Errorf("adding finalizer to %s: %w", cos.Name, err)
	}
	return applied, nil
}

func (r *Reconciler) release(ctx context.Context, cos *orbv1alpha1.ClusterObjectSet) error {
	if err := r.accessManager.FreeWithUser(ctx, cos, cos); err != nil {
		return fmt.Errorf("freeing access manager: %w", err)
	}
	if err := cosutil.RemoveFinalizer(ctx, r.client, cos, fieldOwner, finalizerKey); err != nil {
		return fmt.Errorf("removing finalizer: %w", err)
	}
	if err := cosutil.WaitForFinalizerRemoval(ctx, r.client, client.ObjectKeyFromObject(cos), finalizerKey); err != nil {
		return fmt.Errorf("waiting for cache to sync finalizer removal: %w", err)
	}
	return nil
}
