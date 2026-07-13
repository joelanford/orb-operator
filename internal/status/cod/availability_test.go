package cod

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func TestEvaluateAvailability(t *testing.T) {
	t.Run("no active revisions", func(t *testing.T) {
		cond := EvaluateAvailability(1, nil)
		assert.Equal(t, orbv1alpha1.ConditionTypeAvailable, cond.Type)
		assert.Equal(t, metav1.ConditionFalse, cond.Status)
		assert.Equal(t, orbv1alpha1.ReasonUnavailable, cond.Reason)
		assert.Equal(t, int64(1), cond.ObservedGeneration)
	})

	t.Run("single available revision", func(t *testing.T) {
		active := []orbv1alpha1.ClusterObjectSetStatusSummary{{
			Name: "cos-1",
			Conditions: []metav1.Condition{{
				Type:   orbv1alpha1.ConditionTypeAvailable,
				Status: metav1.ConditionTrue,
			}},
		}}
		cond := EvaluateAvailability(2, active)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)
		assert.Equal(t, orbv1alpha1.ReasonAvailable, cond.Reason)
	})

	t.Run("single unavailable revision", func(t *testing.T) {
		active := []orbv1alpha1.ClusterObjectSetStatusSummary{{
			Name: "cos-1",
			Conditions: []metav1.Condition{{
				Type:   orbv1alpha1.ConditionTypeAvailable,
				Status: metav1.ConditionFalse,
			}},
		}}
		cond := EvaluateAvailability(2, active)
		assert.Equal(t, metav1.ConditionFalse, cond.Status)
		assert.Equal(t, orbv1alpha1.ReasonUnavailable, cond.Reason)
	})

	t.Run("multiple active revisions", func(t *testing.T) {
		active := []orbv1alpha1.ClusterObjectSetStatusSummary{
			{Name: "cos-1"},
			{Name: "cos-2"},
		}
		cond := EvaluateAvailability(3, active)
		assert.Equal(t, metav1.ConditionUnknown, cond.Status)
		assert.Equal(t, orbv1alpha1.ReasonProgressing, cond.Reason)
	})
}

func TestIsAvailable(t *testing.T) {
	t.Run("available", func(t *testing.T) {
		cos := &orbv1alpha1.ClusterObjectSet{}
		cos.Status.Conditions = []metav1.Condition{{
			Type:   orbv1alpha1.ConditionTypeAvailable,
			Status: metav1.ConditionTrue,
		}}
		assert.True(t, IsAvailable(cos))
	})

	t.Run("not available", func(t *testing.T) {
		cos := &orbv1alpha1.ClusterObjectSet{}
		assert.False(t, IsAvailable(cos))
	})
}

func TestActiveRevisionSummaries(t *testing.T) {
	t.Run("filters out archived", func(t *testing.T) {
		coss := []orbv1alpha1.ClusterObjectSet{
			{
				Spec: orbv1alpha1.ClusterObjectSetSpec{LifecycleState: orbv1alpha1.LifecycleStateActive},
			},
			{
				Spec: orbv1alpha1.ClusterObjectSetSpec{LifecycleState: orbv1alpha1.LifecycleStateArchived},
			},
		}
		coss[0].Name = "active"
		coss[1].Name = "archived"

		summaries := ActiveRevisionSummaries(coss)
		assert.Len(t, summaries, 1)
		assert.Equal(t, "active", summaries[0].Name)
	})

	t.Run("empty list", func(t *testing.T) {
		assert.Nil(t, ActiveRevisionSummaries(nil))
	})
}

func TestObjectCountsFromCOS(t *testing.T) {
	t.Run("nil COS returns nil", func(t *testing.T) {
		assert.Nil(t, ObjectCountsFromCOS(nil))
	})

	t.Run("COS with nil objectCounts returns nil", func(t *testing.T) {
		cos := &orbv1alpha1.ClusterObjectSet{}
		assert.Nil(t, ObjectCountsFromCOS(cos))
	})

	t.Run("returns copy of COS objectCounts", func(t *testing.T) {
		cos := &orbv1alpha1.ClusterObjectSet{}
		cos.Status.ObjectCounts = &orbv1alpha1.ObjectCounts{Total: 10, Synced: 8, Available: 6}
		result := ObjectCountsFromCOS(cos)
		require.NotNil(t, result)
		assert.Equal(t, int64(10), result.Total)
		assert.Equal(t, int64(8), result.Synced)
		assert.Equal(t, int64(6), result.Available)
	})
}
