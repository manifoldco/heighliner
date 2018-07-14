package v1alpha1

import (
	"github.com/manifoldco/heighliner/internal/k8sutils"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kube-openapi/pkg/util/proto"
)

// VersioningPolicy describes the configuration options for the VersioningPolicy.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VersioningPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec VersioningPolicySpec `json:"spec"`
}

// VersioningPolicyList is a list of VersioningPolicy CRDs.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VersioningPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []VersioningPolicy `json:"items"`
}

// VersioningPolicySpec describes the specification for Versioning.
type VersioningPolicySpec struct {
	SemVer *SemVerSource `json:"semVer"`
}

type (
	// SemVerLevel indicates a level which we want to monitor the image registry
	// for. It should be in the format of format.
	// Examples:
	// v1.2.3, v1.2.4-rc.0, v1.2.4-pr.1+201804011533
	// 1.2.3, 1.2.4-rc.0, 1.2.4-pr.1+201804011533
	SemVerLevel string

	// SemVerVersion represents the type of version we want to monitor for.
	SemVerVersion string
)

var (
	// SemVerLevelRelease is used for a release that is ready to be rolled out
	// to production.
	SemVerLevelRelease SemVerLevel = "release"

	// SemVerLevelReleaseCandidate is used for a release-candidate that is ready
	// for QA.
	SemVerLevelReleaseCandidate SemVerLevel = "candidate"

	// SemVerLevelPreview is used for a preview release. This is generally
	// associated with development deploys.
	SemVerLevelPreview SemVerLevel = "preview"

	// SemVerVersionMajor indicates that we will release major, minor and patch
	// releases.
	SemVerVersionMajor SemVerVersion = "major"

	// SemVerVersionMinor indicates that we will release minor and patch
	// releases.
	SemVerVersionMinor SemVerVersion = "minor"

	// SemVerVersionPatch indicates that we will release only patch releases.
	SemVerVersionPatch SemVerVersion = "patch"
)

// SemVerSource is a versioning policy based on semver.
// When semver is selected, Heighliner can watch for images on 3 different
// levels.
type SemVerSource struct {
	// Version is the type of Version we want to start tracking with this
	// Policy.
	Version SemVerVersion `json:"version"`

	// Level is the level we want to fetch images for this Microservice for.
	Level SemVerLevel `json:"level"`

	// MinVersion is the minimum version that we want to track for this Policy.
	MinVersion string `json:"minVersion"`
}

// VersioningPolicyValidationSchema represents the OpenAPIV3Schema validation for
// the NetworkPolicy CRD.
var VersioningPolicyValidationSchema = apiextv1beta1.JSONSchemaProps{
	Properties: map[string]apiextv1beta1.JSONSchemaProps{
		"semVer": {
			Properties: map[string]apiextv1beta1.JSONSchemaProps{
				"version": {
					Type: proto.String,
					Enum: []apiextv1beta1.JSON{
						{Raw: k8sutils.JSONBytes(SemVerVersionMajor)},
						{Raw: k8sutils.JSONBytes(SemVerVersionMinor)},
						{Raw: k8sutils.JSONBytes(SemVerVersionPatch)},
					},
				},
				"level": {
					Type: proto.String,
					Enum: []apiextv1beta1.JSON{
						{Raw: k8sutils.JSONBytes(SemVerLevelRelease)},
						{Raw: k8sutils.JSONBytes(SemVerLevelReleaseCandidate)},
						{Raw: k8sutils.JSONBytes(SemVerLevelPreview)},
					},
				},
			},
			Required: []string{"version", "level"},
		},
	},
}
