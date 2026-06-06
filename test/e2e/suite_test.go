package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

var c client.Client

func TestMain(m *testing.M) {
	cfg, err := ctrl.GetConfig()
	if err != nil {
		panic(fmt.Sprintf("getting kubeconfig: %v", err))
	}

	s := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(s))
	utilruntime.Must(apiextensionsv1.AddToScheme(s))
	utilruntime.Must(orbv1alpha1.AddToScheme(s))

	c, err = client.New(cfg, client.Options{Scheme: s})
	if err != nil {
		panic(fmt.Sprintf("creating client: %v", err))
	}

	os.Exit(m.Run())
}

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeScenario,
		Options: &godog.Options{
			Concurrency: 16,
			Format:      "pretty",
			Paths:       []string{"features"},
			Output:      colors.Colored(os.Stdout),
			TestingT:    t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status returned from godog suite")
	}
}

func initializeScenario(sc *godog.ScenarioContext) {
	tc := newTestContext(c)

	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		return ctx, tc.setup(ctx)
	})
	sc.After(func(ctx context.Context, s *godog.Scenario, scenarioErr error) (context.Context, error) {
		if scenarioErr != nil {
			return ctx, nil
		}
		return ctx, tc.teardown(ctx)
	})

	registerSetupSteps(sc, tc)
	registerActionSteps(sc, tc)
	registerAssertSteps(sc, tc)
}
