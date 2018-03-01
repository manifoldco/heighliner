package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VersionedMicroservice represents the combined state of different components
// in time which form a single Microservice.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VersionedMicroservice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec VersionedMicroserviceSpec `json:"spec"`
}

// VersionedMicroserviceList is a list of VersionedMicroservices.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VersionedMicroserviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []*VersionedMicroservice `json:"items"`
}

// VersionedMicroserviceSpec represents the specification for a
// VersionedMicroservice.
type VersionedMicroserviceSpec struct {
	Availability *AvailabilitySpec  `json:"availability,omitempty"`
	Network      *NetworkSpec       `json:"network,omitempty"`
	Volumes      []corev1.Volume    `json:"volumes,omitempty"`
	Containers   []corev1.Container `json:"containers"`
}
