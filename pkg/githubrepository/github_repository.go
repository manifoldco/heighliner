package githubrepository

import (
	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"

	"github.com/jelmersnoeck/kubekit"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

var (
	// GitHubRepositoryResource describes the CRD configuration for the
	// GitHubRepository CRD.
	GitHubRepositoryResource = kubekit.CustomResource{
		Name:       "githubrepository",
		Plural:     "githubrepositories",
		Group:      v1alpha1.GroupName,
		Version:    v1alpha1.Version,
		Scope:      v1beta1.NamespaceScoped,
		Aliases:    []string{"ghr"},
		Object:     &v1alpha1.GitHubRepository{},
		Validation: v1alpha1.GitHubRepositoryValidationSchema,
	}
)
