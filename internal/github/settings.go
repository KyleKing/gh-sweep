package github

import "fmt"

// RepoSettings represents repository settings
type RepoSettings struct {
	Repository          string
	DefaultBranch       string
	AllowMergeCommit    bool
	AllowSquashMerge    bool
	AllowRebaseMerge    bool
	DeleteBranchOnMerge bool
	HasIssues           bool
	HasProjects         bool
	HasWiki             bool
}

type repoResponse struct {
	Name                string `json:"name"`
	DefaultBranch       string `json:"default_branch"`
	AllowMergeCommit    bool   `json:"allow_merge_commit"`
	AllowSquashMerge    bool   `json:"allow_squash_merge"`
	AllowRebaseMerge    bool   `json:"allow_rebase_merge"`
	DeleteBranchOnMerge bool   `json:"delete_branch_on_merge"`
	HasIssues           bool   `json:"has_issues"`
	HasProjects         bool   `json:"has_projects"`
	HasWiki             bool   `json:"has_wiki"`
}

// GetRepoSettings retrieves repository settings
func (c *Client) GetRepoSettings(owner, repo string) (*RepoSettings, error) {
	var response repoResponse
	path := fmt.Sprintf("repos/%s/%s", owner, repo)

	if err := c.Get(path, &response); err != nil {
		return nil, fmt.Errorf("failed to get repo settings: %w", err)
	}

	return &RepoSettings{
		Repository:          fmt.Sprintf("%s/%s", owner, repo),
		DefaultBranch:       response.DefaultBranch,
		AllowMergeCommit:    response.AllowMergeCommit,
		AllowSquashMerge:    response.AllowSquashMerge,
		AllowRebaseMerge:    response.AllowRebaseMerge,
		DeleteBranchOnMerge: response.DeleteBranchOnMerge,
		HasIssues:           response.HasIssues,
		HasProjects:         response.HasProjects,
		HasWiki:             response.HasWiki,
	}, nil
}

// SettingsDiff represents differences between repository settings
type SettingsDiff struct {
	Field    string
	Baseline interface{}
	Current  interface{}
	Severity string // critical, warning, info
}

// CompareSettings compares repository settings against a baseline
func CompareSettings(baseline, current *RepoSettings) []SettingsDiff {
	diffs := []SettingsDiff{}

	if baseline.DefaultBranch != current.DefaultBranch {
		diffs = append(diffs, SettingsDiff{
			Field:    "DefaultBranch",
			Baseline: baseline.DefaultBranch,
			Current:  current.DefaultBranch,
			Severity: "warning",
		})
	}

	if baseline.DeleteBranchOnMerge != current.DeleteBranchOnMerge {
		diffs = append(diffs, SettingsDiff{
			Field:    "DeleteBranchOnMerge",
			Baseline: baseline.DeleteBranchOnMerge,
			Current:  current.DeleteBranchOnMerge,
			Severity: "info",
		})
	}

	if baseline.AllowMergeCommit != current.AllowMergeCommit ||
		baseline.AllowSquashMerge != current.AllowSquashMerge ||
		baseline.AllowRebaseMerge != current.AllowRebaseMerge {
		diffs = append(diffs, SettingsDiff{
			Field:    "MergeStrategies",
			Baseline: fmt.Sprintf("merge:%v squash:%v rebase:%v", baseline.AllowMergeCommit, baseline.AllowSquashMerge, baseline.AllowRebaseMerge),
			Current:  fmt.Sprintf("merge:%v squash:%v rebase:%v", current.AllowMergeCommit, current.AllowSquashMerge, current.AllowRebaseMerge),
			Severity: "info",
		})
	}

	return diffs
}
