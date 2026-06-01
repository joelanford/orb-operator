package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func main() {
	cmd := newRootCommand()
	if err := cmd.Execute(); err != nil {
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

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{})
	if err != nil {
		return fmt.Errorf("creating manager: %w", err)
	}
	return mgr.Start(cmd.Context())
}
