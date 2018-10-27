package hub

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/docker/distribution/manifest/schema2"
	"github.com/heroku/docker-registry-client/registry"
	digest "github.com/opencontainers/go-digest"
	"k8s.io/api/core/v1"

	"github.com/manifoldco/heighliner/apis/heighliner/v1alpha1"
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

		tags   []string
		tagErr error

		manifests   map[string]*schema2.DeserializedManifest
		manifestErr error

		labels   map[string]string
		labelErr error

		match *v1alpha1.ImagePolicyMatch
	}{
		{"ok", "v1.0.0", nil, nil, nil,
			map[string]*schema2.DeserializedManifest{"v1.0.0": {}},
			nil, nil, nil, nil},
		{"can map and lookup on name", "1.0.0", nil, nil, nil,
			map[string]*schema2.DeserializedManifest{"1.0.0": {}},
			nil, nil, nil,
			&v1alpha1.ImagePolicyMatch{
				Name: &v1alpha1.ImagePolicyMatchMapping{From: "v{{.Tag}}"},
			},
		},

		{
			"manifest not found", "",
			reg.NewTagNotFoundError("testrepo", "v1.0.0"),
			nil, nil,
			map[string]*schema2.DeserializedManifest{"v1.0.0": nil},
			&url.Error{Err: &registry.HttpStatusError{Response: &http.Response{StatusCode: 404}}},
			nil, nil,
			&v1alpha1.ImagePolicyMatch{
				Labels: map[string]v1alpha1.ImagePolicyMatchMapping{
					"org.fake.label": {},
				},
			},
		},

		{
			"registry 500 on manifest lookup", "",
			&registry.HttpStatusError{Response: &http.Response{StatusCode: 500}},
			[]string{"v1.0.0"},
			nil,
			map[string]*schema2.DeserializedManifest{"v1.0.0": nil},
			&registry.HttpStatusError{Response: &http.Response{StatusCode: 500}},
			map[string]string{
				"v1.0.0": `{ "container_config": { "Labels": { "org.fake.label": "v1.0.0" } } }`,
			}, nil,
			&v1alpha1.ImagePolicyMatch{
				Labels: map[string]v1alpha1.ImagePolicyMatchMapping{
					"org.fake.label": {},
				},
			},
		},

		{
			"registry non-http error on manifest lookup", "",
			errors.New("bad"),
			[]string{"v1.0.0"},
			nil,
			map[string]*schema2.DeserializedManifest{"v1.0.0": nil},
			errors.New("bad"),
			map[string]string{
				"v1.0.0": `{ "container_config": { "Labels": { "org.fake.label": "v1.0.0" } } }`,
			}, nil,
			&v1alpha1.ImagePolicyMatch{
				Labels: map[string]v1alpha1.ImagePolicyMatchMapping{
					"org.fake.label": {},
				},
			},
		},

		{
			"can match by label", "v1.0.0", nil,
			[]string{"v1.0.0"}, nil,
			map[string]*schema2.DeserializedManifest{"v1.0.0": {}}, nil,
			map[string]string{
				"v1.0.0": `{ "container_config": { "Labels": { "org.fake.label": "v1.0.0" } } }`,
			}, nil,
			&v1alpha1.ImagePolicyMatch{
				Labels: map[string]v1alpha1.ImagePolicyMatchMapping{
					"org.fake.label": {},
				},
			},
		},

		{
			"can exclude by label", "",
			reg.NewTagNotFoundError("testrepo", "v1.0.0"),
			[]string{"v1.0.0"}, nil,
			map[string]*schema2.DeserializedManifest{"v1.0.0": {}}, nil,
			map[string]string{
				"v1.0.0": `{ "container_config": { "Labels": { "org.fake.label": "v1.0.0" } } }`,
			}, nil,
			&v1alpha1.ImagePolicyMatch{
				Labels: map[string]v1alpha1.ImagePolicyMatchMapping{
					"org.fake.other.label": {},
				},
			},
		},

		{
			"can match by label from many", "v1.0.0", nil,
			[]string{"v2.0.0", "v1.0.0", "v0.0.1"}, nil,
			map[string]*schema2.DeserializedManifest{
				"v2.0.0": {},
				"v1.0.0": {},
				"v0.0.1": {},
			}, nil,
			map[string]string{
				"v2.0.0": `{ "container_config": { "Labels": { "org.fake.other.label": "v1.0.0" } } }`,
				"v1.0.0": `{ "container_config": { "Labels": { "org.fake.label": "v1.0.0" } } }`,
				"v0.0.1": `{ "container_config": { "Labels": {} } }`,
			}, nil,
			&v1alpha1.ImagePolicyMatch{
				Labels: map[string]v1alpha1.ImagePolicyMatchMapping{
					"org.fake.label": {},
				},
			},
		},

		{
			"propagates tag list error", "",
			errors.New("bad"),
			nil, errors.New("bad"),
			nil, nil,
			nil, nil,
			&v1alpha1.ImagePolicyMatch{
				Labels: map[string]v1alpha1.ImagePolicyMatchMapping{
					"org.fake.label": {},
				},
			},
		},

		{
			"propagates config download error", "",
			errors.New("bad"),
			[]string{"v1.0.0"}, nil,
			map[string]*schema2.DeserializedManifest{"v1.0.0": {}}, nil,
			map[string]string{"v1.0.0": ``}, errors.New("bad"),
			&v1alpha1.ImagePolicyMatch{
				Labels: map[string]v1alpha1.ImagePolicyMatchMapping{
					"org.fake.label": {},
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {

			ms := make(map[string]*schema2.DeserializedManifest)
			ls := make(map[string]string)

			var i int
			for k, v := range tc.manifests {
				if v != nil {
					dig := digest.Digest(fmt.Sprintf("%d", i))
					v.Config.Digest = dig

					if v, ok := tc.labels[k]; ok {
						ls[string(dig)] = v
					}
				}

				ms[k] = v
				i++
			}

			c := &Client{c: testRegistry{
				ts: tc.tags,
				te: tc.tagErr,
				m:  ms,
				me: tc.manifestErr,
				l:  ls,
				le: tc.labelErr,
			}}

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
	ts []string
	te error

	m  map[string]*schema2.DeserializedManifest
	me error

	l  map[string]string
	le error
}

func (t testRegistry) Tags(repository string) ([]string, error) {
	return t.ts, t.te
}

func (t testRegistry) ManifestV2(repository, image string) (*schema2.DeserializedManifest, error) {
	dm, ok := t.m[image]
	if !ok {
		return nil, errors.New("asked for an image the tests didn't know about: " + image)
	}

	return dm, t.me
}

func (t testRegistry) DownloadLayer(repository string, dig digest.Digest) (io.ReadCloser, error) {
	return ioutil.NopCloser(strings.NewReader(t.l[string(dig)])), t.le
}
