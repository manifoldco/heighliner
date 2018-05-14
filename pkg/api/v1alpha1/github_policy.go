package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GitHubPolicy represents the combined state of different components in time
// which form a single Microservice.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type GitHubPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   GitHubPolicySpec   `json:"spec"`
	Status GitHubPolicyStatus `json:"status"`
}

// GitHubPolicyList is a list of GitHubPolicies.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type GitHubPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []GitHubPolicy `json:"items"`
}

// GitHubPolicySpec represents the specification for a GitHubPolicy.
type GitHubPolicySpec struct {
	// Repositories is a list of repositories we'd like to watch updates for.
	Repositories []GitHubRepository `json:"repositories"`

	// MaxAvailable is the maximum number of releases for a specific level that
	// should be kept. When the number of releases grows over this amount, the
	// oldes release will be sunsetted.
	MaxAvailable int `json:"maxAvailable"`
}

// GitHubRepository represents the configuration for a specific repository.
type GitHubRepository struct {
	// Name is the name of the repository we want to monitor
	Name string `json:"name"`

	// Owner is the owner of the repository, often specified as team.
	Team string `json:"team"`

	// ConfigSecret represent the secret that houses the API token to
	// communicate with the given repository.
	ConfigSecret corev1.LocalObjectReference `json:"configSecret"`
}

// GitHubPolicyStatus represents the current status for the GitHubPolicy.
type GitHubPolicyStatus struct {
	Releases map[string]GitHubRelease `json:"releases"`
}

// GitHubRelease represents a release made in GitHub
type GitHubRelease struct {
	Name       string      `json:"name"`
	Tag        string      `json:"tag"`
	Level      SemVerLevel `json:"level"`
	ReleasedAt metav1.Time `json:"releasedAt"`
}
