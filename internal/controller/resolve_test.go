package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

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

func TestManagedObjectsFromResolved_DeduplicatesGVKs(t *testing.T) {
	resolved := &resolvedPhaseObjects{
		phases: []resolvedPhase{
			{
				name: "phase-1",
				objects: []resolvedObject{
					{obj: newUnstructuredObj("v1", "ConfigMap", "cm1")},
					{obj: newUnstructuredObj("v1", "ConfigMap", "cm2")},
				},
			},
			{
				name: "phase-2",
				objects: []resolvedObject{
					{obj: newUnstructuredObj("apps/v1", "Deployment", "d1")},
					{obj: newUnstructuredObj("v1", "ConfigMap", "cm3")},
				},
			},
		},
	}

	objects := managedObjectsFromResolved(resolved)
	gvks := make(map[schema.GroupVersionKind]bool)
	for _, o := range objects {
		gvks[o.GetObjectKind().GroupVersionKind()] = true
	}
	assert.Len(t, gvks, 2, "should deduplicate GVKs across phases")
	assert.True(t, gvks[schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"}])
	assert.True(t, gvks[schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}])
}

func TestVerifyContentHash_SetsHashOnFirstResolution(t *testing.T) {
	cos := &orbv1alpha1.ClusterObjectSet{}
	resolved := &resolvedPhaseObjects{hash: "abc123"}

	r := &COSReconciler{}
	err := r.verifyContentHash(cos, resolved)
	require.NoError(t, err)
	assert.Equal(t, "abc123", cos.Status.ResolvedContentHash)
}

func TestVerifyContentHash_AcceptsMatchingHash(t *testing.T) {
	cos := &orbv1alpha1.ClusterObjectSet{}
	cos.Status.ResolvedContentHash = "abc123"
	resolved := &resolvedPhaseObjects{hash: "abc123"}

	r := &COSReconciler{}
	err := r.verifyContentHash(cos, resolved)
	require.NoError(t, err)
}

func TestVerifyContentHash_RejectsMismatch(t *testing.T) {
	cos := &orbv1alpha1.ClusterObjectSet{}
	cos.Status.ResolvedContentHash = "abc123"
	resolved := &resolvedPhaseObjects{hash: "def456"}

	r := &COSReconciler{}
	err := r.verifyContentHash(cos, resolved)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hash mismatch")
}

func TestHandleResolutionError_FalseWhenNoHash(t *testing.T) {
	cos := &orbv1alpha1.ClusterObjectSet{}
	cos.Generation = 1

	r := &COSReconciler{}
	_ = r.handleResolutionError(cos, assert.AnError)

	cond := cos.Status.Conditions[0]
	assert.Equal(t, orbv1alpha1.ConditionTypeAvailable, cond.Type)
	assert.Equal(t, "False", string(cond.Status))
	assert.Equal(t, orbv1alpha1.ReasonInvalidRevision, cond.Reason)
}

func TestHandleResolutionError_UnknownWhenHashSet(t *testing.T) {
	cos := &orbv1alpha1.ClusterObjectSet{}
	cos.Generation = 1
	cos.Status.ResolvedContentHash = "previously-resolved"

	r := &COSReconciler{}
	_ = r.handleResolutionError(cos, assert.AnError)

	cond := cos.Status.Conditions[0]
	assert.Equal(t, orbv1alpha1.ConditionTypeAvailable, cond.Type)
	assert.Equal(t, "Unknown", string(cond.Status))
	assert.Equal(t, orbv1alpha1.ReasonInvalidRevision, cond.Reason)
}

func newUnstructuredObj(apiVersion, kind, name string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(apiVersion)
	obj.SetKind(kind)
	obj.SetName(name)
	obj.SetNamespace("default")
	return obj
}
