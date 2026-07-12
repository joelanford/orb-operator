package object

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func TestResolver_InlineObjects(t *testing.T) {
	resolver := NewResolver(fake.NewClientBuilder().Build())

	phases := []orbv1alpha1.Phase{{
		Name: "install",
		Objects: []orbv1alpha1.PhaseObject{{
			Object: runtime.RawExtension{
				Raw: []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"cm1","namespace":"default"}}`),
			},
		}},
	}}

	result, err := resolver.Resolve(context.Background(), phases)
	require.NoError(t, err)
	require.Len(t, result.Phases, 1)
	require.Len(t, result.Phases[0].Objects, 1)
	assert.Equal(t, "ConfigMap", result.Phases[0].Objects[0].Obj.GetKind())
	assert.Equal(t, "cm1", result.Phases[0].Objects[0].Obj.GetName())
	assert.NotEmpty(t, result.Hash)
}

func TestResolver_SliceRef(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, orbv1alpha1.AddToScheme(scheme))

	slice := &orbv1alpha1.ClusterObjectSlice{}
	slice.SetName("test-slice")
	slice.SetGroupVersionKind(orbv1alpha1.GroupVersion.WithKind("ClusterObjectSlice"))
	slice.Objects = []orbv1alpha1.SliceObject{{
		ObjectKey: orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: "ConfigMap", Name: "cm1", Namespace: "default"},
		Content:   []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"cm1","namespace":"default"}}`),
	}}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(slice).Build()
	resolver := NewResolver(cl)

	phases := []orbv1alpha1.Phase{{
		Name: "install",
		Objects: []orbv1alpha1.PhaseObject{{
			ObjectRef: &orbv1alpha1.ObjectRef{
				SliceName: "test-slice",
				ObjectKey: orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: "ConfigMap", Name: "cm1", Namespace: "default"},
			},
		}},
	}}

	result, err := resolver.Resolve(context.Background(), phases)
	require.NoError(t, err)
	require.Len(t, result.Phases, 1)
	require.Len(t, result.Phases[0].Objects, 1)
	assert.Equal(t, "ConfigMap", result.Phases[0].Objects[0].Obj.GetKind())
	assert.Equal(t, "cm1", result.Phases[0].Objects[0].Obj.GetName())
}

func TestResolver_SliceRefNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, orbv1alpha1.AddToScheme(scheme))

	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	resolver := NewResolver(cl)

	phases := []orbv1alpha1.Phase{{
		Name: "install",
		Objects: []orbv1alpha1.PhaseObject{{
			ObjectRef: &orbv1alpha1.ObjectRef{
				SliceName: "missing-slice",
				ObjectKey: orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: "ConfigMap", Name: "cm1"},
			},
		}},
	}}

	_, err := resolver.Resolve(context.Background(), phases)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetching slice")
}

func TestResolver_ObjectNotInSlice(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, orbv1alpha1.AddToScheme(scheme))

	slice := &orbv1alpha1.ClusterObjectSlice{}
	slice.SetName("test-slice")
	slice.SetGroupVersionKind(orbv1alpha1.GroupVersion.WithKind("ClusterObjectSlice"))

	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(slice).Build()
	resolver := NewResolver(cl)

	phases := []orbv1alpha1.Phase{{
		Name: "install",
		Objects: []orbv1alpha1.PhaseObject{{
			ObjectRef: &orbv1alpha1.ObjectRef{
				SliceName: "test-slice",
				ObjectKey: orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: "ConfigMap", Name: "missing"},
			},
		}},
	}}

	_, err := resolver.Resolve(context.Background(), phases)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in slice")
}

func TestResolver_GzipContent(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, orbv1alpha1.AddToScheme(scheme))

	raw := []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"cm1","namespace":"default"}}`)
	compressed := gzipBytes(t, raw)

	slice := &orbv1alpha1.ClusterObjectSlice{}
	slice.SetName("test-slice")
	slice.SetGroupVersionKind(orbv1alpha1.GroupVersion.WithKind("ClusterObjectSlice"))
	slice.Objects = []orbv1alpha1.SliceObject{{
		ObjectKey: orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: "ConfigMap", Name: "cm1", Namespace: "default"},
		Content:   compressed,
	}}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(slice).Build()
	resolver := NewResolver(cl)

	phases := []orbv1alpha1.Phase{{
		Name: "install",
		Objects: []orbv1alpha1.PhaseObject{{
			ObjectRef: &orbv1alpha1.ObjectRef{
				SliceName: "test-slice",
				ObjectKey: orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: "ConfigMap", Name: "cm1", Namespace: "default"},
			},
		}},
	}}

	result, err := resolver.Resolve(context.Background(), phases)
	require.NoError(t, err)
	require.Len(t, result.Phases[0].Objects, 1)
	assert.Equal(t, "ConfigMap", result.Phases[0].Objects[0].Obj.GetKind())
}

func TestResolver_InvalidJSON(t *testing.T) {
	resolver := NewResolver(fake.NewClientBuilder().Build())

	phases := []orbv1alpha1.Phase{{
		Name: "install",
		Objects: []orbv1alpha1.PhaseObject{{
			Object: runtime.RawExtension{Raw: []byte(`not json`)},
		}},
	}}

	_, err := resolver.Resolve(context.Background(), phases)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshalling")
}

func TestResolver_HashStability(t *testing.T) {
	resolver := NewResolver(fake.NewClientBuilder().Build())

	phases := []orbv1alpha1.Phase{{
		Name: "install",
		Objects: []orbv1alpha1.PhaseObject{{
			Object: runtime.RawExtension{
				Raw: []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"cm1"}}`),
			},
		}},
	}}

	r1, err := resolver.Resolve(context.Background(), phases)
	require.NoError(t, err)
	r2, err := resolver.Resolve(context.Background(), phases)
	require.NoError(t, err)
	assert.Equal(t, r1.Hash, r2.Hash)
}

func TestResolver_PreservesPhaseMetadata(t *testing.T) {
	resolver := NewResolver(fake.NewClientBuilder().Build())

	cp := orbv1alpha1.CollisionProtectionNone
	phases := []orbv1alpha1.Phase{{
		Name:                "install",
		CollisionProtection: &cp,
		Objects: []orbv1alpha1.PhaseObject{{
			Object: runtime.RawExtension{
				Raw: []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"cm1"}}`),
			},
			CollisionProtection: &cp,
			Assertions: []orbv1alpha1.Assertion{{
				ConditionEqual: &orbv1alpha1.ConditionEqualAssertion{
					Type:   "Available",
					Status: "True",
				},
			}},
		}},
	}}

	result, err := resolver.Resolve(context.Background(), phases)
	require.NoError(t, err)
	assert.Equal(t, &cp, result.Phases[0].CollisionProtection)
	assert.Equal(t, &cp, result.Phases[0].Objects[0].CollisionProtection)
	require.Len(t, result.Phases[0].Objects[0].Assertions, 1)
}

func TestResult_ManagedObjects_DeduplicatesGVKs(t *testing.T) {
	result := &Result{
		Phases: []Phase{
			{
				Name: "phase-1",
				Objects: []Object{
					{Obj: newUnstructuredObj("v1", "ConfigMap", "cm1")},
					{Obj: newUnstructuredObj("v1", "ConfigMap", "cm2")},
				},
			},
			{
				Name: "phase-2",
				Objects: []Object{
					{Obj: newUnstructuredObj("apps/v1", "Deployment", "d1")},
					{Obj: newUnstructuredObj("v1", "ConfigMap", "cm3")},
				},
			},
		},
	}

	objects := result.ManagedObjects()
	gvks := make(map[schema.GroupVersionKind]bool)
	for _, o := range objects {
		gvks[o.GetObjectKind().GroupVersionKind()] = true
	}
	assert.Len(t, gvks, 2)
	assert.True(t, gvks[schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"}])
	assert.True(t, gvks[schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}])
}

func TestResult_VerifyHash_SetsOnFirst(t *testing.T) {
	result := &Result{Hash: "abc123"}
	err := result.VerifyHash("")
	require.NoError(t, err)
}

func TestResult_VerifyHash_AcceptsMatch(t *testing.T) {
	result := &Result{Hash: "abc123"}
	err := result.VerifyHash("abc123")
	require.NoError(t, err)
}

func TestResult_VerifyHash_RejectsMismatch(t *testing.T) {
	result := &Result{Hash: "def456"}
	err := result.VerifyHash("abc123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hash mismatch")
}

func TestDecompressGzip(t *testing.T) {
	original := []byte(`{"apiVersion":"v1","kind":"ConfigMap"}`)
	compressed := gzipBytes(t, original)

	result, err := decompressGzip(compressed)
	require.NoError(t, err)
	assert.Equal(t, original, result)
}

func TestDecompressGzip_InvalidData(t *testing.T) {
	_, err := decompressGzip([]byte{0x1f, 0x8b, 0x00, 0x00})
	require.Error(t, err)
}

func TestUnmarshalUnstructured(t *testing.T) {
	raw := []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"test","namespace":"default"}}`)
	obj, err := unmarshalUnstructured(raw)
	require.NoError(t, err)
	assert.Equal(t, "v1", obj.GetAPIVersion())
	assert.Equal(t, "ConfigMap", obj.GetKind())
	assert.Equal(t, "test", obj.GetName())
	assert.Equal(t, "default", obj.GetNamespace())
}

func TestUnmarshalUnstructured_InvalidJSON(t *testing.T) {
	_, err := unmarshalUnstructured([]byte(`not json`))
	require.Error(t, err)
}

func newUnstructuredObj(apiVersion, kind, name string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(apiVersion)
	obj.SetKind(kind)
	obj.SetName(name)
	obj.SetNamespace("default")
	return obj
}
