package github

import (
	"fmt"
	"time"
)

const pullRequestsPerPage = 100

// PRRef identifies one side (head or base) of a pull request.
type PRRef struct {
	Ref  string
	SHA  string
	Repo string
}

// PullRequest describes a GitHub pull request.
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

func pullRequestFromResponse(pr prResponse) PullRequest {
	headRepo := ""
	if pr.Head.Repo.FullName != "" {
		headRepo = pr.Head.Repo.FullName
	}
	baseRepo := ""
	if pr.Base.Repo.FullName != "" {
		baseRepo = pr.Base.Repo.FullName
	}

	return PullRequest{
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
	}
}

// ListPullRequests lists pull requests in a repository matching state ("open", "closed", or "all").
func (c *Client) ListPullRequests(owner, repo, state string) ([]PullRequest, error) {
	return fetchPages(c, pullRequestsPerPage, func(page int) string {
		return fmt.Sprintf(
			"repos/%s/%s/pulls?state=%s&per_page=%d&page=%d",
			owner, repo, state, pullRequestsPerPage, page,
		)
	}, pullRequestFromResponse)
}

// GetPullRequestsForBranch lists all pull requests (any state) whose head is owner/branch.
func (c *Client) GetPullRequestsForBranch(owner, repo, branch string) ([]PullRequest, error) {
	return fetchPages(c, pullRequestsPerPage, func(page int) string {
		return fmt.Sprintf(
			"repos/%s/%s/pulls?state=all&head=%s:%s&per_page=%d&page=%d",
			owner, repo, owner, branch, pullRequestsPerPage, page,
		)
	}, pullRequestFromResponse)
}
