package main

import (
	"log"
	"os"

	flags "github.com/jessevdk/go-flags"
	"github.com/manifoldco/heighliner/pkg/k8sutils"
	"github.com/manifoldco/heighliner/pkg/vsvc"

	"github.com/spf13/cobra"
)

var (
	vsvcCmd = &cobra.Command{
		Use:   "vsvc",
		Short: "Run the VersionedMicroservice Controller",
		RunE:  vsvcCommand,
	}

	vsvcFlags struct {
		Namespace string `long:"namespace" env:"NAMESPACE" description:"The namespace to run the controller in. By default we'll watch all namespaces."`
	}
)

func vsvcCommand(cmd *cobra.Command, args []string) error {
	if _, err := flags.ParseArgs(&vsvcFlags, append(args, os.Args...)); err != nil {
		log.Printf("Could not parse flags: %s", err)
		return err
	}

	cfg, cs, acs, err := k8sutils.Clientset()
	if err != nil {
		log.Printf("Could not get Clientset: %s\n", err)
		return err
	}

	if err := k8sutils.CreateCRD(acs, vsvc.CustomResource); err != nil {
		log.Printf("Could not create CRD: %s\n", err)
		return err
	}

	// TODO(jelmer): maybe make a "ControllerContext" or something of the likes?
	ctrl, err := vsvc.NewController(cfg, cs, vsvcFlags.Namespace)
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
	rootCmd.AddCommand(vsvcCmd)
}
