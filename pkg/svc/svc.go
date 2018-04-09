// Package svc manages Microservices.
package svc

import (
	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"

	"github.com/jelmersnoeck/kubekit"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

const (
	// CustomResourceName is the name we'll use for the Microservice CRD.
	CustomResourceName = "microservice"

	// CustomResourceNamePlural is the plural version of CustomResourceName.
	CustomResourceNamePlural = "microservices"
)

var (
	// CustomResource describes the CRD configuration for the Microservice CRD.
	CustomResource = kubekit.CustomResource{
		Name:    CustomResourceName,
		Plural:  CustomResourceNamePlural,
		Group:   v1alpha1.GroupName,
		Version: v1alpha1.Version,
		Scope:   v1beta1.NamespaceScoped,
		Aliases: []string{"msvc"},
		Object:  &v1alpha1.Microservice{},
		Validation: &v1beta1.CustomResourceValidation{
			OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
				Properties: map[string]v1beta1.JSONSchemaProps{
					"spec": v1alpha1.MicroserviceValidationSchema,
				},
			},
		},
	}

	// NetworkPolicyResource describes the CRD configuration for the
	// NetworkPolicy CRD.
	NetworkPolicyResource = kubekit.CustomResource{
		Name:    "networkpolicy",
		Plural:  "networkpolicies",
		Group:   v1alpha1.GroupName,
		Version: v1alpha1.Version,
		Scope:   v1beta1.NamespaceScoped,
		Aliases: []string{"np"},
		Object:  &v1alpha1.NetworkPolicy{},
		Validation: &v1beta1.CustomResourceValidation{
			OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
				Properties: map[string]v1beta1.JSONSchemaProps{
					"spec": v1alpha1.NetworkPolicyValidationSchema,
				},
			},
		},
	}

	// AvailabilityPolicyResource describes the CRD configuration for the
	// AvailabilityPolicy CRD.
	AvailabilityPolicyResource = kubekit.CustomResource{
		Name:    "availabilitypolicy",
		Plural:  "availabilitypolicies",
		Group:   v1alpha1.GroupName,
		Version: v1alpha1.Version,
		Scope:   v1beta1.NamespaceScoped,
		Aliases: []string{"ap"},
		Object:  &v1alpha1.AvailabilityPolicy{},
		Validation: &v1beta1.CustomResourceValidation{
			OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
				Properties: map[string]v1beta1.JSONSchemaProps{
					"spec": v1alpha1.AvailabilityPolicyValidationSchema,
				},
			},
		},
	}

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

	// SecurityPolicyResource describes the CRD configuration for the
	// SecurityPolicy CRD.
	SecurityPolicyResource = kubekit.CustomResource{
		Name:       "securitypolicy",
		Plural:     "securitypolicies",
		Group:      v1alpha1.GroupName,
		Version:    v1alpha1.Version,
		Scope:      v1beta1.NamespaceScoped,
		Aliases:    []string{"sp"},
		Object:     &v1alpha1.SecurityPolicy{},
		Validation: v1alpha1.SecurityPolicyValidationSchema,
	}
)
