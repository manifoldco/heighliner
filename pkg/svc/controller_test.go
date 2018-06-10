package svc

import (
	"testing"

	"github.com/jelmersnoeck/kubekit/patcher"
	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
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

type kubekitClient struct {
	deleteFunc func(runtime.Object, ...patcher.OptionFunc) error
}

func (c *kubekitClient) Delete(obj runtime.Object, ops ...patcher.OptionFunc) error {
	return c.deleteFunc(obj, ops...)
}
