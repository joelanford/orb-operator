package controller

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

const (
	defaultRevisionHistoryLimit int32 = 5

	labelTemplateHash = "orb.operatorframework.io/template-hash"
)

type COSReconciler struct {
	client client.Client
	scheme *runtime.Scheme
}

func NewCOSReconciler(c client.Client, scheme *runtime.Scheme) *COSReconciler {
	return &COSReconciler{client: c, scheme: scheme}
}

func (r *COSReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&orbv1alpha1.ClusterObjectSet{}).
		Owns(&orbv1alpha1.ClusterObjectSetRevision{}).
		Watches(&orbv1alpha1.ClusterObjectSetRevision{},
			handler.EnqueueRequestsFromMapFunc(mapCOSRGroupToCOS),
		).
		Complete(r)
}

func mapCOSRGroupToCOS(_ context.Context, obj client.Object) []reconcile.Request {
	cosr := obj.(*orbv1alpha1.ClusterObjectSetRevision)
	return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: cosr.Spec.Group}}}
}

func (r *COSReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	existing := &orbv1alpha1.ClusterObjectSet{}
	if err := r.client.Get(ctx, req.NamespacedName, existing); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !existing.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	reconciledCOS := existing.DeepCopy()
	reconcileErr := r.reconcile(ctx, reconciledCOS)

	if !equality.Semantic.DeepEqual(existing.Status, reconciledCOS.Status) {
		if err := r.client.Status().Update(ctx, reconciledCOS); err != nil {
			reconcileErr = errors.Join(reconcileErr, fmt.Errorf("updating status: %w", err))
		}
	}

	if reconcileErr != nil {
		log.Error(reconcileErr, "reconcile error")
	}
	return ctrl.Result{}, reconcileErr
}

func (r *COSReconciler) reconcile(ctx context.Context, cos *orbv1alpha1.ClusterObjectSet) error {
	var cosrList orbv1alpha1.ClusterObjectSetRevisionList
	if err := r.client.List(ctx, &cosrList, client.MatchingFields{groupIndex: cos.Name}); err != nil {
		return fmt.Errorf("listing COSRs: %w", err)
	}

	slices.SortFunc(cosrList.Items, func(a, b orbv1alpha1.ClusterObjectSetRevision) int {
		return cmp.Compare(a.Spec.Revision, b.Spec.Revision)
	})

	ownedCOSRs, err := r.adoptAndFilterOwned(ctx, cos, cosrList.Items)
	if err != nil {
		return err
	}

	r.setStatus(cos, ownedCOSRs)

	var latestOwned *orbv1alpha1.ClusterObjectSetRevision
	if len(ownedCOSRs) > 0 {
		latestOwned = &ownedCOSRs[len(ownedCOSRs)-1]
	}

	currentHash := templateHash(cos.Spec.Template)
	if latestOwned == nil || latestOwned.Labels[labelTemplateHash] != currentHash {
		nextRevision := r.nextRevision(cosrList.Items)

		cosr := buildCOSRFromTemplate(cos, nextRevision, currentHash)
		if err := controllerutil.SetControllerReference(cos, cosr, r.scheme); err != nil {
			return fmt.Errorf("setting controller reference: %w", err)
		}
		if err := r.client.Create(ctx, cosr); err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("creating COSR: %w", err)
		}
		return nil
	}

	if err := r.archiveOlderRevisions(ctx, cos, ownedCOSRs); err != nil {
		return err
	}

	r.pruneArchivedCOSRs(ctx, cos, ownedCOSRs)

	return nil
}

func (r *COSReconciler) adoptAndFilterOwned(ctx context.Context, cos *orbv1alpha1.ClusterObjectSet, cosrs []orbv1alpha1.ClusterObjectSetRevision) ([]orbv1alpha1.ClusterObjectSetRevision, error) {
	var owned []orbv1alpha1.ClusterObjectSetRevision
	for i := range cosrs {
		cosr := &cosrs[i]
		ref := metav1.GetControllerOf(cosr)

		if ref == nil {
			if err := r.adoptCOSR(ctx, cos, cosr); err != nil {
				return nil, err
			}
			owned = append(owned, *cosr)
			continue
		}

		if ref.UID == cos.UID {
			owned = append(owned, *cosr)
		}
	}
	return owned, nil
}

func (r *COSReconciler) adoptCOSR(ctx context.Context, cos *orbv1alpha1.ClusterObjectSet, cosr *orbv1alpha1.ClusterObjectSetRevision) error {
	patch := client.MergeFromWithOptions(cosr.DeepCopy(), client.MergeFromWithOptimisticLock{})
	if err := controllerutil.SetControllerReference(cos, cosr, r.scheme); err != nil {
		return fmt.Errorf("adopting COSR %s: %w", cosr.Name, err)
	}
	if err := r.client.Patch(ctx, cosr, patch); err != nil {
		return fmt.Errorf("patching adopted COSR %s: %w", cosr.Name, err)
	}
	return nil
}

func (r *COSReconciler) nextRevision(allGroupCOSRs []orbv1alpha1.ClusterObjectSetRevision) uint32 {
	var maxRevision uint32
	for _, cosr := range allGroupCOSRs {
		if cosr.Spec.Revision > maxRevision {
			maxRevision = cosr.Spec.Revision
		}
	}
	return maxRevision + 1
}

func (r *COSReconciler) archiveOlderRevisions(ctx context.Context, _ *orbv1alpha1.ClusterObjectSet, ownedCOSRs []orbv1alpha1.ClusterObjectSetRevision) error {
	if len(ownedCOSRs) < 2 {
		return nil
	}

	latest := &ownedCOSRs[len(ownedCOSRs)-1]
	if !isCOSRAvailable(latest) {
		return nil
	}

	for i := range ownedCOSRs[:len(ownedCOSRs)-1] {
		cosr := &ownedCOSRs[i]
		if cosr.Spec.LifecycleState == orbv1alpha1.LifecycleStateArchived {
			continue
		}
		cosr.Spec.LifecycleState = orbv1alpha1.LifecycleStateArchived
		if err := r.client.Update(ctx, cosr); err != nil {
			return fmt.Errorf("archiving COSR %s: %w", cosr.Name, err)
		}
	}
	return nil
}

func buildCOSRFromTemplate(cos *orbv1alpha1.ClusterObjectSet, revision uint32, hash string) *orbv1alpha1.ClusterObjectSetRevision {
	src := cos.Spec.Template.Spec
	tmplSpec := orbv1alpha1.ClusterObjectSetTemplateSpec{}
	src.DeepCopyInto(&tmplSpec)

	labels := maps.Clone(cos.Spec.Template.Metadata.Labels)
	if labels == nil {
		labels = map[string]string{}
	}
	labels[labelTemplateHash] = hash

	return &orbv1alpha1.ClusterObjectSetRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-%d", cos.Name, revision),
			Labels:      labels,
			Annotations: maps.Clone(cos.Spec.Template.Metadata.Annotations),
		},
		Spec: orbv1alpha1.ClusterObjectSetRevisionSpec{
			Group:                        cos.Name,
			Revision:                     revision,
			LifecycleState:               orbv1alpha1.LifecycleStateActive,
			ClusterObjectSetTemplateSpec: tmplSpec,
		},
	}
}

func (r *COSReconciler) pruneArchivedCOSRs(ctx context.Context, cos *orbv1alpha1.ClusterObjectSet, cosrs []orbv1alpha1.ClusterObjectSetRevision) {
	log := ctrl.LoggerFrom(ctx)

	limit := defaultRevisionHistoryLimit
	if cos.Spec.RevisionHistoryLimit != nil {
		limit = *cos.Spec.RevisionHistoryLimit
	}

	var prunable []orbv1alpha1.ClusterObjectSetRevision
	for _, cosr := range cosrs {
		if cosr.Spec.LifecycleState == orbv1alpha1.LifecycleStateArchived {
			prunable = append(prunable, cosr)
		}
	}

	slices.SortFunc(prunable, func(a, b orbv1alpha1.ClusterObjectSetRevision) int {
		return cmp.Compare(a.Spec.Revision, b.Spec.Revision)
	})

	excess := len(prunable) - int(limit)
	for i := range excess {
		if err := r.client.Delete(ctx, &prunable[i]); err != nil {
			log.Error(err, "pruning archived COSR", "cosr", prunable[i].Name)
		}
	}
}

func (r *COSReconciler) setStatus(cos *orbv1alpha1.ClusterObjectSet, ownedCOSRs []orbv1alpha1.ClusterObjectSetRevision) {
	var active []orbv1alpha1.ClusterObjectSetRevisionStatusSummary
	for i := range ownedCOSRs {
		if ownedCOSRs[i].Spec.LifecycleState == orbv1alpha1.LifecycleStateArchived {
			continue
		}
		active = append(active, orbv1alpha1.ClusterObjectSetRevisionStatusSummary{
			Name:       ownedCOSRs[i].Name,
			Conditions: ownedCOSRs[i].Status.Conditions,
		})
	}

	cos.Status.ActiveRevisions = active

	condition := metav1.Condition{
		Type:               orbv1alpha1.ConditionTypeAvailable,
		ObservedGeneration: cos.Generation,
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

	meta.SetStatusCondition(&cos.Status.Conditions, condition)
}

func isCOSRAvailable(cosr *orbv1alpha1.ClusterObjectSetRevision) bool {
	return meta.IsStatusConditionTrue(cosr.Status.Conditions, orbv1alpha1.ConditionTypeAvailable)
}
