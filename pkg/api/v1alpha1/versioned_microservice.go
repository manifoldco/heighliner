package v1alpha1

import (
	"github.com/manifoldco/heighliner/pkg/k8sutils"

	corev1 "k8s.io/api/core/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VersionedMicroservice represents the combined state of different components
// in time which form a single Microservice.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VersionedMicroservice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec VersionedMicroserviceSpec `json:"spec"`
}

// VersionedMicroserviceList is a list of VersionedMicroservices.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VersionedMicroserviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []VersionedMicroservice `json:"items"`
}

// VersionedMicroserviceSpec represents the specification for a
// VersionedMicroservice.
type VersionedMicroserviceSpec struct {
	Availability *AvailabilityPolicySpec `json:"availability,omitempty"`
	Network      *NetworkPolicySpec      `json:"network,omitempty"`
	Config       *ConfigPolicySpec       `json:"config,omitempty"`
	Security     *SecurityPolicySpec     `json:"security,omitempty"`
	Containers   []corev1.Container      `json:"containers"`
}

// VersionedMicroserviceValidationSchema represents the OpenAPIV3Scheme which
// defines the validation for the VersionedMicroserviceSpec.
var VersionedMicroserviceValidationSchema = apiextv1beta1.JSONSchemaProps{
	Properties: map[string]apiextv1beta1.JSONSchemaProps{
		"availability": AvailabilityPolicyValidationSchema,
		"network":      NetworkPolicyValidationSchema,
		"config":       *ConfigPolicyValidationSchema.OpenAPIV3Schema,
		"security":     *SecurityPolicyValidationSchema.OpenAPIV3Schema,
		"containers": {
			MinItems: k8sutils.PtrInt64(1),
		},
	},
	Required: []string{
		"containers",
	},
}
