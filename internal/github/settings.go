package github

import "fmt"

// RepoSettings represents repository settings.
type RepoSettings struct {
	Repository          string
	DefaultBranch       string
	AllowMergeCommit    bool
	AllowSquashMerge    bool
	AllowRebaseMerge    bool
	AllowAutoMerge      bool
	AllowUpdateBranch   bool
	DeleteBranchOnMerge bool
	UseSquashPRTitle    bool
	SquashMergeMessage  string
	SquashMergeTitle    string
	MergeCommitMessage  string
	MergeCommitTitle    string
	HasIssues           bool
	HasProjects         bool
	HasWiki             bool
	HasDiscussions      bool
	IsTemplate          bool
	AllowForking        bool
	WebCommitSignoff    bool
	SecurityAndAnalysis SecurityAndAnalysis
}

// SecurityAndAnalysis represents the repo's security_and_analysis feature toggles.
// Each field is "enabled" or "disabled"; an absent feature (e.g. secret scanning
// on a private repo without GHAS) surfaces as an empty string, not a diff target.
type SecurityAndAnalysis struct {
	SecretScanning               string
	SecretScanningPushProtection string
	DependabotSecurityUpdates    string
	SecretScanningNonProvider    string
	SecretScanningValidityChecks string
}

type secretScanningStatus struct {
	Status string `json:"status"`
}

type securityAndAnalysisResponse struct {
	SecretScanning               *secretScanningStatus `json:"secret_scanning"`
	SecretScanningPushProtection *secretScanningStatus `json:"secret_scanning_push_protection"`
	DependabotSecurityUpdates    *secretScanningStatus `json:"dependabot_security_updates"`
	SecretScanningNonProvider    *secretScanningStatus `json:"secret_scanning_non_provider_patterns"`
	SecretScanningValidityChecks *secretScanningStatus `json:"secret_scanning_validity_checks"`
}

type repoResponse struct {
	Name                string                       `json:"name"`
	DefaultBranch       string                       `json:"default_branch"`
	AllowMergeCommit    bool                         `json:"allow_merge_commit"`
	AllowSquashMerge    bool                         `json:"allow_squash_merge"`
	AllowRebaseMerge    bool                         `json:"allow_rebase_merge"`
	AllowAutoMerge      bool                         `json:"allow_auto_merge"`
	AllowUpdateBranch   bool                         `json:"allow_update_branch"`
	DeleteBranchOnMerge bool                         `json:"delete_branch_on_merge"`
	UseSquashPRTitle    bool                         `json:"use_squash_pr_title_as_default"`
	SquashMergeMessage  string                       `json:"squash_merge_commit_message"`
	SquashMergeTitle    string                       `json:"squash_merge_commit_title"`
	MergeCommitMessage  string                       `json:"merge_commit_message"`
	MergeCommitTitle    string                       `json:"merge_commit_title"`
	HasIssues           bool                         `json:"has_issues"`
	HasProjects         bool                         `json:"has_projects"`
	HasWiki             bool                         `json:"has_wiki"`
	HasDiscussions      bool                         `json:"has_discussions"`
	IsTemplate          bool                         `json:"is_template"`
	AllowForking        bool                         `json:"allow_forking"`
	WebCommitSignoff    bool                         `json:"web_commit_signoff_required"`
	SecurityAndAnalysis *securityAndAnalysisResponse `json:"security_and_analysis"`
}

// GetRepoSettings retrieves repository settings.
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
		AllowAutoMerge:      response.AllowAutoMerge,
		AllowUpdateBranch:   response.AllowUpdateBranch,
		DeleteBranchOnMerge: response.DeleteBranchOnMerge,
		UseSquashPRTitle:    response.UseSquashPRTitle,
		SquashMergeMessage:  response.SquashMergeMessage,
		SquashMergeTitle:    response.SquashMergeTitle,
		MergeCommitMessage:  response.MergeCommitMessage,
		MergeCommitTitle:    response.MergeCommitTitle,
		HasIssues:           response.HasIssues,
		HasProjects:         response.HasProjects,
		HasWiki:             response.HasWiki,
		HasDiscussions:      response.HasDiscussions,
		IsTemplate:          response.IsTemplate,
		AllowForking:        response.AllowForking,
		WebCommitSignoff:    response.WebCommitSignoff,
		SecurityAndAnalysis: toSecurityAndAnalysis(response.SecurityAndAnalysis),
	}, nil
}

func toSecurityAndAnalysis(r *securityAndAnalysisResponse) SecurityAndAnalysis {
	if r == nil {
		return SecurityAndAnalysis{}
	}

	return SecurityAndAnalysis{
		SecretScanning:               statusOf(r.SecretScanning),
		SecretScanningPushProtection: statusOf(r.SecretScanningPushProtection),
		DependabotSecurityUpdates:    statusOf(r.DependabotSecurityUpdates),
		SecretScanningNonProvider:    statusOf(r.SecretScanningNonProvider),
		SecretScanningValidityChecks: statusOf(r.SecretScanningValidityChecks),
	}
}

func statusOf(s *secretScanningStatus) string {
	if s == nil {
		return ""
	}

	return s.Status
}

// RepoSettingsPatch carries only the fields to change; nil pointers are left alone.
// Mirrors the subset of GitHub's PATCH /repos/{owner}/{repo} body gh-sweep can set.
type RepoSettingsPatch struct {
	AllowMergeCommit    *bool
	AllowSquashMerge    *bool
	AllowRebaseMerge    *bool
	AllowAutoMerge      *bool
	AllowUpdateBranch   *bool
	DeleteBranchOnMerge *bool
	UseSquashPRTitle    *bool
	HasIssues           *bool
	HasProjects         *bool
	HasWiki             *bool
	HasDiscussions      *bool
	AllowForking        *bool
	WebCommitSignoff    *bool
}

// UpdateRepoSettings applies a partial settings patch via PATCH /repos/{owner}/{repo}.
func (c *Client) UpdateRepoSettings(owner, repo string, patch RepoSettingsPatch) error {
	path := fmt.Sprintf("repos/%s/%s", owner, repo)

	var response repoResponse
	if err := c.Patch(path, patchBody(patch), &response); err != nil {
		return fmt.Errorf("failed to update repo settings: %w", err)
	}

	return nil
}

func patchBody(patch RepoSettingsPatch) map[string]any {
	body := map[string]any{}

	setIfPresent(body, "allow_merge_commit", patch.AllowMergeCommit)
	setIfPresent(body, "allow_squash_merge", patch.AllowSquashMerge)
	setIfPresent(body, "allow_rebase_merge", patch.AllowRebaseMerge)
	setIfPresent(body, "allow_auto_merge", patch.AllowAutoMerge)
	setIfPresent(body, "allow_update_branch", patch.AllowUpdateBranch)
	setIfPresent(body, "delete_branch_on_merge", patch.DeleteBranchOnMerge)
	setIfPresent(body, "use_squash_pr_title_as_default", patch.UseSquashPRTitle)
	setIfPresent(body, "has_issues", patch.HasIssues)
	setIfPresent(body, "has_projects", patch.HasProjects)
	setIfPresent(body, "has_wiki", patch.HasWiki)
	setIfPresent(body, "has_discussions", patch.HasDiscussions)
	setIfPresent(body, "allow_forking", patch.AllowForking)
	setIfPresent(body, "web_commit_signoff_required", patch.WebCommitSignoff)

	return body
}

func setIfPresent(body map[string]any, key string, value *bool) {
	if value != nil {
		body[key] = *value
	}
}

// UpdateSecurityAndAnalysis toggles a single security_and_analysis feature.
// GitHub requires the full nested object per request, so callers pass the one
// feature they want changed; the rest are omitted and left untouched server-side.
func (c *Client) UpdateSecurityAndAnalysis(owner, repo, feature, status string) error {
	path := fmt.Sprintf("repos/%s/%s", owner, repo)
	body := map[string]any{
		"security_and_analysis": map[string]any{
			feature: map[string]string{"status": status},
		},
	}

	var response repoResponse
	if err := c.Patch(path, body, &response); err != nil {
		return fmt.Errorf("failed to update security_and_analysis.%s: %w", feature, err)
	}

	return nil
}

// SettingsDiff represents differences between repository settings.
type SettingsDiff struct {
	Field    string
	Baseline interface{}
	Current  interface{}
	Severity string // critical, warning, info
}

// CompareSettings compares repository settings against a baseline.
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
			Field: "MergeStrategies",
			Baseline: fmt.Sprintf(
				"merge:%v squash:%v rebase:%v",
				baseline.AllowMergeCommit,
				baseline.AllowSquashMerge,
				baseline.AllowRebaseMerge,
			),
			Current: fmt.Sprintf(
				"merge:%v squash:%v rebase:%v",
				current.AllowMergeCommit,
				current.AllowSquashMerge,
				current.AllowRebaseMerge,
			),
			Severity: "info",
		})
	}

	return diffs
}
