package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecurityPolicy describes the configuration options for the SecurityPolicy.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SecurityPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec SecurityPolicySpec `json:"spec"`
}

// SecurityPolicyList is a list of SecurityPolicy CRDs.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SecurityPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []SecurityPolicy `json:"items"`
}

// SecurityPolicySpec describes the specification for Security.
type SecurityPolicySpec struct {
	ServiceAccountName           string                     `json:"serviceAccountName,omitempty"`
	AutomountServiceAccountToken bool                       `json:"automountServiceAccountToken,omitempty"`
	SecurityContext              *corev1.PodSecurityContext `json:"securityContext,omitempty"`
}

// SecurityPolicyValidationSchema represents the OpenAPIV3Schema validation for
// the SecurityPolicy CRD.
var SecurityPolicyValidationSchema = &v1beta1.CustomResourceValidation{
	OpenAPIV3Schema: &v1beta1.JSONSchemaProps{},
}
