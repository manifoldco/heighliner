// Package hub represents the registry implementation for Docker Hub.
package hub

import (
	"encoding/json"
	"errors"
	"net/http"

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
	ManifestDigest(string, string) (digest.Digest, error)
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

// TagFor returns the tag name that matches the provided repo and release.
// It returns a registry.TagNotFound error if no matching tag is found.
func (c *Client) TagFor(repo string, release string, matcher *v1alpha1.ImagePolicyMatch) (string, error) {
	mapped, err := matcher.MapName(release)
	if err != nil {
		return "", err
	}

	_, err = c.c.ManifestDigest(repo, mapped)
	switch t := err.(type) {
	case nil:
		return mapped, nil
	case *registry.HttpStatusError:
		if t.Response.StatusCode == http.StatusNotFound {
			return "", reg.NewTagNotFoundError(repo, release)
		}

		return "", err
	default:
		return "", err
	}
}
