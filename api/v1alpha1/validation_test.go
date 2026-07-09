package v1alpha1_test

import (
	"context"
	stderrors "errors"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

var (
	cfg       *rest.Config
	k8sClient client.Client
	scheme    = runtime.NewScheme()
)

func TestMain(m *testing.M) {
	if err := orbv1alpha1.AddToScheme(scheme); err != nil {
		log.Fatalf("%v", err)
	}

	crdPaths := []string{filepath.Join("..", "..", "deploy", "crds")}
	testEnv := &envtest.Environment{
		CRDDirectoryPaths: crdPaths,
		Scheme:            scheme,
		CRDInstallOptions: envtest.CRDInstallOptions{
			Scheme: scheme,
		},
	}

	var err error
	cfg, err = testEnv.Start()
	if err != nil {
		log.Fatalf("%v", err)
	}

	k8sClient, err = client.New(cfg, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		log.Fatalf("%v", err)
	}

	code := m.Run()
	if err := testEnv.Stop(); err != nil {
		log.Fatalf("%v", err)
	}
	os.Exit(code)
}

func newCOS(name string) *orbv1alpha1.ClusterObjectSet {
	return &orbv1alpha1.ClusterObjectSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: orbv1alpha1.ClusterObjectSetSpec{
			Group:          "test",
			Revision:       1,
			LifecycleState: orbv1alpha1.LifecycleStateActive,
			ClusterObjectDeploymentTemplateSpec: orbv1alpha1.ClusterObjectDeploymentTemplateSpec{
				Phases: []orbv1alpha1.Phase{{
					Name: "default",
					Objects: []orbv1alpha1.PhaseObject{{
						Object: runtime.RawExtension{Raw: []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"` + name + `","namespace":"default"}}`)},
					}},
				}},
			},
		},
	}
}

func createCOS(t *testing.T, ctx context.Context, cos *orbv1alpha1.ClusterObjectSet) {
	t.Helper()
	require.NoError(t, k8sClient.Create(ctx, cos))
	t.Cleanup(func() {
		require.NoError(t, k8sClient.Delete(ctx, cos))
	})
}

func requireStatusError(t *testing.T, err error, field string, msgSubstring string) {
	t.Helper()
	require.Error(t, err)
	var statusErr *errors.StatusError
	ok := stderrors.As(err, &statusErr)
	require.True(t, ok, "expected StatusError, got %T", err)
	found := false
	for _, cause := range statusErr.ErrStatus.Details.Causes {
		if cause.Field == field {
			found = true
			assert.Contains(t, cause.Message, msgSubstring,
				"field %q: expected message containing %q, got %q", field, msgSubstring, cause.Message)
			break
		}
	}
	require.True(t, found, "no cause with field %q in error: %v", field, err)
}
