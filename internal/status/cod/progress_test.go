package cod

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func TestEvaluateDeadline(t *testing.T) {
	deadlineUnit := time.Millisecond
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	deadline := int32(500)
	cod := &orbv1alpha1.ClusterObjectDeployment{
		ObjectMeta: metav1.ObjectMeta{Generation: 1},
		Spec: orbv1alpha1.ClusterObjectDeploymentSpec{
			ProgressDeadlineMinutes: &deadline,
		},
	}
	codNoDeadline := &orbv1alpha1.ClusterObjectDeployment{
		ObjectMeta: metav1.ObjectMeta{Generation: 1},
	}

	t.Run("no active revisions", func(t *testing.T) {
		cond, result := EvaluateDeadline(cod, nil, now, deadlineUnit)
		assert.Equal(t, orbv1alpha1.ConditionTypeProgressing, cond.Type)
		assert.Equal(t, metav1.ConditionFalse, cond.Status)
		assert.Equal(t, orbv1alpha1.ReasonNoActiveRevisions, cond.Reason)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("latest COS completed", func(t *testing.T) {
		completedAt := metav1.NewTime(now.Add(-time.Hour))
		cos := &orbv1alpha1.ClusterObjectSet{
			Status: orbv1alpha1.ClusterObjectSetStatus{
				CompletedAt: &completedAt,
			},
		}
		cond, result := EvaluateDeadline(cod, cos, now, deadlineUnit)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)
		assert.Equal(t, orbv1alpha1.ReasonNewClusterObjectSetProgressed, cond.Reason)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("no deadline set", func(t *testing.T) {
		cos := &orbv1alpha1.ClusterObjectSet{
			ObjectMeta: metav1.ObjectMeta{
				CreationTimestamp: metav1.NewTime(now.Add(-time.Hour)),
			},
		}
		cond, result := EvaluateDeadline(codNoDeadline, cos, now, deadlineUnit)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)
		assert.Equal(t, orbv1alpha1.ReasonNewClusterObjectSetProgressing, cond.Reason)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("within deadline from creation", func(t *testing.T) {
		created := now.Add(-200 * time.Millisecond)
		cos := &orbv1alpha1.ClusterObjectSet{
			ObjectMeta: metav1.ObjectMeta{
				CreationTimestamp: metav1.NewTime(created),
			},
		}
		cond, result := EvaluateDeadline(cod, cos, now, deadlineUnit)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)
		assert.Equal(t, orbv1alpha1.ReasonNewClusterObjectSetProgressing, cond.Reason)
		assert.Equal(t, 300*time.Millisecond, result.RequeueAfter)
	})

	t.Run("deadline exceeded from creation", func(t *testing.T) {
		created := now.Add(-600 * time.Millisecond)
		cos := &orbv1alpha1.ClusterObjectSet{
			ObjectMeta: metav1.ObjectMeta{
				CreationTimestamp: metav1.NewTime(created),
			},
		}
		cond, result := EvaluateDeadline(cod, cos, now, deadlineUnit)
		assert.Equal(t, metav1.ConditionFalse, cond.Status)
		assert.Equal(t, orbv1alpha1.ReasonProgressDeadlineExceeded, cond.Reason)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("phase completedAt extends deadline", func(t *testing.T) {
		created := now.Add(-600 * time.Millisecond)
		phaseCompleted := now.Add(-100 * time.Millisecond)
		phaseCompletedAt := metav1.NewTime(phaseCompleted)
		cos := &orbv1alpha1.ClusterObjectSet{
			ObjectMeta: metav1.ObjectMeta{
				CreationTimestamp: metav1.NewTime(created),
			},
			Status: orbv1alpha1.ClusterObjectSetStatus{
				ObservedPhases: []orbv1alpha1.ObservedPhase{
					{Name: "phase-1", Status: orbv1alpha1.PhaseStatusAvailable, CompletedAt: &phaseCompletedAt},
					{Name: "phase-2", Status: orbv1alpha1.PhaseStatusReconciling},
				},
			},
		}
		cond, result := EvaluateDeadline(cod, cos, now, deadlineUnit)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)
		assert.Equal(t, orbv1alpha1.ReasonNewClusterObjectSetProgressing, cond.Reason)
		assert.Equal(t, 400*time.Millisecond, result.RequeueAfter)
	})

	t.Run("observedGeneration is set", func(t *testing.T) {
		cond, _ := EvaluateDeadline(cod, nil, now, deadlineUnit)
		assert.Equal(t, int64(1), cond.ObservedGeneration)
	})
}
