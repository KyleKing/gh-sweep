package github

import (
	"fmt"
	"time"
)

// WatchState is a repository's notification subscription state for the authenticated user.
type WatchState string

// Subscription states as reported by the GitHub API.
const (
	WatchStateSubscribed WatchState = "subscribed"
	WatchStateIgnored    WatchState = "ignored"
	// WatchStateDefault is GitHub's un-set subscription state ("Participating
	// and @mentions"), not an absence of any relationship to the repo.
	WatchStateDefault WatchState = ""
)

// Subscription describes the authenticated user's notification subscription to a repository.
type Subscription struct {
	Repository string
	Subscribed bool
	Ignored    bool
	Reason     string
	CreatedAt  time.Time
	State      WatchState
}

// RepoBasic holds the minimal repository identity fields used by the watch/subscription APIs.
type RepoBasic struct {
	Name     string
	FullName string
	Owner    string
	Private  bool
}

type userResponse struct {
	Login string `json:"login"`
}

type subscriptionResponse struct {
	Subscribed bool      `json:"subscribed"`
	Ignored    bool      `json:"ignored"`
	Reason     string    `json:"reason"`
	CreatedAt  time.Time `json:"created_at"`
}

// GetAuthenticatedUser returns the login of the user the client is authenticated as.
func (c *Client) GetAuthenticatedUser() (string, error) {
	var response userResponse
	if err := c.Get("user", &response); err != nil {
		return "", fmt.Errorf("failed to get authenticated user: %w", err)
	}

	return response.Login, nil
}

// SetRepoSubscription sets the authenticated user's watch/ignore subscription for a repository.
func (c *Client) SetRepoSubscription(
	owner, repo string,
	subscribed, ignored bool,
) (*Subscription, error) {
	path := fmt.Sprintf("repos/%s/%s/subscription", owner, repo)
	body := map[string]bool{
		"subscribed": subscribed,
		"ignored":    ignored,
	}

	var response subscriptionResponse
	if err := c.Put(path, body, &response); err != nil {
		return nil, fmt.Errorf("failed to set subscription: %w", err)
	}

	state := WatchStateSubscribed
	if response.Ignored {
		state = WatchStateIgnored
	} else if !response.Subscribed {
		state = WatchStateDefault
	}

	return &Subscription{
		Repository: fmt.Sprintf("%s/%s", owner, repo),
		Subscribed: response.Subscribed,
		Ignored:    response.Ignored,
		Reason:     response.Reason,
		CreatedAt:  response.CreatedAt,
		State:      state,
	}, nil
}

// DeleteRepoSubscription removes the authenticated user's subscription to a repository,
// resetting it to the default (un-set) state.
func (c *Client) DeleteRepoSubscription(owner, repo string) error {
	path := fmt.Sprintf("repos/%s/%s/subscription", owner, repo)
	if err := c.Delete(path, nil); err != nil {
		return fmt.Errorf("failed to delete subscription: %w", err)
	}

	return nil
}
