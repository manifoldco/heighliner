package networkpolicy

import (
	"testing"

	"github.com/manifoldco/heighliner/apis/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildServiceForRelease(t *testing.T) {
	release := &v1alpha1.Release{
		SemVer: &v1alpha1.SemVerRelease{
			Name:    "test-application",
			Version: "1.2.3",
		},
		Level: v1alpha1.SemVerLevelRelease,
	}

	ms := &v1alpha1.Microservice{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-application",
		},
	}

	np := &v1alpha1.NetworkPolicy{
		Spec: v1alpha1.NetworkPolicySpec{
			Ports: []v1alpha1.NetworkPort{
				{
					Name:       "headless",
					TargetPort: 8080,
					Port:       80,
				},
			},
		},
	}

	t.Run("with a set of ports", func(t *testing.T) {
		srv := &corev1.Service{}

		obj := buildServiceForRelease(srv, ms, np, release)
		if obj == nil {
			t.Error("Expected object to not be nil")
		}
	})

	t.Run("Maintains existing labels", func(t *testing.T) {
		srv := &corev1.Service{}
		srv.Labels = map[string]string{
			"dummy-label": "value",
		}

		obj := buildServiceForRelease(srv, ms, np, release)
		if obj.Labels["dummy-label"] != "value" {
			t.Error("Expected object to maintain existing label")
		}
	})

	t.Run("Maintains existing annotations", func(t *testing.T) {
		srv := &corev1.Service{}
		srv.Annotations = map[string]string{
			"dummy-annotation": "value",
		}

		obj := buildServiceForRelease(srv, ms, np, release)
		if obj.Annotations["dummy-annotation"] != "value" {
			t.Error("Expected object to maintain existing annotation")
		}
	})

	t.Run("Sets client IP session affinity", func(t *testing.T) {
		srv := &corev1.Service{}

		np = np.DeepCopy()
		np.Spec.SessionAffinity = &corev1.SessionAffinityConfig{
			ClientIP: &corev1.ClientIPConfig{},
		}

		obj := buildServiceForRelease(srv, ms, np, release)
		if obj.Spec.SessionAffinity != corev1.ServiceAffinityClientIP {
			t.Error("Expected service to have client ip session affinity")
		}
	})

}
