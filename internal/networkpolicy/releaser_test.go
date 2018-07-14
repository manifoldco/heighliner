package networkpolicy

import (
	"testing"
	"time"

	"github.com/manifoldco/heighliner/internal/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLatestReleaser(t *testing.T) {
	releases := []v1alpha1.Release{
		{
			Image:       "1",
			ReleaseTime: metav1.Date(2018, time.April, 28, 13, 42, 01, 0, time.UTC),
		},
		{
			Image:       "2",
			ReleaseTime: metav1.Date(2018, time.April, 29, 13, 52, 01, 0, time.UTC),
		},
		{
			Image:       "3",
			ReleaseTime: metav1.Date(2018, time.April, 27, 13, 32, 01, 0, time.UTC),
		},
	}

	releaser := &LatestReleaser{}
	release, err := releaser.ExternalRelease(releases)
	if err != nil {
		t.Errorf("Expected no error, got '%s'", err)
	}

	if release.Image != "2" {
		t.Errorf("Expected release to be '2', got '%s'", release.Image)
	}
}
