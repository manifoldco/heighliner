package main

import (
	"log"
	"os"

	"github.com/manifoldco/heighliner/pkg/svc"

	"github.com/jelmersnoeck/kubekit"
	flags "github.com/jessevdk/go-flags"

	"github.com/spf13/cobra"
)

var (
	svcCmd = &cobra.Command{
		Use:   "msvc",
		Short: "Run the Microservice Controller",
		RunE:  svcCommand,
	}

	svcFlags struct {
		Namespace string `long:"namespace" env:"NAMESPACE" description:"The namespace to run the controller in. By default we'll watch all namespaces."`
	}
)

func svcCommand(cmd *cobra.Command, args []string) error {
	if _, err := flags.ParseArgs(&svcFlags, append(args, os.Args...)); err != nil {
		log.Printf("Could not parse flags: %s", err)
		return err
	}

	cfg, cs, acs, err := kubekit.InClusterClientsets()
	if err != nil {
		log.Printf("Could not get Clientset: %s\n", err)
		return err
	}

	if err := kubekit.CreateCRD(acs, svc.CustomResource); err != nil {
		log.Printf("Could not create Microservice CRD: %s\n", err)
		return err
	}

	if err := kubekit.CreateCRD(acs, svc.AvailabilityPolicyResource); err != nil {
		log.Printf("Could not create AvailabilityPolicy CRD: %s\n", err)
		return err
	}

	if err := kubekit.CreateCRD(acs, svc.HealthPolicyResource); err != nil {
		log.Printf("Could not create HealthPolicy CRD: %s\n", err)
		return err
	}

	if err := kubekit.CreateCRD(acs, svc.SecurityPolicyResource); err != nil {
		log.Printf("Could not create SecurityPolicy CRD: %s\n", err)
		return err
	}

	ctrl, err := svc.NewController(cfg, cs, svcFlags.Namespace)
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
	rootCmd.AddCommand(svcCmd)
}
