package main

import (
	"log"
	"os"

	"github.com/jelmersnoeck/kubekit"
	flags "github.com/jessevdk/go-flags"
	"github.com/manifoldco/heighliner/internal/configpolicy"

	"github.com/spf13/cobra"
)

var (
	cpwCmd = &cobra.Command{
		Use:     "config-policy-watcher",
		Aliases: []string{"cpw"},
		Short:   "Run the ConfigPolicy Watcher",
		RunE:    cpwCommand,
	}

	cpwFlags struct {
		Namespace string `long:"namespace" env:"NAMESPACE" description:"The namespace to run the controller in. By default we'll watch all namespaces."`
	}
)

func cpwCommand(cmd *cobra.Command, args []string) error {
	if _, err := flags.ParseArgs(&cpwFlags, append(args, os.Args...)); err != nil {
		log.Printf("Could not parse flags: %s", err)
		return err
	}

	cfg, cs, acs, err := kubekit.InClusterClientsets()
	if err != nil {
		log.Printf("Could not get Clientset: %s\n", err)
		return err
	}

	if err := kubekit.CreateCRD(acs, configpolicy.ConfigPolicyResource); err != nil {
		log.Printf("Could not create ConfigPolicy CRD: %s\n", err)
		return err
	}

	ctrl, err := configpolicy.NewController(cfg, cs, cpwFlags.Namespace)
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
	rootCmd.AddCommand(cpwCmd)
}
