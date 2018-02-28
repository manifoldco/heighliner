package vsvc

import (
	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	// SchemeBuilder for the vsvc CRD
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme method for the vsvc CRD
	AddToScheme = SchemeBuilder.AddToScheme
)

// addKnownTypes adds the set of types defined in this package to the supplied scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(
		v1alpha1.SchemeGroupVersion,
		&v1alpha1.VersionedMicroservice{},
		&v1alpha1.VersionedMicroserviceList{},
	)

	v1.AddToGroupVersion(scheme, v1alpha1.SchemeGroupVersion)
	return nil
}
