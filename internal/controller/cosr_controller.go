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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"pkg.package-operator.run/boxcutter"
	"pkg.package-operator.run/boxcutter/managedcache"
	"pkg.package-operator.run/boxcutter/ownerhandling"
	"pkg.package-operator.run/boxcutter/probing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
	"github.com/joelanford/orb-operator/internal/assertions"
)

const (
	fieldOwner   = "orb-operator"
	systemPrefix = "orb"
	finalizerKey = "orb.operatorframework.io/cosr-finalizer"
	groupIndex   = ".spec.group"
)

type COSRReconciler struct {
	client          client.Client
	scheme          *runtime.Scheme
	restMapper      meta.RESTMapper
	discoveryClient discovery.OpenAPIV3SchemaInterface
	accessManager   managedcache.ObjectBoundAccessManager[*orbv1alpha1.ClusterObjectSetRevision]
	ownerStrategy   boxcutter.OwnerStrategy
}

func NewCOSRReconciler(
	c client.Client,
	scheme *runtime.Scheme,
	restMapper meta.RESTMapper,
	discoveryClient discovery.OpenAPIV3SchemaInterface,
	accessManager managedcache.ObjectBoundAccessManager[*orbv1alpha1.ClusterObjectSetRevision],
) *COSRReconciler {
	return &COSRReconciler{
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
		&orbv1alpha1.ClusterObjectSetRevision{},
		groupIndex,
		func(obj client.Object) []string {
			cosr := obj.(*orbv1alpha1.ClusterObjectSetRevision)
			return []string{cosr.Spec.Group}
		},
	)
}

func (r *COSRReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("cosr").
		Watches(
			&orbv1alpha1.ClusterObjectSetRevision{},
			handler.EnqueueRequestsFromMapFunc(r.mapToHighestRevInChain),
		).
		WatchesRawSource(
			r.accessManager.Source(
				managedcache.NewEnqueueWatchingObjects(
					r.accessManager,
					&orbv1alpha1.ClusterObjectSetRevision{},
					r.scheme,
				),
			),
		).
		Complete(r)
}

func (r *COSRReconciler) mapToHighestRevInChain(ctx context.Context, obj client.Object) []reconcile.Request {
	cosr := obj.(*orbv1alpha1.ClusterObjectSetRevision)
	var list orbv1alpha1.ClusterObjectSetRevisionList
	if err := r.client.List(ctx, &list, client.MatchingFields{groupIndex: cosr.Spec.Group}); err != nil {
		return nil
	}

	ownerName := controllerOwnerName(cosr)
	var latest *orbv1alpha1.ClusterObjectSetRevision
	for i := range list.Items {
		m := &list.Items[i]
		if controllerOwnerName(m) != ownerName {
			continue
		}
		if latest == nil || m.Spec.Revision > latest.Spec.Revision {
			latest = m
		}
	}
	if latest == nil {
		return nil
	}
	return []reconcile.Request{{NamespacedName: client.ObjectKeyFromObject(latest)}}
}

func (r *COSRReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	existing := &orbv1alpha1.ClusterObjectSetRevision{}
	if err := r.client.Get(ctx, req.NamespacedName, existing); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !existing.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, log, existing)
	}

	if !controllerutil.ContainsFinalizer(existing, finalizerKey) {
		controllerutil.AddFinalizer(existing, finalizerKey)
		if err := r.client.Update(ctx, existing); err != nil {
			return ctrl.Result{}, fmt.Errorf("adding finalizer: %w", err)
		}
		return ctrl.Result{}, nil
	}

	return r.reconcile(ctx, log, existing)
}

func (r *COSRReconciler) reconcile(ctx context.Context, log logr.Logger, cosr *orbv1alpha1.ClusterObjectSetRevision) (ctrl.Result, error) {
	groupMembers, err := r.listGroupMembers(ctx, cosr.Spec.Group)
	if err != nil {
		setCondition(cosr, metav1.ConditionFalse, orbv1alpha1.ReasonUnavailable, fmt.Sprintf("listing group members: %v", err))
		return ctrl.Result{}, err
	}

	ownerName := controllerOwnerName(cosr)
	members := filterByControllerOwner(groupMembers, ownerName)
	chain := buildChain(members)

	return r.reconcileChain(ctx, log, cosr, chain)
}

type revisionChain struct {
	latestActive *orbv1alpha1.ClusterObjectSetRevision
	predecessors []*orbv1alpha1.ClusterObjectSetRevision
	archived     []*orbv1alpha1.ClusterObjectSetRevision
}

func buildChain(members []orbv1alpha1.ClusterObjectSetRevision) revisionChain {
	slices.SortFunc(members, func(a, b orbv1alpha1.ClusterObjectSetRevision) int {
		return cmp.Compare(b.Spec.Revision, a.Spec.Revision)
	})

	var ch revisionChain
	for i := range members {
		m := &members[i]
		switch {
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

func (r *COSRReconciler) reconcileChain(ctx context.Context, log logr.Logger, cosr *orbv1alpha1.ClusterObjectSetRevision, chain revisionChain) (ctrl.Result, error) {
	if err := r.reconcileActiveMembers(ctx, log, cosr, chain); err != nil {
		return ctrl.Result{}, err
	}

	var needsRequeue bool
	for _, m := range chain.archived {
		requeue, err := r.reconcileArchived(ctx, log, m)
		if err != nil {
			return ctrl.Result{}, err
		}
		if requeue {
			needsRequeue = true
		}
	}

	if needsRequeue {
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}
	return ctrl.Result{}, nil
}

func (r *COSRReconciler) reconcileActiveMembers(ctx context.Context, log logr.Logger, _ *orbv1alpha1.ClusterObjectSetRevision, chain revisionChain) error {
	if chain.latestActive == nil {
		return nil
	}

	if err := r.reconcileLatest(ctx, log, chain.latestActive, chain.predecessors); err != nil {
		return err
	}

	for _, p := range chain.predecessors {
		if err := r.reconcilePredecessor(ctx, log, p); err != nil {
			return err
		}
	}
	return nil
}

func (r *COSRReconciler) reconcileLatest(
	ctx context.Context, log logr.Logger,
	cosr *orbv1alpha1.ClusterObjectSetRevision,
	predecessors []*orbv1alpha1.ClusterObjectSetRevision,
) error {
	log.Info("reconciling latest active COSR")

	existing := cosr.DeepCopy()
	reconcileErr := r.doReconcileLatest(ctx, cosr, predecessors)

	if !equality.Semantic.DeepEqual(existing.Status, cosr.Status) {
		if err := r.client.Status().Update(ctx, cosr); err != nil {
			return errors.Join(reconcileErr, fmt.Errorf("updating status for %s: %w", cosr.Name, err))
		}
	}
	return reconcileErr
}

func (r *COSRReconciler) doReconcileLatest(
	ctx context.Context,
	cosr *orbv1alpha1.ClusterObjectSetRevision,
	predecessors []*orbv1alpha1.ClusterObjectSetRevision,
) error {
	engine, err := r.engineForCOSR(ctx, cosr)
	if err != nil {
		setCondition(cosr, metav1.ConditionFalse, orbv1alpha1.ReasonUnavailable, fmt.Sprintf("engine setup: %v", err))
		return err
	}

	rev, err := r.buildRevisionWithPreviousOwners(cosr, predecessors)
	if err != nil {
		setCondition(cosr, metav1.ConditionFalse, orbv1alpha1.ReasonUnavailable, fmt.Sprintf("building revision: %v", err))
		return fmt.Errorf("building revision: %w", err)
	}
	result, err := engine.Reconcile(ctx, rev)
	if err != nil {
		setCondition(cosr, metav1.ConditionFalse, orbv1alpha1.ReasonUnavailable, fmt.Sprintf("reconcile failed: %v", err))
		return fmt.Errorf("reconciling: %w", err)
	}

	if result.IsComplete() {
		setCondition(cosr, metav1.ConditionTrue, orbv1alpha1.ReasonAvailable, "all phases complete")
	} else {
		setCondition(cosr, metav1.ConditionFalse, orbv1alpha1.ReasonUnavailable, "phases not yet complete")
	}
	return nil
}

func (r *COSRReconciler) reconcilePredecessor(ctx context.Context, log logr.Logger, cosr *orbv1alpha1.ClusterObjectSetRevision) error {
	log.Info("reconciling predecessor COSR", "cosr", cosr.Name)

	existing := cosr.DeepCopy()
	r.doReconcilePredecessor(cosr)

	if !equality.Semantic.DeepEqual(existing.Status, cosr.Status) {
		if err := r.client.Status().Update(ctx, cosr); err != nil {
			return fmt.Errorf("updating predecessor status for %s: %w", cosr.Name, err)
		}
	}
	return nil
}

func (r *COSRReconciler) doReconcilePredecessor(cosr *orbv1alpha1.ClusterObjectSetRevision) {
	setCondition(cosr, metav1.ConditionFalse, orbv1alpha1.ReasonSuperseded, "a newer revision exists in this group")
}

func (r *COSRReconciler) reconcileArchived(ctx context.Context, log logr.Logger, cosr *orbv1alpha1.ClusterObjectSetRevision) (bool, error) {
	log.Info("reconciling archived COSR", "cosr", cosr.Name)

	existing := cosr.DeepCopy()
	needsRequeue, reconcileErr := r.doReconcileArchived(ctx, cosr)

	if !equality.Semantic.DeepEqual(existing.Status, cosr.Status) {
		if err := r.client.Status().Update(ctx, cosr); err != nil {
			return false, errors.Join(reconcileErr, fmt.Errorf("updating archived status for %s: %w", cosr.Name, err))
		}
	}
	if reconcileErr != nil {
		return false, reconcileErr
	}

	return needsRequeue, nil
}

func (r *COSRReconciler) doReconcileArchived(ctx context.Context, cosr *orbv1alpha1.ClusterObjectSetRevision) (bool, error) {
	engine, err := r.engineForCOSR(ctx, cosr)
	if err != nil {
		setCondition(cosr, metav1.ConditionFalse, orbv1alpha1.ReasonArchived, fmt.Sprintf("engine setup: %v", err))
		return false, err
	}

	rev, err := r.buildRevision(cosr)
	if err != nil {
		setCondition(cosr, metav1.ConditionFalse, orbv1alpha1.ReasonArchived, fmt.Sprintf("building revision: %v", err))
		return false, fmt.Errorf("building revision: %w", err)
	}
	result, err := engine.Teardown(ctx, rev)
	if err != nil {
		setCondition(cosr, metav1.ConditionFalse, orbv1alpha1.ReasonArchived, fmt.Sprintf("teardown failed: %v", err))
		return false, fmt.Errorf("teardown: %w", err)
	}

	if result.IsComplete() {
		setCondition(cosr, metav1.ConditionFalse, orbv1alpha1.ReasonArchived, "teardown complete")
	} else {
		setCondition(cosr, metav1.ConditionFalse, orbv1alpha1.ReasonArchived, "teardown in progress")
	}
	return !result.IsComplete(), nil
}

func (r *COSRReconciler) handleDeletion(ctx context.Context, log logr.Logger, cosr *orbv1alpha1.ClusterObjectSetRevision) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(cosr, finalizerKey) {
		return ctrl.Result{}, nil
	}

	if res, err := r.teardownForDeletion(ctx, log, cosr); res.RequeueAfter > 0 || err != nil {
		return res, err
	}

	if err := r.accessManager.FreeWithUser(ctx, cosr, cosr); err != nil {
		return ctrl.Result{}, fmt.Errorf("freeing access manager: %w", err)
	}

	if err := removeFinalizer(ctx, r.client, cosr, finalizerKey); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *COSRReconciler) teardownForDeletion(ctx context.Context, log logr.Logger, cosr *orbv1alpha1.ClusterObjectSetRevision) (ctrl.Result, error) {
	// The VAP "cosr-orphan-finalizer-ordering" guarantees the "orphan" finalizer
	// cannot be removed while our finalizer is still present. So if the "orphan"
	// finalizer is set, we can safely skip teardown.
	if controllerutil.ContainsFinalizer(cosr, "orphan") {
		log.Info("orphan finalizer present, skipping teardown")
		return ctrl.Result{}, nil
	}

	log.Info("tearing down for deletion")
	engine, err := r.engineForCOSR(ctx, cosr)
	if err != nil {
		return ctrl.Result{}, err
	}

	rev, err := r.buildRevision(cosr)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("building revision: %w", err)
	}
	result, err := engine.Teardown(ctx, rev)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("teardown: %w", err)
	}

	if !result.IsComplete() {
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}
	return ctrl.Result{}, nil
}

func (r *COSRReconciler) engineForCOSR(ctx context.Context, cosr *orbv1alpha1.ClusterObjectSetRevision) (*boxcutter.RevisionEngine, error) {
	usedFor, err := r.managedObjectsForCOSR(cosr)
	if err != nil {
		return nil, fmt.Errorf("listing managed objects: %w", err)
	}
	accessor, err := r.accessManager.GetWithUser(ctx, cosr, cosr, usedFor)
	if err != nil {
		return nil, fmt.Errorf("getting accessor: %w", err)
	}
	engine, err := boxcutter.NewRevisionEngine(boxcutter.RevisionEngineOptions{
		Scheme:           r.scheme,
		FieldOwner:       fieldOwner,
		SystemPrefix:     systemPrefix,
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

func (r *COSRReconciler) managedObjectsForCOSR(cosr *orbv1alpha1.ClusterObjectSetRevision) ([]client.Object, error) {
	seen := map[schema.GroupVersionKind]struct{}{}
	var objects []client.Object
	for _, p := range cosr.Spec.Phases {
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

func (r *COSRReconciler) buildRevision(cosr *orbv1alpha1.ClusterObjectSetRevision) (boxcutter.Revision, error) {
	return r.buildRevisionWithPreviousOwners(cosr, nil)
}

func (r *COSRReconciler) buildRevisionWithPreviousOwners(
	cosr *orbv1alpha1.ClusterObjectSetRevision,
	previousOwners []*orbv1alpha1.ClusterObjectSetRevision,
) (boxcutter.Revision, error) {
	phases := make([]boxcutter.Phase, 0, len(cosr.Spec.Phases))

	for _, p := range cosr.Spec.Phases {
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

		phase := boxcutter.NewPhaseWithOwner(p.Name, objects, cosr, r.ownerStrategy)
		if len(phaseReconcileOpts) > 0 {
			phase.WithReconcileOptions(phaseReconcileOpts...)
		}
		phases = append(phases, phase)
	}

	var reconcileOpts []boxcutter.RevisionReconcileOption

	if cosr.Spec.CollisionProtection != nil {
		reconcileOpts = append(reconcileOpts, mapCollisionProtection(*cosr.Spec.CollisionProtection))
	} else {
		reconcileOpts = append(reconcileOpts, mapCollisionProtection(orbv1alpha1.CollisionProtectionPrevent))
	}

	if len(previousOwners) > 0 {
		prevOwners := make(boxcutter.WithPreviousOwners, 0, len(previousOwners))
		for _, po := range previousOwners {
			prevOwners = append(prevOwners, po)
		}
		reconcileOpts = append(reconcileOpts, prevOwners)
	}

	rev := boxcutter.NewRevisionWithOwner(
		cosr.Name,
		int64(cosr.Spec.Revision),
		phases,
		cosr,
		r.ownerStrategy,
	)
	if len(reconcileOpts) > 0 {
		rev.WithReconcileOptions(reconcileOpts...)
	}
	return rev, nil
}

func (r *COSRReconciler) listGroupMembers(ctx context.Context, group string) ([]orbv1alpha1.ClusterObjectSetRevision, error) {
	var list orbv1alpha1.ClusterObjectSetRevisionList
	if err := r.client.List(ctx, &list, client.MatchingFields{groupIndex: group}); err != nil {
		return nil, fmt.Errorf("listing group members: %w", err)
	}
	slices.SortFunc(list.Items, func(a, b orbv1alpha1.ClusterObjectSetRevision) int {
		return cmp.Compare(a.Spec.Revision, b.Spec.Revision)
	})
	return list.Items, nil
}

func controllerOwnerName(cosr *orbv1alpha1.ClusterObjectSetRevision) string {
	ref := metav1.GetControllerOf(cosr)
	if ref == nil {
		return ""
	}
	return ref.Name
}

func filterByControllerOwner(members []orbv1alpha1.ClusterObjectSetRevision, ownerName string) []orbv1alpha1.ClusterObjectSetRevision {
	var result []orbv1alpha1.ClusterObjectSetRevision
	for _, m := range members {
		if controllerOwnerName(&m) == ownerName {
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

func setCondition(cosr *orbv1alpha1.ClusterObjectSetRevision, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&cosr.Status.Conditions, metav1.Condition{
		Type:               orbv1alpha1.ConditionTypeAvailable,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: cosr.Generation,
	})
}

func removeFinalizer(ctx context.Context, c client.Client, obj client.Object, finalizer string) error {
	if !controllerutil.ContainsFinalizer(obj, finalizer) {
		return nil
	}
	controllerutil.RemoveFinalizer(obj, finalizer)
	patch, err := json.Marshal(map[string]any{
		"metadata": map[string]any{
			"resourceVersion": obj.GetResourceVersion(),
			"finalizers":      obj.GetFinalizers(),
		},
	})
	if err != nil {
		return fmt.Errorf("marshalling finalizer patch: %w", err)
	}
	if err := c.Patch(ctx, obj, client.RawPatch(types.MergePatchType, patch)); err != nil {
		return fmt.Errorf("removing finalizer: %w", err)
	}
	return nil
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
