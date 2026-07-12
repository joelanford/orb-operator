package revision

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"pkg.package-operator.run/boxcutter/ownerhandling"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
	"github.com/joelanford/orb-operator/internal/object"
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

func TestBuild(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, orbv1alpha1.AddToScheme(scheme))

	cos := &orbv1alpha1.ClusterObjectSet{}
	cos.Name = "test-cos"
	cos.UID = "test-uid"
	cos.Spec.Revision = 1
	cos.Spec.Group = "test-group"

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("v1")
	obj.SetKind("ConfigMap")
	obj.SetName("cm1")
	obj.SetNamespace("default")

	resolved := &object.Result{
		Phases: []object.Phase{{
			Name: "install",
			Objects: []object.Object{{
				Obj: obj,
			}},
		}},
	}

	ownerStrategy := ownerhandling.NewNative(scheme)
	rev := Build(cos, resolved, nil, ownerStrategy)

	assert.Equal(t, "test-cos", rev.GetName())
	assert.Equal(t, int64(1), rev.GetRevisionNumber())
	require.Len(t, rev.GetPhases(), 1)
	assert.Equal(t, "install", rev.GetPhases()[0].GetName())
}

func TestBuild_WithSiblings(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, orbv1alpha1.AddToScheme(scheme))

	cos := &orbv1alpha1.ClusterObjectSet{}
	cos.Name = "test-cos"
	cos.UID = "test-uid"
	cos.Spec.Revision = 2

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("v1")
	obj.SetKind("ConfigMap")
	obj.SetName("cm1")

	resolved := &object.Result{
		Phases: []object.Phase{{
			Name:    "install",
			Objects: []object.Object{{Obj: obj}},
		}},
	}

	sibling := &orbv1alpha1.ClusterObjectSet{}
	sibling.Name = "sibling-cos"

	ownerStrategy := ownerhandling.NewNative(scheme)
	rev := Build(cos, resolved, []*orbv1alpha1.ClusterObjectSet{sibling}, ownerStrategy)

	assert.Equal(t, "test-cos", rev.GetName())
	assert.Equal(t, int64(2), rev.GetRevisionNumber())
}

func TestBuild_WithCollisionProtection(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, orbv1alpha1.AddToScheme(scheme))

	cp := orbv1alpha1.CollisionProtectionNone
	cos := &orbv1alpha1.ClusterObjectSet{}
	cos.Name = "test-cos"
	cos.UID = "test-uid"
	cos.Spec.Revision = 1
	cos.Spec.CollisionProtection = &cp

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("v1")
	obj.SetKind("ConfigMap")
	obj.SetName("cm1")

	objCP := orbv1alpha1.CollisionProtectionIfNoController
	resolved := &object.Result{
		Phases: []object.Phase{{
			Name:                "install",
			CollisionProtection: &cp,
			Objects: []object.Object{{
				Obj:                 obj,
				CollisionProtection: &objCP,
			}},
		}},
	}

	ownerStrategy := ownerhandling.NewNative(scheme)
	rev := Build(cos, resolved, nil, ownerStrategy)
	assert.NotNil(t, rev)
}
