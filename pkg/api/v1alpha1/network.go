package v1alpha1

import (
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kube-openapi/pkg/util/proto"
)

// NetworkPolicy describes the configuration options for the NetworkPolicy.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NetworkPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec NetworkSpec `json:"spec"`
}

// NetworkPolicyList is a list of NetworkPolicy CRDs.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NetworkPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []NetworkPolicy `json:"items"`
}

// NetworkSpec describes the specification for Network.
type NetworkSpec struct {
	IngressClass string        `json:"ingressClass"`
	Ports        []NetworkPort `json:"ports"`
	DNS          []NetworkDNS  `json:"dns"`
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

// NetworkDNS describes a DNS entry for a given service, allowing external
// access to the service.
// If no port is provided but a DNS entry is provided, a default headless port
// will be created with the internalPort `8080`.
type NetworkDNS struct {
	// The domain name that will be linked to the service.
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

// NetworkPolicyValidationSchema represents the OpenAPIV3Schema validation for
// the NetworkPolicy CRD.
var NetworkPolicyValidationSchema = apiextv1beta1.JSONSchemaProps{
	Properties: map[string]apiextv1beta1.JSONSchemaProps{
		"ingressClass": {
			Type: "string",
			Enum: []apiextv1beta1.JSON{
				{
					Raw: jsonBytes("nginx"),
				},
			},
		},
		"ports": {
			Items: &apiextv1beta1.JSONSchemaPropsOrArray{
				Schema: &apiextv1beta1.JSONSchemaProps{
					Required: []string{"name", "targetPort", "port"},
				},
			},
		},
		"dns": {
			Items: &apiextv1beta1.JSONSchemaPropsOrArray{
				Schema: &apiextv1beta1.JSONSchemaProps{
					Required: []string{"domain"},
					Properties: map[string]apiextv1beta1.JSONSchemaProps{
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
}
