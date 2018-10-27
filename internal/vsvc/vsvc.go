// Package vsvc manages Versioned Microservices.
package vsvc

import (
	"github.com/manifoldco/heighliner/apis/heighliner/v1alpha1"

	"github.com/jelmersnoeck/kubekit"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

const (
	// CustomResourceName is the name we'll use for the Versioned Microservice
	// CRD.
	CustomResourceName = "versionedmicroservice"

	// CustomResourceNamePlural is the plural version of CustomResourceName.
	CustomResourceNamePlural = "versionedmicroservices"
)

var (
	// CustomResource describes the CRD configuration for the VersionedMicroservice CRD.
	CustomResource = kubekit.CustomResource{
		Name:    CustomResourceName,
		Plural:  CustomResourceNamePlural,
		Group:   v1alpha1.GroupName,
		Version: v1alpha1.Version,
		Scope:   v1beta1.NamespaceScoped,
		Aliases: []string{"vsvc"},
		Object:  &v1alpha1.VersionedMicroservice{},
		Validation: &v1beta1.CustomResourceValidation{
			OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
				Properties: map[string]v1beta1.JSONSchemaProps{
					"spec": v1alpha1.VersionedMicroserviceValidationSchema,
				},
			},
		},
	}
)
