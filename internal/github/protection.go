package github

import "fmt"

// ProtectionRule represents branch protection settings.
type ProtectionRule struct {
	Repository              string
	Branch                  string
	RequiredReviews         int
	RequireCodeOwnerReviews bool
	RequireStatusChecks     []string
	EnforceAdmins           bool
	RequireLinearHistory    bool
	AllowForcePushes        bool
	AllowDeletions          bool
}

type protectionResponse struct {
	RequiredPullRequestReviews *struct {
		RequiredApprovingReviewCount int  `json:"required_approving_review_count"`
		RequireCodeOwnerReviews      bool `json:"require_code_owner_reviews"`
	} `json:"required_pull_request_reviews"`
	RequiredStatusChecks *struct {
		Contexts []string `json:"contexts"`
	} `json:"required_status_checks"`
	EnforceAdmins struct {
		Enabled bool `json:"enabled"`
	} `json:"enforce_admins"`
	RequireLinearHistory struct {
		Enabled bool `json:"enabled"`
	} `json:"required_linear_history"`
	AllowForcePushes struct {
		Enabled bool `json:"enabled"`
	} `json:"allow_force_pushes"`
	AllowDeletions struct {
		Enabled bool `json:"enabled"`
	} `json:"allow_deletions"`
}

// GetBranchProtection retrieves branch protection rules.
func (c *Client) GetBranchProtection(owner, repo, branch string) (*ProtectionRule, error) {
	var response protectionResponse
	path := fmt.Sprintf("repos/%s/%s/branches/%s/protection", owner, repo, branch)

	if err := c.Get(path, &response); err != nil {
		return nil, fmt.Errorf("failed to get branch protection: %w", err)
	}

	rule := &ProtectionRule{
		Repository:           fmt.Sprintf("%s/%s", owner, repo),
		Branch:               branch,
		EnforceAdmins:        response.EnforceAdmins.Enabled,
		RequireLinearHistory: response.RequireLinearHistory.Enabled,
		AllowForcePushes:     response.AllowForcePushes.Enabled,
		AllowDeletions:       response.AllowDeletions.Enabled,
	}

	if response.RequiredPullRequestReviews != nil {
		rule.RequiredReviews = response.RequiredPullRequestReviews.RequiredApprovingReviewCount
		rule.RequireCodeOwnerReviews = response.RequiredPullRequestReviews.RequireCodeOwnerReviews
	}

	if response.RequiredStatusChecks != nil {
		rule.RequireStatusChecks = response.RequiredStatusChecks.Contexts
	}

	return rule, nil
}

// UpdateBranchProtection replaces branch protection rules via PUT, which GitHub
// requires as a full replacement rather than a partial patch. Fields left at their
// zero value (e.g. no status checks) are sent as such, not omitted.
func (c *Client) UpdateBranchProtection(owner, repo, branch string, desired ProtectionRule) error {
	path := fmt.Sprintf("repos/%s/%s/branches/%s/protection", owner, repo, branch)

	body := map[string]any{
		"required_status_checks": map[string]any{
			"strict":   false,
			"contexts": desired.RequireStatusChecks,
		},
		"enforce_admins": desired.EnforceAdmins,
		"required_pull_request_reviews": map[string]any{
			"required_approving_review_count": desired.RequiredReviews,
			"require_code_owner_reviews":      desired.RequireCodeOwnerReviews,
		},
		"restrictions":            nil,
		"required_linear_history": desired.RequireLinearHistory,
		"allow_force_pushes":      desired.AllowForcePushes,
		"allow_deletions":         desired.AllowDeletions,
	}

	if desired.RequiredReviews == 0 {
		body["required_pull_request_reviews"] = nil
	}

	var response protectionResponse
	if err := c.Put(path, body, &response); err != nil {
		return fmt.Errorf("failed to update branch protection: %w", err)
	}

	return nil
}

// GetDefaultBranchProtection retrieves protection rules for the repo's default branch.
func (c *Client) GetDefaultBranchProtection(owner, repo string) (*ProtectionRule, error) {
	branch := "main"
	if settings, err := c.GetRepoSettings(owner, repo); err == nil && settings.DefaultBranch != "" {
		branch = settings.DefaultBranch
	}

	return c.GetBranchProtection(owner, repo, branch)
}

// CompareProtectionRules compares protection rules across repositories.
func CompareProtectionRules(rules []*ProtectionRule) map[string][]string {
	differences := make(map[string][]string)

	if len(rules) < 2 {
		return differences
	}

	baseline := rules[0]

	for i := 1; i < len(rules); i++ {
		rule := rules[i]

		if rule.RequiredReviews != baseline.RequiredReviews {
			differences["RequiredReviews"] = append(
				differences["RequiredReviews"],
				fmt.Sprintf(
					"%s: %d (baseline: %d)",
					rule.Repository,
					rule.RequiredReviews,
					baseline.RequiredReviews,
				),
			)
		}

		if rule.RequireCodeOwnerReviews != baseline.RequireCodeOwnerReviews {
			differences["RequireCodeOwnerReviews"] = append(
				differences["RequireCodeOwnerReviews"],
				fmt.Sprintf(
					"%s: %v (baseline: %v)",
					rule.Repository,
					rule.RequireCodeOwnerReviews,
					baseline.RequireCodeOwnerReviews,
				),
			)
		}

		if rule.EnforceAdmins != baseline.EnforceAdmins {
			differences["EnforceAdmins"] = append(
				differences["EnforceAdmins"],
				fmt.Sprintf(
					"%s: %v (baseline: %v)",
					rule.Repository,
					rule.EnforceAdmins,
					baseline.EnforceAdmins,
				),
			)
		}
	}

	return differences
}
