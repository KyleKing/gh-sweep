package github

import (
	"fmt"
	"time"
)

// Comment represents a PR review comment
type Comment struct {
	ID          int
	Repository  string
	PRNumber    int
	Author      string
	Body        string
	Path        string
	Line        int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	InReplyToID *int
	Resolved    bool
}

type commentResponse struct {
	ID        int    `json:"id"`
	Body      string `json:"body"`
	Path      string `json:"path"`
	Line      int    `json:"line"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	InReplyToID *int      `json:"in_reply_to_id"`
}

// ListPRComments lists all comments for a pull request
func (c *Client) ListPRComments(owner, repo string, prNumber int) ([]Comment, error) {
	var response []commentResponse
	path := fmt.Sprintf("repos/%s/%s/pulls/%d/comments", owner, repo, prNumber)

	if err := c.Get(path, &response); err != nil {
		return nil, fmt.Errorf("failed to list PR comments: %w", err)
	}

	comments := make([]Comment, len(response))
	for i, cr := range response {
		comments[i] = Comment{
			ID:          cr.ID,
			Repository:  fmt.Sprintf("%s/%s", owner, repo),
			PRNumber:    prNumber,
			Author:      cr.User.Login,
			Body:        cr.Body,
			Path:        cr.Path,
			Line:        cr.Line,
			CreatedAt:   cr.CreatedAt,
			UpdatedAt:   cr.UpdatedAt,
			InReplyToID: cr.InReplyToID,
			Resolved:    false, // TODO: Implement resolution detection
		}
	}

	return comments, nil
}

// FilterUnresolvedComments filters comments to only unresolved ones
func FilterUnresolvedComments(comments []Comment) []Comment {
	// Simple heuristic: a comment is unresolved if it's not a reply
	// and doesn't have recent replies
	unresolved := []Comment{}

	// Group by ID for reply detection
	repliesTo := make(map[int]bool)
	for _, c := range comments {
		if c.InReplyToID != nil {
			repliesTo[*c.InReplyToID] = true
		}
	}

	for _, c := range comments {
		// Skip if it's a reply
		if c.InReplyToID != nil {
			continue
		}

		// Consider unresolved if no replies
		if !repliesTo[c.ID] {
			unresolved = append(unresolved, c)
		}
	}

	return unresolved
}
