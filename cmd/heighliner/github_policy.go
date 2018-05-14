package main

import (
	"log"
	"os"

	"github.com/jelmersnoeck/kubekit"
	flags "github.com/jessevdk/go-flags"
	"github.com/manifoldco/heighliner/pkg/githubpolicy"

	"github.com/spf13/cobra"
)

var (
	ghpcCmd = &cobra.Command{
		Use:     "github-policy-controller",
		Aliases: []string{"ghpc"},
		Short:   "Run the GitHubPolicy Controller",
		RunE:    ghpcCommand,
	}

	ghpcFlags struct {
		Namespace string `long:"namespace" env:"NAMESPACE" description:"The namespace we'll watch for CRDs. By default we'll watch all namespaces."`
	}
)

func ghpcCommand(cmd *cobra.Command, args []string) error {
	if _, err := flags.ParseArgs(&ghpcFlags, append(args, os.Args...)); err != nil {
		log.Printf("Could not parse flags: %s", err)
		return err
	}

	cfg, cs, acs, err := kubekit.InClusterClientsets()
	if err != nil {
		log.Printf("Could not get Clientset: %s\n", err)
		return err
	}

	if err := kubekit.CreateCRD(acs, githubpolicy.GitHubPolicyResource); err != nil {
		log.Printf("Could not create GitHubPolicy CRD: %s\n", err)
		return err
	}

	ctrl, err := githubpolicy.NewController(cfg, cs, ghpcFlags.Namespace)
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
	rootCmd.AddCommand(ghpcCmd)
}
