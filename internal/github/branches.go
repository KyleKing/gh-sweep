package github

import (
	"fmt"
	"sync"
	"time"
)

// Branch represents a GitHub branch.
type Branch struct {
	Name           string
	SHA            string
	Protected      bool
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

// ListBranchesWithDates lists branches with LastCommitDate populated. The
// branches endpoint omits commit dates, so this costs one extra request per
// branch; callers that classify branches by age need it, and a caller that
// only needs names should use ListBranches.
func (c *Client) ListBranchesWithDates(owner, repo string) ([]Branch, error) {
	branches, err := c.ListBranches(owner, repo)
	if err != nil {
		return nil, err
	}

	c.fillCommitDates(owner, repo, branches)

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
