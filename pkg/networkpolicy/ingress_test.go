package networkpolicy

import (
	"testing"

	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildIngressForRelease(t *testing.T) {
	ms := &v1alpha1.Microservice{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hello-world",
		},
	}

	release := &v1alpha1.Release{
		SemVer: &v1alpha1.SemVerRelease{
			Name: "hello-world",
		},
		Level: v1alpha1.SemVerLevelRelease,
	}

	np := &v1alpha1.NetworkPolicy{}

	srv := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind: "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "srv",
		},
	}

	t.Run("Sets OwnerReference to the service", func(t *testing.T) {
		np := &v1alpha1.NetworkPolicy{
			Spec: v1alpha1.NetworkPolicySpec{
				ExternalDNS: []v1alpha1.ExternalDNS{
					{Domain: "fake.fake"},
				},
			},
		}

		ing, err := buildIngressForRelease(ms, np, release, srv)
		if err != nil {
			t.Error("Expected no err. got:", err)
		}

		if len(ing.OwnerReferences) != 1 {
			t.Error("Wrong number of owners:", len(ing.OwnerReferences))
		}

		if ing.OwnerReferences[0].Name != srv.Name ||
			ing.OwnerReferences[0].Kind != srv.Kind ||
			ing.OwnerReferences[0].Controller == nil ||
			!*ing.OwnerReferences[0].Controller {

			t.Error("Bad OwnerReference seen. got", ing.OwnerReferences)
		}
	})

	t.Run("It is nil with no external dns", func(t *testing.T) {

		ing, err := buildIngressForRelease(ms, np, release, srv)
		if err != nil {
			t.Error("Expected no err. got:", err)
		}

		if ing != nil {
			t.Error("Expected no ingress. got:", ing)
		}
	})
}

func TestTemplatedDomain(t *testing.T) {
	release := &v1alpha1.Release{
		SemVer: &v1alpha1.SemVerRelease{
			Name: "hello-world",
		},
		Level: v1alpha1.SemVerLevelRelease,
	}

	ms := &v1alpha1.Microservice{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hello-world",
		},
	}

	testData := []struct {
		domain   string
		expected string
		err      error
	}{
		{"{{.FullName}}.pr.arigato.tools", "hello-world-flica9pe.pr.arigato.tools", nil},
		{"{{.Name}}.pr.arigato.tools", "hello-world.pr.arigato.tools", nil},
	}

	for _, item := range testData {
		str, err := templatedDomain(ms, release, item.domain)
		if err != item.err {
			t.Errorf("Expected error to be '%s', got '%s'", item.err, err)
			continue
		}

		if str != item.expected {
			t.Errorf("Expected domain to be '%s', got '%s'", item.expected, str)
		}
	}
}
