package v1alpha1

import (
	"k8s.io/api/rbac/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// GroupName defines the name of the group we'll use for our components.
	GroupName = "hlnr.io"

	// Version defines the version of this API.
	Version = "v1alpha1"
)

var (
	// SchemeBuilder for the svc CRD
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme method for the svc CRD
	AddToScheme = SchemeBuilder.AddToScheme

	// SchemeGroupVersion is the Group Version used for this scheme.
	SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: Version}
)

// Resource gets a Heighliner GroupResource for a specified resource.
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

// addKnownTypes adds the set of types defined in this package to the supplied scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(
		SchemeGroupVersion,
		&Microservice{},
		&MicroserviceList{},
		&VersionedMicroservice{},
		&VersionedMicroserviceList{},
		&ImagePolicy{},
		&ImagePolicyList{},
		&NetworkPolicy{},
		&NetworkPolicyList{},
		&AvailabilityPolicy{},
		&AvailabilityPolicyList{},
		&VersioningPolicy{},
		&VersioningPolicyList{},
		&ConfigPolicy{},
		&ConfigPolicyList{},
		&SecurityPolicy{},
		&SecurityPolicyList{},
		&GitHubRepository{},
		&GitHubRepositoryList{},
		&HealthPolicy{},
		&HealthPolicyList{},
	)

	v1.AddToGroupVersion(scheme, v1alpha1.SchemeGroupVersion)
	return nil
}
