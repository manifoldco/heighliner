package imagepolicy

import (
	"fmt"
	"testing"

	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
	"k8s.io/api/core/v1"
)

func TestFilterImages(t *testing.T) {

	repo := &v1alpha1.GitHubRepository{
		Spec: v1alpha1.GitHubRepositorySpec{
			Repo: "github.com//manifoldco/heighliner",
		},
		Status: v1alpha1.GitHubRepositoryStatus{
			Releases: []v1alpha1.GitHubRelease{
				{
					Name:  "heighliner",
					Tag:   "latest",
					Level: v1alpha1.SemVerLevelRelease,
				},
				{
					Name:  "heighliner",
					Tag:   "rc",
					Level: v1alpha1.SemVerLevelReleaseCandidate,
				},
				{
					Name:  "heighliner",
					Tag:   "pr",
					Level: v1alpha1.SemVerLevelPreview,
				},
			},
		},
	}

	registry := &mockRegistryClient{}

	tcs := []struct {
		level v1alpha1.SemVerLevel
		tag   string
	}{
		{v1alpha1.SemVerLevelRelease, "latest"},
		{v1alpha1.SemVerLevelReleaseCandidate, "rc"},
		{v1alpha1.SemVerLevelPreview, "pr"},
	}

	for _, tc := range tcs {
		t.Run(fmt.Sprintf("with %s version policy for %s", tc.level, tc.tag), func(t *testing.T) {
			ip := &v1alpha1.ImagePolicy{
				Spec: v1alpha1.ImagePolicySpec{
					Image: "manifoldco/heighliner",
					Filter: v1alpha1.ImagePolicyFilter{
						GitHub: &v1.ObjectReference{
							Name: "github.com/manifoldco/heighliner",
						},
					},
				},
			}
			vp := &v1alpha1.VersioningPolicy{
				Spec: v1alpha1.VersioningPolicySpec{
					SemVer: &v1alpha1.SemVerSource{
						Level: tc.level,
					},
				},
			}

			expectedReleases := []v1alpha1.Release{
				{
					SemVer: &v1alpha1.SemVerRelease{
						Name:    "heighliner",
						Version: tc.tag,
					},

					Image: "manifoldco/heighliner:" + tc.tag,
				},
			}

			actualReleases, err := filterImages(ip.Spec.Image, repo, registry, vp)
			if err != nil {
				t.Errorf("Error filtering images for %s", ip.Name)
			}

			if expectedReleases[0].Image != actualReleases[0].Image {
				t.Errorf("Expected release image for release to be %s instead got %s", expectedReleases[0].Image, actualReleases[0].Image)
			}
		})
	}
}

type mockRegistryClient struct{}

func (c *mockRegistryClient) Ping() error {
	return nil
}

func (c *mockRegistryClient) GetManifest(image string, tag string) (bool, error) {
	return true, nil
}
