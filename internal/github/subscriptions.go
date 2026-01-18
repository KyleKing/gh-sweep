package github

import (
	"fmt"
	"strings"
	"time"
)

type WatchState string

const (
	WatchStateSubscribed  WatchState = "subscribed"
	WatchStateIgnored     WatchState = "ignored"
	WatchStateNotWatching WatchState = ""
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

type repoListResponse struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Owner    struct {
		Login string `json:"login"`
	} `json:"owner"`
	Private bool `json:"private"`
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

func (c *Client) ListUserRepos() ([]RepoBasic, error) {
	var allRepos []RepoBasic
	page := 1
	perPage := 100

	for {
		var response []repoListResponse
		path := fmt.Sprintf("user/repos?affiliation=owner&per_page=%d&page=%d", perPage, page)

		if err := c.Get(path, &response); err != nil {
			return nil, fmt.Errorf("failed to list user repos: %w", err)
		}

		if len(response) == 0 {
			break
		}

		for _, repo := range response {
			allRepos = append(allRepos, RepoBasic{
				Name:     repo.Name,
				FullName: repo.FullName,
				Owner:    repo.Owner.Login,
				Private:  repo.Private,
			})
		}

		if len(response) < perPage {
			break
		}
		page++
	}

	return allRepos, nil
}

func (c *Client) GetRepoSubscription(owner, repo string) (*Subscription, error) {
	var response subscriptionResponse
	path := fmt.Sprintf("repos/%s/%s/subscription", owner, repo)

	if err := c.Get(path, &response); err != nil {
		if strings.Contains(err.Error(), "404") {
			return &Subscription{
				Repository: fmt.Sprintf("%s/%s", owner, repo),
				Subscribed: false,
				Ignored:    false,
				State:      WatchStateNotWatching,
			}, nil
		}
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	state := WatchStateSubscribed
	if response.Ignored {
		state = WatchStateIgnored
	} else if !response.Subscribed {
		state = WatchStateNotWatching
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
		state = WatchStateNotWatching
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
