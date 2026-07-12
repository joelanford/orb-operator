package cod

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func EvaluateDeadline(cod *orbv1alpha1.ClusterObjectDeployment, latestCOS *orbv1alpha1.ClusterObjectSet, now time.Time, deadlineUnit time.Duration) (metav1.Condition, ctrl.Result) {
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

		deadline := time.Duration(*cod.Spec.ProgressDeadlineMinutes) * deadlineUnit
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
