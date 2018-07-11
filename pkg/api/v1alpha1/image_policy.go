package v1alpha1

import (
	"bytes"
	"errors"
	"strings"
	"text/template"

	"k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const defaultMatch = "{{.Tag}}"

var (
	errTagNotFound = errors.New("no Tag template value found")
	errTooManyTags = errors.New("only one Tag template must be provided")
	errNoMatch     = errors.New("no match found for from template")
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
	Match            *ImagePolicyMatch         `json:"match,omitempty"`
}

// ImagePolicyStatus represents the latest version of the ImagePolicy that
// matches the VersioningPolicy associated with it.
// The Status will be used by the Microservice component to build the actual
// Deployment.
type ImagePolicyStatus struct {
	Releases []Release `json:"releases"`
}

// ImagePolicyMatch defines how a release is matched to an image tag.
type ImagePolicyMatch struct {
	// Name defines a match on the image tag name.
	Name *ImagePolicyMatchMapping `json:"name,omitempty"`
}

// ImagePolicyMatchMapping defines how a release is transformed to match an
// image tag or label value
type ImagePolicyMatchMapping struct {

	// From transforms the release value, extracting the tag. The value is
	// formatted as a Go template string, and matches on on `{{.Tag}}`. If no
	// value is provided, "{{.Tag}}" is assumed.
	From string `json:"from,omitempty"`

	// To formats the extrated Tag value from From. If no value is provided,
	// "{{.Tag}}" is assumed.
	To string `json:"to,omitempty"`
}

// Map translates the provided from string to a to string. It returns an
// error if either the configured From or To value cannot be compiled.
func (m *ImagePolicyMatchMapping) Map(from string) (string, error) {
	mfrom := m.From
	if mfrom == "" {
		mfrom = defaultMatch
	}

	parts := strings.Split(mfrom, defaultMatch)
	switch len(parts) {
	case 1:
		return "", errTagNotFound
	case 2: //ok
	default:
		return "", errTooManyTags
	}

	if parts[0] != "" {
		newFrom := strings.TrimPrefix(from, parts[0])
		if from == newFrom {
			return "", errNoMatch
		}
		from = newFrom
	}

	if parts[1] != "" {
		newFrom := strings.TrimSuffix(from, parts[1])
		if from == newFrom {
			return "", errNoMatch
		}
		from = newFrom
	}

	to := m.To
	if to == "" {
		to = defaultMatch
	}

	toT, err := template.New("").Parse(to)

	if err != nil {
		return "", err
	}

	w := &bytes.Buffer{}
	if err := toT.Execute(w, struct{ Tag string }{from}); err != nil {
		return "", err
	}

	if w.String() == m.To {
		return "", errTagNotFound
	}

	return w.String(), nil
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
					"match":  matchValidationSchema,
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

var matchValidationSchema = v1beta1.JSONSchemaProps{
	Type: "object",
	Properties: map[string]v1beta1.JSONSchemaProps{
		"name": mappingValidationSchema,
	},
}

var mappingValidationSchema = v1beta1.JSONSchemaProps{
	Type: "object",
	Properties: map[string]v1beta1.JSONSchemaProps{
		"from": {Type: "string"},
		"to":   {Type: "string"},
	},
}
