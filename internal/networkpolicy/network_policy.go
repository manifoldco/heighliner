package networkpolicy

import (
	"github.com/manifoldco/heighliner/apis/v1alpha1"

	"github.com/jelmersnoeck/kubekit"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

var (
	// NetworkPolicyResource describes the CRD networkuration for the
	// NetworkPolicy CRD.
	NetworkPolicyResource = kubekit.CustomResource{
		Name:    "networkpolicy",
		Plural:  "networkpolicies",
		Group:   v1alpha1.GroupName,
		Version: v1alpha1.Version,
		Scope:   v1beta1.NamespaceScoped,
		// TODO(jelmer): find appropriate alias that doesn't conflict with the
		// default kubernetes network policy. Maybe needs a full rename.
		Aliases:    []string{},
		Object:     &v1alpha1.NetworkPolicy{},
		Validation: v1alpha1.NetworkPolicyValidationSchema,
	}

	// VersioningPolicyResource describes the CRD configuration for the
	// VersioningPolicy CRD.
	VersioningPolicyResource = kubekit.CustomResource{
		Name:    "versioningpolicy",
		Plural:  "versioningpolicies",
		Group:   v1alpha1.GroupName,
		Version: v1alpha1.Version,
		Scope:   v1beta1.NamespaceScoped,
		Aliases: []string{"vp"},
		Object:  &v1alpha1.VersioningPolicy{},
		Validation: &v1beta1.CustomResourceValidation{
			OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
				Properties: map[string]v1beta1.JSONSchemaProps{
					"spec": v1alpha1.VersioningPolicyValidationSchema,
				},
			},
		},
	}
)
