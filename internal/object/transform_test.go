package object

import (
	"bytes"
	"compress/gzip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func TestTransformClusterObjectSlice_BuildsObjectMap(t *testing.T) {
	slice := &orbv1alpha1.ClusterObjectSlice{
		ObjectMeta: metav1.ObjectMeta{Name: "test-slice"},
		Objects: []orbv1alpha1.SliceObject{
			{
				ObjectKey: orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: "ConfigMap", Name: "cm1", Namespace: "default"},
				Content:   []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"cm1","namespace":"default"}}`),
			},
			{
				ObjectKey: orbv1alpha1.ObjectKey{APIVersion: "apps/v1", Kind: "Deployment", Name: "deploy1", Namespace: "default"},
				Content:   []byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"deploy1","namespace":"default"}}`),
			},
		},
	}

	result, err := TransformClusterObjectSlice(slice)
	require.NoError(t, err)

	transformed := result.(*orbv1alpha1.ClusterObjectSlice)
	assert.Nil(t, transformed.Objects, "Objects should be nil after transform")
	assert.Len(t, transformed.ObjectMap, 2, "ObjectMap should have 2 entries")

	cm1Key := orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: "ConfigMap", Name: "cm1", Namespace: "default"}
	assert.Contains(t, transformed.ObjectMap, cm1Key)
	assert.JSONEq(t,
		`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"cm1","namespace":"default"}}`,
		string(transformed.ObjectMap[cm1Key]),
	)
}

func TestTransformClusterObjectSlice_PreservesCompressedContent(t *testing.T) {
	raw := []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"cm1","namespace":"ns"}}`)
	compressed := gzipBytes(t, raw)

	slice := &orbv1alpha1.ClusterObjectSlice{
		Objects: []orbv1alpha1.SliceObject{{
			ObjectKey: orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: "ConfigMap", Name: "cm1", Namespace: "ns"},
			Content:   compressed,
		}},
	}

	result, err := TransformClusterObjectSlice(slice)
	require.NoError(t, err)

	transformed := result.(*orbv1alpha1.ClusterObjectSlice)
	key := orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: "ConfigMap", Name: "cm1", Namespace: "ns"}
	assert.Equal(t, compressed, transformed.ObjectMap[key], "transform should not decompress content")
}

func TestTransformClusterObjectSlice_WrongType(t *testing.T) {
	_, err := TransformClusterObjectSlice("not-a-slice")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected type")
}

func gzipBytes(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, err := w.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}
