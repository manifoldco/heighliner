package v1alpha1

import (
	"fmt"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Release represents a specific release for a version of an image.
type Release struct {
	// Name represents the name of the service to be released. For releases and
	// release candidates, this will generally be the name of the application,
	// for previews this will be the preview tag (generally the branch name).
	Name string `json:"name"`

	// Version is the specific version for this release. For releases and
	// release candidates, this will be the specific version. For previews this
	// will be the build version.
	Version string `json:"version"`

	// Image is the fully qualified image name that can be used to download the
	// image.
	Image string `json:"image"`

	// Released represents when this version became available to be deployed.
	Released metav1.Time `json:"released"`
}

// String concatenates the Release values into a single unique string.
func (r Release) String() string {
	return fmt.Sprintf("%s-%s-%s", r.Name, r.Version, r.Released)
}

// ReleaseValidationSchema represents the OpenAPIv3 validation schema for a
// release object.
var ReleaseValidationSchema = v1beta1.JSONSchemaProps{
	Required: []string{"releases"},
	Properties: map[string]v1beta1.JSONSchemaProps{
		"releases": {
			Required: []string{"name", "version", "image", "released"},
		},
	},
}
