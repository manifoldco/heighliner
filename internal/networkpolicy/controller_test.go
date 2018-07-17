package networkpolicy

import (
	"errors"
	"fmt"
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/typed/core/v1/fake"

	"github.com/jelmersnoeck/kubekit/patcher"
	"github.com/manifoldco/heighliner/apis/v1alpha1"
	"github.com/manifoldco/heighliner/internal/tester"
)

func TestGroupReleases(t *testing.T) {
	t.Run("with a set of semver releases", func(t *testing.T) {
		t.Run("with different PR applications", func(t *testing.T) {
			releases := []v1alpha1.Release{
				{
					Image: "hlnr.io/test:1.2.3-pr.456-pr+201804281301",
					SemVer: &v1alpha1.SemVerRelease{
						Name:    "456-pr",
						Version: "0.1.0",
					},
					Level: v1alpha1.SemVerLevelPreview,
				},
				{
					Image: "hlnr.io/test:1.2.3-pr.456-pr+201804281308",
					SemVer: &v1alpha1.SemVerRelease{
						Name:    "456-pr",
						Version: "0.1.1",
					},
					Level: v1alpha1.SemVerLevelPreview,
				},
				{
					Image: "hlnr.io/test:1.2.3-pr.457-pr+201804281307",
					SemVer: &v1alpha1.SemVerRelease{
						Name:    "457-pr",
						Version: "0.1.0",
					},
					Level: v1alpha1.SemVerLevelPreview,
				},
			}

			results := groupReleases("test-deploy", releases)
			expectedLength := 2
			if len(results) != expectedLength {
				t.Errorf("Expected length to be %d, got %d", expectedLength, len(results))
			}
		})
	})
}

func TestController_syncNetworking(t *testing.T) {
	pc := new(tester.PatchClient)
	ctrl := &Controller{
		cs:      new(fake.FakeCoreV1),
		patcher: pc,
	}

	t.Run("without linked microservice", func(t *testing.T) {
		defer pc.Flush()

		myErr := errors.New("my error")
		pc.GetFunc = func(obj interface{}, namespace, name string) error {
			if _, ok := obj.(*v1alpha1.Microservice); !ok {
				t.Errorf("Expected object to be Microservice, got %T", obj)
			}

			return myErr
		}

		np := &v1alpha1.NetworkPolicy{}
		if err := ctrl.syncNetworking(np); err != myErr {
			t.Errorf("Expected '%s', got '%s'", myErr, err)
		}
	})

	t.Run("with linked microservice", func(t *testing.T) {
		t.Run("without any releases", func(t *testing.T) {
			defer pc.Flush()

			np := &v1alpha1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unit-tests",
					Namespace: "testing",
				},
				Spec: v1alpha1.NetworkPolicySpec{
					Microservice: &v1.LocalObjectReference{
						Name: "unit-test",
					},
				},
			}

			pc.GetFunc = func(obj interface{}, namespace, name string) error {
				switch obj := obj.(type) {
				case *v1alpha1.Microservice:
					if name != "unit-test" {
						t.Fatalf("Expected Microservice name to be `unit-test`, got `%s`", name)
					}

					if namespace != "testing" {
						t.Fatalf("Expected Microservice namespace to be `testing`, got `%s`", namespace)
					}

					obj.Status.Releases = []v1alpha1.Release{}
					return nil
				}

				return fmt.Errorf("Object %T not supported", obj)
			}

			if err := ctrl.syncNetworking(np); err != nil {
				t.Errorf("Expected no error, got '%s'", err)
			}

			if l := len(np.Status.Domains); l != 0 {
				t.Errorf("Expected no domains to be set, got %d", l)
			}
		})

		t.Run("with a release", func(t *testing.T) {
			defer pc.Flush()

			np := &v1alpha1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unit-tests",
					Namespace: "testing",
				},
				Spec: v1alpha1.NetworkPolicySpec{
					Microservice: &v1.LocalObjectReference{
						Name: "unit-test",
					},
				},
			}

			pc.GetFunc = func(obj interface{}, namespace, name string) error {
				switch obj := obj.(type) {
				case *v1alpha1.Microservice:
					if name != "unit-test" {
						t.Fatalf("Expected Microservice name to be `unit-test`, got `%s`", name)
					}

					if namespace != "testing" {
						t.Fatalf("Expected Microservice namespace to be `testing`, got `%s`", namespace)
					}

					obj.Status.Releases = []v1alpha1.Release{
						{
							Level: v1alpha1.SemVerLevelRelease,
							SemVer: &v1alpha1.SemVerRelease{
								Name:    "unit-test",
								Version: "1.2.3",
							},
						},
					}
					return nil
				}

				return fmt.Errorf("Object %T not supported", obj)
			}

			pc.ApplyFunc = func(obj runtime.Object, opts ...patcher.OptionFunc) ([]byte, error) {
				// XXX test this more specifically
				return nil, nil
			}

			if err := ctrl.syncNetworking(np); err != nil {
				t.Errorf("Expected no error, got '%s'", err)
			}

			if l := len(np.Status.Domains); l != 0 {
				t.Errorf("Expected no domains to be set, got %d", l)
			}
		})
	})
}
