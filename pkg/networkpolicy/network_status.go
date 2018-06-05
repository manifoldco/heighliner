package networkpolicy

import (
	"fmt"

	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
)

func buildNetworkStatusForRelease(ms *v1alpha1.Microservice, np *v1alpha1.NetworkPolicy, release *v1alpha1.Release) (v1alpha1.NetworkPolicyStatus, error) {
	ns := v1alpha1.NetworkPolicyStatus{
		Domains: []v1alpha1.Domain{},
	}

	for _, record := range np.Spec.ExternalDNS {
		url, err := templatedDomain(ms, release, getFullURL(record))
		if err != nil {
			// XXX: handle gracefully
			panic(err)
		}

		domain := v1alpha1.Domain{
			URL:    url,
			SemVer: release.SemVer,
		}
		ns.Domains = append(ns.Domains, domain)
	}

	return ns, nil
}

func getFullURL(dns v1alpha1.ExternalDNS) string {
	scheme := "https://"
	if dns.DisableTLS {
		scheme = "http://"
	}

	return fmt.Sprintf("%s%s", scheme, dns.Domain)
}
