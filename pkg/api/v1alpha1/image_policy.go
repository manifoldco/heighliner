package v1alpha1

import (
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

// ImagePolicy describes the configuration options for the ImagePolicy.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ImagePolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec ImageSpec `json:"spec"`
}

// ImagePolicyList is a list of ImagePolicy CRDs.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ImagePolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ImagePolicy `json:"items"`
}

// ImageSpec describes the specification for Image.
type ImageSpec struct {
	Image            string                      `json:"image"`
	ImagePullSecrets []core.LocalObjectReference `json:"imagePullSecrets"`
	VersioningPolicy core.LocalObjectReference   `json:"versioningPolicy"`
}

// ImagePolicyValidationSchema represents the OpenAPIV3Schema validation for
// the NetworkPolicy CRD.
var ImagePolicyValidationSchema = apiextv1beta1.JSONSchemaProps{
	Required: []string{"image", "versioningPolicy"},
}
