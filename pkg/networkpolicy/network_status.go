package networkpolicy

import (
	"fmt"

	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
)

func buildNetworkStatusDomainsForRelease(ms *v1alpha1.Microservice, np *v1alpha1.NetworkPolicy, release *v1alpha1.Release) ([]v1alpha1.Domain, error) {
	domains := []v1alpha1.Domain{}

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
		domains = append(domains, domain)
	}

	return domains, nil
}

func getFullURL(dns v1alpha1.ExternalDNS) string {
	scheme := "https://"
	if dns.DisableTLS {
		scheme = "http://"
	}

	return fmt.Sprintf("%s%s", scheme, dns.Domain)
}

func statusDomainsEqual(old, new []v1alpha1.Domain) bool {
	if len(old) != len(new) {
		return false
	}

oldLoop:
	for _, o := range old {
		for _, n := range new {
			if o.URL != n.URL {
				continue
			}

			if o.SemVer == nil && n.SemVer != nil ||
				o.SemVer != nil && n.SemVer == nil {
				continue
			}

			if o.SemVer != nil &&
				(o.SemVer.Name != n.SemVer.Name ||
					o.SemVer.Version != n.SemVer.Version) {
				continue
			}

			continue oldLoop // found a match!
		}

		return false // didn't find a match :(
	}

	return true
}
