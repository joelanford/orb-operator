package controller

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"pkg.package-operator.run/boxcutter"
	"pkg.package-operator.run/boxcutter/machinery"
	"pkg.package-operator.run/boxcutter/machinery/types"
	"pkg.package-operator.run/boxcutter/managedcache"
	"pkg.package-operator.run/boxcutter/ownerhandling"
	"pkg.package-operator.run/boxcutter/probing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
	cosac "github.com/joelanford/orb-operator/applyconfigurations/api/v1alpha1"
	"github.com/joelanford/orb-operator/internal/assertions"
)

const (
	cosFieldOwner = "cos-controller"
	managedBy     = "orb-operator"
	systemPrefix  = "orb.operatorframework.io"
	finalizerKey  = "orb.operatorframework.io/cos-finalizer"
	groupIndex    = ".spec.group"
)

type COSReconciler struct {
	client          client.Client
	scheme          *runtime.Scheme
	restMapper      meta.RESTMapper
	discoveryClient discovery.OpenAPIV3SchemaInterface
	accessManager   managedcache.ObjectBoundAccessManager[*orbv1alpha1.ClusterObjectSet]
	ownerStrategy   boxcutter.OwnerStrategy
}

func NewCOSReconciler(
	c client.Client,
	scheme *runtime.Scheme,
	restMapper meta.RESTMapper,
	discoveryClient discovery.OpenAPIV3SchemaInterface,
	accessManager managedcache.ObjectBoundAccessManager[*orbv1alpha1.ClusterObjectSet],
) *COSReconciler {
	return &COSReconciler{
		client:          c,
		scheme:          scheme,
		restMapper:      restMapper,
		discoveryClient: discoveryClient,
		accessManager:   accessManager,
		ownerStrategy:   ownerhandling.NewNative(scheme),
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

func (r *COSReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("cos").
		For(&orbv1alpha1.ClusterObjectSet{}).
		WatchesRawSource(
			r.accessManager.Source(
				handler.EnqueueRequestForOwner(r.scheme, mgr.GetRESTMapper(), &orbv1alpha1.ClusterObjectSet{}, handler.OnlyControllerOwner()),
			),
		).
		WithOptions(controller.Options{MaxConcurrentReconciles: 4}).
		Complete(r)
}

func (r *COSReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	existing := &orbv1alpha1.ClusterObjectSet{}
	if err := r.client.Get(ctx, req.NamespacedName, existing); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return r.reconcile(ctx, log, existing)
}

func (r *COSReconciler) reconcile(ctx context.Context, log logr.Logger, cos *orbv1alpha1.ClusterObjectSet) (ctrl.Result, error) {
	if !cos.DeletionTimestamp.IsZero() || cos.Spec.LifecycleState == orbv1alpha1.LifecycleStateArchived {
		return r.teardownAndRelease(ctx, log, cos)
	}

	groupMembers, err := r.listGroupMembers(ctx, cos.Spec.Group)
	if err != nil {
		return ctrl.Result{}, err
	}

	ownerKey := controllerOwnerKeyOf(cos)
	members := filterByControllerOwner(groupMembers, ownerKey)
	chain := buildChain(members)

	if applied, err := r.ensureFinalizer(ctx, cos); applied || err != nil {
		return ctrl.Result{}, err
	}

	siblings := chain.siblingsOf(cos)
	return ctrl.Result{}, r.reconcileActive(ctx, log, cos, siblings)
}

type revisionChain struct {
	latestActive *orbv1alpha1.ClusterObjectSet
	predecessors []*orbv1alpha1.ClusterObjectSet
	archived     []*orbv1alpha1.ClusterObjectSet
	deleted      []*orbv1alpha1.ClusterObjectSet
}

func buildChain(members []orbv1alpha1.ClusterObjectSet) revisionChain {
	slices.SortFunc(members, func(a, b orbv1alpha1.ClusterObjectSet) int {
		return cmp.Compare(b.Spec.Revision, a.Spec.Revision)
	})

	var ch revisionChain
	for i := range members {
		m := &members[i]
		switch {
		case !m.DeletionTimestamp.IsZero():
			ch.deleted = append(ch.deleted, m)
		case m.Spec.LifecycleState == orbv1alpha1.LifecycleStateArchived:
			ch.archived = append(ch.archived, m)
		case ch.latestActive == nil:
			ch.latestActive = m
		default:
			ch.predecessors = append(ch.predecessors, m)
		}
	}
	return ch
}

func (ch revisionChain) siblingsOf(cos *orbv1alpha1.ClusterObjectSet) []*orbv1alpha1.ClusterObjectSet {
	var siblings []*orbv1alpha1.ClusterObjectSet
	if ch.latestActive != nil && ch.latestActive.Name != cos.Name {
		siblings = append(siblings, ch.latestActive)
	}
	for _, p := range ch.predecessors {
		if p.Name != cos.Name {
			siblings = append(siblings, p)
		}
	}
	return siblings
}

func (r *COSReconciler) ensureFinalizer(ctx context.Context, cos *orbv1alpha1.ClusterObjectSet) (bool, error) {
	applied, err := applyCOS(ctx, r.client, cos, cosFieldOwner,
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

func (r *COSReconciler) reconcileActive(ctx context.Context, log logr.Logger, cos *orbv1alpha1.ClusterObjectSet, siblings []*orbv1alpha1.ClusterObjectSet) error {
	log.Info("reconciling active COS")

	existing := cos.DeepCopy()
	reconcileErr := r.doReconcileActive(ctx, cos, siblings)

	if !equality.Semantic.DeepEqual(existing.Status, cos.Status) {
		if err := r.client.Status().Update(ctx, cos); err != nil {
			return errors.Join(reconcileErr, fmt.Errorf("updating status for %s: %w", cos.Name, err))
		}
	}
	return reconcileErr
}

func (r *COSReconciler) doReconcileActive(ctx context.Context, cos *orbv1alpha1.ClusterObjectSet, siblings []*orbv1alpha1.ClusterObjectSet) error {
	engine, err := r.engineForCOS(ctx, cos)
	if err != nil {
		setInternalErrorStatus(cos, fmt.Sprintf("engine setup: %v", err))
		return err
	}

	rev, err := r.buildRevisionWithSiblings(cos, siblings)
	if err != nil {
		setInternalErrorStatus(cos, fmt.Sprintf("building revision: %v", err))
		return fmt.Errorf("building revision: %w", err)
	}
	result, err := engine.Reconcile(ctx, rev, types.WithAggregatePhaseReconcileErrors())
	cos.Status.ObservedPhases = observedPhasesFromReconcileResult(cos.Spec.Phases, result)
	if err != nil {
		setCondition(cos, metav1.ConditionUnknown, orbv1alpha1.ReasonReconcileError, fmt.Sprintf("reconcile failed: %v", err))
		return fmt.Errorf("reconciling: %w", err)
	}

	if verr := result.GetValidationError(); verr != nil {
		setCondition(cos, metav1.ConditionFalse, orbv1alpha1.ReasonInvalidRevision, verr.Error())
		return nil
	}

	// HasProgressed implies IsComplete (progressed objects pass their probes),
	// so check HasProgressed first to distinguish "all objects adopted by a
	// sibling" from "all objects healthy under this revision."
	switch {
	case result.HasProgressed():
		setCondition(cos, metav1.ConditionFalse, orbv1alpha1.ReasonSuperseded, "all objects adopted by a newer revision")
	case result.IsComplete():
		if cos.Status.CompletedAt == nil {
			now := metav1.Now()
			cos.Status.CompletedAt = &now
		}
		setCondition(cos, metav1.ConditionTrue, orbv1alpha1.ReasonAvailable, "all phases complete")
	default:
		setCondition(cos, metav1.ConditionFalse, orbv1alpha1.ReasonUnavailable, "phases not yet complete")
	}
	return nil
}

func (r *COSReconciler) teardownAndRelease(ctx context.Context, log logr.Logger, cos *orbv1alpha1.ClusterObjectSet) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(cos, finalizerKey) {
		return ctrl.Result{}, nil
	}

	// The VAP "cos-orphan-finalizer-ordering" guarantees the "orphan" finalizer
	// cannot be removed while our finalizer is still present. When the "orphan"
	// finalizer is set, skip teardown but still release the finalizer so the
	// deletion can proceed.
	if !cos.DeletionTimestamp.IsZero() && controllerutil.ContainsFinalizer(cos, "orphan") {
		log.Info("orphan finalizer present, skipping teardown")
		return ctrl.Result{}, r.releaseCOS(ctx, cos)
	}

	existing := cos.DeepCopy()
	requeue, reconcileErr := r.doTeardownCOS(ctx, cos)

	if !equality.Semantic.DeepEqual(existing.Status, cos.Status) {
		if err := r.client.Status().Update(ctx, cos); err != nil {
			return ctrl.Result{}, errors.Join(reconcileErr, fmt.Errorf("updating status for %s: %w", cos.Name, err))
		}
	}
	if reconcileErr != nil {
		return ctrl.Result{}, reconcileErr
	}
	if requeue {
		return ctrl.Result{Requeue: true}, nil
	}

	if err := r.releaseCOS(ctx, cos); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *COSReconciler) doTeardownCOS(ctx context.Context, cos *orbv1alpha1.ClusterObjectSet) (bool, error) {
	engine, err := r.engineForCOS(ctx, cos)
	if err != nil {
		setInternalErrorStatus(cos, fmt.Sprintf("engine setup: %v", err))
		return false, fmt.Errorf("engine setup: %w", err)
	}

	rev, err := r.buildRevision(cos)
	if err != nil {
		setInternalErrorStatus(cos, fmt.Sprintf("building revision: %v", err))
		return false, fmt.Errorf("building revision: %w", err)
	}

	result, teardownErr := engine.Teardown(ctx, rev, types.WithAggregatePhaseTeardownErrors())
	setTeardownStatus(cos, result, teardownErr)

	if teardownErr != nil {
		return false, fmt.Errorf("teardown: %w", teardownErr)
	}
	if !result.IsComplete() {
		return true, nil
	}
	return false, nil
}

func setTeardownStatus(cos *orbv1alpha1.ClusterObjectSet, result machinery.RevisionTeardownResult, teardownErr error) {
	cos.Status.ObservedPhases = observedPhasesFromTeardownResult(cos.Spec.Phases, result)
	switch {
	case teardownErr != nil:
		setCondition(cos, metav1.ConditionUnknown, orbv1alpha1.ReasonTeardownError,
			fmt.Sprintf("teardown failed: %v", teardownErr))
	case result != nil && !result.IsComplete():
		setCondition(cos, metav1.ConditionFalse, orbv1alpha1.ReasonArchived, "teardown in progress")
	default:
		setCondition(cos, metav1.ConditionFalse, orbv1alpha1.ReasonArchived, "teardown complete")
	}
}

func (r *COSReconciler) releaseCOS(ctx context.Context, cos *orbv1alpha1.ClusterObjectSet) error {
	if err := r.accessManager.FreeWithUser(ctx, cos, cos); err != nil {
		return fmt.Errorf("freeing access manager: %w", err)
	}
	if err := removeFinalizer(ctx, r.client, cos, finalizerKey); err != nil {
		return fmt.Errorf("removing finalizer: %w", err)
	}
	// Wait for the informer cache to reflect the finalizer removal (or
	// deletion) before returning. controller-runtime serializes reconciles
	// per key, so blocking here ensures the next queued reconcile reads the
	// updated state and exits early at the ContainsFinalizer check instead
	// of re-acquiring the cache for a doomed COS.
	if err := waitForFinalizerRemoval(ctx, r.client, client.ObjectKeyFromObject(cos)); err != nil {
		return fmt.Errorf("waiting for cache to sync finalizer removal: %w", err)
	}
	return nil
}

func (r *COSReconciler) engineForCOS(ctx context.Context, cos *orbv1alpha1.ClusterObjectSet) (*boxcutter.RevisionEngine, error) {
	usedFor, err := r.managedObjectsForCOS(cos)
	if err != nil {
		return nil, fmt.Errorf("listing managed objects: %w", err)
	}
	accessor, err := r.accessManager.GetWithUser(ctx, cos, cos, usedFor)
	if err != nil {
		return nil, fmt.Errorf("getting accessor: %w", err)
	}
	engine, err := boxcutter.NewRevisionEngine(boxcutter.RevisionEngineOptions{
		Scheme:           r.scheme,
		FieldOwner:       "cos-group/" + cos.Spec.Group,
		SystemPrefix:     systemPrefix,
		ManagedBy:        managedBy,
		DiscoveryClient:  r.discoveryClient,
		RestMapper:       r.restMapper,
		Writer:           accessor,
		Reader:           accessor,
		UnfilteredReader: accessor.UnfilteredReader(),
	})
	if err != nil {
		return nil, fmt.Errorf("creating revision engine: %w", err)
	}
	return engine, nil
}

func (r *COSReconciler) managedObjectsForCOS(cos *orbv1alpha1.ClusterObjectSet) ([]client.Object, error) {
	seen := map[schema.GroupVersionKind]struct{}{}
	var objects []client.Object
	for _, p := range cos.Spec.Phases {
		for _, o := range p.Objects {
			obj, err := objectFromRawExtension(o.Object)
			if err != nil {
				return nil, fmt.Errorf("phase %q: %w", p.Name, err)
			}
			gvk := obj.GetObjectKind().GroupVersionKind()
			if _, ok := seen[gvk]; ok {
				continue
			}
			seen[gvk] = struct{}{}
			objects = append(objects, obj)
		}
	}
	return objects, nil
}

func (r *COSReconciler) buildRevision(cos *orbv1alpha1.ClusterObjectSet) (boxcutter.Revision, error) {
	return r.buildRevisionWithSiblings(cos, nil)
}

func (r *COSReconciler) buildRevisionWithSiblings(
	cos *orbv1alpha1.ClusterObjectSet,
	siblings []*orbv1alpha1.ClusterObjectSet,
) (boxcutter.Revision, error) {
	phases := make([]boxcutter.Phase, 0, len(cos.Spec.Phases))

	for _, p := range cos.Spec.Phases {
		objects := make([]client.Object, 0, len(p.Objects))
		var phaseReconcileOpts []boxcutter.PhaseReconcileOption

		if p.CollisionProtection != nil {
			phaseReconcileOpts = append(phaseReconcileOpts, mapCollisionProtection(*p.CollisionProtection))
		}

		for _, o := range p.Objects {
			obj, err := objectFromRawExtension(o.Object)
			if err != nil {
				return nil, fmt.Errorf("phase %q: %w", p.Name, err)
			}
			objects = append(objects, obj)

			var objOpts []boxcutter.ObjectReconcileOption

			probe, err := assertions.ProbeForAssertions(o.Assertions)
			if err != nil {
				probe = boxcutter.ProbeFunc(func(_ client.Object) probing.Result {
					return probing.FalseResult(fmt.Sprintf("invalid assertion: %v", err))
				})
			}
			if probe != nil {
				objOpts = append(objOpts, boxcutter.WithProbe(boxcutter.ProgressProbeType, probe))
			}

			if o.CollisionProtection != nil {
				objOpts = append(objOpts, mapCollisionProtection(*o.CollisionProtection))
			}

			if len(objOpts) > 0 {
				phaseReconcileOpts = append(phaseReconcileOpts,
					boxcutter.WithObjectReconcileOptions(obj, objOpts...),
				)
			}
		}

		phase := boxcutter.NewPhaseWithOwner(p.Name, objects, cos, r.ownerStrategy)
		if len(phaseReconcileOpts) > 0 {
			phase.WithReconcileOptions(phaseReconcileOpts...)
		}
		phases = append(phases, phase)
	}

	var reconcileOpts []boxcutter.RevisionReconcileOption

	if cos.Spec.CollisionProtection != nil {
		reconcileOpts = append(reconcileOpts, mapCollisionProtection(*cos.Spec.CollisionProtection))
	} else {
		reconcileOpts = append(reconcileOpts, mapCollisionProtection(orbv1alpha1.CollisionProtectionPrevent))
	}

	if len(siblings) > 0 {
		siblingObjs := make([]client.Object, 0, len(siblings))
		for _, s := range siblings {
			siblingObjs = append(siblingObjs, s)
		}
		reconcileOpts = append(reconcileOpts, boxcutter.WithSiblingOwners(siblingObjs))
	}

	rev := boxcutter.NewRevisionWithOwner(
		cos.Name,
		int64(cos.Spec.Revision),
		phases,
		cos,
		r.ownerStrategy,
	)
	if len(reconcileOpts) > 0 {
		rev.WithReconcileOptions(reconcileOpts...)
	}
	return rev, nil
}

func (r *COSReconciler) listGroupMembers(ctx context.Context, group string) ([]orbv1alpha1.ClusterObjectSet, error) {
	var list orbv1alpha1.ClusterObjectSetList
	if err := r.client.List(ctx, &list, client.MatchingFields{groupIndex: group}); err != nil {
		return nil, fmt.Errorf("listing group members: %w", err)
	}
	slices.SortFunc(list.Items, func(a, b orbv1alpha1.ClusterObjectSet) int {
		return cmp.Compare(a.Spec.Revision, b.Spec.Revision)
	})
	return list.Items, nil
}

type controllerOwnerKey struct {
	Kind string
	Name string
}

func controllerOwnerKeyOf(cos *orbv1alpha1.ClusterObjectSet) controllerOwnerKey {
	ref := metav1.GetControllerOf(cos)
	if ref == nil {
		return controllerOwnerKey{}
	}
	return controllerOwnerKey{Kind: ref.Kind, Name: ref.Name}
}

func filterByControllerOwner(members []orbv1alpha1.ClusterObjectSet, key controllerOwnerKey) []orbv1alpha1.ClusterObjectSet {
	var result []orbv1alpha1.ClusterObjectSet
	for _, m := range members {
		if controllerOwnerKeyOf(&m) == key {
			result = append(result, m)
		}
	}
	return result
}

func objectFromRawExtension(raw runtime.RawExtension) (*unstructured.Unstructured, error) {
	if raw.Object != nil {
		u := &unstructured.Unstructured{}
		data, err := runtime.DefaultUnstructuredConverter.ToUnstructured(raw.Object)
		if err != nil {
			return nil, fmt.Errorf("converting to unstructured: %w", err)
		}
		u.Object = data
		return u, nil
	}
	u := &unstructured.Unstructured{}
	if err := u.UnmarshalJSON(raw.Raw); err != nil {
		return nil, fmt.Errorf("unmarshalling raw extension: %w", err)
	}
	return u, nil
}

func setInternalErrorStatus(cos *orbv1alpha1.ClusterObjectSet, message string) {
	cos.Status.ObservedPhases = nil
	setCondition(cos, metav1.ConditionUnknown, orbv1alpha1.ReasonInternalError, message)
}

func setCondition(cos *orbv1alpha1.ClusterObjectSet, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&cos.Status.Conditions, metav1.Condition{
		Type:               orbv1alpha1.ConditionTypeAvailable,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: cos.Generation,
	})
}

func removeFinalizer(ctx context.Context, c client.Client, cos *orbv1alpha1.ClusterObjectSet, finalizer string) error {
	if !controllerutil.ContainsFinalizer(cos, finalizer) {
		return nil
	}
	patch := client.MergeFromWithOptions(cos.DeepCopy(), client.MergeFromWithOptimisticLock{})
	controllerutil.RemoveFinalizer(cos, finalizer)
	clearFinalizerFieldOwnership(cos.ManagedFields, cosFieldOwner, finalizer)
	return c.Patch(ctx, cos, patch)
}

func waitForFinalizerRemoval(ctx context.Context, c client.Client, key client.ObjectKey) error {
	return wait.PollUntilContextTimeout(ctx, 50*time.Millisecond, 5*time.Second, true, func(ctx context.Context) (bool, error) {
		var cos orbv1alpha1.ClusterObjectSet
		if err := c.Get(ctx, key, &cos); err != nil {
			return apierrors.IsNotFound(err), client.IgnoreNotFound(err)
		}
		return !controllerutil.ContainsFinalizer(&cos, finalizerKey), nil
	})
}

func clearFinalizerFieldOwnership(managedFields []metav1.ManagedFieldsEntry, manager, finalizer string) {
	key := "v:" + finalizer
	for i := range managedFields {
		e := &managedFields[i]
		if e.Manager != manager || e.FieldsV1 == nil {
			continue
		}
		var fields map[string]any
		if err := json.Unmarshal(e.FieldsV1.GetRawBytes(), &fields); err != nil {
			continue
		}
		fMeta, _ := fields["f:metadata"].(map[string]any)
		if fMeta == nil {
			continue
		}
		fFinalizers, _ := fMeta["f:finalizers"].(map[string]any)
		if fFinalizers == nil {
			continue
		}
		delete(fFinalizers, key)
		if len(fFinalizers) == 0 {
			delete(fMeta, "f:finalizers")
		}
		if len(fMeta) == 0 {
			delete(fields, "f:metadata")
		}
		raw, err := json.Marshal(fields)
		if err != nil {
			continue
		}
		e.FieldsV1.SetRawBytes(raw)
	}
}

func mapCollisionProtection(cp orbv1alpha1.CollisionProtection) boxcutter.WithCollisionProtection {
	switch cp {
	case orbv1alpha1.CollisionProtectionIfNoController:
		return boxcutter.WithCollisionProtection(boxcutter.CollisionProtectionIfNoController)
	case orbv1alpha1.CollisionProtectionNone:
		return boxcutter.WithCollisionProtection(boxcutter.CollisionProtectionNone)
	default:
		return boxcutter.WithCollisionProtection(boxcutter.CollisionProtectionPrevent)
	}
}
