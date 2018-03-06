package vsvc

import (
	"log"
	"strconv"
	"strings"

	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
	"github.com/manifoldco/heighliner/pkg/k8sutils"

	"github.com/jelmersnoeck/kubekit"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// getService creates the Service Object for a VersionedMicroservice.
func getIngress(crd *v1alpha1.VersionedMicroservice) (runtime.Object, error) {
	if crd.Spec.Network == nil || crd.Spec.Network.DNS == nil {
		log.Printf("No DNS specified for %s, skipping ingress.", crd.Name)
		return nil, nil
	}

	dns := crd.Spec.Network.DNS
	labels := k8sutils.Labels(crd.Labels, crd.ObjectMeta)

	ingressClass := crd.Spec.Network.IngressClass
	if ingressClass == "" {
		ingressClass = "nginx"
	}

	domains := make([]string, len(dns))
	for i, record := range dns {
		domains[i] = record.Domain
	}

	annotations := k8sutils.Annotations(crd.Annotations, v1alpha1.Version, crd)
	annotations["kubernetes.io/ingress.class"] = ingressClass
	annotations["external-dns.alpha.kubernetes.io/hostname"] = strings.Join(domains, ",")
	annotations["external-dns.alpha.kubernetes.io/ttl"] = ttlValue(dns[0].TTL)

	ing := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        crd.Name,
			Namespace:   crd.Namespace,
			Labels:      labels,
			Annotations: annotations,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(
					crd,
					v1alpha1.SchemeGroupVersion.WithKind(kubekit.TypeName(crd)),
				),
			},
		},
		Spec: v1beta1.IngressSpec{
			TLS:   getIngressTLS(dns),
			Rules: getIngressRules(crd.Name, dns),
		},
	}

	return ing, nil
}

func getIngressRules(serviceName string, records []v1alpha1.NetworkDNS) []v1beta1.IngressRule {
	rules := make([]v1beta1.IngressRule, len(records))
	for i, r := range records {
		servicePort := "headless"
		if r.Port != "" {
			servicePort = r.Port
		}

		rules[i] = v1beta1.IngressRule{
			Host: r.Domain,
			IngressRuleValue: v1beta1.IngressRuleValue{
				HTTP: &v1beta1.HTTPIngressRuleValue{
					Paths: []v1beta1.HTTPIngressPath{
						{
							// TODO(jelmer): do we want to make this
							// configurable?
							// This might be where we start putting our
							// "highly opinionated" goals forward.
							Path: "/",
							Backend: v1beta1.IngressBackend{
								ServiceName: serviceName,
								ServicePort: intstr.FromString(servicePort),
							},
						},
					},
				},
			},
		}
	}

	return rules
}

func getIngressTLS(records []v1alpha1.NetworkDNS) []v1beta1.IngressTLS {
	tls := make([]v1beta1.IngressTLS, len(records))

	for i, dns := range records {
		if dns.DisableTLS {
			return nil
		}

		secretName := "heighliner-components"
		if dns.TLSGroup != "" {
			secretName = dns.TLSGroup
		}

		tls[i] = v1beta1.IngressTLS{
			Hosts:      []string{dns.Domain},
			SecretName: "certificates-" + secretName,
		}
	}

	return tls
}

func ttlValue(ttl int32) string {
	if ttl == 0 {
		return "300"
	}

	return strconv.Itoa(int(ttl))
}
