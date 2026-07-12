package cosutil

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestClearFinalizerFieldOwnership(t *testing.T) {
	t.Run("removes finalizer from field ownership", func(t *testing.T) {
		fields := map[string]any{
			"f:metadata": map[string]any{
				"f:finalizers": map[string]any{
					"v:my-finalizer":    map[string]any{},
					"v:other-finalizer": map[string]any{},
				},
			},
		}
		raw, _ := json.Marshal(fields)

		managedFields := []metav1.ManagedFieldsEntry{{
			Manager:  "test-manager",
			FieldsV1: &metav1.FieldsV1{Raw: raw},
		}}

		ClearFinalizerFieldOwnership(managedFields, "test-manager", "my-finalizer")

		var result map[string]any
		_ = json.Unmarshal(managedFields[0].FieldsV1.GetRawBytes(), &result)

		fMeta := result["f:metadata"].(map[string]any)
		fFinalizers := fMeta["f:finalizers"].(map[string]any)
		assert.NotContains(t, fFinalizers, "v:my-finalizer")
		assert.Contains(t, fFinalizers, "v:other-finalizer")
	})

	t.Run("cleans up empty maps", func(t *testing.T) {
		fields := map[string]any{
			"f:metadata": map[string]any{
				"f:finalizers": map[string]any{
					"v:only-finalizer": map[string]any{},
				},
			},
		}
		raw, _ := json.Marshal(fields)

		managedFields := []metav1.ManagedFieldsEntry{{
			Manager:  "test-manager",
			FieldsV1: &metav1.FieldsV1{Raw: raw},
		}}

		ClearFinalizerFieldOwnership(managedFields, "test-manager", "only-finalizer")

		var result map[string]any
		_ = json.Unmarshal(managedFields[0].FieldsV1.GetRawBytes(), &result)
		assert.NotContains(t, result, "f:metadata")
	})

	t.Run("skips wrong manager", func(t *testing.T) {
		fields := map[string]any{
			"f:metadata": map[string]any{
				"f:finalizers": map[string]any{
					"v:my-finalizer": map[string]any{},
				},
			},
		}
		raw, _ := json.Marshal(fields)

		managedFields := []metav1.ManagedFieldsEntry{{
			Manager:  "other-manager",
			FieldsV1: &metav1.FieldsV1{Raw: raw},
		}}

		ClearFinalizerFieldOwnership(managedFields, "test-manager", "my-finalizer")

		var result map[string]any
		_ = json.Unmarshal(managedFields[0].FieldsV1.GetRawBytes(), &result)
		fMeta := result["f:metadata"].(map[string]any)
		fFinalizers := fMeta["f:finalizers"].(map[string]any)
		assert.Contains(t, fFinalizers, "v:my-finalizer")
	})

	t.Run("handles nil FieldsV1", func(t *testing.T) {
		managedFields := []metav1.ManagedFieldsEntry{{
			Manager: "test-manager",
		}}
		ClearFinalizerFieldOwnership(managedFields, "test-manager", "my-finalizer")
	})

	t.Run("handles empty managed fields", func(t *testing.T) {
		ClearFinalizerFieldOwnership(nil, "test-manager", "my-finalizer")
	})
}
