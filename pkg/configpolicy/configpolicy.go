package configpolicy

import (
	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"

	"github.com/jelmersnoeck/kubekit"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

var (
	// ConfigPolicyResource describes the CRD configuration for the
	// ConfigPolicy CRD.
	ConfigPolicyResource = kubekit.CustomResource{
		Name:       "configpolicy",
		Plural:     "configpolicies",
		Group:      v1alpha1.GroupName,
		Version:    v1alpha1.Version,
		Scope:      v1beta1.NamespaceScoped,
		Aliases:    []string{"cp"},
		Object:     &v1alpha1.ConfigPolicy{},
		Validation: v1alpha1.ConfigPolicyValidationSchema,
	}
)
