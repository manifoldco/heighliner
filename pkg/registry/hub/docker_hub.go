// Package hub represents the registry implementation for Docker Hub.
package hub

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type (
	resultsResponse struct {
		Count    int           `json:"count"`
		Next     *string       `json:"next,omitempty"`
		Previous *string       `json:"previous,omitempty"`
		Results  []interface{} `json:"results"`
	}

	tagsResponse struct {
		Name string `json:"name"`
	}
)

var (
	// ErrNoMorePages is used when the API doesn't have any more pages
	// available.
	ErrNoMorePages = errors.New("No more pages")
)

// Tags returns a list of available tags for the given repository.
func (c *Client) Tags(repo string, limit int) ([]string, error) {
	url := expandURL(c.Config.URL, fmt.Sprintf("/v2/repositories/%s/tags", repo))

	var tags []string
	var err error
	rsp := []tagsResponse{}
	for {
		url, err = c.getPaginatedResults(url, &rsp)
		switch err {
		case ErrNoMorePages:
			for _, v := range rsp {
				tags = append(tags, v.Name)
			}

			return tags, nil
		case nil:
			for _, v := range rsp {
				tags = append(tags, v.Name)
			}

			if limit > 0 && len(tags) >= limit {
				return tags, nil
			}

			continue
		default:
			return nil, err
		}
	}
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

func (c *Client) getPaginatedResults(url string, response interface{}) (string, error) {
	rsp, err := c.Client.Get(url)
	if err != nil {
		return "", err
	}
	defer rsp.Body.Close()

	resultsRsp := &resultsResponse{}
	if err := json.NewDecoder(rsp.Body).Decode(resultsRsp); err != nil {
		return "", err
	}

	results, err := json.Marshal(resultsRsp.Results)
	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(results, response); err != nil {
		return "", err
	}

	if resultsRsp.Next == nil {
		return "", ErrNoMorePages
	}

	return *resultsRsp.Next, nil
}

type (
	tokenTransport struct {
		URL      string
		Username string
		Password string

		Next http.RoundTripper
	}

	authToken struct {
		Token string `json:"token"`
	}

	loginData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	// Config represents the options to create a new registry client.
	Config struct {
		URL      string
		Username string
		Password string
	}

	// Client interacts with the Docker Hub API.
	Client struct {
		Config
		Client *http.Client
	}
)

// New creates a new registry client for Docker Hub.
func New(cfg Config) (*Client, error) {
	// TODO(jelmer): we need to abstract this out. Docker Hub - hosted - has a
	// different interface than a local registry. We can do this detection based
	// on the hostname.
	// For now, we'll focus on docker hub.
	return &Client{
		Config: cfg,
		Client: newHTTPClient(cfg),
	}, nil
}

func newHTTPClient(cfg Config) *http.Client {
	transport := http.DefaultTransport
	transport = &tokenTransport{
		URL:      cfg.URL,
		Username: cfg.Username,
		Password: cfg.Password,
		Next:     transport,
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
	ld := loginData{
		Username: t.Username,
		Password: t.Password,
	}
	bts, err := json.Marshal(ld)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", expandURL(t.URL, "/v2/users/login/"), bytes.NewBuffer(bts))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

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
