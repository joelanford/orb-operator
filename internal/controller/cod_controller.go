package controller

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
	cosac "github.com/joelanford/orb-operator/applyconfigurations/api/v1alpha1"
	"github.com/joelanford/orb-operator/internal/cosutil"
	codstatus "github.com/joelanford/orb-operator/internal/status/cod"
	"github.com/joelanford/orb-operator/internal/template"
)

const (
	codFieldOwner                     = "cod-controller"
	defaultRevisionHistoryLimit int32 = 5
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

	currentHash, err := template.Hash(cod.Spec.Template)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("computing template hash: %w", err)
	}
	if latestOwned == nil || latestOwned.Labels[template.LabelTemplateHash] != currentHash {
		nextRevision := r.nextRevision(cosList.Items)

		cos, err := template.BuildCOS(cod, nextRevision, currentHash)
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

	desired, err := template.BuildCOS(cod, latestOwned.Spec.Revision, currentHash)
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
	_, err := cosutil.Apply(ctx, r.client, cos, codFieldOwner,
		func(cos *orbv1alpha1.ClusterObjectSet) bool {
			return true
		},
		func(ac *cosac.ClusterObjectSetApplyConfiguration) {
			template.SetControllerReference(cod, ac)
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
	if !codstatus.IsAvailable(latest) {
		return nil
	}

	for i := range ownedCOSs[:len(ownedCOSs)-1] {
		cos := &ownedCOSs[i]
		if _, err := cosutil.Apply(ctx, r.client, cos, codFieldOwner,
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
	active := codstatus.ActiveRevisionSummaries(ownedCOSs)
	cod.Status.ActiveRevisions = active

	var activeCOSs []orbv1alpha1.ClusterObjectSet
	for i := range ownedCOSs {
		if ownedCOSs[i].Spec.LifecycleState != orbv1alpha1.LifecycleStateArchived {
			activeCOSs = append(activeCOSs, ownedCOSs[i])
		}
	}

	meta.SetStatusCondition(&cod.Status.Conditions, codstatus.EvaluateAvailability(cod.Generation, active))

	var latestCOS *orbv1alpha1.ClusterObjectSet
	if len(activeCOSs) > 0 {
		latestCOS = &activeCOSs[len(activeCOSs)-1]
	}
	progressingCondition, requeueAfter := codstatus.EvaluateDeadline(cod, latestCOS, time.Now(), r.deadlineUnit)
	meta.SetStatusCondition(&cod.Status.Conditions, progressingCondition)

	return requeueAfter
}

