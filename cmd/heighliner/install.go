package main

import (
	"bytes"
	"html/template"
	"os"

	"github.com/spf13/cobra"
)

const namespaceName = "docs/kube/00-heighliner-namespace.yaml"

var (
	installCmd = &cobra.Command{
		Use:   "install",
		Short: "Create installation Manifests to install Heighliner in your Kubernetes cluster.",
		RunE:  installCommand,
	}

	installFlags struct {
		GitHubCallbackDomain string
		Version              string
		DNSProvider          string
	}
)

func installCommand(cmd *cobra.Command, args []string) error {
	data := bytes.NewBuffer(nil)

	// the namespace should be created first, otherwise all other components
	// won't be installed.
	nsData, err := Asset(namespaceName)
	if err != nil {
		return err
	}
	data.Write(nsData)

	for _, name := range AssetNames() {
		if name == namespaceName {
			continue
		}

		assetData, err := Asset(name)
		if err != nil {
			return err
		}

		data.Write([]byte("\n---\n"))
		data.Write(assetData)
	}

	tpl, err := template.New("heighliner-install").Parse(data.String())
	if err != nil {
		return err
	}

	if err := tpl.Execute(os.Stdout, installFlags); err != nil {
		return err
	}

	return nil
}

func init() {
	installCmd.Flags().StringVar(&installFlags.GitHubCallbackDomain, "github-callback-domain", "", "The domain used for GitHub to do callbacks to")
	installCmd.MarkFlagRequired("github-callback-url")
	installCmd.Flags().StringVar(&installFlags.Version, "version", "latest", "The version of Heighliner to install")
	installCmd.Flags().StringVar(&installFlags.DNSProvider, "dns-provider", "route53-dns", "The DNS Provider configured through ExternalDNS")

	rootCmd.AddCommand(installCmd)
}
