package github

import (
	"fmt"
	"strings"
)

const repositoriesPerPage = 100

// Repository describes a GitHub repository.
type Repository struct {
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Owner         string `json:"owner"`
	Private       bool   `json:"private"`
	Archived      bool   `json:"archived"`
	DefaultBranch string `json:"default_branch"`
}

type repoListItemResponse struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Owner    struct {
		Login string `json:"login"`
	} `json:"owner"`
	Private       bool   `json:"private"`
	Archived      bool   `json:"archived"`
	DefaultBranch string `json:"default_branch"`
}

func repositoryFromResponse(repo repoListItemResponse) Repository {
	return Repository{
		Name:          repo.Name,
		FullName:      repo.FullName,
		Owner:         repo.Owner.Login,
		Private:       repo.Private,
		Archived:      repo.Archived,
		DefaultBranch: repo.DefaultBranch,
	}
}

// ListOrgRepositories lists all repositories belonging to an organization.
func (c *Client) ListOrgRepositories(org string) ([]Repository, error) {
	return fetchPages(c, repositoriesPerPage, func(page int) string {
		return fmt.Sprintf("orgs/%s/repos?per_page=%d&page=%d", org, repositoriesPerPage, page)
	}, repositoryFromResponse)
}

// ListUserRepositories lists all repositories belonging to a user.
func (c *Client) ListUserRepositories(username string) ([]Repository, error) {
	return fetchPages(c, repositoriesPerPage, func(page int) string {
		return fmt.Sprintf("users/%s/repos?per_page=%d&page=%d", username, repositoriesPerPage, page)
	}, repositoryFromResponse)
}

// ListNamespaceRepositories lists repositories for a namespace that may be either an
// organization or a user, reporting which kind it resolved to.
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
