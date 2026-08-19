package github

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// Sentinel errors returned by BranchStatus.DeleteBlocked.
var (
	ErrDefaultBranchDeletion   = errors.New("cannot delete the default branch")
	ErrOpenPRBranchDeletion    = errors.New("branch has an open pull request")
	ErrProtectedBranchDeletion = errors.New("cannot delete a protected branch")
)

// Branch represents a GitHub branch.
type Branch struct {
	Name           string
	SHA            string
	Protected      bool
	Ahead          int
	Behind         int
	LastCommitDate time.Time
}

// BranchListResponse is the response from the GitHub API.
type branchListResponse struct {
	Name   string `json:"name"`
	Commit struct {
		SHA    string `json:"sha"`
		Commit struct {
			Author struct {
				Date time.Time `json:"date"`
			} `json:"author"`
		} `json:"commit"`
	} `json:"commit"`
	Protected bool `json:"protected"`
}

// ListBranches lists all branches for a repository.
func (c *Client) ListBranches(owner, repo string) ([]Branch, error) {
	var response []branchListResponse
	path := fmt.Sprintf("repos/%s/%s/branches", owner, repo)

	if err := c.Get(path, &response); err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	branches := make([]Branch, len(response))
	for i, br := range response {
		branches[i] = Branch{
			Name:           br.Name,
			SHA:            br.Commit.SHA,
			Protected:      br.Protected,
			LastCommitDate: br.Commit.Commit.Author.Date,
		}
	}

	return branches, nil
}

const commitDateWorkers = 4

func (c *Client) fillCommitDates(owner, repo string, branches []Branch) {
	var wg sync.WaitGroup

	sem := make(chan struct{}, commitDateWorkers)

	for i := range branches {
		if !branches[i].LastCommitDate.IsZero() {
			continue
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(b *Branch) {
			defer wg.Done()
			defer func() { <-sem }()

			var response struct {
				Commit struct {
					Author struct {
						Date time.Time `json:"date"`
					} `json:"author"`
				} `json:"commit"`
			}

			path := fmt.Sprintf("repos/%s/%s/commits/%s", owner, repo, b.SHA)
			if err := c.Get(path, &response); err == nil {
				b.LastCommitDate = response.Commit.Author.Date
			}
		}(&branches[i])
	}

	wg.Wait()
}

// CompareBranches compares two branches and returns ahead/behind counts.
func (c *Client) CompareBranches(owner, repo, base, head string) (int, int, error) {
	var response struct {
		AheadBy  int `json:"ahead_by"`
		BehindBy int `json:"behind_by"`
	}

	path := fmt.Sprintf("repos/%s/%s/compare/%s...%s", owner, repo, base, head)

	if err := c.Get(path, &response); err != nil {
		return 0, 0, fmt.Errorf("failed to compare branches: %w", err)
	}

	return response.AheadBy, response.BehindBy, nil
}

// DeleteBranch deletes a branch.
func (c *Client) DeleteBranch(owner, repo, branch string) error {
	path := fmt.Sprintf("repos/%s/%s/git/refs/heads/%s", owner, repo, branch)

	if err := c.Delete(path, nil); err != nil {
		return fmt.Errorf("failed to delete branch: %w", err)
	}

	return nil
}

// CreatePullRequest creates a new pull request.
func (c *Client) CreatePullRequest(owner, repo, title, body, head, base string) (int, error) {
	requestBody := map[string]string{
		"title": title,
		"body":  body,
		"head":  head,
		"base":  base,
	}

	var response struct {
		Number int `json:"number"`
	}

	path := fmt.Sprintf("repos/%s/%s/pulls", owner, repo)

	if err := c.Post(path, requestBody, &response); err != nil {
		return 0, fmt.Errorf("failed to create pull request: %w", err)
	}

	return response.Number, nil
}

// BranchWithComparison extends Branch with comparison data.
type BranchWithComparison struct {
	Branch
	ComparedTo string
}

// GetBranchesWithComparison fetches branches and compares them to a base branch.
func (c *Client) GetBranchesWithComparison(
	owner, repo, baseBranch string,
) ([]BranchWithComparison, error) {
	branches, err := c.ListBranches(owner, repo)
	if err != nil {
		return nil, err
	}

	result := make([]BranchWithComparison, 0, len(branches))

	for _, branch := range branches {
		if branch.Name == baseBranch {
			result = append(result, BranchWithComparison{
				Branch:     branch,
				ComparedTo: baseBranch,
			})

			continue
		}

		// Compare to base branch
		ahead, behind, err := c.CompareBranches(owner, repo, baseBranch, branch.Name)
		if err != nil {
			// Log error but continue
			result = append(result, BranchWithComparison{
				Branch:     branch,
				ComparedTo: baseBranch,
			})

			continue
		}

		branch.Ahead = ahead
		branch.Behind = behind

		result = append(result, BranchWithComparison{
			Branch:     branch,
			ComparedTo: baseBranch,
		})
	}

	return result, nil
}

// BranchStatus extends Branch with default-branch and pull request context.
type BranchStatus struct {
	Branch
	ComparedTo string
	IsDefault  bool
	PR         *PullRequest
}

// DeleteBlocked reports why the branch must not be deleted, or nil when deletion is safe.
func (b BranchStatus) DeleteBlocked() error {
	switch {
	case b.IsDefault:
		return ErrDefaultBranchDeletion
	case b.Protected:
		return ErrProtectedBranchDeletion
	case b.PR != nil && b.PR.State == "open":
		return ErrOpenPRBranchDeletion
	default:
		return nil
	}
}

// MatchBranchPR returns the open PR whose head is the branch, or the most recent closed one.
func MatchBranchPR(prs []PullRequest, repoFullName, branch string) *PullRequest {
	var match *PullRequest

	for i := range prs {
		pr := &prs[i]
		if pr.Head.Ref != branch {
			continue
		}
		if pr.Head.Repo != "" && pr.Head.Repo != repoFullName {
			continue
		}
		if pr.State == "open" {
			return pr
		}
		if match == nil || pr.Number > match.Number {
			match = pr
		}
	}

	return match
}

// ListBranchStatuses lists branches enriched with default-branch, comparison, and PR data.
// An empty baseBranch compares against the repository default branch.
func (c *Client) ListBranchStatuses(owner, repo, baseBranch string) ([]BranchStatus, error) {
	defaultBranch, err := c.GetDefaultBranch(owner, repo)
	if err != nil {
		return nil, err
	}

	if baseBranch == "" {
		baseBranch = defaultBranch
	}

	branches, err := c.ListBranches(owner, repo)
	if err != nil {
		return nil, err
	}

	c.fillCommitDates(owner, repo, branches)

	prs, err := c.ListPullRequests(owner, repo, "all")
	if err != nil {
		return nil, err
	}

	repoFullName := owner + "/" + repo
	statuses := make([]BranchStatus, 0, len(branches))

	for _, branch := range branches {
		if branch.Name != baseBranch {
			ahead, behind, compareErr := c.CompareBranches(owner, repo, baseBranch, branch.Name)
			if compareErr == nil {
				branch.Ahead = ahead
				branch.Behind = behind
			}
		}

		statuses = append(statuses, BranchStatus{
			Branch:     branch,
			ComparedTo: baseBranch,
			IsDefault:  branch.Name == defaultBranch,
			PR:         MatchBranchPR(prs, repoFullName, branch.Name),
		})
	}

	return statuses, nil
}

// GetDefaultBranch fetches the default branch for a repository.
func (c *Client) GetDefaultBranch(owner, repo string) (string, error) {
	var response struct {
		DefaultBranch string `json:"default_branch"`
	}

	path := fmt.Sprintf("repos/%s/%s", owner, repo)

	if err := c.Get(path, &response); err != nil {
		return "", fmt.Errorf("failed to get default branch: %w", err)
	}

	return response.DefaultBranch, nil
}
