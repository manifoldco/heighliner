package v1alpha1

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GitHubRepository represents the configuration for a specific GitHub
// repository.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type GitHubRepository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   GitHubRepositorySpec   `json:"spec"`
	Status GitHubRepositoryStatus `json:"status"`
}

// GitHubRepositoryList is a list of GitHubRepositories.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type GitHubRepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []GitHubRepository `json:"items"`
}

// GitHubRepositorySpec represents the specification for a GitHubRepository.
type GitHubRepositorySpec struct {
	// MaxAvailable is the maximum number of releases for a specific level that
	// should be kept. When the number of releases grows over this amount, the
	// oldes release will be sunsetted.
	MaxAvailable int `json:"maxAvailable"`

	// Repo is the name of the repository we want to monitor
	Repo string `json:"repo"`

	// Owner is the owner of the repository, often specified as team.
	Owner string `json:"owner"`

	// ConfigSecret represent the secret that houses the API token to
	// communicate with the given repository.
	ConfigSecret corev1.LocalObjectReference `json:"configSecret"`
}

// Slug returns the slug of the repository.
func (r *GitHubRepositorySpec) Slug() string {
	return fmt.Sprintf("%s/%s", r.Owner, r.Repo)
}

// GitHubRepositoryStatus represents the current status for the GitHubRepository.
type GitHubRepositoryStatus struct {
	// Releases represents the available releases on GitHub for the associated
	// repositories.
	Releases []GitHubRelease `json:"releases"`

	// Webhook represents the installed Webhook information for the GitHub
	// Repository.
	Webhook *GitHubHook `json:"webhook"`
}

// GitHubHook represents the status object for a GiHub Webhook for the CRD.
type GitHubHook struct {
	// ID is the ID on GitHub for the installed hooks. This is needed to perform
	// updates and deletes.
	ID *int64 `json:"id"`

	// Secret is the secret used in GHs communication to our server.
	Secret string `json:"secret"`
}

// GitHubRelease represents a release made in GitHub
type GitHubRelease struct {
	Name       string      `json:"name"`
	Tag        string      `json:"tag"`
	Level      SemVerLevel `json:"level"`
	ReleasedAt metav1.Time `json:"releasedAt"`
	Deployment *Deployment `json:"deployment,omitempty"`
}

// Deployment represents a linking between a GitHub deployment and a network
// policy. Through the release information we can determine a specific domain.
type Deployment struct {
	ID            int64                       `json:"deployment"`
	NetworkPolicy corev1.LocalObjectReference `json:"networkPolicy"`
	State         string                      `json:"state"`
	URL           *string                     `json:"url,omitempty"`
}

// GitHubRepositoryValidationSchema represents the OpenAPIV3Schema
// validation for the GitHubRepository CRD.
var GitHubRepositoryValidationSchema = &v1beta1.CustomResourceValidation{
	OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
		Required: []string{"spec"},
		Properties: map[string]v1beta1.JSONSchemaProps{
			"spec": {
				Required: []string{"repo", "owner", "configSecret"},
			},
		},
	},
}
