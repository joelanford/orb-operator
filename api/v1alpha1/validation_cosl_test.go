package v1alpha1_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func newSlice(name string) *orbv1alpha1.ClusterObjectSlice {
	cmJSON, _ := json.Marshal(map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   map[string]interface{}{"name": name, "namespace": "default"},
	})
	return &orbv1alpha1.ClusterObjectSlice{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Objects: []orbv1alpha1.SliceObject{{
			ObjectKey: orbv1alpha1.ObjectKey{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Name:       name,
				Namespace:  "default",
			},
			Content: cmJSON,
		}},
	}
}

func createSlice(t *testing.T, ctx context.Context, slice *orbv1alpha1.ClusterObjectSlice) {
	t.Helper()
	require.NoError(t, k8sClient.Create(ctx, slice))
	t.Cleanup(func() {
		_ = k8sClient.Delete(ctx, slice)
	})
}

func TestCOSL_Count(t *testing.T) {
	ctx := context.Background()

	t.Run("MAP sets count when not provided", func(t *testing.T) {
		slice := newSlice("cosl-count-auto")
		createSlice(t, ctx, slice)

		got := &orbv1alpha1.ClusterObjectSlice{}
		require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(slice), got))
		require.Equal(t, int32(1), got.Count)
	})

	t.Run("MAP overwrites incorrect count", func(t *testing.T) {
		slice := newSlice("cosl-count-wrong")
		slice.Count = 99
		createSlice(t, ctx, slice)

		got := &orbv1alpha1.ClusterObjectSlice{}
		require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(slice), got))
		require.Equal(t, int32(1), got.Count)
	})
}

func TestCOSL_Objects_Immutable(t *testing.T) {
	ctx := context.Background()

	t.Run("updating objects is rejected", func(t *testing.T) {
		slice := newSlice("cosl-immut")
		createSlice(t, ctx, slice)

		cmJSON, _ := json.Marshal(map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata":   map[string]interface{}{"name": "different", "namespace": "default"},
		})
		slice.Objects = []orbv1alpha1.SliceObject{{
			ObjectKey: orbv1alpha1.ObjectKey{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Name:       "different",
				Namespace:  "default",
			},
			Content: cmJSON,
		}}
		err := k8sClient.Update(ctx, slice)
		require.ErrorContains(t, err, "objects is immutable")
	})
}

func TestCOSL_Objects_MinMaxItems(t *testing.T) {
	ctx := context.Background()

	t.Run("empty objects is rejected", func(t *testing.T) {
		slice := &orbv1alpha1.ClusterObjectSlice{
			ObjectMeta: metav1.ObjectMeta{Name: "cosl-empty"},
			Objects:    []orbv1alpha1.SliceObject{},
		}
		err := k8sClient.Create(ctx, slice)
		requireStatusError(t, err, "objects", "should have at least 1 items")
	})
}

func TestCOSL_Objects_DuplicateKeysRejected(t *testing.T) {
	ctx := context.Background()

	cmJSON, _ := json.Marshal(map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   map[string]interface{}{"name": "dup", "namespace": "default"},
	})
	key := orbv1alpha1.ObjectKey{
		APIVersion: "v1",
		Kind:       "ConfigMap",
		Name:       "dup",
		Namespace:  "default",
	}
	slice := &orbv1alpha1.ClusterObjectSlice{
		ObjectMeta: metav1.ObjectMeta{Name: "cosl-dup"},
		Objects: []orbv1alpha1.SliceObject{
			{ObjectKey: key, Content: cmJSON},
			{ObjectKey: key, Content: cmJSON},
		},
	}
	err := k8sClient.Create(ctx, slice)
	require.Error(t, err)
}

func TestCOSL_ObjectKey_Validation(t *testing.T) {
	ctx := context.Background()

	createWithKey := func(name string, key orbv1alpha1.ObjectKey) error {
		cmJSON, _ := json.Marshal(map[string]interface{}{
			"apiVersion": key.APIVersion,
			"kind":       key.Kind,
			"metadata":   map[string]interface{}{"name": key.Name, "namespace": key.Namespace},
		})
		slice := &orbv1alpha1.ClusterObjectSlice{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Objects: []orbv1alpha1.SliceObject{{
				ObjectKey: key,
				Content:   cmJSON,
			}},
		}
		return k8sClient.Create(ctx, slice)
	}

	t.Run("valid key is accepted", func(t *testing.T) {
		key := orbv1alpha1.ObjectKey{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "my-deploy",
			Namespace:  "default",
		}
		require.NoError(t, createWithKey("cosl-key-ok", key))
		t.Cleanup(func() {
			s := &orbv1alpha1.ClusterObjectSlice{ObjectMeta: metav1.ObjectMeta{Name: "cosl-key-ok"}}
			_ = k8sClient.Delete(ctx, s)
		})
	})

	t.Run("cluster-scoped (empty namespace) is accepted", func(t *testing.T) {
		key := orbv1alpha1.ObjectKey{
			APIVersion: "v1",
			Kind:       "Namespace",
			Name:       "my-ns",
		}
		require.NoError(t, createWithKey("cosl-key-cluster", key))
		t.Cleanup(func() {
			s := &orbv1alpha1.ClusterObjectSlice{ObjectMeta: metav1.ObjectMeta{Name: "cosl-key-cluster"}}
			_ = k8sClient.Delete(ctx, s)
		})
	})

	t.Run("empty apiVersion is rejected", func(t *testing.T) {
		key := orbv1alpha1.ObjectKey{APIVersion: "", Kind: "ConfigMap", Name: "cm"}
		requireStatusError(t, createWithKey("cosl-key-noav", key),
			"objects[0].apiVersion", "should be at least 1")
	})

	t.Run("empty kind is rejected", func(t *testing.T) {
		key := orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: "", Name: "cm"}
		requireStatusError(t, createWithKey("cosl-key-nokind", key),
			"objects[0].kind", "should be at least 1")
	})

	t.Run("empty name is rejected", func(t *testing.T) {
		key := orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: "ConfigMap", Name: ""}
		requireStatusError(t, createWithKey("cosl-key-noname", key),
			"objects[0].name", "should be at least 1")
	})

	t.Run("kind exceeding 63 chars is rejected", func(t *testing.T) {
		key := orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: strings.Repeat("K", 64), Name: "cm"}
		requireStatusError(t, createWithKey("cosl-key-longkind", key),
			"objects[0].kind", "may not be more than 63")
	})

	t.Run("name exceeding 253 chars is rejected", func(t *testing.T) {
		key := orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: "ConfigMap", Name: strings.Repeat("n", 254)}
		requireStatusError(t, createWithKey("cosl-key-longname", key),
			"objects[0].name", "may not be more than 253")
	})

	t.Run("namespace exceeding 63 chars is rejected", func(t *testing.T) {
		key := orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: "ConfigMap", Name: "cm", Namespace: strings.Repeat("n", 64)}
		requireStatusError(t, createWithKey("cosl-key-longns", key),
			"objects[0].namespace", "may not be more than 63")
	})
}

func TestCOSL_ObjectKey_APIVersion_Pattern(t *testing.T) {
	ctx := context.Background()

	for _, tc := range []struct {
		name       string
		apiVersion string
	}{
		{"core version", "v1"},
		{"named group", "apps/v1"},
		{"dotted group", "apiextensions.k8s.io/v1"},
		{"alpha version", "v1alpha1"},
		{"beta version", "v1beta2"},
	} {
		t.Run(tc.name+" is accepted", func(t *testing.T) {
			cosName := "cosl-av-ok-" + strings.ReplaceAll(tc.apiVersion, "/", "-")
			cosName = strings.ReplaceAll(cosName, ".", "-")
			key := orbv1alpha1.ObjectKey{APIVersion: tc.apiVersion, Kind: "ConfigMap", Name: "cm"}
			require.NoError(t, createSliceWithKey(t, ctx, cosName, key))
		})
	}

	for _, tc := range []struct {
		name       string
		cosName    string
		apiVersion string
	}{
		{"uppercase group", "cosl-av-upper", "Apps/v1"},
		{"trailing slash", "cosl-av-trail", "apps/"},
		{"leading slash", "cosl-av-lead", "/v1"},
		{"double slash", "cosl-av-dslash", "apps//v1"},
		{"group starts with dot", "cosl-av-gdot", ".apps/v1"},
		{"group ends with dot", "cosl-av-gdote", "apps./v1"},
		{"version starts with digit", "cosl-av-vdig", "1v"},
		{"version starts with hyphen", "cosl-av-vhyp", "-v1"},
	} {
		t.Run(tc.name+" is rejected", func(t *testing.T) {
			key := orbv1alpha1.ObjectKey{APIVersion: tc.apiVersion, Kind: "ConfigMap", Name: "cm"}
			requireStatusError(t, createSliceWithKey(t, ctx, tc.cosName, key),
				"objects[0].apiVersion", "must be a valid API version")
		})
	}
}

func TestCOSL_ObjectKey_Kind_Pattern(t *testing.T) {
	ctx := context.Background()

	for _, tc := range []struct {
		name string
		kind string
	}{
		{"simple", "ConfigMap"},
		{"lowercase start", "configMap"},
		{"with hyphens", "My-Kind"},
		{"single char", "C"},
	} {
		t.Run(tc.name+" is accepted", func(t *testing.T) {
			cosName := "cosl-k-ok-" + strings.ToLower(tc.kind)
			key := orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: tc.kind, Name: "obj"}
			require.NoError(t, createSliceWithKey(t, ctx, cosName, key))
		})
	}

	for _, tc := range []struct {
		name    string
		cosName string
		kind    string
	}{
		{"starts with digit", "cosl-k-digit", "1Kind"},
		{"starts with hyphen", "cosl-k-hyp", "-Kind"},
		{"ends with hyphen", "cosl-k-endhyp", "Kind-"},
		{"contains underscore", "cosl-k-uscore", "My_Kind"},
		{"contains dot", "cosl-k-dot", "My.Kind"},
	} {
		t.Run(tc.name+" is rejected", func(t *testing.T) {
			key := orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: tc.kind, Name: "obj"}
			requireStatusError(t, createSliceWithKey(t, ctx, tc.cosName, key),
				"objects[0].kind", "must be a DNS-1035 label")
		})
	}
}

func TestCOSL_ObjectKey_Name_Pattern(t *testing.T) {
	ctx := context.Background()

	for _, tc := range []struct {
		name    string
		objName string
	}{
		{"simple", "my-cm"},
		{"with dots", "my.cm.name"},
		{"single char", "a"},
		{"digits", "cm1"},
		{"double dots", "my..cm"},
		{"dot-hyphen", "my.-cm"},
		{"hyphen-dot", "my-.cm"},
	} {
		t.Run(tc.name+" is accepted", func(t *testing.T) {
			key := orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: "ConfigMap", Name: tc.objName}
			require.NoError(t, createSliceWithKey(t, ctx, "cosl-n-ok", key))
		})
	}

	for _, tc := range []struct {
		name    string
		cosName string
		objName string
	}{
		{"starts with hyphen", "cosl-n-hyp", "-cm"},
		{"starts with dot", "cosl-n-dot", ".cm"},
		{"ends with hyphen", "cosl-n-endhyp", "cm-"},
		{"ends with dot", "cosl-n-enddot", "cm."},
		{"uppercase", "cosl-n-upper", "MyCm"},
		{"contains underscore", "cosl-n-uscore", "my_cm"},
	} {
		t.Run(tc.name+" is rejected", func(t *testing.T) {
			key := orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: "ConfigMap", Name: tc.objName}
			requireStatusError(t, createSliceWithKey(t, ctx, tc.cosName, key),
				"objects[0].name", "must start and end with a lowercase alphanumeric character")
		})
	}
}

func TestCOSL_ObjectKey_Namespace_Pattern(t *testing.T) {
	ctx := context.Background()

	for _, tc := range []struct {
		name      string
		namespace string
	}{
		{"empty (cluster-scoped)", ""},
		{"simple", "default"},
		{"with hyphens", "my-ns"},
		{"digits", "ns1"},
	} {
		t.Run(tc.name+" is accepted", func(t *testing.T) {
			ns := tc.namespace
			if ns == "" {
				ns = "empty"
			}
			cosName := "cosl-ns-ok-" + ns
			key := orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: "ConfigMap", Name: "cm", Namespace: tc.namespace}
			require.NoError(t, createSliceWithKey(t, ctx, cosName, key))
		})
	}

	for _, tc := range []struct {
		name      string
		cosName   string
		namespace string
	}{
		{"starts with hyphen", "cosl-ns-hyp", "-ns"},
		{"ends with hyphen", "cosl-ns-endhyp", "ns-"},
		{"contains dot", "cosl-ns-dot", "my.ns"},
		{"uppercase", "cosl-ns-upper", "MyNs"},
		{"contains underscore", "cosl-ns-uscore", "my_ns"},
	} {
		t.Run(tc.name+" is rejected", func(t *testing.T) {
			key := orbv1alpha1.ObjectKey{APIVersion: "v1", Kind: "ConfigMap", Name: "cm", Namespace: tc.namespace}
			requireStatusError(t, createSliceWithKey(t, ctx, tc.cosName, key),
				"objects[0].namespace", "must be empty or a valid DNS-1123 label")
		})
	}
}

func createSliceWithKey(t *testing.T, ctx context.Context, name string, key orbv1alpha1.ObjectKey) error {
	t.Helper()
	cmJSON, _ := json.Marshal(map[string]interface{}{
		"apiVersion": key.APIVersion,
		"kind":       key.Kind,
		"metadata":   map[string]interface{}{"name": key.Name, "namespace": key.Namespace},
	})
	slice := &orbv1alpha1.ClusterObjectSlice{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Objects: []orbv1alpha1.SliceObject{{
			ObjectKey: key,
			Content:   cmJSON,
		}},
	}
	err := k8sClient.Create(ctx, slice)
	if err == nil {
		t.Cleanup(func() { _ = k8sClient.Delete(ctx, slice) })
	}
	return err
}
