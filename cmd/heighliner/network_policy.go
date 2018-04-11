package main

import (
	"log"
	"os"

	"github.com/jelmersnoeck/kubekit"
	flags "github.com/jessevdk/go-flags"
	"github.com/manifoldco/heighliner/pkg/networkpolicy"

	"github.com/spf13/cobra"
)

var (
	npwCmd = &cobra.Command{
		Use:     "network-policy-watcher",
		Aliases: []string{"npw"},
		Short:   "Run the NetworkPolicy Watcher",
		RunE:    npwCommand,
	}

	npwFlags struct {
		Namespace string `long:"namespace" env:"NAMESPACE" description:"The namespace we'll watch for CRDs. By default we'll watch all namespaces."`
	}
)

func npwCommand(cmd *cobra.Command, args []string) error {
	if _, err := flags.ParseArgs(&npwFlags, append(args, os.Args...)); err != nil {
		log.Printf("Could not parse flags: %s", err)
		return err
	}

	cfg, cs, acs, err := kubekit.InClusterClientsets()
	if err != nil {
		log.Printf("Could not get Clientset: %s\n", err)
		return err
	}

	if err := kubekit.CreateCRD(acs, networkpolicy.NetworkPolicyResource); err != nil {
		log.Printf("Could not create NetworkPolicy CRD: %s\n", err)
		return err
	}

	if err := kubekit.CreateCRD(acs, networkpolicy.VersioningPolicyResource); err != nil {
		log.Printf("Could not create VersioningPolicy CRD: %s\n", err)
		return err
	}

	ctrl, err := networkpolicy.NewController(cfg, cs, npwFlags.Namespace)
	if err != nil {
		log.Printf("Could not create controller: %s\n", err)
		return err
	}

	if err := ctrl.Run(); err != nil {
		log.Printf("Error running controller: %s\n", err)
		return err
	}

	return nil
}

func init() {
	rootCmd.AddCommand(npwCmd)
}
