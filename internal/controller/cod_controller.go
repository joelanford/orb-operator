package controller

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	metav1ac "k8s.io/client-go/applyconfigurations/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
	cosac "github.com/joelanford/orb-operator/applyconfigurations/api/v1alpha1"
)

const (
	codFieldOwner                     = "cod-controller"
	defaultRevisionHistoryLimit int32 = 5

	labelTemplateHash = "orb.operatorframework.io/template-hash"
)

type CODReconciler struct {
	client       client.Client
	scheme       *runtime.Scheme
	deadlineUnit time.Duration
}

func NewCODReconciler(c client.Client, scheme *runtime.Scheme, deadlineUnit time.Duration) *CODReconciler {
	return &CODReconciler{client: c, scheme: scheme, deadlineUnit: deadlineUnit}
}

func (r *CODReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&orbv1alpha1.ClusterObjectDeployment{}).
		Owns(&orbv1alpha1.ClusterObjectSet{}).
		Watches(&orbv1alpha1.ClusterObjectSet{},
			handler.EnqueueRequestsFromMapFunc(mapCOSGroupToCOD),
		).
		WithOptions(controller.Options{MaxConcurrentReconciles: 4}).
		Complete(r)
}

func mapCOSGroupToCOD(_ context.Context, obj client.Object) []reconcile.Request {
	cos := obj.(*orbv1alpha1.ClusterObjectSet)
	return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: cos.Spec.Group}}}
}

func (r *CODReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	existing := &orbv1alpha1.ClusterObjectDeployment{}
	if err := r.client.Get(ctx, req.NamespacedName, existing); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !existing.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	reconciledCOD := existing.DeepCopy()
	result, reconcileErr := r.reconcile(ctx, reconciledCOD)

	if !equality.Semantic.DeepEqual(existing.Status, reconciledCOD.Status) {
		if err := r.client.Status().Update(ctx, reconciledCOD); err != nil {
			reconcileErr = errors.Join(reconcileErr, fmt.Errorf("updating status: %w", err))
		}
	}

	if reconcileErr != nil {
		log.Error(reconcileErr, "reconcile error")
	}
	return result, reconcileErr
}

func (r *CODReconciler) reconcile(ctx context.Context, cod *orbv1alpha1.ClusterObjectDeployment) (ctrl.Result, error) {
	var cosList orbv1alpha1.ClusterObjectSetList
	if err := r.client.List(ctx, &cosList, client.MatchingFields{groupIndex: cod.Name}); err != nil {
		return ctrl.Result{}, fmt.Errorf("listing COSs: %w", err)
	}

	slices.SortFunc(cosList.Items, func(a, b orbv1alpha1.ClusterObjectSet) int {
		return cmp.Compare(a.Spec.Revision, b.Spec.Revision)
	})

	ownedCOSs, err := r.adoptAndFilterOwned(ctx, cod, cosList.Items)
	if err != nil {
		return ctrl.Result{}, err
	}

	requeueAfter := r.setStatus(cod, ownedCOSs)

	var latestOwned *orbv1alpha1.ClusterObjectSet
	if len(ownedCOSs) > 0 {
		latestOwned = &ownedCOSs[len(ownedCOSs)-1]
	}

	currentHash, err := templateHash(cod.Spec.Template)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("computing template hash: %w", err)
	}
	if latestOwned == nil || latestOwned.Labels[labelTemplateHash] != currentHash {
		nextRevision := r.nextRevision(cosList.Items)

		cos, err := buildCOSFromTemplate(cod, nextRevision, currentHash)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("building COS from template: %w", err)
		}

		cosUnstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cos)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("converting COS to unstructured: %w", err)
		}

		u := &unstructured.Unstructured{Object: cosUnstructuredObj}
		if err := r.client.Create(ctx, u); err != nil {
			return ctrl.Result{}, fmt.Errorf("creating COS: %w", err)
		}
		if err := r.client.Apply(ctx, cos, client.FieldOwner(codFieldOwner), client.ForceOwnership); err != nil {
			return ctrl.Result{}, fmt.Errorf("claiming field ownership for new COS: %w", err)
		}
		return ctrl.Result{}, nil
	}

	desired, err := buildCOSFromTemplate(cod, latestOwned.Spec.Revision, currentHash)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("building desired COS apply config: %w", err)
	}
	existing, err := cosac.ExtractClusterObjectSet(latestOwned, codFieldOwner)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("extracting COS apply config: %w", err)
	}
	if !equality.Semantic.DeepEqual(existing, desired) {
		ctrl.LoggerFrom(ctx).Info("fixing up COS field owners")
		if err := r.client.Apply(ctx, desired, client.FieldOwner(codFieldOwner), client.ForceOwnership); err != nil {
			return ctrl.Result{}, fmt.Errorf("applying COS: %w", err)
		}
	}

	if err := r.archiveOlderRevisions(ctx, cod, ownedCOSs); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.pruneArchivedCOSs(ctx, cod, ownedCOSs); err != nil {
		return ctrl.Result{}, err
	}

	return requeueAfter, nil
}

func (r *CODReconciler) adoptAndFilterOwned(ctx context.Context, cod *orbv1alpha1.ClusterObjectDeployment, coss []orbv1alpha1.ClusterObjectSet) ([]orbv1alpha1.ClusterObjectSet, error) {
	var owned []orbv1alpha1.ClusterObjectSet
	for i := range coss {
		cos := &coss[i]
		ref := metav1.GetControllerOf(cos)

		if ref != nil {
			if ref.UID == cod.UID {
				owned = append(owned, *cos)
			}
			continue
		}

		if err := r.adoptCOS(ctx, cod, cos); err != nil {
			return nil, err
		}
		owned = append(owned, *cos)
	}
	return owned, nil
}

func (r *CODReconciler) adoptCOS(ctx context.Context, cod *orbv1alpha1.ClusterObjectDeployment, cos *orbv1alpha1.ClusterObjectSet) error {
	_, err := applyCOS(ctx, r.client, cos, codFieldOwner,
		func(cos *orbv1alpha1.ClusterObjectSet) bool {
			return true
		},
		func(ac *cosac.ClusterObjectSetApplyConfiguration) {
			setCODControllerReference(cod, ac)
		},
	)
	if err != nil {
		return fmt.Errorf("adopting COS %s: %w", cos.Name, err)
	}
	return nil
}

func (r *CODReconciler) nextRevision(allGroupCOSs []orbv1alpha1.ClusterObjectSet) uint32 {
	var maxRevision uint32
	for _, cos := range allGroupCOSs {
		if cos.Spec.Revision > maxRevision {
			maxRevision = cos.Spec.Revision
		}
	}
	return maxRevision + 1
}

func (r *CODReconciler) archiveOlderRevisions(ctx context.Context, _ *orbv1alpha1.ClusterObjectDeployment, ownedCOSs []orbv1alpha1.ClusterObjectSet) error {
	if len(ownedCOSs) < 2 {
		return nil
	}

	latest := &ownedCOSs[len(ownedCOSs)-1]
	if !isCOSAvailable(latest) {
		return nil
	}

	for i := range ownedCOSs[:len(ownedCOSs)-1] {
		cos := &ownedCOSs[i]
		if _, err := applyCOS(ctx, r.client, cos, codFieldOwner,
			func(cos *orbv1alpha1.ClusterObjectSet) bool {
				return cos.Spec.LifecycleState != orbv1alpha1.LifecycleStateArchived
			},
			func(ac *cosac.ClusterObjectSetApplyConfiguration) {
				ac.WithSpec(cosac.ClusterObjectSetSpec().
					WithLifecycleState(orbv1alpha1.LifecycleStateArchived))
			},
		); err != nil {
			return fmt.Errorf("archiving COS %s: %w", cos.Name, err)
		}
	}
	return nil
}

func buildCOSFromTemplate(cod *orbv1alpha1.ClusterObjectDeployment, revision uint32, hash string) (*cosac.ClusterObjectSetApplyConfiguration, error) {
	labels := maps.Clone(cod.Spec.Template.Metadata.Labels)
	if labels == nil {
		labels = map[string]string{}
	}
	labels[labelTemplateHash] = hash

	tmplSpecJSON, err := json.Marshal(cod.Spec.Template.Spec)
	if err != nil {
		return nil, err
	}

	var cosSpec cosac.ClusterObjectSetSpecApplyConfiguration
	if err := json.Unmarshal(tmplSpecJSON, &cosSpec); err != nil {
		return nil, err
	}

	cosSpec.WithGroup(cod.Name).
		WithRevision(revision).
		WithLifecycleState(orbv1alpha1.LifecycleStateActive)

	name := fmt.Sprintf("%s-%d", cod.Name, revision)
	cos := cosac.ClusterObjectSet(name).
		WithLabels(labels).
		WithAnnotations(maps.Clone(cod.Spec.Template.Metadata.Annotations)).
		WithSpec(&cosSpec)

	setCODControllerReference(cod, cos)
	return cos, nil
}

func setCODControllerReference(cod *orbv1alpha1.ClusterObjectDeployment, cos *cosac.ClusterObjectSetApplyConfiguration) {
	gvk := orbv1alpha1.GroupVersion.WithKind("ClusterObjectDeployment")
	cos.WithOwnerReferences(metav1ac.OwnerReference().
		WithAPIVersion(gvk.GroupVersion().String()).
		WithKind(gvk.Kind).
		WithName(cod.Name).
		WithUID(cod.UID).
		WithController(true).
		WithBlockOwnerDeletion(true),
	)
}

func (r *CODReconciler) pruneArchivedCOSs(ctx context.Context, cod *orbv1alpha1.ClusterObjectDeployment, coss []orbv1alpha1.ClusterObjectSet) error {
	limit := defaultRevisionHistoryLimit
	if cod.Spec.RevisionHistoryLimit != nil {
		limit = *cod.Spec.RevisionHistoryLimit
	}

	var prunable []orbv1alpha1.ClusterObjectSet
	for _, cos := range coss {
		if cos.Spec.LifecycleState == orbv1alpha1.LifecycleStateArchived {
			prunable = append(prunable, cos)
		}
	}

	slices.SortFunc(prunable, func(a, b orbv1alpha1.ClusterObjectSet) int {
		return cmp.Compare(a.Spec.Revision, b.Spec.Revision)
	})

	excess := len(prunable) - int(limit)
	for i := range excess {
		if err := r.client.Delete(ctx, &prunable[i]); err != nil {
			return fmt.Errorf("pruning archived COS %s: %w", prunable[i].Name, err)
		}
	}
	return nil
}

func (r *CODReconciler) setStatus(cod *orbv1alpha1.ClusterObjectDeployment, ownedCOSs []orbv1alpha1.ClusterObjectSet) ctrl.Result {
	var activeCOSs []orbv1alpha1.ClusterObjectSet
	var active []orbv1alpha1.ClusterObjectSetStatusSummary
	for i := range ownedCOSs {
		if ownedCOSs[i].Spec.LifecycleState == orbv1alpha1.LifecycleStateArchived {
			continue
		}
		activeCOSs = append(activeCOSs, ownedCOSs[i])
		active = append(active, orbv1alpha1.ClusterObjectSetStatusSummary{
			Name:       ownedCOSs[i].Name,
			Conditions: ownedCOSs[i].Status.Conditions,
		})
	}

	cod.Status.ActiveRevisions = active

	meta.SetStatusCondition(&cod.Status.Conditions, evaluateAvailability(cod.Generation, active))

	var latestCOS *orbv1alpha1.ClusterObjectSet
	if len(activeCOSs) > 0 {
		latestCOS = &activeCOSs[len(activeCOSs)-1]
	}
	progressingCondition, requeueAfter := r.evaluateProgressDeadline(cod, latestCOS, time.Now())
	meta.SetStatusCondition(&cod.Status.Conditions, progressingCondition)

	return requeueAfter
}

func evaluateAvailability(generation int64, active []orbv1alpha1.ClusterObjectSetStatusSummary) metav1.Condition {
	condition := metav1.Condition{
		Type:               orbv1alpha1.ConditionTypeAvailable,
		ObservedGeneration: generation,
	}

	switch len(active) {
	case 0:
		condition.Status = metav1.ConditionFalse
		condition.Reason = orbv1alpha1.ReasonUnavailable
		condition.Message = "no active revisions"
	case 1:
		if meta.IsStatusConditionTrue(active[0].Conditions, orbv1alpha1.ConditionTypeAvailable) {
			condition.Status = metav1.ConditionTrue
			condition.Reason = orbv1alpha1.ReasonAvailable
			condition.Message = "active revision is available"
		} else {
			condition.Status = metav1.ConditionFalse
			condition.Reason = orbv1alpha1.ReasonUnavailable
			condition.Message = "active revision is not yet available"
		}
	default:
		condition.Status = metav1.ConditionUnknown
		condition.Reason = orbv1alpha1.ReasonProgressing
		condition.Message = "revision transition in progress"
	}

	return condition
}

func (r *CODReconciler) evaluateProgressDeadline(cod *orbv1alpha1.ClusterObjectDeployment, latestCOS *orbv1alpha1.ClusterObjectSet, now time.Time) (metav1.Condition, ctrl.Result) {
	condition := metav1.Condition{
		Type:               orbv1alpha1.ConditionTypeProgressing,
		ObservedGeneration: cod.Generation,
	}

	if latestCOS == nil {
		condition.Status = metav1.ConditionFalse
		condition.Reason = orbv1alpha1.ReasonNoActiveRevisions
		condition.Message = "no active revisions"
		return condition, ctrl.Result{}
	}

	if latestCOS.Status.CompletedAt != nil {
		condition.Status = metav1.ConditionTrue
		condition.Reason = orbv1alpha1.ReasonNewClusterObjectSetProgressed
		condition.Message = "latest revision has progressed"
		return condition, ctrl.Result{}
	}

	var requeueAfter time.Duration
	if cod.Spec.ProgressDeadlineMinutes != nil {
		lastMilestone := latestCOS.CreationTimestamp
		for _, phase := range latestCOS.Status.ObservedPhases {
			if phase.CompletedAt != nil && phase.CompletedAt.After(lastMilestone.Time) {
				lastMilestone = *phase.CompletedAt
			}
		}

		deadline := time.Duration(*cod.Spec.ProgressDeadlineMinutes) * r.deadlineUnit
		elapsed := now.Sub(lastMilestone.Time)

		if elapsed >= deadline {
			condition.Status = metav1.ConditionFalse
			condition.Reason = orbv1alpha1.ReasonProgressDeadlineExceeded
			condition.Message = "latest revision has not made progress within the deadline"
			return condition, ctrl.Result{}
		}
		requeueAfter = deadline - elapsed
	}

	condition.Status = metav1.ConditionTrue
	condition.Reason = orbv1alpha1.ReasonNewClusterObjectSetProgressing
	condition.Message = "latest revision is progressing"
	return condition, ctrl.Result{RequeueAfter: requeueAfter}
}

func isCOSAvailable(cos *orbv1alpha1.ClusterObjectSet) bool {
	return meta.IsStatusConditionTrue(cos.Status.Conditions, orbv1alpha1.ConditionTypeAvailable)
}
