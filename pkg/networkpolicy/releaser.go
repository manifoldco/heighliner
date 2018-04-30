package networkpolicy

import (
	"errors"

	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
)

// LatestReleaser is able to select a release based on the releasedat date.
type LatestReleaser struct{}

// ExternalRelease goes over all releases and releases the latest release based
// on the ReleasedAt timestamp.
func (r *LatestReleaser) ExternalRelease(releases []v1alpha1.Release) (*v1alpha1.Release, error) {
	if len(releases) == 0 {
		return nil, errors.New("Need at least one release to link to an external release")
	}

	latestRelease := releases[0]
	for _, release := range releases {
		if latestRelease.Released.Before(&release.Released) {
			latestRelease = release
		}
	}

	return &latestRelease, nil
}
