package v1alpha1

import (
	"testing"
)

func TestReleaseNaming(t *testing.T) {
	t.Run("with SemVer Release", func(t *testing.T) {
		tcs := []struct {
			tcName        string
			name          string
			semVerName    string
			semVerVersion string
			semVerLevel   SemVerLevel
			streamName    string
			fullName      string
		}{
			{"release level", "hello-world", "hello-world", "v1.2.3", SemVerLevelRelease, "hello-world", "hello-world-hqo6t73v"},
			{"release level ignores semver name", "hello-world", "other-world", "v1.2.3", SemVerLevelRelease, "hello-world", "hello-world-hqo6t73v"},
			{"release level full name uses version", "hello-world", "hello-world", "v1.2.4", SemVerLevelRelease, "hello-world", "hello-world-ubdj93q6"},

			{"candidate level", "hello-world", "hello-world", "v1.2.3", SemVerLevelReleaseCandidate, "hello-world-rc", "hello-world-rc-hqo6t73v"},
			{"candidate level ignores semver name", "hello-world", "other-world", "v1.2.3", SemVerLevelReleaseCandidate, "hello-world-rc", "hello-world-rc-hqo6t73v"},
			{"candidate level full name uses version", "hello-world", "hello-world", "v1.2.4", SemVerLevelReleaseCandidate, "hello-world-rc", "hello-world-rc-ubdj93q6"},

			{"pr level", "hello-world", "hello-world", "v1.2.3", SemVerLevelPreview, "hello-world-pr-cmqolv9f", "hello-world-pr-cmqolv9f-hqo6t73v"},
			{"pr level uses semver name", "hello-world", "other-world", "v1.2.3", SemVerLevelPreview, "hello-world-pr-hulm66p0", "hello-world-pr-hulm66p0-hqo6t73v"},
			{"pr level full name uses version", "hello-world", "hello-world", "v1.2.4", SemVerLevelPreview, "hello-world-pr-cmqolv9f", "hello-world-pr-cmqolv9f-ubdj93q6"},
		}

		for _, tc := range tcs {
			release := &Release{
				SemVer: &SemVerRelease{
					Name:    tc.semVerName,
					Version: tc.semVerVersion,
				},
				Level: tc.semVerLevel,
			}

			t.Run(tc.tcName+" (stream name)", func(t *testing.T) {
				if name := release.StreamName(tc.name); name != tc.streamName {
					t.Errorf("Expected '%s', got '%s'", tc.streamName, name)
				}
			})

			t.Run(tc.tcName+" (full name)", func(t *testing.T) {
				if name := release.FullName(tc.name); name != tc.fullName {
					t.Errorf("Expected '%s', got '%s'", tc.fullName, name)
				}
			})
		}
	})
}
