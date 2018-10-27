package v1alpha1

import (
	"github.com/manifoldco/heighliner/internal/k8sutils"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// AvailabilityPolicy defines the configuration options for the AvailabilityPolicy.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AvailabilityPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec AvailabilityPolicySpec `json:"spec"`
}

// AvailabilityPolicyList is a list of AvailabilityPolicy CRDs.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AvailabilityPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []AvailabilityPolicy `json:"items"`
}

// AvailabilityPolicySpec is the specification for Availability.
type AvailabilityPolicySpec struct {
	// Number of desired replicas of the service.
	Replicas *int32 `json:"replicas"`

	// An eviction is allowed if at least "minAvailable" pods selected by
	// "selector" will still be available after the eviction, i.e. even in the
	// absence of the evicted pod.  So for example you can prevent all voluntary
	// evictions by specifying "100%".
	MinAvailable *intstr.IntOrString `json:"minAvailable,omitempty"`

	// An eviction is allowed if at most "maxUnavailable" pods selected by
	// "selector" are unavailable after the eviction, i.e. even in absence of
	// the evicted pod. For example, one can prevent all voluntary evictions
	// by specifying 0. This is a mutually exclusive setting with "minAvailable".
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`

	// RestartPolicy describes how the container should be restarted. Only one
	// of `Always`, `OnFailure` or `Never` restart policies may be specified.
	// If none of the policies is specified, the default one is `Always`.
	RestartPolicy corev1.RestartPolicy `json:"restartPolicy,omitempty"`

	// The deployment strategy to use to replace existing pods with new ones.
	DeploymentStrategy v1beta1.DeploymentStrategy `json:"deploymentStrategy,omitempty"`

	// The microservice's scheduling constraints.
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
}

// DefaultAvailabilityPolicySpec is the default availability spec that will be
// used when it's not defined.
// Affinity is DeploymentSpecific so will be filled in later on.
var DefaultAvailabilityPolicySpec = AvailabilityPolicySpec{
	Replicas:       func(i int32) *int32 { return &i }(2),
	MinAvailable:   k8sutils.PtrIntOrString(intstr.FromInt(1)),
	MaxUnavailable: k8sutils.PtrIntOrString(intstr.FromString("25%")),
	RestartPolicy:  corev1.RestartPolicyAlways,
	DeploymentStrategy: v1beta1.DeploymentStrategy{
		Type: v1beta1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &v1beta1.RollingUpdateDeployment{
			MaxUnavailable: k8sutils.PtrIntOrString(intstr.FromString("25%")),
			MaxSurge:       k8sutils.PtrIntOrString(intstr.FromString("25%")),
		},
	},
}

// AvailabilityPolicyValidationSchema represents the OpenAPIV3Schema validation for
// the Availability CRD.
var AvailabilityPolicyValidationSchema = apiextv1beta1.JSONSchemaProps{}
