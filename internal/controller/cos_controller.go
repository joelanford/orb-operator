package controller

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"

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
	cosrac "github.com/joelanford/orb-operator/applyconfigurations/api/v1alpha1"
)

const (
	cosFieldOwner                     = "cos-controller"
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
		WithOptions(controller.Options{MaxConcurrentReconciles: 4}).
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

	currentHash, err := templateHash(cos.Spec.Template)
	if err != nil {
		return fmt.Errorf("computing template hash: %w", err)
	}
	if latestOwned == nil || latestOwned.Labels[labelTemplateHash] != currentHash {
		nextRevision := r.nextRevision(cosrList.Items)

		cosr, err := buildCOSRFromTemplate(cos, nextRevision, currentHash)
		if err != nil {
			return fmt.Errorf("building COSR from template: %w", err)
		}

		cosrUnstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cosr)
		if err != nil {
			return fmt.Errorf("converting COSR to unstructured: %w", err)
		}

		u := &unstructured.Unstructured{Object: cosrUnstructuredObj}
		if err := r.client.Create(ctx, u); err != nil {
			return fmt.Errorf("creating COSR: %w", err)
		}
		if err := r.client.Apply(ctx, cosr, client.FieldOwner(cosFieldOwner), client.ForceOwnership); err != nil {
			return fmt.Errorf("claiming field ownership for new COSR: %w", err)
		}
		return nil
	}

	desired, err := buildCOSRFromTemplate(cos, latestOwned.Spec.Revision, currentHash)
	if err != nil {
		return fmt.Errorf("building desired COSR apply config: %w", err)
	}
	existing, err := cosrac.ExtractClusterObjectSetRevision(latestOwned, cosFieldOwner)
	if err != nil {
		return fmt.Errorf("extracting COSR apply config: %w", err)
	}
	if !equality.Semantic.DeepEqual(existing, desired) {
		ctrl.LoggerFrom(ctx).Info("fixing up COSR field owners")
		if err := r.client.Apply(ctx, desired, client.FieldOwner(cosFieldOwner), client.ForceOwnership); err != nil {
			return fmt.Errorf("applying COSR: %w", err)
		}
	}

	if err := r.archiveOlderRevisions(ctx, cos, ownedCOSRs); err != nil {
		return err
	}

	if err := r.pruneArchivedCOSRs(ctx, cos, ownedCOSRs); err != nil {
		return err
	}

	return nil
}

func (r *COSReconciler) adoptAndFilterOwned(ctx context.Context, cos *orbv1alpha1.ClusterObjectSet, cosrs []orbv1alpha1.ClusterObjectSetRevision) ([]orbv1alpha1.ClusterObjectSetRevision, error) {
	var owned []orbv1alpha1.ClusterObjectSetRevision
	for i := range cosrs {
		cosr := &cosrs[i]
		ref := metav1.GetControllerOf(cosr)

		if ref != nil {
			if ref.UID == cos.UID {
				owned = append(owned, *cosr)
			}
			continue
		}

		if err := r.adoptCOSR(ctx, cos, cosr); err != nil {
			return nil, err
		}
		owned = append(owned, *cosr)
	}
	return owned, nil
}

func (r *COSReconciler) adoptCOSR(ctx context.Context, cos *orbv1alpha1.ClusterObjectSet, cosr *orbv1alpha1.ClusterObjectSetRevision) error {
	_, err := applyCOSR(ctx, r.client, cosr, cosFieldOwner,
		func(cosr *orbv1alpha1.ClusterObjectSetRevision) bool {
			return true
		},
		func(ac *cosrac.ClusterObjectSetRevisionApplyConfiguration) {
			setCOSControllerReference(cos, ac)
		},
	)
	if err != nil {
		return fmt.Errorf("adopting COSR %s: %w", cosr.Name, err)
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
		if _, err := applyCOSR(ctx, r.client, cosr, cosFieldOwner,
			func(cosr *orbv1alpha1.ClusterObjectSetRevision) bool {
				return cosr.Spec.LifecycleState != orbv1alpha1.LifecycleStateArchived
			},
			func(ac *cosrac.ClusterObjectSetRevisionApplyConfiguration) {
				ac.WithSpec(cosrac.ClusterObjectSetRevisionSpec().
					WithLifecycleState(orbv1alpha1.LifecycleStateArchived))
			},
		); err != nil {
			return fmt.Errorf("archiving COSR %s: %w", cosr.Name, err)
		}
	}
	return nil
}

func buildCOSRFromTemplate(cos *orbv1alpha1.ClusterObjectSet, revision uint32, hash string) (*cosrac.ClusterObjectSetRevisionApplyConfiguration, error) {
	labels := maps.Clone(cos.Spec.Template.Metadata.Labels)
	if labels == nil {
		labels = map[string]string{}
	}
	labels[labelTemplateHash] = hash

	tmplSpecJSON, err := json.Marshal(cos.Spec.Template.Spec)
	if err != nil {
		return nil, err
	}

	var cosrSpec cosrac.ClusterObjectSetRevisionSpecApplyConfiguration
	if err := json.Unmarshal(tmplSpecJSON, &cosrSpec); err != nil {
		return nil, err
	}

	cosrSpec.WithGroup(cos.Name).
		WithRevision(revision).
		WithLifecycleState(orbv1alpha1.LifecycleStateActive)

	name := fmt.Sprintf("%s-%d", cos.Name, revision)
	cosr := cosrac.ClusterObjectSetRevision(name).
		WithLabels(labels).
		WithAnnotations(maps.Clone(cos.Spec.Template.Metadata.Annotations)).
		WithSpec(&cosrSpec)

	setCOSControllerReference(cos, cosr)
	return cosr, nil
}

func setCOSControllerReference(cos *orbv1alpha1.ClusterObjectSet, cosr *cosrac.ClusterObjectSetRevisionApplyConfiguration) {
	gvk := orbv1alpha1.GroupVersion.WithKind("ClusterObjectSet")
	cosr.WithOwnerReferences(metav1ac.OwnerReference().
		WithAPIVersion(gvk.GroupVersion().String()).
		WithKind(gvk.Kind).
		WithName(cos.Name).
		WithUID(cos.UID).
		WithController(true).
		WithBlockOwnerDeletion(true),
	)
}

func (r *COSReconciler) pruneArchivedCOSRs(ctx context.Context, cos *orbv1alpha1.ClusterObjectSet, cosrs []orbv1alpha1.ClusterObjectSetRevision) error {
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
			return fmt.Errorf("pruning archived COSR %s: %w", prunable[i].Name, err)
		}
	}
	return nil
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
