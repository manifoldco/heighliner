package networkpolicy

import (
	"testing"

	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
)

func TestGroupReleases(t *testing.T) {
	t.Run("with a set of semver releases", func(t *testing.T) {
		t.Run("with different PR applications", func(t *testing.T) {

			releases := []v1alpha1.Release{
				{
					Image: "hlnr.io/test:1.2.3-pr.456-pr+201804281301",
					SemVer: &v1alpha1.SemVerRelease{
						Name: "456-pr",
					},
				},
				{
					Image: "hlnr.io/test:1.2.3-pr.456-pr+201804281308",
					SemVer: &v1alpha1.SemVerRelease{
						Name: "456-pr",
					},
				},
				{
					Image: "hlnr.io/test:1.2.3-pr.457-pr+201804281307",
					SemVer: &v1alpha1.SemVerRelease{
						Name: "457-pr",
					},
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
