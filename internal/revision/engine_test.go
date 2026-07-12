package revision

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func TestEngine_completedPhaseNames(t *testing.T) {
	earlier := metav1.Now()

	t.Run("returns completed phases", func(t *testing.T) {
		e := &Engine{
			existingOPs: []orbv1alpha1.ObservedPhase{
				{Name: "phase-1", CompletedAt: &earlier},
				{Name: "phase-2"},
				{Name: "phase-3", CompletedAt: &earlier},
			},
		}
		names := e.completedPhaseNames()
		assert.True(t, names["phase-1"])
		assert.False(t, names["phase-2"])
		assert.True(t, names["phase-3"])
	})

	t.Run("empty phases", func(t *testing.T) {
		e := &Engine{}
		names := e.completedPhaseNames()
		assert.Empty(t, names)
	})
}
