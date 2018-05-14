package githubpolicy

import (
	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"

	"github.com/jelmersnoeck/kubekit"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

var (
	// GitHubPolicyResource describes the CRD configuration for the GitHubPolicy CRD.
	GitHubPolicyResource = kubekit.CustomResource{
		Name:       "githubpolicy",
		Plural:     "githubpolicies",
		Group:      v1alpha1.GroupName,
		Version:    v1alpha1.Version,
		Scope:      v1beta1.NamespaceScoped,
		Aliases:    []string{"ghp"},
		Object:     &v1alpha1.GitHubPolicy{},
		Validation: v1alpha1.GitHubPolicyValidationSchema,
	}
)
