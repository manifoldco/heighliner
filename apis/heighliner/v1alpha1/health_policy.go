package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HealthPolicy describes the configuration options for the HealthPolicy.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type HealthPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec HealthPolicySpec `json:"spec"`
}

// HealthPolicyList is a list of HealthPolicy CRDs.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type HealthPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []HealthPolicy `json:"items"`
}

// HealthPolicySpec describes the specification which will be used for health
// checks.
type HealthPolicySpec struct {
	LivenessProbe  *v1.Probe `json:"livenessProbe,omitempty"`
	ReadinessProbe *v1.Probe `json:"readinessProbe,omitempty"`
}

// HealthPolicyValidationSchema represents the OpenAPIV3Schema validation for
// the NetworkPolicy CRD.
var HealthPolicyValidationSchema = &v1beta1.CustomResourceValidation{
	OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
		Required: []string{"spec"},
	},
}
