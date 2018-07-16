// Package hub represents the registry implementation for Docker Hub.
package hub

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/docker/distribution/manifest/schema2"
	"github.com/heroku/docker-registry-client/registry"
	"github.com/opencontainers/go-digest"
	"k8s.io/api/core/v1"

	"github.com/manifoldco/heighliner/apis/v1alpha1"
	reg "github.com/manifoldco/heighliner/internal/registry"
)

const dockerHubRegistryURL string = "https://registry-1.docker.io"

var errNoUsername = errors.New("username missing from configuration")
var errNoPassword = errors.New("password missing from configuration")

type regClient interface {
	Tags(string) ([]string, error)
	ManifestV2(string, string) (*schema2.DeserializedManifest, error)
	DownloadLayer(string, digest.Digest) (io.ReadCloser, error)
}

// Client is a docker registry client
type Client struct {
	c regClient
}

// New creates a new registry client for Docker Hub.
func New(secret *v1.Secret) (*Client, error) {
	// TODO(jelmer): we need to abstract this out. Docker Hub - hosted - has a
	// different interface than a local registry. We can do this detection based
	// on the hostname.
	// For now, we'll focus on docker hub.

	// get cfg from k8s secret
	u, p, err := configFromSecret(secret)
	if err != nil {
		return nil, err
	}

	c, err := registry.New(dockerHubRegistryURL, u, p)
	if err != nil {
		return nil, err
	}

	c.Logf = registry.Quiet

	return &Client{c: c}, nil
}

type auth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func configFromSecret(secret *v1.Secret) (string, string, error) {
	var creds map[string]auth
	credsJSON := secret.Data[".dockercfg"]
	if err := json.Unmarshal(credsJSON, &creds); err != nil {
		return "", "", err
	}

	// .dockercfg has one key which is the url of the docker registry
	var stanza auth
	for _, s := range creds {
		stanza = s
		break
	}

	if stanza.Username == "" {
		return "", "", errNoUsername
	}

	if stanza.Password == "" {
		return "", "", errNoPassword
	}

	return stanza.Username, stanza.Password, nil
}

type containerConfig struct {
	Labels map[string]string
}

type config struct {
	ContainerConfig containerConfig `json:"container_config"`
}

// TagFor returns the tag name that matches the provided repo and release.
// It returns a registry.TagNotFound error if no matching tag is found.
func (c *Client) TagFor(repo string, release string, matcher *v1alpha1.ImagePolicyMatch) (string, error) {

	hasName, hasLabels := matcher.Config()

	ts := []string{}

	if hasName {
		n, err := matcher.MapName(release)
		if err != nil {
			return "", err
		}
		ts = append(ts, n)
	} else {
		var err error
		ts, err = c.c.Tags(repo)
		if err != nil {
			return "", normalizeErr(repo, release, err)
		}
	}

	for _, t := range ts {
		m, err := c.c.ManifestV2(repo, t)
		if err != nil {
			return "", normalizeErr(repo, release, err)
		}

		var labels map[string]string
		if hasLabels {
			l, err := c.c.DownloadLayer(repo, m.Config.Digest)
			if err != nil {
				return "", normalizeErr(repo, release, err)
			}

			var c config
			d := json.NewDecoder(l)
			if err := d.Decode(&c); err != nil {
				return "", err
			}

			labels = c.ContainerConfig.Labels
		}

		matches, err := matcher.Matches(release, t, labels)
		if err != nil {
			return "", err
		}

		if matches {
			return t, nil
		}
	}

	return "", reg.NewTagNotFoundError(repo, release)
}

func normalizeErr(repo, release string, err error) error {
	if t, ok := err.(*registry.HttpStatusError); ok {
		if t.Response.StatusCode == http.StatusNotFound {
			return reg.NewTagNotFoundError(repo, release)
		}
	}

	return err
}
