package v1alpha1

import (
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kube-openapi/pkg/util/proto"
)

// VersioningPolicy describes the configuration options for the VersioningPolicy.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VersioningPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec VersioningSpec `json:"spec"`
}

// VersioningPolicyList is a list of VersioningPolicy CRDs.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VersioningPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []VersioningPolicy `json:"items"`
}

// VersioningSpec describes the specification for Versioning.
type VersioningSpec struct {
	SemverRef *SemverSource `json:"semverRef"`
}

type (
	// SemverLevel indicates a level which we want to monitor the image registry
	// for. It should be in the format of format.
	// Examples:
	// v1.2.3, v1.2.4-rc.0, v1.2.4-pr.1
	// 1.2.3, 1.2.4-rc.0, 1.2.4-pr.1
	SemverLevel string

	// SemverVersion represents the type of version we want to monitor for.
	SemverVersion string
)

var (
	// SemverLevelRelease is used for a release that is ready to be rolled out
	// to production.
	SemverLevelRelease = "release"

	// SemverLevelReleaseCandidate is used for a release-candidate that is ready
	// for QA.
	SemverLevelReleaseCandidate = "rc"

	// SemverLevelPreview is used for a preview release. This is generally
	// associated with development deploys.
	SemverLevelPreview = "preview"

	// SemverVersionMajor indicates that we will release major, minor and patch
	// releases.
	SemverVersionMajor = "major"

	// SemverVersionMinor indicates that we will release minor and patch
	// releases.
	SemverVersionMinor = "minor"

	// SemverVersionPatch indicates that we will release only patch releases.
	SemverVersionPatch = "patch"
)

// SemverSource is a versioning policy based on semver.
// When semver is selected, Heighliner can watch for images on 3 different
// levels.
// When `release` level is selected, we will only get
type SemverSource struct {
	Version SemverVersion `json:"version"`
	Level   SemverLevel   `json:"level"`
}

// VersioningPolicyValidationSchema represents the OpenAPIV3Schema validation for
// the NetworkPolicy CRD.
var VersioningPolicyValidationSchema = apiextv1beta1.JSONSchemaProps{
	Required: []string{"version", "level"},
	Properties: map[string]apiextv1beta1.JSONSchemaProps{
		"version": {
			Type: proto.String,
			Enum: []apiextv1beta1.JSON{
				{Raw: jsonBytes(SemverVersionMajor)},
				{Raw: jsonBytes(SemverVersionMinor)},
				{Raw: jsonBytes(SemverVersionPatch)},
			},
		},
		"level": {
			Type: proto.String,
			Enum: []apiextv1beta1.JSON{
				{Raw: jsonBytes(SemverLevelRelease)},
				{Raw: jsonBytes(SemverLevelReleaseCandidate)},
				{Raw: jsonBytes(SemverLevelPreview)},
			},
		},
	},
}
