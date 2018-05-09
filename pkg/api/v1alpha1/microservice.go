package v1alpha1

import (
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core "k8s.io/kubernetes/pkg/apis/core"
)

// Microservice represents the definition which we'll use to define a deployable
// microservice.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Microservice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   MicroserviceSpec   `json:"spec"`
	Status MicroserviceStatus `json:"status"`
}

// MicroserviceList is a list of Microservices.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MicroserviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Microservice `json:"items"`
}

// MicroserviceSpec represents the specification for a Microservice. It houses
// all the policies which we'll use to build a VersionedMicroservice.
type MicroserviceSpec struct {
	ImagePolicy        core.LocalObjectReference `json:"imagePolicy"`
	AvailabilityPolicy core.LocalObjectReference `json:"availabilityPolicy,omitempty"`
	ConfigPolicy       core.LocalObjectReference `json:"configPolicy,omitempty"`
	SecurityPolicy     core.LocalObjectReference `json:"securityPolicy,omitempty"`
}

// MicroserviceStatus represents the status a specific Microservice is in.
type MicroserviceStatus struct {
	Releases []Release `json:"releases"`
}

// MicroserviceValidationSchema represents the OpenAPIV3Scheme which
// defines the validation for the MicroserviceSpec.
var MicroserviceValidationSchema = &v1beta1.CustomResourceValidation{
	OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
		Properties: map[string]v1beta1.JSONSchemaProps{
			"spec": {
				Required: []string{"imagePolicy"},
				Properties: map[string]v1beta1.JSONSchemaProps{
					"imagePolicy": requiredObjectReference,
				},
			},
			"status": ReleaseValidationSchema,
		},
		Required: []string{"spec"},
	},
}

var requiredObjectReference = v1beta1.JSONSchemaProps{
	Required: []string{"name"},
}
