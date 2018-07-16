package imagepolicy

import (
	"github.com/jelmersnoeck/kubekit"
	"github.com/manifoldco/heighliner/apis/v1alpha1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

var (
	// ImagePolicyResource describes the CRD configuration for the ImagePolicy CRD.
	ImagePolicyResource = kubekit.CustomResource{
		Name:       "imagepolicy",
		Plural:     "imagepolicies",
		Group:      v1alpha1.GroupName,
		Version:    v1alpha1.Version,
		Scope:      v1beta1.NamespaceScoped,
		Aliases:    []string{"ip"},
		Object:     &v1alpha1.ImagePolicy{},
		Validation: v1alpha1.ImagePolicyValidationSchema,
	}
)
