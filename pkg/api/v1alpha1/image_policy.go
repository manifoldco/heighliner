package v1alpha1

import (
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

// ImagePolicy describes the configuration options for the ImagePolicy.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ImagePolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   ImagePolicySpec   `json:"spec"`
	Status ImagePolicyStatus `json:"status"`
}

// ImagePolicyList is a list of ImagePolicy CRDs.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ImagePolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ImagePolicy `json:"items"`
}

// ImagePolicySpec describes the specification for Image.
type ImagePolicySpec struct {
	Image            string                      `json:"image"`
	ImagePullSecrets []core.LocalObjectReference `json:"imagePullSecrets"`
	VersioningPolicy core.LocalObjectReference   `json:"versioningPolicy"`
}

// ImagePolicyStatus represents the latest version of the ImagePolicy that
// matches the VersioningPolicy associated with it.
// The Status will be used by the Microservice component to build the actual
// Deployment.
type ImagePolicyStatus struct {
	Image string `json:"image"`
}

// ImagePolicyValidationSchema represents the OpenAPIV3Schema validation for
// the NetworkPolicy CRD.
var ImagePolicyValidationSchema = &v1beta1.CustomResourceValidation{
	OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
		Properties: map[string]v1beta1.JSONSchemaProps{
			"spec": {
				Required: []string{"image", "versioningPolicy"},
			},
			"status": {
				Required: []string{"image"},
			},
		},
		Required: []string{"spec"},
	},
}
