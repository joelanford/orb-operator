package cosutil_test

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
	cosac "github.com/joelanford/orb-operator/applyconfigurations/api/v1alpha1"
	"github.com/joelanford/orb-operator/internal/cosutil"
	"github.com/joelanford/orb-operator/internal/testutil"
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

	testEnv := &envtest.Environment{
		Scheme: scheme,
	}

	var err error
	cfg, err = testEnv.Start()
	if err != nil {
		log.Fatalf("%v", err)
	}

	httpClient, err := rest.HTTPClientFor(cfg)
	if err != nil {
		log.Fatalf("%v", err)
	}
	mapper, err := apiutil.NewDynamicRESTMapper(cfg, httpClient)
	if err != nil {
		log.Fatalf("%v", err)
	}

	k8sClient, err = client.New(cfg, client.Options{
		Scheme: scheme,
		Mapper: mapper,
	})
	if err != nil {
		log.Fatalf("%v", err)
	}

	if err := testutil.InstallAPI(context.Background(), k8sClient); err != nil {
		log.Fatalf("%v", err)
	}

	code := m.Run()
	if err := testEnv.Stop(); err != nil {
		log.Fatalf("%v", err)
	}
	os.Exit(code)
}

func TestApply_AppliesMutation(t *testing.T) {
	ctx := context.Background()
	cos := createCOS(t, ctx, "test-apply")

	applied, err := cosutil.Apply(ctx, k8sClient, cos, "test-owner",
		func(_ *orbv1alpha1.ClusterObjectSet) bool { return true },
		func(ac *cosac.ClusterObjectSetApplyConfiguration) {
			ac.WithFinalizers("test-finalizer")
		},
	)
	require.NoError(t, err)
	assert.True(t, applied)

	var updated orbv1alpha1.ClusterObjectSet
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(cos), &updated))
	assert.True(t, controllerutil.ContainsFinalizer(&updated, "test-finalizer"))
}

func TestRemoveFinalizer_RemovesExistingFinalizer(t *testing.T) {
	ctx := context.Background()
	cos := createCOS(t, ctx, "test-remove-fin")

	controllerutil.AddFinalizer(cos, "test-finalizer")
	require.NoError(t, k8sClient.Update(ctx, cos))

	require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(cos), cos))
	require.True(t, controllerutil.ContainsFinalizer(cos, "test-finalizer"))

	err := cosutil.RemoveFinalizer(ctx, k8sClient, cos, "test-owner", "test-finalizer")
	require.NoError(t, err)

	var updated orbv1alpha1.ClusterObjectSet
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(cos), &updated))
	assert.False(t, controllerutil.ContainsFinalizer(&updated, "test-finalizer"))
}

func TestWaitForFinalizerRemoval_ReturnsWhenAbsent(t *testing.T) {
	ctx := context.Background()
	cos := createCOS(t, ctx, "test-wait-fin")

	err := cosutil.WaitForFinalizerRemoval(ctx, k8sClient, client.ObjectKeyFromObject(cos), "nonexistent")
	require.NoError(t, err)
}

func createCOS(t *testing.T, ctx context.Context, group string) *orbv1alpha1.ClusterObjectSet {
	t.Helper()
	cos := &orbv1alpha1.ClusterObjectSet{
		ObjectMeta: metav1.ObjectMeta{Name: group + "-1"},
		Spec: orbv1alpha1.ClusterObjectSetSpec{
			Group:          group,
			Revision:       1,
			LifecycleState: orbv1alpha1.LifecycleStateActive,
			ClusterObjectDeploymentTemplateSpec: orbv1alpha1.ClusterObjectDeploymentTemplateSpec{
				Phases: []orbv1alpha1.Phase{{
					Name: "phase-1",
					Objects: []orbv1alpha1.PhaseObject{{
						Object: runtime.RawExtension{Raw: []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"cm-` + group + `","namespace":"default"}}`)},
					}},
				}},
			},
		},
	}
	require.NoError(t, k8sClient.Create(ctx, cos))
	t.Cleanup(func() {
		_ = k8sClient.Delete(context.Background(), cos)
	})
	return cos
}
