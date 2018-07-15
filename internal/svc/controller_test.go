package svc

import (
	"errors"
	"reflect"
	"testing"

	"github.com/jelmersnoeck/kubekit/patcher"
	"github.com/manifoldco/heighliner/internal/api/v1alpha1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestDeprecatedReleases(t *testing.T) {
	released := metav1.Now()
	t.Run("with no difference", func(t *testing.T) {
		releases := []v1alpha1.Release{
			{
				SemVer: &v1alpha1.SemVerRelease{
					Name:    "my-test1",
					Version: "1.2.3",
				},
				Level:       v1alpha1.SemVerLevelRelease,
				ReleaseTime: released,
			},
			{
				SemVer: &v1alpha1.SemVerRelease{
					Name:    "my-test1",
					Version: "1.2.4",
				},
				Level:       v1alpha1.SemVerLevelRelease,
				ReleaseTime: released,
			},
		}

		if ln := len(deprecatedReleases(releases, releases)); ln != 0 {
			t.Errorf("Expected length to equal 0, got %d", ln)
		}
	})

	t.Run("with differences", func(t *testing.T) {
		t.Run("with different versions", func(t *testing.T) {
			desired := []v1alpha1.Release{
				{
					SemVer: &v1alpha1.SemVerRelease{
						Name:    "my-test1",
						Version: "1.2.3",
					},
					Level:       v1alpha1.SemVerLevelRelease,
					ReleaseTime: released,
				},
				{
					SemVer: &v1alpha1.SemVerRelease{
						Name:    "my-test1",
						Version: "1.2.4",
					},
					Level:       v1alpha1.SemVerLevelRelease,
					ReleaseTime: released,
				},
			}

			current := []v1alpha1.Release{
				{
					SemVer: &v1alpha1.SemVerRelease{
						Name:    "my-test1",
						Version: "1.2.2",
					},
					Level:       v1alpha1.SemVerLevelRelease,
					ReleaseTime: released,
				},
				{
					SemVer: &v1alpha1.SemVerRelease{
						Name:    "my-test1",
						Version: "1.2.3",
					},
					Level:       v1alpha1.SemVerLevelRelease,
					ReleaseTime: released,
				},
				{
					SemVer: &v1alpha1.SemVerRelease{
						Name:    "my-test1",
						Version: "1.2.4",
					},
					Level:       v1alpha1.SemVerLevelRelease,
					ReleaseTime: released,
				},
			}

			if ln := len(deprecatedReleases(desired, current)); ln != 1 {
				t.Errorf("Expected length to equal 1, got %d", ln)
			}
		})

		t.Run("with missing versions", func(t *testing.T) {
			desired := []v1alpha1.Release{
				{
					SemVer: &v1alpha1.SemVerRelease{
						Name:    "my-test1",
						Version: "1.2.3",
					},
					Level:       v1alpha1.SemVerLevelRelease,
					ReleaseTime: released,
				},
				{
					SemVer: &v1alpha1.SemVerRelease{
						Name:    "my-test1",
						Version: "1.2.4",
					},
					Level:       v1alpha1.SemVerLevelRelease,
					ReleaseTime: released,
				},
			}

			current := []v1alpha1.Release{
				{
					SemVer: &v1alpha1.SemVerRelease{
						Name:    "my-test1",
						Version: "1.2.4",
					},
					Level:       v1alpha1.SemVerLevelRelease,
					ReleaseTime: released,
				},
			}

			if ln := len(deprecatedReleases(desired, current)); ln != 0 {
				t.Errorf("Expected length to equal 0, got %d", ln)
			}
		})
	})
}

func TestDeprecateReleases(t *testing.T) {
	cl := &kubekitClient{}
	cl.deleteFunc = func(obj runtime.Object, objs ...patcher.OptionFunc) error {
		vsvc := obj.(*v1alpha1.VersionedMicroservice)
		expected := "test-service-1mpl3547"
		if vsvc.Name != expected {
			t.Errorf("Expected name to be '%s', got '%s'", expected, vsvc.Name)
		}
		return nil
	}

	released := metav1.Now()
	releases := []v1alpha1.Release{
		{
			SemVer: &v1alpha1.SemVerRelease{
				Name:    "my-test1",
				Version: "1.2.3",
			},
			Level:       v1alpha1.SemVerLevelRelease,
			ReleaseTime: released,
		},
		{
			SemVer: &v1alpha1.SemVerRelease{
				Name:    "my-test1",
				Version: "1.2.4",
			},
			Level:       v1alpha1.SemVerLevelRelease,
			ReleaseTime: released,
		},
	}

	svc := &v1alpha1.Microservice{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-service",
		},
		Status: v1alpha1.MicroserviceStatus{
			Releases: []v1alpha1.Release{
				{
					SemVer: &v1alpha1.SemVerRelease{
						Name:    "my-test1",
						Version: "1.2.2",
					},
					Level:       v1alpha1.SemVerLevelRelease,
					ReleaseTime: released,
				},
				{
					SemVer: &v1alpha1.SemVerRelease{
						Name:    "my-test1",
						Version: "1.2.3",
					},
					Level:       v1alpha1.SemVerLevelRelease,
					ReleaseTime: released,
				},
				{
					SemVer: &v1alpha1.SemVerRelease{
						Name:    "my-test1",
						Version: "1.2.4",
					},
					Level:       v1alpha1.SemVerLevelRelease,
					ReleaseTime: released,
				},
			},
		},
	}

	if err := deprecateReleases(cl, svc, releases); err != nil {
		t.Errorf("Didn't expect error deprecating releases but got '%s'", err)
	}
}

func TestController_PatchMicroservice(t *testing.T) {
	cl := new(kubekitClient)
	ctrl := &Controller{patcher: cl}
	deploy := &v1alpha1.Microservice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deploy",
			Namespace: "testing",
		},
	}

	t.Run("without releases", func(t *testing.T) {
		defer cl.flush()

		cl.getFunc = func(obj interface{}, namespace, name string) error {
			if obj, ok := obj.(*v1alpha1.ImagePolicy); ok {
				obj.Status = v1alpha1.ImagePolicyStatus{
					Releases: []v1alpha1.Release{},
				}

				return nil
			}

			return errors.New("Object not supported")
		}

		cl.applyFunc = func(obj runtime.Object, opts ...patcher.OptionFunc) ([]byte, error) {
			msvc := obj.(*v1alpha1.Microservice)
			if len(msvc.Status.Releases) > 0 {
				t.Errorf("Expected no releases to be set up")
			}

			return nil, nil
		}

		err := ctrl.patchMicroservice(deploy)
		if err != nil {
			t.Errorf("Expected no error, got %s", err)
		}
	})

	t.Run("with a release", func(t *testing.T) {
		t.Run("without extra config", func(t *testing.T) {
			defer cl.flush()

			fullName := "test-deploy-pr-2l6fggiv-ribi3jce"
			cl.getFunc = func(obj interface{}, namespace, name string) error {
				if obj, ok := obj.(*v1alpha1.ImagePolicy); ok {
					obj.Status = v1alpha1.ImagePolicyStatus{
						Releases: []v1alpha1.Release{
							{
								Level: v1alpha1.SemVerLevelPreview,
								SemVer: &v1alpha1.SemVerRelease{
									Name:    "pr-branch",
									Version: "46ef87b86b66a301c3ac1f072d630d08bbd77420",
								},
							},
						},
					}

					return nil
				}

				// object will already be patched, no need to refresh in tests
				if _, ok := obj.(*v1alpha1.VersionedMicroservice); ok {
					return nil
				}

				return errors.New("Object not supported")
			}

			cl.applyFunc = func(obj runtime.Object, opts ...patcher.OptionFunc) ([]byte, error) {
				if vsvc, vok := obj.(*v1alpha1.VersionedMicroservice); vok {
					expected := &v1alpha1.VersionedMicroservice{
						TypeMeta: metav1.TypeMeta{
							Kind:       "VersionedMicroservice",
							APIVersion: "hlnr.io/v1alpha1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      fullName,
							Namespace: "testing",
							Labels: map[string]string{
								"hlnr.io/microservice.full_name": fullName,
								"hlnr.io/microservice.name":      "test-deploy",
								"hlnr.io/microservice.release":   "pr-branch",
								"hlnr.io/microservice.version":   "46ef87b86b66a301c3ac1f072d630d08bbd77420",
								"hlnr.io/service":                "test-deploy",
							},
							OwnerReferences: vsvc.OwnerReferences, // XXX test nicely
						},
						Spec: v1alpha1.VersionedMicroserviceSpec{
							Containers: []v1.Container{
								{
									Name:            "test-deploy",
									ImagePullPolicy: v1.PullIfNotPresent,
								},
							},
						},
					}

					if !reflect.DeepEqual(vsvc, expected) {
						t.Errorf("Expected %T to equal %T", vsvc, expected)
					}
				}

				if svc, ok := obj.(*v1alpha1.Microservice); ok {
					if l := len(svc.Status.Releases); l != 1 {
						t.Errorf("Expected 1 release, got %d", l)
					}
				}

				return nil, nil
			}

			err := ctrl.patchMicroservice(deploy)
			if err != nil {
				t.Errorf("Expected no error, got %s", err)
			}
		})

	})
}

type kubekitClient struct {
	applyFunc  func(obj runtime.Object, opts ...patcher.OptionFunc) ([]byte, error)
	getFunc    func(obj interface{}, namespace, name string) error
	deleteFunc func(runtime.Object, ...patcher.OptionFunc) error
}

func (c *kubekitClient) flush() {
	c.applyFunc = nil
	c.getFunc = nil
	c.deleteFunc = nil
}

func (c *kubekitClient) Apply(obj runtime.Object, opts ...patcher.OptionFunc) ([]byte, error) {
	return c.applyFunc(obj, opts...)
}

func (c *kubekitClient) Get(obj interface{}, namespace, name string) error {
	return c.getFunc(obj, namespace, name)
}

func (c *kubekitClient) Delete(obj runtime.Object, ops ...patcher.OptionFunc) error {
	return c.deleteFunc(obj, ops...)
}
