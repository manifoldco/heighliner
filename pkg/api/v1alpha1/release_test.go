package v1alpha1

import (
	"testing"
)

func TestRelease_FullName(t *testing.T) {
	t.Run("with SemVer Release", func(t *testing.T) {
		testData := []struct {
			name          string
			semVerName    string
			semVerVersion string
			semVerBuild   string
			expected      string
		}{
			{"hello-world", "hello-world", "v1.2.3", "", "hello-world-594abfa937"},
			{"hello-world", "456-pr-branch", "v1.2.3", "201805011532", "hello-world-8a2f764d29"},
			{"hello-world", "456-pr-branch", "1.2.3", "201805011532", "hello-world-60882cbb00"},
			{"hello-world", "456-pr-branch", "v1.2.3", "201805011531", "hello-world-dcd827915f"},
			{"demo-app", "hello-world", "v1.2.3", "", "demo-app-594abfa937"},
		}

		for _, item := range testData {
			release := &Release{
				SemVer: &SemVerRelease{
					Name:    item.semVerName,
					Version: item.semVerVersion,
					Build:   item.semVerBuild,
				},
			}

			if fName := release.FullName(item.name); fName != item.expected {
				t.Errorf("Expected '%s', got '%s'", item.expected, fName)
			}
		}
	})
}
