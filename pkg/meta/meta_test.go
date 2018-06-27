package meta

import (
	"reflect"
	"testing"

	"github.com/manifoldco/heighliner/pkg/api/v1alpha1"
)

func TestMicroserviceLabels(t *testing.T) {

	ms := &v1alpha1.Microservice{}
	ms.Name = "test"
	r := &v1alpha1.Release{
		SemVer: &v1alpha1.SemVerRelease{
			Name:    "a-branch",
			Version: "0.0.1",
		},
		Level: v1alpha1.SemVerLevelPreview,
	}

	l := MicroserviceLabels(ms, r, ms)

	expected := map[string]string{
		"hlnr.io/service":                "test",
		"hlnr.io/microservice.name":      "test",
		"hlnr.io/microservice.full_name": "test-pr-ebq4dofr-svek39uq",
		"hlnr.io/microservice.release":   "a-branch",
		"hlnr.io/microservice.version":   "0.0.1",
	}

	if !reflect.DeepEqual(l, expected) {
		t.Error("labels did not match. got:", l, "wanted:", expected)
	}
}
