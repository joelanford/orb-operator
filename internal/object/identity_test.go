package object

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func TestResolveIdentities_ObjectRef(t *testing.T) {
	phases := []orbv1alpha1.Phase{{
		Name: "install",
		Objects: []orbv1alpha1.PhaseObject{{
			ObjectRef: &orbv1alpha1.ObjectRef{
				SliceName: "my-slice",
				ObjectKey: orbv1alpha1.ObjectKey{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "nginx",
					Namespace:  "default",
				},
			},
		}},
	}}

	result, err := ResolveIdentities(phases)
	require.NoError(t, err)
	require.Len(t, result.Phases, 1)
	require.Len(t, result.Phases[0].Objects, 1)

	obj := result.Phases[0].Objects[0].Obj
	assert.Equal(t, "apps/v1", obj.GetAPIVersion())
	assert.Equal(t, "Deployment", obj.GetKind())
	assert.Equal(t, "nginx", obj.GetName())
	assert.Equal(t, "default", obj.GetNamespace())
	assert.Empty(t, result.Hash)
}

func TestResolveIdentities_Inline(t *testing.T) {
	phases := []orbv1alpha1.Phase{{
		Name: "install",
		Objects: []orbv1alpha1.PhaseObject{{
			Object: runtime.RawExtension{
				Raw: []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"cm1","namespace":"test-ns"},"data":{"key":"value"}}`),
			},
		}},
	}}

	result, err := ResolveIdentities(phases)
	require.NoError(t, err)
	require.Len(t, result.Phases, 1)
	require.Len(t, result.Phases[0].Objects, 1)

	obj := result.Phases[0].Objects[0].Obj
	assert.Equal(t, "v1", obj.GetAPIVersion())
	assert.Equal(t, "ConfigMap", obj.GetKind())
	assert.Equal(t, "cm1", obj.GetName())
	assert.Equal(t, "test-ns", obj.GetNamespace())
}

func TestResolveIdentities_MixedPhase(t *testing.T) {
	phases := []orbv1alpha1.Phase{{
		Name: "install",
		Objects: []orbv1alpha1.PhaseObject{
			{
				ObjectRef: &orbv1alpha1.ObjectRef{
					SliceName: "slice1",
					ObjectKey: orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: "ConfigMap", Name: "from-slice", Namespace: "ns1"},
				},
			},
			{
				Object: runtime.RawExtension{
					Raw: []byte(`{"apiVersion":"v1","kind":"Secret","metadata":{"name":"inline-secret","namespace":"ns1"}}`),
				},
			},
		},
	}}

	result, err := ResolveIdentities(phases)
	require.NoError(t, err)
	require.Len(t, result.Phases[0].Objects, 2)
	assert.Equal(t, "ConfigMap", result.Phases[0].Objects[0].Obj.GetKind())
	assert.Equal(t, "from-slice", result.Phases[0].Objects[0].Obj.GetName())
	assert.Equal(t, "Secret", result.Phases[0].Objects[1].Obj.GetKind())
	assert.Equal(t, "inline-secret", result.Phases[0].Objects[1].Obj.GetName())
}

func TestResolveIdentities_MultiplePhases_ManagedObjectsDedup(t *testing.T) {
	phases := []orbv1alpha1.Phase{
		{
			Name: "phase-1",
			Objects: []orbv1alpha1.PhaseObject{
				{ObjectRef: &orbv1alpha1.ObjectRef{SliceName: "s", ObjectKey: orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: "ConfigMap", Name: "cm1", Namespace: "ns"}}},
				{ObjectRef: &orbv1alpha1.ObjectRef{SliceName: "s", ObjectKey: orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: "ConfigMap", Name: "cm2", Namespace: "ns"}}},
			},
		},
		{
			Name: "phase-2",
			Objects: []orbv1alpha1.PhaseObject{
				{ObjectRef: &orbv1alpha1.ObjectRef{SliceName: "s", ObjectKey: orbv1alpha1.ObjectKey{APIVersion: "apps/v1", Kind: "Deployment", Name: "d1", Namespace: "ns"}}},
				{ObjectRef: &orbv1alpha1.ObjectRef{SliceName: "s", ObjectKey: orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: "ConfigMap", Name: "cm3", Namespace: "ns"}}},
			},
		},
	}

	result, err := ResolveIdentities(phases)
	require.NoError(t, err)

	managed := result.ManagedObjects()
	gvks := map[schema.GroupVersionKind]bool{}
	for _, o := range managed {
		gvks[o.GetObjectKind().GroupVersionKind()] = true
	}
	assert.Len(t, gvks, 2)
	assert.True(t, gvks[schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"}])
	assert.True(t, gvks[schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}])
}

func TestResolveIdentities_ClusterScoped(t *testing.T) {
	phases := []orbv1alpha1.Phase{{
		Name: "install",
		Objects: []orbv1alpha1.PhaseObject{{
			Object: runtime.RawExtension{
				Raw: []byte(`{"apiVersion":"apiextensions.k8s.io/v1","kind":"CustomResourceDefinition","metadata":{"name":"widgets.example.com"}}`),
			},
		}},
	}}

	result, err := ResolveIdentities(phases)
	require.NoError(t, err)

	obj := result.Phases[0].Objects[0].Obj
	assert.Equal(t, "CustomResourceDefinition", obj.GetKind())
	assert.Equal(t, "widgets.example.com", obj.GetName())
	assert.Empty(t, obj.GetNamespace())
}

func TestResolveIdentities_MalformedJSON(t *testing.T) {
	phases := []orbv1alpha1.Phase{{
		Name: "install",
		Objects: []orbv1alpha1.PhaseObject{{
			Object: runtime.RawExtension{Raw: []byte(`not json`)},
		}},
	}}

	_, err := ResolveIdentities(phases)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "extracting identity")
}

func TestResolveIdentities_NeitherObjectNorRef(t *testing.T) {
	phases := []orbv1alpha1.Phase{{
		Name:    "install",
		Objects: []orbv1alpha1.PhaseObject{{}},
	}}

	_, err := ResolveIdentities(phases)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "neither inline content nor objectRef")
}

func TestResolveIdentities_PreservesCollisionProtection(t *testing.T) {
	phaseCp := orbv1alpha1.CollisionProtectionNone
	objCp := orbv1alpha1.CollisionProtectionPrevent
	phases := []orbv1alpha1.Phase{{
		Name:                "install",
		CollisionProtection: &phaseCp,
		Objects: []orbv1alpha1.PhaseObject{{
			ObjectRef: &orbv1alpha1.ObjectRef{
				SliceName: "s",
				ObjectKey: orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: "ConfigMap", Name: "cm1"},
			},
			CollisionProtection: &objCp,
		}},
	}}

	result, err := ResolveIdentities(phases)
	require.NoError(t, err)
	assert.Equal(t, &phaseCp, result.Phases[0].CollisionProtection)
	assert.Equal(t, &objCp, result.Phases[0].Objects[0].CollisionProtection)
}

func TestResolveIdentities_PreservesPhaseName(t *testing.T) {
	phases := []orbv1alpha1.Phase{
		{
			Name: "crds",
			Objects: []orbv1alpha1.PhaseObject{{
				ObjectRef: &orbv1alpha1.ObjectRef{SliceName: "s", ObjectKey: orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: "ConfigMap", Name: "cm1"}},
			}},
		},
		{
			Name: "resources",
			Objects: []orbv1alpha1.PhaseObject{{
				ObjectRef: &orbv1alpha1.ObjectRef{SliceName: "s", ObjectKey: orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: "ConfigMap", Name: "cm2"}},
			}},
		},
	}

	result, err := ResolveIdentities(phases)
	require.NoError(t, err)
	assert.Equal(t, "crds", result.Phases[0].Name)
	assert.Equal(t, "resources", result.Phases[1].Name)
}
