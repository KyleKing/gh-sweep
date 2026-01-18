package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/api"
)

// Client wraps the GitHub API client
type Client struct {
	httpClient *http.Client
	apiClient  api.RESTClient
	ctx        context.Context
}

// NewClient creates a new GitHub API client
// It will use gh CLI authentication if available, or fall back to GITHUB_TOKEN env var
func NewClient(ctx context.Context) (*Client, error) {
	opts := &api.ClientOptions{}

	// Create REST client (will use gh CLI auth or GITHUB_TOKEN)
	restClient, err := gh.RESTClient(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Create HTTP client
	httpClient, err := gh.HTTPClient(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return &Client{
		httpClient: httpClient,
		apiClient:  restClient,
		ctx:        ctx,
	}, nil
}

// NewClientWithToken creates a new GitHub API client with an explicit token
func NewClientWithToken(ctx context.Context, token string) (*Client, error) {
	opts := &api.ClientOptions{
		AuthToken: token,
	}

	restClient, err := gh.RESTClient(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	httpClient, err := gh.HTTPClient(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return &Client{
		httpClient: httpClient,
		apiClient:  restClient,
		ctx:        ctx,
	}, nil
}

// Get performs a GET request to the GitHub API
func (c *Client) Get(path string, response interface{}) error {
	return c.apiClient.Get(path, response)
}

// Post performs a POST request to the GitHub API
func (c *Client) Post(path string, body interface{}, response interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}
	return c.apiClient.Post(path, bytes.NewReader(jsonBody), response)
}

// Patch performs a PATCH request to the GitHub API
func (c *Client) Patch(path string, body interface{}, response interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}
	return c.apiClient.Patch(path, bytes.NewReader(jsonBody), response)
}

// Put performs a PUT request to the GitHub API
func (c *Client) Put(path string, body interface{}, response interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}
	return c.apiClient.Put(path, bytes.NewReader(jsonBody), response)
}

// Delete performs a DELETE request to the GitHub API
func (c *Client) Delete(path string, response interface{}) error {
	return c.apiClient.Delete(path, response)
}

// Context returns the client's context
func (c *Client) Context() context.Context {
	return c.ctx
}
