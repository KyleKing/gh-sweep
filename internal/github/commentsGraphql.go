package github

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cli/go-gh"
)

// DefaultOpenPRCap bounds how many of the newest open PRs are scanned for review threads.
const DefaultOpenPRCap = 20

const threadFetchConcurrency = 4

const reviewThreadsQuery = `query($owner: String!, $name: String!, $number: Int!, $cursor: String) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      title
      reviewThreads(first: 100, after: $cursor) {
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
          isResolved
          isOutdated
          path
          comments(first: 50) {
            nodes {
              author {
                login
              }
              body
              createdAt
              url
            }
          }
        }
      }
    }
  }
}`

type gqlDoer interface {
	Do(query string, variables map[string]interface{}, response interface{}) error
}

// GQLClient wraps the GitHub GraphQL API for review thread queries.
type GQLClient struct {
	doer gqlDoer
}

// NewGQLClient creates a GraphQL client using gh CLI auth or GITHUB_TOKEN.
func NewGQLClient() (*GQLClient, error) {
	client, err := gh.GQLClient(clientOptions())
	if err != nil {
		return nil, fmt.Errorf("failed to create GraphQL client: %w", err)
	}

	return &GQLClient{doer: client}, nil
}

type reviewThreadsResponse struct {
	Repository struct {
		PullRequest struct {
			Title         string `json:"title"`
			ReviewThreads struct {
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
				Nodes []reviewThreadNode `json:"nodes"`
			} `json:"reviewThreads"`
		} `json:"pullRequest"`
	} `json:"repository"`
}

type reviewThreadNode struct {
	IsResolved bool   `json:"isResolved"`
	IsOutdated bool   `json:"isOutdated"`
	Path       string `json:"path"`
	Comments   struct {
		Nodes []reviewCommentNode `json:"nodes"`
	} `json:"comments"`
}

type reviewCommentNode struct {
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
	URL       string    `json:"url"`
}

func mapReviewThreads(
	owner, repo string,
	prNumber int,
	prTitle string,
	nodes []reviewThreadNode,
) []ReviewThread {
	threads := make([]ReviewThread, len(nodes))
	for i, node := range nodes {
		comments := make([]ReviewComment, len(node.Comments.Nodes))
		for j, c := range node.Comments.Nodes {
			comments[j] = ReviewComment{
				Author:    c.Author.Login,
				Body:      c.Body,
				CreatedAt: c.CreatedAt,
				URL:       c.URL,
			}
		}
		threads[i] = ReviewThread{
			Repository: fmt.Sprintf("%s/%s", owner, repo),
			PRNumber:   prNumber,
			PRTitle:    prTitle,
			Path:       node.Path,
			IsResolved: node.IsResolved,
			IsOutdated: node.IsOutdated,
			Comments:   comments,
		}
	}

	return threads
}

// ListPRReviewThreads fetches all review threads for a single pull request.
func (g *GQLClient) ListPRReviewThreads(owner, repo string, prNumber int) ([]ReviewThread, error) {
	var threads []ReviewThread
	var cursor interface{}

	for {
		variables := map[string]interface{}{
			"owner":  owner,
			"name":   repo,
			"number": prNumber,
			"cursor": cursor,
		}

		var response reviewThreadsResponse
		if err := g.doer.Do(reviewThreadsQuery, variables, &response); err != nil {
			return nil, fmt.Errorf("failed to list review threads for PR #%d: %w", prNumber, err)
		}

		pr := response.Repository.PullRequest
		threads = append(
			threads,
			mapReviewThreads(owner, repo, prNumber, pr.Title, pr.ReviewThreads.Nodes)...)

		if !pr.ReviewThreads.PageInfo.HasNextPage {
			break
		}
		cursor = pr.ReviewThreads.PageInfo.EndCursor
	}

	return threads, nil
}

// ListOpenPRReviewThreads fetches review threads across the newest open PRs, capped at maxPRs.
func (g *GQLClient) ListOpenPRReviewThreads(
	client *Client,
	owner, repo string,
	maxPRs int,
) ([]ReviewThread, error) {
	prs, err := client.ListPullRequests(owner, repo, "open")
	if err != nil {
		return nil, fmt.Errorf("failed to list open pull requests: %w", err)
	}

	if maxPRs > 0 && len(prs) > maxPRs {
		prs = prs[:maxPRs]
	}

	results := make([][]ReviewThread, len(prs))
	errs := make([]error, len(prs))
	sem := make(chan struct{}, threadFetchConcurrency)

	var wg sync.WaitGroup
	for i, pr := range prs {
		wg.Add(1)
		go func(idx, number int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[idx], errs[idx] = g.ListPRReviewThreads(owner, repo, number)
		}(i, pr.Number)
	}
	wg.Wait()

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

	var threads []ReviewThread
	for _, r := range results {
		threads = append(threads, r...)
	}

	return threads, nil
}

// ListRepoReviewThreads fetches threads for one PR when prNumber > 0, otherwise across open PRs.
func (g *GQLClient) ListRepoReviewThreads(
	client *Client,
	owner, repo string,
	prNumber, maxPRs int,
) ([]ReviewThread, error) {
	if prNumber > 0 {
		return g.ListPRReviewThreads(owner, repo, prNumber)
	}

	return g.ListOpenPRReviewThreads(client, owner, repo, maxPRs)
}
