package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
)

// Client wraps the GitHub API client.
//
//nolint:containedctx // deferred: threading ctx through every method and caller is a larger change
type Client struct {
	httpClient *http.Client
	apiClient  *api.RESTClient
	ctx        context.Context
}

// NewClient creates a new GitHub API client
// It will use gh CLI authentication if available, or fall back to GITHUB_TOKEN env var.
func NewClient(ctx context.Context) (*Client, error) {
	opts := clientOptions()

	// Create REST client (will use gh CLI auth or GITHUB_TOKEN)
	restClient, err := api.NewRESTClient(*opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Create HTTP client
	httpClient, err := api.NewHTTPClient(*opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return &Client{
		httpClient: httpClient,
		apiClient:  restClient,
		ctx:        ctx,
	}, nil
}

// NewClientWithTransport creates a client whose requests are served by rt,
// bypassing gh CLI auth resolution. Intended for tests using httptest or
// in-memory round-trip fakes; no network is reached.
func NewClientWithTransport(ctx context.Context, rt http.RoundTripper) (*Client, error) {
	opts := api.ClientOptions{
		Host:      testAPIHost,
		AuthToken: testAuthToken,
		Transport: rt,
	}

	restClient, err := api.NewRESTClient(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	httpClient, err := api.NewHTTPClient(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return &Client{
		httpClient: httpClient,
		apiClient:  restClient,
		ctx:        ctx,
	}, nil
}

// NewClientWithRealAuthAndTransport creates a client that resolves real gh CLI
// auth (host and token) but routes requests through rt instead of the default
// transport. For cassette-recording tools only: it panics under `go test` so
// tests keep using the fake-token seam in NewClientWithTransport.
func NewClientWithRealAuthAndTransport(ctx context.Context, rt http.RoundTripper) (*Client, error) {
	if testing.Testing() {
		panic("github.NewClientWithRealAuthAndTransport must not be used under go test")
	}

	opts := api.ClientOptions{Transport: rt}

	restClient, err := api.NewRESTClient(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	httpClient, err := api.NewHTTPClient(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return &Client{
		httpClient: httpClient,
		apiClient:  restClient,
		ctx:        ctx,
	}, nil
}

// NewClientWithToken creates a new GitHub API client with an explicit token.
func NewClientWithToken(ctx context.Context, token string) (*Client, error) {
	opts := api.ClientOptions{
		AuthToken: token,
	}

	restClient, err := api.NewRESTClient(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	httpClient, err := api.NewHTTPClient(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return &Client{
		httpClient: httpClient,
		apiClient:  restClient,
		ctx:        ctx,
	}, nil
}

// Get performs a GET request to the GitHub API.
func (c *Client) Get(path string, response interface{}) error {
	if err := c.apiClient.Get(path, response); err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}

	return nil
}

// Post performs a POST request to the GitHub API.
func (c *Client) Post(path string, body, response interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	if err := c.apiClient.Post(path, bytes.NewReader(jsonBody), response); err != nil {
		return fmt.Errorf("POST %s: %w", path, err)
	}

	return nil
}

// Patch performs a PATCH request to the GitHub API.
func (c *Client) Patch(path string, body, response interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	if err := c.apiClient.Patch(path, bytes.NewReader(jsonBody), response); err != nil {
		return fmt.Errorf("PATCH %s: %w", path, err)
	}

	return nil
}

// Put performs a PUT request to the GitHub API.
func (c *Client) Put(path string, body, response interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	if err := c.apiClient.Put(path, bytes.NewReader(jsonBody), response); err != nil {
		return fmt.Errorf("PUT %s: %w", path, err)
	}

	return nil
}

// Delete performs a DELETE request to the GitHub API.
func (c *Client) Delete(path string, response interface{}) error {
	if err := c.apiClient.Delete(path, response); err != nil {
		return fmt.Errorf("DELETE %s: %w", path, err)
	}

	return nil
}

// Context returns the client's context.
func (c *Client) Context() context.Context {
	return c.ctx
}
