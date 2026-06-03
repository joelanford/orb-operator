package controller

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"

	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

const defaultRevisionHistoryLimit int32 = 5

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
		Complete(r)
}

func (r *COSReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	existing := &orbv1alpha1.ClusterObjectSet{}
	if err := r.client.Get(ctx, req.NamespacedName, existing); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	reconciledCOS := existing.DeepCopy()
	res, reconcileErr := r.reconcile(ctx, reconciledCOS)

	if !equality.Semantic.DeepEqual(existing.Status, reconciledCOS.Status) {
		if err := r.client.Status().Update(ctx, reconciledCOS); err != nil {
			return res, fmt.Errorf("updating status: %w", err)
		}
	}

	return res, reconcileErr
}

func (r *COSReconciler) reconcile(ctx context.Context, cos *orbv1alpha1.ClusterObjectSet) (ctrl.Result, error) {
	var cosrList orbv1alpha1.ClusterObjectSetRevisionList
	if err := r.client.List(ctx, &cosrList, client.MatchingFields{groupIndex: cos.Name}); err != nil {
		return ctrl.Result{}, fmt.Errorf("listing COSRs: %w", err)
	}

	slices.SortFunc(cosrList.Items, func(a, b orbv1alpha1.ClusterObjectSetRevision) int {
		return cmp.Compare(a.Spec.Revision, b.Spec.Revision)
	})

	ownedCOSRs := filterOwnedCOSRs(cosrList.Items, cos)

	statusBefore := cos.Status.DeepCopy()
	setStatus(cos, ownedCOSRs)
	if !equality.Semantic.DeepEqual(*statusBefore, cos.Status) {
		return ctrl.Result{}, nil
	}

	var latestCOSR *orbv1alpha1.ClusterObjectSetRevision
	if len(ownedCOSRs) > 0 {
		latestCOSR = &ownedCOSRs[len(ownedCOSRs)-1]
	}

	if latestCOSR == nil || !templateEqual(cos, latestCOSR) {
		nextRevision := uint32(1)
		if latestCOSR != nil {
			nextRevision = latestCOSR.Spec.Revision + 1
		}

		cosr := buildCOSRFromTemplate(cos, nextRevision)
		if err := controllerutil.SetControllerReference(cos, cosr, r.scheme); err != nil {
			return ctrl.Result{}, fmt.Errorf("setting controller reference: %w", err)
		}
		if err := r.client.Create(ctx, cosr); err != nil && !apierrors.IsAlreadyExists(err) {
			return ctrl.Result{}, fmt.Errorf("creating COSR: %w", err)
		}
		return ctrl.Result{}, nil
	}

	r.pruneArchivedCOSRs(ctx, cos, ownedCOSRs)

	return ctrl.Result{}, nil
}

func filterOwnedCOSRs(cosrs []orbv1alpha1.ClusterObjectSetRevision, cos *orbv1alpha1.ClusterObjectSet) []orbv1alpha1.ClusterObjectSetRevision {
	var owned []orbv1alpha1.ClusterObjectSetRevision
	for _, cosr := range cosrs {
		for _, ref := range cosr.OwnerReferences {
			if ref.Kind == "ClusterObjectSet" && ref.Name == cos.Name && ref.Controller != nil && *ref.Controller {
				owned = append(owned, cosr)
				break
			}
		}
	}
	return owned
}

func templateEqual(cos *orbv1alpha1.ClusterObjectSet, cosr *orbv1alpha1.ClusterObjectSetRevision) bool {
	if !equality.Semantic.DeepEqual(cos.Spec.Template.Spec, cosr.Spec.ClusterObjectSetTemplateSpec) {
		return false
	}
	if !maps.Equal(cos.Spec.Template.Metadata.Labels, cosr.Labels) {
		return false
	}
	if !maps.Equal(cos.Spec.Template.Metadata.Annotations, cosr.Annotations) {
		return false
	}
	return true
}

func buildCOSRFromTemplate(cos *orbv1alpha1.ClusterObjectSet, revision uint32) *orbv1alpha1.ClusterObjectSetRevision {
	src := cos.Spec.Template.Spec
	tmplSpec := orbv1alpha1.ClusterObjectSetTemplateSpec{}
	src.DeepCopyInto(&tmplSpec)
	return &orbv1alpha1.ClusterObjectSetRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-%d", cos.Name, revision),
			Labels:      maps.Clone(cos.Spec.Template.Metadata.Labels),
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
	limit := defaultRevisionHistoryLimit
	if cos.Spec.RevisionHistoryLimit != nil {
		limit = *cos.Spec.RevisionHistoryLimit
	}

	var archived []orbv1alpha1.ClusterObjectSetRevision
	for _, cosr := range cosrs {
		if cosr.Spec.LifecycleState == orbv1alpha1.LifecycleStateArchived {
			archived = append(archived, cosr)
		}
	}

	slices.SortFunc(archived, func(a, b orbv1alpha1.ClusterObjectSetRevision) int {
		return cmp.Compare(a.Spec.Revision, b.Spec.Revision)
	})

	excess := len(archived) - int(limit)
	for i := range excess {
		_ = r.client.Delete(ctx, &archived[i])
	}
}

func setStatus(cos *orbv1alpha1.ClusterObjectSet, cosrs []orbv1alpha1.ClusterObjectSetRevision) {
	var latest *orbv1alpha1.ClusterObjectSetRevision
	var allActive []orbv1alpha1.ClusterObjectSetRevisionStatusSummary
	for i := range cosrs {
		if cosrs[i].Spec.LifecycleState != orbv1alpha1.LifecycleStateArchived {
			allActive = append(allActive, orbv1alpha1.ClusterObjectSetRevisionStatusSummary{
				Name:       cosrs[i].Name,
				Conditions: cosrs[i].Status.Conditions,
			})
		}
		if latest == nil || cosrs[i].Spec.Revision > latest.Spec.Revision {
			latest = &cosrs[i]
		}
	}

	condition := metav1.Condition{
		Type:               orbv1alpha1.ConditionTypeAvailable,
		ObservedGeneration: cos.Generation,
	}

	switch {
	case latest != nil && isCOSRAvailable(latest):
		// Latest revision is available: only report it — superseded revisions are being cleaned up.
		// This prevents a second status write when they archive, which would race with spec updates.
		cos.Status.ActiveRevisions = []orbv1alpha1.ClusterObjectSetRevisionStatusSummary{{
			Name:       latest.Name,
			Conditions: latest.Status.Conditions,
		}}
		condition.Status = metav1.ConditionTrue
		condition.Reason = orbv1alpha1.ReasonAvailable
		condition.Message = "active revision is available"
	case len(allActive) > 1:
		cos.Status.ActiveRevisions = allActive
		condition.Status = metav1.ConditionUnknown
		condition.Reason = orbv1alpha1.ReasonProgressing
		condition.Message = "revision transition in progress"
	default:
		cos.Status.ActiveRevisions = allActive
		condition.Status = metav1.ConditionFalse
		condition.Reason = orbv1alpha1.ReasonUnavailable
		condition.Message = "active revision is not yet available"
	}

	meta.SetStatusCondition(&cos.Status.Conditions, condition)
}

func isCOSRAvailable(cosr *orbv1alpha1.ClusterObjectSetRevision) bool {
	return meta.IsStatusConditionTrue(cosr.Status.Conditions, orbv1alpha1.ConditionTypeAvailable)
}
