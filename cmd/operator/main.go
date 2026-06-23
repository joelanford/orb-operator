package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"pkg.package-operator.run/boxcutter/managedcache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
	"github.com/joelanford/orb-operator/internal/controller"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(orbv1alpha1.AddToScheme(scheme))
}

func main() {
	cmd := newRootCommand()
	if err := cmd.ExecuteContext(ctrl.SetupSignalHandler()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "orb-operator",
		Short: "Kubernetes operator for phased extension object management",
		RunE:  run,
	}
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	klog.InitFlags(fs)
	vFlag := fs.Lookup("v")
	cmd.Flags().AddFlag(pflag.PFlagFromGoFlag(&flag.Flag{
		Name:     vFlag.Name,
		Usage:    vFlag.Usage,
		Value:    vFlag.Value,
		DefValue: vFlag.DefValue,
	}))
	return cmd
}

func run(cmd *cobra.Command, _ []string) error {
	log.SetLogger(klog.NewKlogr())

	cfg := ctrl.GetConfigOrDie()

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress:    ":8443",
			SecureServing:  true,
			FilterProvider: filters.WithAuthenticationAndAuthorization,
		},
	})
	if err != nil {
		return fmt.Errorf("creating manager: %w", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return fmt.Errorf("creating discovery client: %w", err)
	}

	accessManager := managedcache.NewObjectBoundAccessManager[*orbv1alpha1.ClusterObjectSetRevision](
		ctrl.Log.WithName("managed-cache"),
		func(_ context.Context, _ *orbv1alpha1.ClusterObjectSetRevision, c *rest.Config, opts cache.Options) (*rest.Config, cache.Options, error) {
			opts.Scheme = scheme
			opts.Mapper = mgr.GetRESTMapper()
			return c, opts, nil
		},
		cfg,
		cache.Options{
			Scheme: scheme,
			Mapper: mgr.GetRESTMapper(),
		},
	)
	if err := mgr.Add(accessManager); err != nil {
		return fmt.Errorf("adding access manager: %w", err)
	}

	if err := controller.SetupIndexes(mgr); err != nil {
		return fmt.Errorf("setting up indexes: %w", err)
	}

	cosrReconciler := controller.NewCOSRReconciler(
		mgr.GetClient(),
		mgr.GetScheme(),
		mgr.GetRESTMapper(),
		discoveryClient,
		accessManager,
	)
	if err := cosrReconciler.SetupWithManager(mgr); err != nil {
		return fmt.Errorf("setting up COSR controller: %w", err)
	}

	cosReconciler := controller.NewCOSReconciler(mgr.GetClient(), mgr.GetScheme())
	if err := cosReconciler.SetupWithManager(mgr); err != nil {
		return fmt.Errorf("setting up COS controller: %w", err)
	}

	return mgr.Start(cmd.Context())
}
