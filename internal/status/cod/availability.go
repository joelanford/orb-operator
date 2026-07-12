package cod

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func EvaluateAvailability(generation int64, active []orbv1alpha1.ClusterObjectSetStatusSummary) metav1.Condition {
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

func IsAvailable(cos *orbv1alpha1.ClusterObjectSet) bool {
	return meta.IsStatusConditionTrue(cos.Status.Conditions, orbv1alpha1.ConditionTypeAvailable)
}

func ActiveRevisionSummaries(ownedCOSs []orbv1alpha1.ClusterObjectSet) []orbv1alpha1.ClusterObjectSetStatusSummary {
	var active []orbv1alpha1.ClusterObjectSetStatusSummary
	for i := range ownedCOSs {
		if ownedCOSs[i].Spec.LifecycleState == orbv1alpha1.LifecycleStateArchived {
			continue
		}
		active = append(active, orbv1alpha1.ClusterObjectSetStatusSummary{
			Name:       ownedCOSs[i].Name,
			Conditions: ownedCOSs[i].Status.Conditions,
		})
	}
	return active
}
