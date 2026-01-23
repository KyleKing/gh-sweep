package github

import (
	"fmt"
	"time"
)

type PRRef struct {
	Ref  string
	SHA  string
	Repo string
}

type PullRequest struct {
	Number   int
	Title    string
	State    string
	Head     PRRef
	Base     PRRef
	MergedAt *time.Time
	ClosedAt *time.Time
}

type prResponse struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Head   struct {
		Ref  string `json:"ref"`
		SHA  string `json:"sha"`
		Repo struct {
			FullName string `json:"full_name"`
		} `json:"repo"`
	} `json:"head"`
	Base struct {
		Ref  string `json:"ref"`
		SHA  string `json:"sha"`
		Repo struct {
			FullName string `json:"full_name"`
		} `json:"repo"`
	} `json:"base"`
	MergedAt *time.Time `json:"merged_at"`
	ClosedAt *time.Time `json:"closed_at"`
}

func (c *Client) ListPullRequests(owner, repo, state string) ([]PullRequest, error) {
	var allPRs []PullRequest
	page := 1
	perPage := 100

	for {
		var response []prResponse
		path := fmt.Sprintf("repos/%s/%s/pulls?state=%s&per_page=%d&page=%d", owner, repo, state, perPage, page)

		if err := c.Get(path, &response); err != nil {
			return nil, fmt.Errorf("failed to list pull requests: %w", err)
		}

		if len(response) == 0 {
			break
		}

		for _, pr := range response {
			headRepo := ""
			if pr.Head.Repo.FullName != "" {
				headRepo = pr.Head.Repo.FullName
			}
			baseRepo := ""
			if pr.Base.Repo.FullName != "" {
				baseRepo = pr.Base.Repo.FullName
			}

			allPRs = append(allPRs, PullRequest{
				Number: pr.Number,
				Title:  pr.Title,
				State:  pr.State,
				Head: PRRef{
					Ref:  pr.Head.Ref,
					SHA:  pr.Head.SHA,
					Repo: headRepo,
				},
				Base: PRRef{
					Ref:  pr.Base.Ref,
					SHA:  pr.Base.SHA,
					Repo: baseRepo,
				},
				MergedAt: pr.MergedAt,
				ClosedAt: pr.ClosedAt,
			})
		}

		if len(response) < perPage {
			break
		}
		page++
	}

	return allPRs, nil
}

func (c *Client) GetPullRequestsForBranch(owner, repo, branch string) ([]PullRequest, error) {
	var allPRs []PullRequest
	page := 1
	perPage := 100

	for {
		var response []prResponse
		path := fmt.Sprintf("repos/%s/%s/pulls?state=all&head=%s:%s&per_page=%d&page=%d", owner, repo, owner, branch, perPage, page)

		if err := c.Get(path, &response); err != nil {
			return nil, fmt.Errorf("failed to get pull requests for branch: %w", err)
		}

		if len(response) == 0 {
			break
		}

		for _, pr := range response {
			headRepo := ""
			if pr.Head.Repo.FullName != "" {
				headRepo = pr.Head.Repo.FullName
			}
			baseRepo := ""
			if pr.Base.Repo.FullName != "" {
				baseRepo = pr.Base.Repo.FullName
			}

			allPRs = append(allPRs, PullRequest{
				Number: pr.Number,
				Title:  pr.Title,
				State:  pr.State,
				Head: PRRef{
					Ref:  pr.Head.Ref,
					SHA:  pr.Head.SHA,
					Repo: headRepo,
				},
				Base: PRRef{
					Ref:  pr.Base.Ref,
					SHA:  pr.Base.SHA,
					Repo: baseRepo,
				},
				MergedAt: pr.MergedAt,
				ClosedAt: pr.ClosedAt,
			})
		}

		if len(response) < perPage {
			break
		}
		page++
	}

	return allPRs, nil
}
