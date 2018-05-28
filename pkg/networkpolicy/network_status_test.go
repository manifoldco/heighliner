package networkpolicy

import (
	"testing"

	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
)

func TestNetworkStatus(t *testing.T) {
	release := &v1alpha1.Release{
		SemVer: &v1alpha1.SemVerRelease{
			Name: "hello-world",
		},
	}

	t.Run("with a single domain network policy", func(t *testing.T) {
		np := &v1alpha1.NetworkPolicy{
			Spec: v1alpha1.NetworkPolicySpec{
				ExternalDNS: []v1alpha1.ExternalDNS{
					{
						Domain: "my.cool.domain",
					},
				},
			},
		}

		status, _ := buildNetworkStatusForRelease(np, release)
		if len(status.Domains) != 1 {
			t.Errorf("Expected status domains to be of length 1, got '%d'", len(status.Domains))
		}

		actualDomainURL := status.Domains[0].URL
		expectedDomainURL := "my.cool.domain"
		if actualDomainURL != expectedDomainURL {
			t.Errorf("Expected domain URL to be %s, got '%s'", expectedDomainURL, actualDomainURL)
		}

	})

	t.Run("with a multi domain network policy", func(t *testing.T) {
		np := &v1alpha1.NetworkPolicy{
			Spec: v1alpha1.NetworkPolicySpec{
				ExternalDNS: []v1alpha1.ExternalDNS{
					{
						Domain: "my.cool.domain",
					},
					{
						Domain: "my.other.cool.domain",
					},
				},
			},
		}

		status, _ := buildNetworkStatusForRelease(np, release)
		if len(status.Domains) != 2 {
			t.Errorf("Expected status domains to be of length 2, got '%d'", len(status.Domains))
		}

		actualFirstDomainURL := status.Domains[0].URL
		expectedFirstDomainURL := "my.cool.domain"
		if actualFirstDomainURL != expectedFirstDomainURL {
			t.Errorf("Expected domain URL to be %s, got '%s'", expectedFirstDomainURL, actualFirstDomainURL)
		}

		actualSecondDomainURL := status.Domains[1].URL
		expectedSecondDomainURL := "my.other.cool.domain"
		if actualSecondDomainURL != expectedSecondDomainURL {
			t.Errorf("Expected domain URL to be %s, got '%s'", expectedSecondDomainURL, actualSecondDomainURL)
		}

	})
}
