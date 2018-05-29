package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kube-openapi/pkg/util/proto"
)

// NetworkPolicy describes the configuration options for the NetworkPolicy.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NetworkPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   NetworkPolicySpec   `json:"spec"`
	Status NetworkPolicyStatus `json:"status"`
}

// NetworkPolicyList is a list of NetworkPolicy CRDs.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NetworkPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []NetworkPolicy `json:"items"`
}

// NetworkPolicySpec describes the specification for Network.
type NetworkPolicySpec struct {
	// Microservice represents the name of the Microservice which we want to
	// create DNS entries for.
	// If the Microservice Name is not provided, the name of the NetworkPolicy
	// CRD will be used.
	Microservice *corev1.LocalObjectReference `json:"microservice,omitempty"`

	// SessionAffinity lets you define a config for SessionAffinity. If no
	// config is provided, SessionAffinity will be "None".
	SessionAffinity *corev1.SessionAffinityConfig `json:"sessionAffinity"`

	// Ports which we want to be accessible for the associated Microservice.
	Ports []NetworkPort `json:"ports"`

	// ExternalDNS represents the domain specification for a Microservice
	// externally.
	ExternalDNS []ExternalDNS `json:"externalDNS"`

	// UpdateStrategy defines how Heighliner will transition DNS from one
	// version to another.
	UpdateStrategy UpdateStrategy `json:"updateStrategy"`
}

// NetworkPort describes a port that is exposed for a given service.
type NetworkPort struct {
	// The name the port will be given. This will be used to link DNS entries.
	Name string `json:"name"`

	// The port that is exposed within the service container and which the
	// application is running on.
	TargetPort int32 `json:"targetPort"`

	// The port this service will be available on from within the cluster.
	Port int32 `json:"port"`
}

// ExternalDNS describes a DNS entry for a given service, allowing external
// access to the service.
// If no port is provided but a DNS entry is provided, a default headless port
// will be created with the internalPort `8080`.
type ExternalDNS struct {
	// IngressClass represents the class that is given to the Ingress controller
	// to handle DNS entries. This defaults to the default at the controller
	// configuration level.
	IngressClass string `json:"ingressClass"`

	// The domain name that will be linked to the service. This can be a full
	// fledged domain like `dashboard.heighliner.com` or it could be a templated
	// domain like `{.Version}.{.Name}.pr.heighliner.com`. Templated domains get
	// the data from a Release object, possible values are `Version` and `Name`.
	Domain string `json:"domain"`

	// TTL in seconds for the DNS entry, defaults to `300`.
	// Note: if multiple DNS entries are provided, the TTL of the first record
	// will be used.
	TTL int32 `json:"ttl"`

	// By default, TLS will be enabled for external access to a service.
	// Defaults to `false`.
	DisableTLS bool `json:"disableTLS"`

	// TLSGroup specifies the certificate group in which we'll store the SSL
	// Certificates. This defaults to "heighliner-components". It is recommended
	// to set this up per group of applications, this way the certificates will
	// be stored together.
	TLSGroup string `json:"tlsGroup"`

	// Port links back to a NetworkPort and will be used to guide traffic for
	// this hostname through the specified port. Defaults to `headless`.
	Port string `json:"port"`
}

// UpdateStrategy allows a strategy to be defined which will allow the
// NetworkPolicy controller to determine when and how to transition from one
// version to another for a specific Microservice.
// The fields defined on each strategy will be used as label selectors to select
// the correct VersionedMicroservice.
type UpdateStrategy struct {
	Manual *ManualUpdateStrategy `json:"manual"`
	Latest *LatestUpdateStrategy `json:"latest"`
}

// ManualUpdateStrategy is an UpdateStrategy that is purely manual. The
// Controller will put in the labels as provided and won't take any other action
// to detect possible versions.
type ManualUpdateStrategy struct {
	// SemVer is the SemVer annotation of the specific release we want to use
	// for this Microservice.
	SemVer *SemVerRelease `json:"semVer"`
}

// LatestUpdateStrategy will monitor the available release for a given
// Microservice and use the latest available release to link to the internal and
// external DNS.
type LatestUpdateStrategy struct{}

// NetworkPolicyStatus provides external domains and associated SemVer from the release
type NetworkPolicyStatus struct {
	Domains []Domain `json:"domains"`
}

// Domain is represents a url associated with the NetworkPolicy and the associated SemVer
type Domain struct {
	// Url is the url that points to the application
	URL string `json:"url"`

	// SemVer is the SemVer release object linked to this NetworkPolicyStatus if the
	// VersioningPolicy associated with it is SemVer.
	SemVer *SemVerRelease `json:"semVer,omitempty"`
}

// NetworkPolicyValidationSchema represents the OpenAPIV3Schema validation for
// the NetworkPolicy CRD.
var NetworkPolicyValidationSchema = &v1beta1.CustomResourceValidation{
	OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
		Required: []string{"spec"},
		Properties: map[string]v1beta1.JSONSchemaProps{
			"spec": {
				Required: []string{"updateStrategy"},
				Properties: map[string]v1beta1.JSONSchemaProps{
					"ingressClass": {
						Type: proto.String,
					},
					"ports": {
						Items: &v1beta1.JSONSchemaPropsOrArray{
							Schema: &v1beta1.JSONSchemaProps{
								Required: []string{"name", "targetPort", "port"},
							},
						},
					},
					"externalDNS": {
						Items: &v1beta1.JSONSchemaPropsOrArray{
							Schema: &v1beta1.JSONSchemaProps{
								Required: []string{"domain"},
								Properties: map[string]v1beta1.JSONSchemaProps{
									"ttl": {
										Type: proto.Integer,
									},
									"disableTLS": {
										Type: proto.Boolean,
									},
									"tlsGroup": {
										Type: proto.String,
									},
									"port": {
										Type: proto.String,
									},
								},
							},
						},
					},
				},
			},
		},
	},
}
