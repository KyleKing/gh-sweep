package github

import (
	"fmt"
	"time"
)

// Branch represents a GitHub branch
type Branch struct {
	Name           string
	SHA            string
	Protected      bool
	Ahead          int
	Behind         int
	LastCommitDate time.Time
}

// BranchListResponse is the response from the GitHub API
type branchListResponse struct {
	Name      string `json:"name"`
	Commit    struct {
		SHA    string `json:"sha"`
		Commit struct {
			Author struct {
				Date time.Time `json:"date"`
			} `json:"author"`
		} `json:"commit"`
	} `json:"commit"`
	Protected bool `json:"protected"`
}

// ListBranches lists all branches for a repository
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

// CompareBranches compares two branches and returns ahead/behind counts
func (c *Client) CompareBranches(owner, repo, base, head string) (ahead, behind int, err error) {
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

// DeleteBranch deletes a branch
func (c *Client) DeleteBranch(owner, repo, branch string) error {
	path := fmt.Sprintf("repos/%s/%s/git/refs/heads/%s", owner, repo, branch)

	if err := c.Delete(path, nil); err != nil {
		return fmt.Errorf("failed to delete branch: %w", err)
	}

	return nil
}

// CreatePullRequest creates a new pull request
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

// BranchWithComparison extends Branch with comparison data
type BranchWithComparison struct {
	Branch
	ComparedTo string
}

// GetBranchesWithComparison fetches branches and compares them to a base branch
func (c *Client) GetBranchesWithComparison(owner, repo, baseBranch string) ([]BranchWithComparison, error) {
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

// GetDefaultBranch fetches the default branch for a repository
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
