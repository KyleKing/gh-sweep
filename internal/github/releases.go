package github

import (
	"fmt"
	"time"
)

// Release represents a GitHub release
type Release struct {
	ID          int
	Repository  string
	TagName     string
	Name        string
	Body        string
	Author      string
	CreatedAt   time.Time
	PublishedAt time.Time
	Draft       bool
	Prerelease  bool
}

type releaseResponse struct {
	ID        int    `json:"id"`
	TagName   string `json:"tag_name"`
	Name      string `json:"name"`
	Body      string `json:"body"`
	Author    struct {
		Login string `json:"login"`
	} `json:"author"`
	CreatedAt   time.Time `json:"created_at"`
	PublishedAt time.Time `json:"published_at"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
}

// ListReleases lists all releases for a repository
func (c *Client) ListReleases(owner, repo string) ([]Release, error) {
	var response []releaseResponse
	path := fmt.Sprintf("repos/%s/%s/releases", owner, repo)

	if err := c.Get(path, &response); err != nil {
		return nil, fmt.Errorf("failed to list releases: %w", err)
	}

	releases := make([]Release, len(response))
	for i, r := range response {
		releases[i] = Release{
			ID:          r.ID,
			Repository:  fmt.Sprintf("%s/%s", owner, repo),
			TagName:     r.TagName,
			Name:        r.Name,
			Body:        r.Body,
			Author:      r.Author.Login,
			CreatedAt:   r.CreatedAt,
			PublishedAt: r.PublishedAt,
			Draft:       r.Draft,
			Prerelease:  r.Prerelease,
		}
	}

	return releases, nil
}

// GetLatestRelease returns the most recent release
func (c *Client) GetLatestRelease(owner, repo string) (*Release, error) {
	var response releaseResponse
	path := fmt.Sprintf("repos/%s/%s/releases/latest", owner, repo)

	if err := c.Get(path, &response); err != nil {
		return nil, fmt.Errorf("failed to get latest release: %w", err)
	}

	return &Release{
		ID:          response.ID,
		Repository:  fmt.Sprintf("%s/%s", owner, repo),
		TagName:     response.TagName,
		Name:        response.Name,
		Body:        response.Body,
		Author:      response.Author.Login,
		CreatedAt:   response.CreatedAt,
		PublishedAt: response.PublishedAt,
		Draft:       response.Draft,
		Prerelease:  response.Prerelease,
	}, nil
}

// ReleaseComparison compares releases across repositories
type ReleaseComparison struct {
	Repositories     []string
	LatestReleases   map[string]*Release
	OutdatedRepos    []string // Repos with no release in 90+ days
	NonSemVerRepos   []string // Repos not following semver
}

// CompareReleases compares releases across multiple repositories
func CompareReleases(releases map[string]*Release) ReleaseComparison {
	comparison := ReleaseComparison{
		LatestReleases: releases,
		Repositories:   make([]string, 0, len(releases)),
	}

	threshold := time.Now().AddDate(0, -3, 0) // 90 days ago

	for repo, release := range releases {
		comparison.Repositories = append(comparison.Repositories, repo)

		if release == nil {
			comparison.OutdatedRepos = append(comparison.OutdatedRepos, repo)
			continue
		}

		if release.PublishedAt.Before(threshold) {
			comparison.OutdatedRepos = append(comparison.OutdatedRepos, repo)
		}

		// Simple semver check (starts with v followed by numbers)
		if len(release.TagName) < 2 || release.TagName[0] != 'v' {
			comparison.NonSemVerRepos = append(comparison.NonSemVerRepos, repo)
		}
	}

	return comparison
}
