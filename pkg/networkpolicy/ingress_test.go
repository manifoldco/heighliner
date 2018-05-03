package networkpolicy

import (
	"testing"

	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTemplatedDomain(t *testing.T) {
	release := &v1alpha1.Release{
		SemVer: &v1alpha1.SemVerRelease{
			Name: "hello-world",
		},
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
		{"{{.FullName}}.pr.arigato.tools", "hello-world-c92fbe4899.pr.arigato.tools", nil},
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
