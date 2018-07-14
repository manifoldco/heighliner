package hub

import (
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/heroku/docker-registry-client/registry"
	digest "github.com/opencontainers/go-digest"
	"k8s.io/api/core/v1"

	"github.com/manifoldco/heighliner/internal/api/v1alpha1"
	reg "github.com/manifoldco/heighliner/internal/registry"
)

func TestConfigFromSecret(t *testing.T) {
	tcs := []struct {
		name string

		username string
		password string
		noErr    bool

		data string
	}{
		{"ok", "hlnr-user", "s3cr4t", true,
			`{
				"https://index.docker.io/v1/": {
				  "username": "hlnr-user",
				  "password": "s3cr4t"
			    }
			}`,
		},

		{"empty file", "", "", false, ``},
		{"bad json", "", "", false, `{ this isn't json`},
		{"json with the wrong structure", "", "", false, `[]`},

		{"json missing username", "", "", false,
			`{
				"https://index.docker.io/v1/": {
				  "password": "s3cr4t"
			    }
			}`,
		},

		{"json missing password", "", "", false,
			`{
				"https://index.docker.io/v1/": {
				  "username": "hlnr-user"
			    }
			}`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			secret := v1.Secret{Data: map[string][]byte{
				".dockercfg": []byte(tc.data),
			}}
			u, p, err := configFromSecret(&secret)

			if tc.noErr && err != nil {
				t.Fatal("expected no err but got one:", err)
			}

			if !tc.noErr && err == nil {
				t.Fatal("expected err but got none.")
			}

			if u != tc.username {
				t.Error("Wrong username. expected:", tc.username, "got:", u)
			}
			if p != tc.password {
				t.Error("Wrong username. expected:", tc.password, "got:", p)
			}
		})
	}
}

func TestClientTagFor(t *testing.T) {
	tcs := []struct {
		name string

		out string
		err error

		resp    digest.Digest
		errResp error

		match *v1alpha1.ImagePolicyMatch
	}{
		{"ok", "v1.0.0", nil, digest.Digest("fake"), nil, nil},
		{"can map", "1.0.0", nil, digest.Digest("fake"), nil, &v1alpha1.ImagePolicyMatch{
			Name: &v1alpha1.ImagePolicyMatchMapping{From: "v{{.Tag}}"},
		}},

		{
			"not found", "",
			reg.NewTagNotFoundError("testrepo", "v1.0.0"),
			digest.Digest(""),
			&registry.HttpStatusError{Response: &http.Response{StatusCode: 404}},
			nil,
		},

		{
			"registry 500", "",
			&registry.HttpStatusError{Response: &http.Response{StatusCode: 500}},
			digest.Digest(""),
			&registry.HttpStatusError{Response: &http.Response{StatusCode: 500}},
			nil,
		},
		{
			"registry non-http error", "",
			errors.New("bad"),
			digest.Digest(""),
			errors.New("bad"),
			nil,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			c := &Client{c: testRegistry{tc.resp, tc.errResp}}

			out, err := c.TagFor("testrepo", "v1.0.0", tc.match)

			if !reflect.DeepEqual(err, tc.err) {
				t.Fatal("Wrong err result. expected:", tc.err, "got:", err)
			}
			if out != tc.out {
				t.Error("Wrong tag. expected:", tc.out, "got:", out)
			}
		})
	}
}

type testRegistry struct {
	d digest.Digest
	e error
}

func (t testRegistry) ManifestDigest(repository, image string) (digest.Digest, error) {
	return t.d, t.e
}
