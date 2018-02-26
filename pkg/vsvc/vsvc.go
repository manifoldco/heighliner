// Package vsvc manages Versioned Microservices.
package vsvc

import (
	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
	"github.com/manifoldco/heighliner/pkg/k8sutils"

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
	CustomResource = k8sutils.CustomResource{
		Name:    CustomResourceName,
		Plural:  CustomResourceNamePlural,
		Group:   v1alpha1.GroupName,
		Version: v1alpha1.Version,
		Scope:   v1beta1.NamespaceScoped,
		Aliases: []string{"vsvc"},
		Object:  &v1alpha1.VersionedMicroservice{},
	}
)
