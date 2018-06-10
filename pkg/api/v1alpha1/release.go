package v1alpha1

import (
	"fmt"

	"github.com/manifoldco/heighliner/pkg/k8sutils"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Release represents a specific release for a version of an image.
type Release struct {
	// OwnerReferences represents who the owner is of this release. This will
	// be set by the Microservice controller and reference a
	// VersionedMicroservice.
	OwnerReferences []metav1.OwnerReference `json:"ownerReference,omitempty"`

	// Image is the fully qualified image name that can be used to download the
	// image.
	Image string `json:"image"`

	// ReleaseTime represents when this version became available to be deployed.
	ReleaseTime metav1.Time `json:"releaseTime"`

	// SemVer is the SemVer release object linked to this Release if the
	// VersioningPolicy associated with it is SemVer.
	SemVer *SemVerRelease `json:"semVer,omitempty"`

	// Level is the detected maturity level for this release
	Level SemVerLevel `json:"level"`
}

// String concatenates the Release values into a single unique string.
func (r Release) String() string {
	return fmt.Sprintf("%s-%s", r.SemVer.String(), r.ReleaseTime)
}

// StreamName creates the release stream name for a release. It takes the name
// of a Microservice as a prefix.
//
// Stream names are based on release level:
//     release - `<prefix>`
//     candidate - `<prefix>-rc`
//     preview - `<prefix>-pr-<hash of branch/release name>
func (r Release) StreamName(prefix string) string {
	if r.SemVer != nil {
		switch r.Level {
		case SemVerLevelRelease:
			return prefix
		case SemVerLevelReleaseCandidate:
			return fmt.Sprintf("%s-rc", prefix)
		case SemVerLevelPreview:
			return fmt.Sprintf("%s-pr-%s", prefix, k8sutils.ShortHash(r.SemVer.Name, 8))
		default:
			panic("Unknown SemVerLevel")
		}
	}

	panic("No release type specified")
}

// FullName creates the full name for a release. This is the stream name
// suffixed by a version derived hash.
// Microservice as a prefix.
func (r Release) FullName(prefix string) string {
	if r.SemVer != nil {
		return fmt.Sprintf("%s-%s", r.StreamName(prefix), k8sutils.ShortHash(r.SemVer.Version, 8))
	}

	panic("No release type specified")
}

// Name returns the name of the actual version.
func (r Release) Name() string {
	if r.SemVer != nil {
		return r.SemVer.Name
	}

	panic("No release type specified")
}

// Version returns the version of the release.
func (r Release) Version() string {
	if r.SemVer != nil {
		return r.SemVer.Version
	}

	panic("No release type specified")
}

// SemVerRelease represents a release which is linked to a SemVer
// VersioningPolicy.
type SemVerRelease struct {
	// Name represents the name of the service to be released. For releases and
	// release candidates, this will be the name of the application, for
	// previews this will be the preview tag (generally the branch name).
	Name string `json:"name"`

	// Version is the specific version for this release in a SemVer annotation.
	Version string `json:"version"`
}

// String concatenates the SemVer Release values into a single unique string.
func (r *SemVerRelease) String() string {
	return fmt.Sprintf("%s-%s", r.Name, r.Version)
}

// ReleaseValidationSchema represents the OpenAPIv3 validation schema for a
// release object.
var ReleaseValidationSchema = v1beta1.JSONSchemaProps{
	Properties: map[string]v1beta1.JSONSchemaProps{
		"releases": {
			Required: []string{"semVer", "image", "releaseTime"},
			Properties: map[string]v1beta1.JSONSchemaProps{
				"semVer": semVerReleaseValidation,
			},
		},
	},
}

var semVerReleaseValidation = v1beta1.JSONSchemaProps{
	Required: []string{"version"},
}
