package networkpolicy

import (
	"bytes"
	"html/template"
	"strconv"
	"strings"

	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
	"github.com/manifoldco/heighliner/pkg/k8sutils"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func buildIngressForRelease(ms *v1alpha1.Microservice, np *v1alpha1.NetworkPolicy, release *v1alpha1.Release) (*v1beta1.Ingress, error) {
	if len(np.Spec.ExternalDNS) == 0 {
		return nil, nil
	}

	// TODO (jelmer): if there's different ingress classes, this should deploy
	// different ingress objects. For now, this will do.
	ingressClass := np.Spec.ExternalDNS[0].IngressClass
	if ingressClass == "" {
		ingressClass = "nginx"
	}

	domains := make([]string, len(np.Spec.ExternalDNS))
	for i, record := range np.Spec.ExternalDNS {
		domains[i] = record.Domain
	}

	labels := k8sutils.Labels(np.Labels, np.ObjectMeta)
	labels["hlnr.io/microservice.full_name"] = release.FullName(ms.Name)
	labels["hlnr.io/microservice.name"] = ms.Name
	labels["hlnr.io/microservice.release"] = release.Name()
	labels["hlnr.io/microservice.version"] = release.Version()

	annotations := k8sutils.Annotations(np.Annotations, v1alpha1.Version, np)
	annotations["kubernetes.io/ingress.class"] = ingressClass
	annotations["external-dns.alpha.kubernetes.io/hostname"] = strings.Join(domains, ",")
	// TODO (jelmer): different TTLs should mean different Ingresses
	annotations["external-dns.alpha.kubernetes.io/ttl"] = ttlValue(np.Spec.ExternalDNS[0].TTL)

	// Disable SSL redirects when we don't have TLS enabled.
	if np.Spec.ExternalDNS[0].DisableTLS {
		annotations["nginx.ingress.kubernetes.io/ssl-redirect"] = "false"
	}

	ingressTLS, err := getIngressTLS(ms, release, np.Spec.ExternalDNS)
	if err != nil {
		return nil, err
	}

	ingressRules, err := getIngressRules(ms, release, np.Spec.ExternalDNS)
	if err != nil {
		return nil, err
	}

	ing := &v1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            release.FullName(ms.Name),
			Namespace:       ms.Namespace,
			Labels:          labels,
			Annotations:     annotations,
			OwnerReferences: release.OwnerReferences,
		},
		Spec: v1beta1.IngressSpec{
			TLS:   ingressTLS,
			Rules: ingressRules,
		},
	}

	return ing, nil
}

func getIngressRules(ms *v1alpha1.Microservice, release *v1alpha1.Release, records []v1alpha1.ExternalDNS) ([]v1beta1.IngressRule, error) {
	rules := make([]v1beta1.IngressRule, len(records))
	for i, r := range records {
		servicePort := "headless"
		if r.Port != "" {
			servicePort = r.Port
		}

		domain, err := templatedDomain(ms, release, r.Domain)
		if err != nil {
			return nil, err
		}

		rules[i] = v1beta1.IngressRule{
			Host: domain,
			IngressRuleValue: v1beta1.IngressRuleValue{
				HTTP: &v1beta1.HTTPIngressRuleValue{
					Paths: []v1beta1.HTTPIngressPath{
						{
							Path: "/",
							Backend: v1beta1.IngressBackend{
								ServiceName: ms.Name,
								ServicePort: intstr.FromString(servicePort),
							},
						},
					},
				},
			},
		}
	}

	return rules, nil
}

func getIngressTLS(ms *v1alpha1.Microservice, release *v1alpha1.Release, records []v1alpha1.ExternalDNS) ([]v1beta1.IngressTLS, error) {
	tls := make([]v1beta1.IngressTLS, len(records))

	for i, dns := range records {
		if dns.DisableTLS {
			continue
		}

		secretName := "heighliner-components"
		if dns.TLSGroup != "" {
			secretName = dns.TLSGroup
		}

		domain, err := templatedDomain(ms, release, dns.Domain)
		if err != nil {
			return nil, err
		}

		tls[i] = v1beta1.IngressTLS{
			Hosts:      []string{domain},
			SecretName: "certificates-" + secretName,
		}
	}

	return tls, nil
}

func templatedDomain(ms *v1alpha1.Microservice, release *v1alpha1.Release, domain string) (string, error) {
	tmpl, err := template.New("domain").Parse(domain)
	if err != nil {
		return "", err
	}

	data := struct {
		FullName string
		Name     string
	}{

		FullName: release.FullName(ms.Name),
		Name:     release.Name(),
	}

	buf := bytes.NewBufferString("")
	if err := tmpl.Execute(buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func ttlValue(ttl int32) string {
	if ttl == 0 {
		return "300"
	}

	return strconv.Itoa(int(ttl))
}
