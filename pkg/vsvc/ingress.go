package vsvc

import (
	"log"
	"strconv"

	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
	"github.com/manifoldco/heighliner/pkg/k8sutils"

	"github.com/jelmersnoeck/kubekit"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// getService creates the Service Object for a VersionedMicroservice.
func getIngress(crd *v1alpha1.VersionedMicroservice) (*v1beta1.Ingress, error) {
	if crd.Spec.Network == nil || crd.Spec.Network.DNS == nil {
		log.Printf("No DNS specified for %s, skipping ingress.", crd.Name)
		return nil, nil
	}

	dns := crd.Spec.Network.DNS
	labels := k8sutils.Labels(nil, crd.ObjectMeta)

	ingressClass := crd.Spec.Network.IngressClass
	if ingressClass == "" {
		ingressClass = "nginx"
	}

	annotations := crd.Annotations
	annotations = k8sutils.Annotations(annotations, v1alpha1.Version, crd)
	annotations["kubernetes.io/ingress.class"] = ingressClass
	annotations["external-dns.alpha.kubernetes.io/hostname"] = dns.Domain + "."
	annotations["external-dns.alpha.kubernetes.io/ttl"] = ttlValue(dns.TTL)

	servicePort := "headless"
	if dns.Port != "" {
		servicePort = dns.Port
	}

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
			TLS: getIngressTLS(dns),
			Rules: []v1beta1.IngressRule{
				{
					Host: dns.Domain,
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
										ServiceName: crd.Name,
										ServicePort: intstr.FromString(servicePort),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return ing, nil
}

func getIngressTLS(dns *v1alpha1.NetworkDNS) []v1beta1.IngressTLS {
	if dns.DisableTLS {
		return nil
	}

	secretName := "heighliner-components"
	if dns.TLSGroup != "" {
		secretName = dns.TLSGroup
	}

	return []v1beta1.IngressTLS{
		{
			Hosts:      []string{dns.Domain},
			SecretName: "certificates-" + secretName,
		},
	}
}

func ttlValue(ttl int32) string {
	if ttl == 0 {
		return "300"
	}

	return strconv.Itoa(int(ttl))
}
