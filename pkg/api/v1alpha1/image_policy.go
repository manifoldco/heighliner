package v1alpha1

import (
	"k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	Image            string                    `json:"image"`
	ImagePullSecrets []v1.LocalObjectReference `json:"imagePullSecrets"`
	ImagePullPolicy  *v1.PullPolicy            `json:"imagePullPolicy"`
	VersioningPolicy v1.ObjectReference        `json:"versioningPolicy"`
	Filter           ImagePolicyFilter         `json:"filter"`
}

// ImagePolicyStatus represents the latest version of the ImagePolicy that
// matches the VersioningPolicy associated with it.
// The Status will be used by the Microservice component to build the actual
// Deployment.
type ImagePolicyStatus struct {
	Releases []Release `json:"releases"`
}

// ImagePolicyFilter will define how we can filter where images come from
type ImagePolicyFilter struct {
	GitHub *v1.ObjectReference `json:"github,omitempty"`
}

// ImagePolicyValidationSchema represents the OpenAPIV3Schema validation for
// the NetworkPolicy CRD.
var ImagePolicyValidationSchema = &v1beta1.CustomResourceValidation{
	OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
		Required: []string{"spec"},
		Properties: map[string]v1beta1.JSONSchemaProps{
			"spec": {
				Required: []string{"image", "versioningPolicy", "filter"},
				Properties: map[string]v1beta1.JSONSchemaProps{
					"filter": filterValidationSchema,
				},
			},
			"status": ReleaseValidationSchema,
		},
	},
}

var filterValidationSchema = v1beta1.JSONSchemaProps{
	OneOf: []v1beta1.JSONSchemaProps{
		{
			Required: []string{"github"},
		},
	},
}
