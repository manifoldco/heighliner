package meta

import (
	"reflect"
	"testing"

	"github.com/manifoldco/heighliner/apis/v1alpha1"
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

func TestTrim(t *testing.T) {
	tcs := []struct {
		name string
		in   string
		l    int
		out  string
	}{
		{"empty", "", 2, ""},
		{"short", "abc", 4, "abc"},
		{"long", "abcdef", 4, "abcd"},
		{"unicode", "😍cool stuff", 4, "😍coo"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			if trim(tc.in, tc.l) != tc.out {
				t.Error("trim did not match. got:", trim(tc.in, tc.l), "wanted:", tc.out)
			}
		})
	}
}

func TestElide(t *testing.T) {
	tcs := []struct {
		name string
		in   string
		l    int
		out  string
	}{
		{"empty", "", 4, ""},
		{"skip elide", "abc", 2, "ab"},
		{"short", "abc", 4, "abc"},
		{"long", "abcdef", 5, "a...0"},
		{"unicode", "😍cool stuff", 7, "😍co...0"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			if elide(tc.in, tc.l) != tc.out {
				t.Error("elide did not match. got:", elide(tc.in, tc.l), "wanted:", tc.out)
			}
		})
	}
}

func TestLabelize(t *testing.T) {
	tcs := []struct {
		name string
		in   string
		out  string
	}{
		{"empty", "", ""},
		{"same", "ab0-c", "ab0-c"},
		{"pad with 0s", ".abc.", "0.abc.0"},
		{"replace chars", "replace space? great", "replace_space__great"},
		{"replace unicode", "😍cool stuff", "0_cool_stuff"},
		{"elided",
			"The quick brown fox jumps over the lazy dog. The quick brown fox jumps over the lazy dog",
			"The_quick_brown_fox_jumps_over_the_lazy_dog._The_quick_brow...0"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			if labelize(tc.in) != tc.out {
				t.Error("labelize did not match. got:", labelize(tc.in), "wanted:", tc.out)
			}
		})
	}
}
