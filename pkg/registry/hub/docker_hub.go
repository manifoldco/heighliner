// Package hub represents the registry implementation for Docker Hub.
package hub

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"k8s.io/api/core/v1"
)

const (
	// DockerHubRegistryURL refers to the registry that will be used to access image on DockerHub
	DockerHubRegistryURL string = "https://registry-1.docker.io"

	// DockerHubManifestAcceptString refers to the application type to request for the image manifests
	DockerHubManifestAcceptString string = "application/vnd.docker.distribution.manifest.v2+json"

	// DockerHubRepoAuthString is the URL that must be authed to in order to get a repository access token
	DockerHubRepoAuthString string = "https://auth.docker.io/token?scope=repository:%s:pull&service=registry.docker.io"
)

// GetManifest returns a bool indicated weather or not the tag for that image is available
func (c *Client) GetManifest(repo string, tag string) (bool, error) {
	url := expandURL(c.Config.URL, fmt.Sprintf("/v2/%s/manifests/%s", repo, tag))

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", DockerHubManifestAcceptString)
	rsp, err := c.Client.Do(req)

	if err != nil {
		return false, err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		return false, nil
	}

	return true, nil
}

// Ping validates if the given credentials are valid for the given registry.
func (c *Client) Ping() error {
	rsp, err := c.Client.Get(expandURL(c.Config.URL, "/v2/"))
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		return errors.New(rsp.Status)
	}

	return nil
}

type (
	tokenTransport struct {
		URL       string
		Username  string
		Password  string
		ImageRepo string
		Next      http.RoundTripper
	}

	authToken struct {
		Token string `json:"token"`
	}

	// Config represents the options to create a new registry client.
	Config struct {
		URL       string
		Username  string
		Password  string
		ImageRepo string
	}

	// Client interacts with the Docker Hub API.
	Client struct {
		Config
		Client *http.Client
	}
)

// New creates a new registry client for Docker Hub.
func New(secret *v1.Secret, imageRepo string) (*Client, error) {
	// TODO(jelmer): we need to abstract this out. Docker Hub - hosted - has a
	// different interface than a local registry. We can do this detection based
	// on the hostname.
	// For now, we'll focus on docker hub.

	// get cfg from k8s secret
	cfg := configFromSecret(secret)
	cfg.ImageRepo = imageRepo

	return &Client{
		Config: cfg,
		Client: newHTTPClient(cfg),
	}, nil
}

func configFromSecret(secret *v1.Secret) Config {
	var creds = map[string]map[string]string{}
	credsJSON := secret.Data[".dockercfg"]
	json.Unmarshal(credsJSON, &creds)

	// .dockercfg has one key which is the url of the docker registry
	var url string
	for key := range creds {
		url = key
	}

	cfg := Config{
		URL:      DockerHubRegistryURL,
		Username: creds[url]["username"],
		Password: creds[url]["password"],
	}

	return cfg
}

func newHTTPClient(cfg Config) *http.Client {
	transport := http.DefaultTransport
	transport = &tokenTransport{
		URL:       cfg.URL,
		Username:  cfg.Username,
		Password:  cfg.Password,
		ImageRepo: cfg.ImageRepo,
		Next:      transport,
	}

	return &http.Client{
		Transport: transport,
	}
}

func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := t.newToken()
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	return t.Next.RoundTrip(req)
}

func (t *tokenTransport) newToken() (string, error) {

	authURL := fmt.Sprintf(DockerHubRepoAuthString, t.ImageRepo)
	req, err := http.NewRequest("GET", authURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(t.Username, t.Password)
	rsp, err := t.Next.RoundTrip(req)
	if err != nil {
		return "", err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode >= 400 {
		return "", errors.New(rsp.Status)
	}

	tokenData := &authToken{}
	decoder := json.NewDecoder(rsp.Body)
	if err := decoder.Decode(tokenData); err != nil {
		return "", err
	}

	return tokenData.Token, nil
}

func expandURL(url, path string) string {
	return fmt.Sprintf("%s%s", strings.TrimSuffix(url, "/"), path)
}
