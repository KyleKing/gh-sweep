package linear

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
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

// ExtractLinearIssueIDs extracts Linear issue IDs from PR body
// Pure function: regex-based extraction
func ExtractLinearIssueIDs(body string) []string {
	// Match common linking patterns:
	// - Fixes LIN-123
	// - Closes LIN-456
	// - Resolves LIN-789
	// - Refs LIN-101
	pattern := regexp.MustCompile(`(?i)(?:fixes|closes|resolves|refs?)\s+([A-Z]+-\d+)`)
	matches := pattern.FindAllStringSubmatch(body, -1)

	// Deduplicate IDs
	idSet := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			idSet[match[1]] = true
		}
	}

	// Convert to slice
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	return ids
}

// PRIssuePair represents a GitHub PR linked to a Linear issue
type PRIssuePair struct {
	Repository  string
	PRNumber    int
	PRStatus    string // open, merged, closed
	PRTitle     string
	IssueID     string
	Issue       *Issue
	InSync      bool
	DriftReason string // Why they're out of sync
}

// CheckPRIssueSyncStatus determines if a PR and issue are in sync
// Pure function: compares states
func CheckPRIssueSyncStatus(prStatus string, issueState string) (bool, string) {
	// Expected state transitions:
	// PR open -> Issue should be "In Progress" or "Todo"
	// PR merged -> Issue should be "Done" or "Closed"
	// PR closed (not merged) -> Issue should be "Canceled" or "Closed"

	switch prStatus {
	case "merged":
		if issueState == "Done" || issueState == "Closed" || issueState == "Completed" {
			return true, ""
		}
		return false, fmt.Sprintf("PR merged but issue is '%s' (expected Done/Closed)", issueState)

	case "closed":
		if issueState == "Canceled" || issueState == "Cancelled" || issueState == "Closed" {
			return true, ""
		}
		return false, fmt.Sprintf("PR closed but issue is '%s' (expected Canceled/Closed)", issueState)

	case "open":
		if issueState == "Done" || issueState == "Closed" || issueState == "Completed" {
			return false, fmt.Sprintf("PR open but issue is '%s' (expected In Progress/Todo)", issueState)
		}
		return true, ""

	default:
		return true, "" // Unknown status, assume in sync
	}
}

// AnalyzePRIssueLinks analyzes PR-issue pairs for sync status
// Pure function: maps over pairs to check sync
func AnalyzePRIssueLinks(pairs []PRIssuePair) []PRIssuePair {
	analyzed := make([]PRIssuePair, len(pairs))

	for i, pair := range pairs {
		analyzed[i] = pair

		if pair.Issue != nil {
			inSync, reason := CheckPRIssueSyncStatus(pair.PRStatus, pair.Issue.State)
			analyzed[i].InSync = inSync
			analyzed[i].DriftReason = reason
		} else {
			analyzed[i].InSync = false
			analyzed[i].DriftReason = "Issue not found"
		}
	}

	return analyzed
}

// FilterOutOfSyncPairs filters pairs that are out of sync
// Pure function: filter predicate
func FilterOutOfSyncPairs(pairs []PRIssuePair) []PRIssuePair {
	outOfSync := make([]PRIssuePair, 0)

	for _, pair := range pairs {
		if !pair.InSync {
			outOfSync = append(outOfSync, pair)
		}
	}

	return outOfSync
}
