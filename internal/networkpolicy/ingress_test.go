package networkpolicy

import (
	"reflect"
	"testing"

	"github.com/jelmersnoeck/kubekit"
	"github.com/manifoldco/heighliner/apis/v1alpha1"
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

		ownerReference := *metav1.NewControllerRef(
			srv,
			corev1.SchemeGroupVersion.WithKind(kubekit.TypeName(srv)),
		)

		if !reflect.DeepEqual(ing.OwnerReferences[0], ownerReference) {
			t.Errorf("Bad OwnerReference seen. got\n%#v\n\nwanted\n%#v", ing.OwnerReferences[0], ownerReference)
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
			Name:    "hello-world",
			Version: "0.0.1",
		},
		Level: v1alpha1.SemVerLevelPreview,
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
		{"{{.FullName}}.arigato.tools", "hello-world-pr-cmqolv9f-svek39uq.arigato.tools", nil},
		{"{{.Name}}.arigato.tools", "hello-world.arigato.tools", nil},
		{"{{.StreamName}}.arigato.tools", "hello-world-pr-cmqolv9f.arigato.tools", nil},
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
