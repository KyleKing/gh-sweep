package github

import (
	"fmt"
	"time"
)

type WatchState string

const (
	WatchStateSubscribed WatchState = "subscribed"
	WatchStateIgnored    WatchState = "ignored"
	// WatchStateDefault is GitHub's un-set subscription state ("Participating
	// and @mentions"), not an absence of any relationship to the repo.
	WatchStateDefault WatchState = ""
)

type Subscription struct {
	Repository string
	Subscribed bool
	Ignored    bool
	Reason     string
	CreatedAt  time.Time
	State      WatchState
}

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

func (c *Client) GetAuthenticatedUser() (string, error) {
	var response userResponse
	if err := c.Get("user", &response); err != nil {
		return "", fmt.Errorf("failed to get authenticated user: %w", err)
	}

	return response.Login, nil
}

func (c *Client) SetRepoSubscription(owner, repo string, subscribed, ignored bool) (*Subscription, error) {
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

func (c *Client) DeleteRepoSubscription(owner, repo string) error {
	path := fmt.Sprintf("repos/%s/%s/subscription", owner, repo)
	if err := c.Delete(path, nil); err != nil {
		return fmt.Errorf("failed to delete subscription: %w", err)
	}

	return nil
}
