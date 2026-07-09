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
	cosrac "github.com/joelanford/orb-operator/applyconfigurations/api/v1alpha1"
	"github.com/joelanford/orb-operator/internal/assertions"
)

const (
	cosrFieldOwner = "cosr-controller"
	managedBy      = "orb-operator"
	systemPrefix   = "orb.operatorframework.io"
	finalizerKey   = "orb.operatorframework.io/cosr-finalizer"
	groupIndex     = ".spec.group"
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
		For(&orbv1alpha1.ClusterObjectSetRevision{}).
		WatchesRawSource(
			r.accessManager.Source(
				handler.EnqueueRequestForOwner(r.scheme, mgr.GetRESTMapper(), &orbv1alpha1.ClusterObjectSetRevision{}, handler.OnlyControllerOwner()),
			),
		).
		WithOptions(controller.Options{MaxConcurrentReconciles: 4}).
		Complete(r)
}

func (r *COSRReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	existing := &orbv1alpha1.ClusterObjectSetRevision{}
	if err := r.client.Get(ctx, req.NamespacedName, existing); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return r.reconcile(ctx, log, existing)
}

func (r *COSRReconciler) reconcile(ctx context.Context, log logr.Logger, cosr *orbv1alpha1.ClusterObjectSetRevision) (ctrl.Result, error) {
	if !cosr.DeletionTimestamp.IsZero() || cosr.Spec.LifecycleState == orbv1alpha1.LifecycleStateArchived {
		return r.teardownAndRelease(ctx, log, cosr)
	}

	groupMembers, err := r.listGroupMembers(ctx, cosr.Spec.Group)
	if err != nil {
		return ctrl.Result{}, err
	}

	ownerKey := controllerOwnerKeyOf(cosr)
	members := filterByControllerOwner(groupMembers, ownerKey)
	chain := buildChain(members)

	if applied, err := r.ensureFinalizer(ctx, cosr); applied || err != nil {
		return ctrl.Result{}, err
	}

	siblings := chain.siblingsOf(cosr)
	return ctrl.Result{}, r.reconcileActive(ctx, log, cosr, siblings)
}

type revisionChain struct {
	latestActive *orbv1alpha1.ClusterObjectSetRevision
	predecessors []*orbv1alpha1.ClusterObjectSetRevision
	archived     []*orbv1alpha1.ClusterObjectSetRevision
	deleted      []*orbv1alpha1.ClusterObjectSetRevision
}

func buildChain(members []orbv1alpha1.ClusterObjectSetRevision) revisionChain {
	slices.SortFunc(members, func(a, b orbv1alpha1.ClusterObjectSetRevision) int {
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

func (ch revisionChain) siblingsOf(cosr *orbv1alpha1.ClusterObjectSetRevision) []*orbv1alpha1.ClusterObjectSetRevision {
	var siblings []*orbv1alpha1.ClusterObjectSetRevision
	if ch.latestActive != nil && ch.latestActive.Name != cosr.Name {
		siblings = append(siblings, ch.latestActive)
	}
	for _, p := range ch.predecessors {
		if p.Name != cosr.Name {
			siblings = append(siblings, p)
		}
	}
	return siblings
}

func (r *COSRReconciler) ensureFinalizer(ctx context.Context, cosr *orbv1alpha1.ClusterObjectSetRevision) (bool, error) {
	applied, err := applyCOSR(ctx, r.client, cosr, cosrFieldOwner,
		func(cosr *orbv1alpha1.ClusterObjectSetRevision) bool {
			return !controllerutil.ContainsFinalizer(cosr, finalizerKey)
		},
		func(ac *cosrac.ClusterObjectSetRevisionApplyConfiguration) {
			ac.WithFinalizers(finalizerKey)
		},
	)
	if err != nil {
		return false, fmt.Errorf("adding finalizer to %s: %w", cosr.Name, err)
	}
	return applied, nil
}

func (r *COSRReconciler) reconcileActive(ctx context.Context, log logr.Logger, cosr *orbv1alpha1.ClusterObjectSetRevision, siblings []*orbv1alpha1.ClusterObjectSetRevision) error {
	log.Info("reconciling active COSR")

	existing := cosr.DeepCopy()
	reconcileErr := r.doReconcileActive(ctx, cosr, siblings)

	if !equality.Semantic.DeepEqual(existing.Status, cosr.Status) {
		if err := r.client.Status().Update(ctx, cosr); err != nil {
			return errors.Join(reconcileErr, fmt.Errorf("updating status for %s: %w", cosr.Name, err))
		}
	}
	return reconcileErr
}

func (r *COSRReconciler) doReconcileActive(ctx context.Context, cosr *orbv1alpha1.ClusterObjectSetRevision, siblings []*orbv1alpha1.ClusterObjectSetRevision) error {
	engine, err := r.engineForCOSR(ctx, cosr)
	if err != nil {
		setInternalErrorStatus(cosr, fmt.Sprintf("engine setup: %v", err))
		return err
	}

	rev, err := r.buildRevisionWithSiblings(cosr, siblings)
	if err != nil {
		setInternalErrorStatus(cosr, fmt.Sprintf("building revision: %v", err))
		return fmt.Errorf("building revision: %w", err)
	}
	result, err := engine.Reconcile(ctx, rev, types.WithAggregatePhaseReconcileErrors())
	cosr.Status.ObservedPhases = observedPhasesFromReconcileResult(cosr.Spec.Phases, result)
	if err != nil {
		setCondition(cosr, metav1.ConditionUnknown, orbv1alpha1.ReasonReconcileError, fmt.Sprintf("reconcile failed: %v", err))
		return fmt.Errorf("reconciling: %w", err)
	}

	if verr := result.GetValidationError(); verr != nil {
		setCondition(cosr, metav1.ConditionFalse, orbv1alpha1.ReasonInvalidRevision, verr.Error())
		return nil
	}

	// HasProgressed implies IsComplete (progressed objects pass their probes),
	// so check HasProgressed first to distinguish "all objects adopted by a
	// sibling" from "all objects healthy under this revision."
	switch {
	case result.HasProgressed():
		setCondition(cosr, metav1.ConditionFalse, orbv1alpha1.ReasonSuperseded, "all objects adopted by a newer revision")
	case result.IsComplete():
		if cosr.Status.CompletedAt == nil {
			now := metav1.Now()
			cosr.Status.CompletedAt = &now
		}
		setCondition(cosr, metav1.ConditionTrue, orbv1alpha1.ReasonAvailable, "all phases complete")
	default:
		setCondition(cosr, metav1.ConditionFalse, orbv1alpha1.ReasonUnavailable, "phases not yet complete")
	}
	return nil
}

func (r *COSRReconciler) teardownAndRelease(ctx context.Context, log logr.Logger, cosr *orbv1alpha1.ClusterObjectSetRevision) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(cosr, finalizerKey) {
		return ctrl.Result{}, nil
	}

	// The VAP "cosr-orphan-finalizer-ordering" guarantees the "orphan" finalizer
	// cannot be removed while our finalizer is still present. When the "orphan"
	// finalizer is set, skip teardown but still release the finalizer so the
	// deletion can proceed.
	if !cosr.DeletionTimestamp.IsZero() && controllerutil.ContainsFinalizer(cosr, "orphan") {
		log.Info("orphan finalizer present, skipping teardown")
		return ctrl.Result{}, r.releaseCOSR(ctx, cosr)
	}

	existing := cosr.DeepCopy()
	requeue, reconcileErr := r.doTeardownCOSR(ctx, cosr)

	if !equality.Semantic.DeepEqual(existing.Status, cosr.Status) {
		if err := r.client.Status().Update(ctx, cosr); err != nil {
			return ctrl.Result{}, errors.Join(reconcileErr, fmt.Errorf("updating status for %s: %w", cosr.Name, err))
		}
	}
	if reconcileErr != nil {
		return ctrl.Result{}, reconcileErr
	}
	if requeue {
		return ctrl.Result{Requeue: true}, nil
	}

	if err := r.releaseCOSR(ctx, cosr); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *COSRReconciler) doTeardownCOSR(ctx context.Context, cosr *orbv1alpha1.ClusterObjectSetRevision) (bool, error) {
	engine, err := r.engineForCOSR(ctx, cosr)
	if err != nil {
		setInternalErrorStatus(cosr, fmt.Sprintf("engine setup: %v", err))
		return false, fmt.Errorf("engine setup: %w", err)
	}

	rev, err := r.buildRevision(cosr)
	if err != nil {
		setInternalErrorStatus(cosr, fmt.Sprintf("building revision: %v", err))
		return false, fmt.Errorf("building revision: %w", err)
	}

	result, teardownErr := engine.Teardown(ctx, rev, types.WithAggregatePhaseTeardownErrors())
	setTeardownStatus(cosr, result, teardownErr)

	if teardownErr != nil {
		return false, fmt.Errorf("teardown: %w", teardownErr)
	}
	if !result.IsComplete() {
		return true, nil
	}
	return false, nil
}

func setTeardownStatus(cosr *orbv1alpha1.ClusterObjectSetRevision, result machinery.RevisionTeardownResult, teardownErr error) {
	cosr.Status.ObservedPhases = observedPhasesFromTeardownResult(cosr.Spec.Phases, result)
	switch {
	case teardownErr != nil:
		setCondition(cosr, metav1.ConditionUnknown, orbv1alpha1.ReasonTeardownError,
			fmt.Sprintf("teardown failed: %v", teardownErr))
	case result != nil && !result.IsComplete():
		setCondition(cosr, metav1.ConditionFalse, orbv1alpha1.ReasonArchived, "teardown in progress")
	default:
		setCondition(cosr, metav1.ConditionFalse, orbv1alpha1.ReasonArchived, "teardown complete")
	}
}

func (r *COSRReconciler) releaseCOSR(ctx context.Context, cosr *orbv1alpha1.ClusterObjectSetRevision) error {
	if err := r.accessManager.FreeWithUser(ctx, cosr, cosr); err != nil {
		return fmt.Errorf("freeing access manager: %w", err)
	}
	if err := removeFinalizer(ctx, r.client, cosr, finalizerKey); err != nil {
		return fmt.Errorf("removing finalizer: %w", err)
	}
	// Wait for the informer cache to reflect the finalizer removal (or
	// deletion) before returning. controller-runtime serializes reconciles
	// per key, so blocking here ensures the next queued reconcile reads the
	// updated state and exits early at the ContainsFinalizer check instead
	// of re-acquiring the cache for a doomed COSR.
	if err := waitForFinalizerRemoval(ctx, r.client, client.ObjectKeyFromObject(cosr)); err != nil {
		return fmt.Errorf("waiting for cache to sync finalizer removal: %w", err)
	}
	return nil
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
		FieldOwner:       "cosr-group/" + cosr.Spec.Group,
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
	return r.buildRevisionWithSiblings(cosr, nil)
}

func (r *COSRReconciler) buildRevisionWithSiblings(
	cosr *orbv1alpha1.ClusterObjectSetRevision,
	siblings []*orbv1alpha1.ClusterObjectSetRevision,
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

	if len(siblings) > 0 {
		siblingObjs := make([]client.Object, 0, len(siblings))
		for _, s := range siblings {
			siblingObjs = append(siblingObjs, s)
		}
		reconcileOpts = append(reconcileOpts, boxcutter.WithSiblingOwners(siblingObjs))
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

type controllerOwnerKey struct {
	Kind string
	Name string
}

func controllerOwnerKeyOf(cosr *orbv1alpha1.ClusterObjectSetRevision) controllerOwnerKey {
	ref := metav1.GetControllerOf(cosr)
	if ref == nil {
		return controllerOwnerKey{}
	}
	return controllerOwnerKey{Kind: ref.Kind, Name: ref.Name}
}

func filterByControllerOwner(members []orbv1alpha1.ClusterObjectSetRevision, key controllerOwnerKey) []orbv1alpha1.ClusterObjectSetRevision {
	var result []orbv1alpha1.ClusterObjectSetRevision
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

func setInternalErrorStatus(cosr *orbv1alpha1.ClusterObjectSetRevision, message string) {
	cosr.Status.ObservedPhases = nil
	setCondition(cosr, metav1.ConditionUnknown, orbv1alpha1.ReasonInternalError, message)
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

func removeFinalizer(ctx context.Context, c client.Client, cosr *orbv1alpha1.ClusterObjectSetRevision, finalizer string) error {
	if !controllerutil.ContainsFinalizer(cosr, finalizer) {
		return nil
	}
	patch := client.MergeFromWithOptions(cosr.DeepCopy(), client.MergeFromWithOptimisticLock{})
	controllerutil.RemoveFinalizer(cosr, finalizer)
	clearFinalizerFieldOwnership(cosr.ManagedFields, cosrFieldOwner, finalizer)
	return c.Patch(ctx, cosr, patch)
}

func waitForFinalizerRemoval(ctx context.Context, c client.Client, key client.ObjectKey) error {
	return wait.PollUntilContextTimeout(ctx, 50*time.Millisecond, 5*time.Second, true, func(ctx context.Context) (bool, error) {
		var cosr orbv1alpha1.ClusterObjectSetRevision
		if err := c.Get(ctx, key, &cosr); err != nil {
			return apierrors.IsNotFound(err), client.IgnoreNotFound(err)
		}
		return !controllerutil.ContainsFinalizer(&cosr, finalizerKey), nil
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
