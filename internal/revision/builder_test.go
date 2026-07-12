package revision

import (
	"testing"

	"github.com/stretchr/testify/assert"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func TestMapCollisionProtection(t *testing.T) {
	tests := []struct {
		name  string
		input orbv1alpha1.CollisionProtection
	}{
		{"Prevent", orbv1alpha1.CollisionProtectionPrevent},
		{"IfNoController", orbv1alpha1.CollisionProtectionIfNoController},
		{"None", orbv1alpha1.CollisionProtectionNone},
		{"unknown defaults to Prevent", orbv1alpha1.CollisionProtection("unknown")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapCollisionProtection(tt.input)
			assert.NotNil(t, result)
		})
	}
}
