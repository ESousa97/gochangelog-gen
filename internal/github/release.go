// Package github handles interactions with the GitHub API for release management.
package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// ErrReleaseFailed is returned when the GitHub API rejects a release creation request.
var ErrReleaseFailed = errors.New("failed to create GitHub release")

// Client wraps an HTTP client to interact with the GitHub Releases API.
type Client struct {
	httpClient *http.Client
	token      string
}

// NewClient creates a new GitHub API client with sensible timeouts.
func NewClient(token string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DialContext:           (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
				TLSHandshakeTimeout:   5 * time.Second,
				ResponseHeaderTimeout: 10 * time.Second,
				IdleConnTimeout:       90 * time.Second,
				MaxIdleConns:          10,
				MaxIdleConnsPerHost:   5,
			},
		},
		token: token,
	}
}

// releasePayload is the JSON body sent to the GitHub create release endpoint.
type releasePayload struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Body    string `json:"body"`
	Draft   bool   `json:"draft"`
}

// CreateDraftRelease creates a draft release on GitHub for the given repository.
// It returns [ErrReleaseFailed] wrapped with status details if the API returns
// a non-201 response.
func (c *Client) CreateDraftRelease(ctx context.Context, owner, repo, version, changelog string) error {
	if c.token == "" {
		return errors.New("GITHUB_TOKEN is empty; cannot create release")
	}

	payload := releasePayload{
		TagName: version,
		Name:    version,
		Body:    changelog,
		Draft:   true,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling release payload: %w", err)
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("creating HTTP request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing HTTP request: %w", err)
	}
	defer func() {
		// Drain and close body to allow connection reuse.
		_, _ = io.Copy(io.Discard, resp.Body) //nolint:errcheck
		_ = resp.Body.Close()                 //nolint:errcheck
	}()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck
		return fmt.Errorf("%w: status %d, body: %s", ErrReleaseFailed, resp.StatusCode, string(body))
	}

	return nil
}
