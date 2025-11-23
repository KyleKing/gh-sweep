package linear

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Client represents a Linear API client
type Client struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new Linear API client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{},
		baseURL:    "https://api.linear.app/graphql",
	}
}

// Issue represents a Linear issue
type Issue struct {
	ID       string
	Title    string
	State    string
	Assignee string
	Project  string
	Cycle    string
}

// graphQLRequest represents a GraphQL request
type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// graphQLResponse represents a GraphQL response
type graphQLResponse struct {
	Data   json.RawMessage        `json:"data"`
	Errors []graphQLError         `json:"errors,omitempty"`
}

type graphQLError struct {
	Message string `json:"message"`
}

// query executes a GraphQL query
func (c *Client) query(query string, variables map[string]interface{}) (json.RawMessage, error) {
	reqBody := graphQLRequest{
		Query:     query,
		Variables: variables,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	var gqlResp graphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", gqlResp.Errors[0].Message)
	}

	return gqlResp.Data, nil
}

// GetIssue retrieves an issue by ID
func (c *Client) GetIssue(issueID string) (*Issue, error) {
	query := `
		query GetIssue($id: String!) {
			issue(id: $id) {
				id
				title
				state { name }
				assignee { name }
				project { name }
				cycle { name }
			}
		}
	`

	variables := map[string]interface{}{
		"id": issueID,
	}

	data, err := c.query(query, variables)
	if err != nil {
		return nil, err
	}

	var result struct {
		Issue struct {
			ID    string `json:"id"`
			Title string `json:"title"`
			State struct {
				Name string `json:"name"`
			} `json:"state"`
			Assignee *struct {
				Name string `json:"name"`
			} `json:"assignee"`
			Project *struct {
				Name string `json:"name"`
			} `json:"project"`
			Cycle *struct {
				Name string `json:"name"`
			} `json:"cycle"`
		} `json:"issue"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal issue: %w", err)
	}

	issue := &Issue{
		ID:    result.Issue.ID,
		Title: result.Issue.Title,
		State: result.Issue.State.Name,
	}

	if result.Issue.Assignee != nil {
		issue.Assignee = result.Issue.Assignee.Name
	}

	if result.Issue.Project != nil {
		issue.Project = result.Issue.Project.Name
	}

	if result.Issue.Cycle != nil {
		issue.Cycle = result.Issue.Cycle.Name
	}

	return issue, nil
}

// ExtractLinearIssueID extracts a Linear issue ID from PR body
func ExtractLinearIssueID(body string) string {
	// Simple extraction: look for "LIN-" pattern
	// In production, use regex
	if len(body) > 0 {
		// Placeholder implementation
		return ""
	}
	return ""
}
