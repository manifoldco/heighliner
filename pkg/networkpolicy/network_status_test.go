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

	})
}
