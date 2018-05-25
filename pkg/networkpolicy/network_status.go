package networkpolicy

import "github.com/manifoldco/heighliner/pkg/api/v1alpha1"

func buildNetworkStatusForRelease(np *v1alpha1.NetworkPolicy, release *v1alpha1.Release) (v1alpha1.NetworkStatus, error) {

	ns := v1alpha1.NetworkStatus{
		Domains: []v1alpha1.Domain{},
	}

	for _, record := range np.Spec.ExternalDNS {
		domain := v1alpha1.Domain{
			URL: record.Domain, SemVer: release.SemVer,
		}
		ns.Domains = append(ns.Domains, domain)
	}

	return ns, nil
}
