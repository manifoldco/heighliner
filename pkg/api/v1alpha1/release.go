package v1alpha1

import (
	"fmt"
	"strings"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Release represents a specific release for a version of an image.
type Release struct {
	// Image is the fully qualified image name that can be used to download the
	// image.
	Image string `json:"image"`

	// Released represents when this version became available to be deployed.
	Released metav1.Time `json:"released"`

	// SemVer is the SemVer release object linked to this Release if the
	// VersioningPolicy associated with it is SemVer.
	SemVer *SemVerRelease `json:"semVer,omitempty"`
}

// String concatenates the Release values into a single unique string.
func (r Release) String() string {
	return fmt.Sprintf("%s-%s", r.SemVer.String(), r.Released)
}

// FullName creates the full name for a release. It takes the name of a
// Microservice as a prefix.
func (r Release) FullName(name string) string {
	if r.SemVer != nil {
		return r.SemVer.fullName(name)
	}

	panic("No release type specified")
}

// Name returns the name of the release.
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

	// Build is the specific build for a preview release for a specific version.
	Build string `json:"build,omitempty"`
}

// String concatenates the SemVer Release values into a single unique string.
func (r *SemVerRelease) String() string {
	build := ""
	if r.Build != "" {
		build = fmt.Sprintf("-%s", r.Build)
	}

	return fmt.Sprintf("%s-%s%s", r.Name, r.Version, build)
}

func (r *SemVerRelease) fullName(name string) string {
	releaseName := ""
	if r.Name != name {
		releaseName = fmt.Sprintf("-%s", r.Name)
	}

	build := ""
	if r.Build != "" {
		build = fmt.Sprintf("-%s", r.Build)
	}

	return strings.ToLower(fmt.Sprintf("%s%s-%s%s", name, releaseName, r.Version, build))
}

// ReleaseValidationSchema represents the OpenAPIv3 validation schema for a
// release object.
var ReleaseValidationSchema = v1beta1.JSONSchemaProps{
	Required: []string{"releases"},
	Properties: map[string]v1beta1.JSONSchemaProps{
		"releases": {
			Required: []string{"semVer", "image", "released"},
			Properties: map[string]v1beta1.JSONSchemaProps{
				"semVer": semVerReleaseValidation,
			},
		},
	},
}

var semVerReleaseValidation = v1beta1.JSONSchemaProps{
	Required: []string{"version"},
}
