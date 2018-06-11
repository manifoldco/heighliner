package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigPolicy describes the configuration options for the ConfigPolicy.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ConfigPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   ConfigPolicySpec   `json:"spec"`
	Status ConfigPolicyStatus `json:"status"`
}

// ConfigPolicyList is a list of ConfigPolicy CRDs.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ConfigPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ConfigPolicy `json:"items"`
}

// ConfigPolicySpec describes the specification for Config.
type ConfigPolicySpec struct {
	Env          []corev1.EnvVar        `json:"env,omitempty"`
	EnvFrom      []corev1.EnvFromSource `json:"envFrom,omitempty"`
	VolumeMounts []corev1.VolumeMount   `json:"volumeMounts,omitempty"`
	Volumes      []corev1.Volume        `json:"volumes,omitempty"`
}

// ConfigPolicyStatus represents the current status of a ConfigPolicy.
type ConfigPolicyStatus struct {
	LastUpdatedTime metav1.Time `json:"lastUpdatedTime"`
	Hashed          string      `json:"hashed"`
}

// ConfigPolicyValidationSchema represents the OpenAPIV3Schema validation for
// the ConfigPolicy CRD.
var ConfigPolicyValidationSchema = &v1beta1.CustomResourceValidation{
	OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
		Properties: map[string]v1beta1.JSONSchemaProps{
			"status": {
				Required: []string{"lastUpdatedTime", "hashed"},
			},
		},
	},
}
