package github

import (
	"fmt"
	"strings"
)

type Repository struct {
	Name          string
	FullName      string
	Owner         string
	Private       bool
	Archived      bool
	DefaultBranch string
}

type repoListItemResponse struct {
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Owner         struct {
		Login string `json:"login"`
	} `json:"owner"`
	Private       bool   `json:"private"`
	Archived      bool   `json:"archived"`
	DefaultBranch string `json:"default_branch"`
}

func (c *Client) ListOrgRepositories(org string) ([]Repository, error) {
	var allRepos []Repository
	page := 1
	perPage := 100

	for {
		var response []repoListItemResponse
		path := fmt.Sprintf("orgs/%s/repos?per_page=%d&page=%d", org, perPage, page)

		if err := c.Get(path, &response); err != nil {
			return nil, fmt.Errorf("failed to list org repos: %w", err)
		}

		if len(response) == 0 {
			break
		}

		for _, repo := range response {
			allRepos = append(allRepos, Repository{
				Name:          repo.Name,
				FullName:      repo.FullName,
				Owner:         repo.Owner.Login,
				Private:       repo.Private,
				Archived:      repo.Archived,
				DefaultBranch: repo.DefaultBranch,
			})
		}

		if len(response) < perPage {
			break
		}
		page++
	}

	return allRepos, nil
}

func (c *Client) ListUserRepositories(username string) ([]Repository, error) {
	var allRepos []Repository
	page := 1
	perPage := 100

	for {
		var response []repoListItemResponse
		path := fmt.Sprintf("users/%s/repos?per_page=%d&page=%d", username, perPage, page)

		if err := c.Get(path, &response); err != nil {
			return nil, fmt.Errorf("failed to list user repos: %w", err)
		}

		if len(response) == 0 {
			break
		}

		for _, repo := range response {
			allRepos = append(allRepos, Repository{
				Name:          repo.Name,
				FullName:      repo.FullName,
				Owner:         repo.Owner.Login,
				Private:       repo.Private,
				Archived:      repo.Archived,
				DefaultBranch: repo.DefaultBranch,
			})
		}

		if len(response) < perPage {
			break
		}
		page++
	}

	return allRepos, nil
}

func (c *Client) ListNamespaceRepositories(namespace string) ([]Repository, bool, error) {
	repos, err := c.ListOrgRepositories(namespace)
	if err == nil {
		return repos, true, nil
	}

	if !strings.Contains(err.Error(), "404") {
		return nil, false, err
	}

	repos, err = c.ListUserRepositories(namespace)
	if err != nil {
		return nil, false, fmt.Errorf("failed to list namespace repos: %w", err)
	}

	return repos, false, nil
}
