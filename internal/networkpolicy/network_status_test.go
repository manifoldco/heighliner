package networkpolicy

import (
	"fmt"
	"testing"

	"github.com/manifoldco/heighliner/internal/api/v1alpha1"
)

func TestNetworkStatus(t *testing.T) {
	release := &v1alpha1.Release{
		SemVer: &v1alpha1.SemVerRelease{
			Name: "hello-world",
		},
		Level: v1alpha1.SemVerLevelRelease,
	}

	ms := &v1alpha1.Microservice{}

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

		domains, _ := buildNetworkStatusDomainsForRelease(ms, np, release)
		if len(domains) != 1 {
			t.Errorf("Expected domains to be of length 1, got '%d'", len(domains))
		}

		actualDomainURL := domains[0].URL
		expectedDomainURL := "https://my.cool.domain"
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

		domains, _ := buildNetworkStatusDomainsForRelease(ms, np, release)
		if len(domains) != 2 {
			t.Errorf("Expected domains to be of length 2, got '%d'", len(domains))
		}

		actualFirstDomainURL := domains[0].URL
		expectedFirstDomainURL := "https://my.cool.domain"
		if actualFirstDomainURL != expectedFirstDomainURL {
			t.Errorf("Expected domain URL to be %s, got '%s'", expectedFirstDomainURL, actualFirstDomainURL)
		}

		actualSecondDomainURL := domains[1].URL
		expectedSecondDomainURL := "https://my.other.cool.domain"
		if actualSecondDomainURL != expectedSecondDomainURL {
			t.Errorf("Expected domain URL to be %s, got '%s'", expectedSecondDomainURL, actualSecondDomainURL)
		}

	})
}

func TestFullDomain(t *testing.T) {
	t.Run("without TLS disabled", func(t *testing.T) {
		domain := "my.cool.domain"
		url := fmt.Sprintf("https://%s", domain)

		dns := v1alpha1.ExternalDNS{
			Domain: domain,
		}

		if actual := getFullURL(dns); actual != url {
			t.Errorf("Expected url to be '%s', got '%s'", url, actual)
		}
	})

	t.Run("with TLS disabled", func(t *testing.T) {
		domain := "my.cool.domain"
		url := fmt.Sprintf("http://%s", domain)

		dns := v1alpha1.ExternalDNS{
			Domain:     domain,
			DisableTLS: true,
		}

		if actual := getFullURL(dns); actual != url {
			t.Errorf("Expected url to be '%s', got '%s'", url, actual)
		}
	})
}

func TestStatusDomainsEqual(t *testing.T) {
	tcs := []struct {
		name     string
		old      []v1alpha1.Domain
		new      []v1alpha1.Domain
		expected bool
	}{
		{"empty (nil/nil)", nil, nil, true},
		{"empty (nil/0)", nil, []v1alpha1.Domain{}, true},

		{"one entry (equal)",
			[]v1alpha1.Domain{{URL: "https://fake.fake"}},
			[]v1alpha1.Domain{{URL: "https://fake.fake"}},
			true,
		},

		{"one entry (equal with semver)",
			[]v1alpha1.Domain{{URL: "https://fake.fake", SemVer: &v1alpha1.SemVerRelease{Name: "foo", Version: "0.1.0"}}},
			[]v1alpha1.Domain{{URL: "https://fake.fake", SemVer: &v1alpha1.SemVerRelease{Name: "foo", Version: "0.1.0"}}},
			true,
		},

		{"one entry (mismatched semver)",
			[]v1alpha1.Domain{{URL: "https://fake.fake", SemVer: &v1alpha1.SemVerRelease{Name: "foo", Version: "0.1.0"}}},
			[]v1alpha1.Domain{{URL: "https://fake.fake", SemVer: &v1alpha1.SemVerRelease{Name: "foo", Version: "0.1.1"}}},
			false,
		},

		{"one entry (mismatched URL)",
			[]v1alpha1.Domain{{URL: "https://fake.fake", SemVer: &v1alpha1.SemVerRelease{Name: "foo", Version: "0.1.0"}}},
			[]v1alpha1.Domain{{URL: "https://anotherfake.fake", SemVer: &v1alpha1.SemVerRelease{Name: "foo", Version: "0.1.0"}}},
			false,
		},

		{"two entries out of order",
			[]v1alpha1.Domain{{URL: "https://fake.fake"}, {URL: "https://other.fake"}},
			[]v1alpha1.Domain{{URL: "https://other.fake"}, {URL: "https://fake.fake"}},
			true,
		},

		{"mismatched length",
			[]v1alpha1.Domain{{URL: "https://fake.fake"}},
			[]v1alpha1.Domain{{URL: "https://other.fake"}, {URL: "https://fake.fake"}},
			false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			if statusDomainsEqual(tc.old, tc.new) != tc.expected {
				t.Error("bad result for statusDomainsEqual")
			}
		})
	}
}
