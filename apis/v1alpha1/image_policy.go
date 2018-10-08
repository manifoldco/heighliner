package v1alpha1

import (
	"bytes"
	"errors"
	"strings"
	"text/template"

	"github.com/manifoldco/heighliner/internal/k8sutils"
	"k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const defaultMatch = "{{.Tag}}"

var defaultImagePolicyMatch = &ImagePolicyMatch{
	Name: &ImagePolicyMatchMapping{},
}

var defaultContainerRegistry = &ContainerRegistry{
	Name: "docker",
}

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
	Image             string             `json:"image"`
	ImagePullPolicy   *v1.PullPolicy     `json:"imagePullPolicy"`
	VersioningPolicy  v1.ObjectReference `json:"versioningPolicy"`
	Filter            ImagePolicyFilter  `json:"filter"`
	Match             *ImagePolicyMatch  `json:"match,omitempty"`
	ContainerRegistry *ContainerRegistry `json:"containerRegistry,omitempty"`
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

	// Labels defines matches on image labels.
	Labels map[string]ImagePolicyMatchMapping `json:"labels,omitempty"`
}

// Config returns two booleans indicating if there is name matching and label
// matching in this match.
func (m *ImagePolicyMatch) Config() (bool, bool) {
	if m == nil || (m.Name == nil && len(m.Labels) == 0) {
		m = defaultImagePolicyMatch
	}

	return m.Name != nil, len(m.Labels) > 0
}

// MapName returns the Name mapping for the provided release value.
// It returns an error if the name mapping errors.
func (m *ImagePolicyMatch) MapName(release string) (string, error) {
	if m == nil || m.Name == nil {
		m = defaultImagePolicyMatch
	}

	mapped, err := m.Name.Map(release)
	return mapped, err
}

// Matches returns a bool indicating if the provided image tag and labels match
// this match stanza. It returns an error if any of the match mappings error.
// If m is nil or the zero value, it uses the default match value of:
//   match:
//     name:
//       from: "{{.Tag}}"
//       to: "{{.Tag}}"
func (m *ImagePolicyMatch) Matches(release, tag string, labels map[string]string) (bool, error) {
	if m == nil || (m.Name == nil && len(m.Labels) == 0) {
		m = defaultImagePolicyMatch
	}

	if m.Name != nil {
		mapped, err := m.Name.Map(release)
		if err != nil {
			return false, err
		}
		if mapped != tag {
			return false, nil
		}
	}

	for l, lm := range m.Labels {
		t, ok := labels[l]
		if !ok {
			return false, nil
		}

		mapped, err := lm.Map(release)
		if err != nil {
			return false, err
		}
		if mapped != t {
			return false, nil
		}
	}

	return true, nil
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
	Pinned *SemVerRelease      `json:"pinned,omitempty"`
}

// ContainerRegistry will define how to fetch images from a container registry.
// Docker Hub is the only one supported at the moment.
type ContainerRegistry struct {
	Name             string                    `json:"name"`
	ImagePullSecrets []v1.LocalObjectReference `json:"imagePullSecrets"`
}

// Registry returns the name of the container registry. If nil, returns the
// default value.
func (c *ContainerRegistry) Registry() string {
	if c == nil {
		c = defaultContainerRegistry
	}

	if c.Name == "" {
		return defaultContainerRegistry.Name
	}

	return c.Name
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
					"filter":            filterValidationSchema,
					"match":             matchValidationSchema,
					"containerRegistry": containerRegistryValidationSchema,
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
		{
			Required: []string{"pinned"},
			Properties: map[string]v1beta1.JSONSchemaProps{
				"pinned": {
					Required: []string{"version"},
					Properties: map[string]v1beta1.JSONSchemaProps{
						"version": {
							Type: "string",
						},
					},
				},
			},
		},
	},
}

var matchValidationSchema = v1beta1.JSONSchemaProps{
	Type: "object",
	Properties: map[string]v1beta1.JSONSchemaProps{
		"name": mappingValidationSchema,
		"labels": {
			Type: "object",
		},
	},
}

var mappingValidationSchema = v1beta1.JSONSchemaProps{
	Type: "object",
	Properties: map[string]v1beta1.JSONSchemaProps{
		"from": {Type: "string"},
		"to":   {Type: "string"},
	},
}

var containerRegistryValidationSchema = v1beta1.JSONSchemaProps{
	Type: "object",
	Properties: map[string]v1beta1.JSONSchemaProps{
		"name": {
			Enum: []v1beta1.JSON{
				{Raw: k8sutils.JSONBytes("docker")},
			},
		},
	},
}
