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

	// NetworkPolicyResourceName is the name used for the NetworkPolicy CRD.
	NetworkPolicyResourceName = "networkpolicy"

	// NetworkPolicyNamePlural is the plural version of NetworkPolicyResourceName.
	NetworkPolicyNamePlural = "networkpolicies"

	// AvailabilityPolicyResourceName is the name used for the AvailabilityPolicy CRD.
	AvailabilityPolicyResourceName = "availabilitypolicy"

	// AvailabilityPolicyNamePlural is the plural version of AvailabilityPolicyResourceName.
	AvailabilityPolicyNamePlural = "availabilitypolicies"
)

var (
	// CustomResource describes the CRD configuration for the Microservice CRD.
	CustomResource = kubekit.CustomResource{
		Name:    CustomResourceName,
		Plural:  CustomResourceNamePlural,
		Group:   v1alpha1.GroupName,
		Version: v1alpha1.Version,
		Scope:   v1beta1.NamespaceScoped,
		Aliases: []string{"svc"},
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
		Name:    NetworkPolicyResourceName,
		Plural:  NetworkPolicyNamePlural,
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
		Name:    AvailabilityPolicyResourceName,
		Plural:  AvailabilityPolicyNamePlural,
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
)
