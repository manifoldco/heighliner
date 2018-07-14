package main

import (
	"log"
	"os"

	"github.com/jelmersnoeck/kubekit"
	flags "github.com/jessevdk/go-flags"
	"github.com/manifoldco/heighliner/internal/imagepolicy"

	"github.com/spf13/cobra"
)

var (
	ipcCmd = &cobra.Command{
		Use:   "ipc",
		Short: "Run the Image Policy Controller",
		RunE:  ipcCommand,
	}

	ipcFlags struct {
		Namespace string `long:"namespace" env:"NAMESPACE" description:"The namespace to run the controller in. By default we'll watch all namespaces."`
	}
)

func ipcCommand(cmd *cobra.Command, args []string) error {
	if _, err := flags.ParseArgs(&ipcFlags, append(args, os.Args...)); err != nil {
		log.Printf("Could not parse flags: %s", err)
		return err
	}

	cfg, cs, acs, err := kubekit.InClusterClientsets()
	if err != nil {
		log.Printf("Could not get Clientset: %s\n", err)
		return err
	}

	if err := kubekit.CreateCRD(acs, imagepolicy.ImagePolicyResource); err != nil {
		log.Printf("Could not create ImagePolicy CRD: %s\n", err)
		return err
	}

	ctrl, err := imagepolicy.NewController(cfg, cs, ipcFlags.Namespace)
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
	rootCmd.AddCommand(ipcCmd)
}
