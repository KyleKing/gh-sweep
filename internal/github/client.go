package github

import (
	"context"
	"fmt"
	"net/http"

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
	opts := api.ClientOptions{}

	// Create REST client (will use gh CLI auth or GITHUB_TOKEN)
	restClient, err := api.DefaultRESTClient(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Create HTTP client
	httpClient, err := api.DefaultHTTPClient()
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

// Get performs a GET request to the GitHub API
func (c *Client) Get(path string, response interface{}) error {
	return c.apiClient.Get(path, response)
}

// Post performs a POST request to the GitHub API
func (c *Client) Post(path string, body interface{}, response interface{}) error {
	return c.apiClient.Post(path, body, response)
}

// Patch performs a PATCH request to the GitHub API
func (c *Client) Patch(path string, body interface{}, response interface{}) error {
	return c.apiClient.Patch(path, body, response)
}

// Delete performs a DELETE request to the GitHub API
func (c *Client) Delete(path string, response interface{}) error {
	return c.apiClient.Delete(path, response)
}

// Context returns the client's context
func (c *Client) Context() context.Context {
	return c.ctx
}
