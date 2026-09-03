package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ErrPolicyFileNotFound means none of the candidate policy file paths exist.
var ErrPolicyFileNotFound = errors.New("no policy file found")

// PolicyConfig declares the desired GitHub state for a set of repositories:
// the settings gh-sweep's policy command diffs live repos against and can
// sync toward. Unlike Config, this is desired state, not flag defaults —
// every field is a pointer or has an explicit "unset means don't manage
// this" meaning, so a narrow policy only touches what it declares.
type PolicyConfig struct {
	DefaultOrg   string                    `yaml:"default_org"`
	Repositories []string                  `yaml:"repositories"`
	Settings     PolicySettings            `yaml:"settings"`
	Security     PolicySecurity            `yaml:"security"`
	Releases     PolicyReleases            `yaml:"releases"`
	Protection   PolicyProtection          `yaml:"protection"`
	Ruleset      PolicyRuleset             `yaml:"ruleset"`
	Branches     PolicyBranches            `yaml:"branches"`
	Overrides    map[string]PolicyOverride `yaml:"overrides"`
}

// PolicyOverride declares one repo's deviation from the base policy. It is
// keyed in PolicyConfig.Overrides by the same repo form as Repositories
// (bare name qualified by DefaultOrg, or "owner/repo"). Within each domain,
// an unset field (nil pointer, "" string) keeps the base policy's value for
// this repo; a set field replaces it. Ruleset is the exception: because a
// ruleset is matched by Name as one unit, declaring any Name in an override
// replaces the base Ruleset entirely for this repo rather than merging
// field by field.
type PolicyOverride struct {
	Settings   PolicySettings   `yaml:"settings"`
	Security   PolicySecurity   `yaml:"security"`
	Releases   PolicyReleases   `yaml:"releases"`
	Protection PolicyProtection `yaml:"protection"`
	Ruleset    PolicyRuleset    `yaml:"ruleset"`
	Branches   PolicyBranches   `yaml:"branches"`
}

// ForRepo resolves the policy that applies to fullName, merging any declared
// override onto the base policy's fields. Callers that already have a
// qualified name (from QualifiedRepos) get an exact map hit; a bare name is
// also tried qualified by DefaultOrg so overrides can be written either way.
func (p *PolicyConfig) ForRepo(fullName string) PolicyConfig {
	override, ok := p.Overrides[fullName]
	if !ok {
		bare := strings.TrimPrefix(fullName, p.DefaultOrg+"/")
		override, ok = p.Overrides[bare]
	}
	if !ok {
		return *p
	}

	resolved := *p
	resolved.Settings = mergeSettings(p.Settings, override.Settings)
	resolved.Security = mergeSecurity(p.Security, override.Security)
	resolved.Releases = mergeReleases(p.Releases, override.Releases)
	resolved.Protection = mergeProtection(p.Protection, override.Protection)
	resolved.Branches = mergeBranches(p.Branches, override.Branches)

	if override.Ruleset.Managed() {
		resolved.Ruleset = override.Ruleset
	}

	return resolved
}

func mergeSettings(base, override PolicySettings) PolicySettings {
	return PolicySettings{
		AllowMergeCommit:    firstNonNil(override.AllowMergeCommit, base.AllowMergeCommit),
		AllowSquashMerge:    firstNonNil(override.AllowSquashMerge, base.AllowSquashMerge),
		AllowRebaseMerge:    firstNonNil(override.AllowRebaseMerge, base.AllowRebaseMerge),
		AllowAutoMerge:      firstNonNil(override.AllowAutoMerge, base.AllowAutoMerge),
		AllowUpdateBranch:   firstNonNil(override.AllowUpdateBranch, base.AllowUpdateBranch),
		DeleteBranchOnMerge: firstNonNil(override.DeleteBranchOnMerge, base.DeleteBranchOnMerge),
		UseSquashPRTitle:    firstNonNil(override.UseSquashPRTitle, base.UseSquashPRTitle),
		HasIssues:           firstNonNil(override.HasIssues, base.HasIssues),
		HasProjects:         firstNonNil(override.HasProjects, base.HasProjects),
		HasWiki:             firstNonNil(override.HasWiki, base.HasWiki),
		HasDiscussions:      firstNonNil(override.HasDiscussions, base.HasDiscussions),
		AllowForking:        firstNonNil(override.AllowForking, base.AllowForking),
		WebCommitSignoff:    firstNonNil(override.WebCommitSignoff, base.WebCommitSignoff),
	}
}

func mergeSecurity(base, override PolicySecurity) PolicySecurity {
	return PolicySecurity{
		SecretScanning: firstNonEmpty(override.SecretScanning, base.SecretScanning),
		SecretScanningPushProtection: firstNonEmpty(
			override.SecretScanningPushProtection, base.SecretScanningPushProtection,
		),
		DependabotSecurityUpdates: firstNonEmpty(override.DependabotSecurityUpdates, base.DependabotSecurityUpdates),
	}
}

func mergeReleases(base, override PolicyReleases) PolicyReleases {
	return PolicyReleases{
		Immutable: firstNonNil(override.Immutable, base.Immutable),
	}
}

func mergeProtection(base, override PolicyProtection) PolicyProtection {
	return PolicyProtection{
		RequiredReviews:         firstNonNil(override.RequiredReviews, base.RequiredReviews),
		RequireCodeOwnerReviews: firstNonNil(override.RequireCodeOwnerReviews, base.RequireCodeOwnerReviews),
		RequireStatusChecks:     firstNonEmptySlice(override.RequireStatusChecks, base.RequireStatusChecks),
		EnforceAdmins:           firstNonNil(override.EnforceAdmins, base.EnforceAdmins),
		RequireLinearHistory:    firstNonNil(override.RequireLinearHistory, base.RequireLinearHistory),
		AllowForcePushes:        firstNonNil(override.AllowForcePushes, base.AllowForcePushes),
		AllowDeletions:          firstNonNil(override.AllowDeletions, base.AllowDeletions),
	}
}

func mergeBranches(base, override PolicyBranches) PolicyBranches {
	return PolicyBranches{
		PruneMerged:     firstNonNil(override.PruneMerged, base.PruneMerged),
		PruneClosed:     firstNonNil(override.PruneClosed, base.PruneClosed),
		PruneNoPR:       firstNonNil(override.PruneNoPR, base.PruneNoPR),
		NoPRGraceDays:   firstNonNil(override.NoPRGraceDays, base.NoPRGraceDays),
		ExcludePatterns: firstNonEmptySlice(override.ExcludePatterns, base.ExcludePatterns),
	}
}

func firstNonNil[T any](override, base *T) *T {
	if override != nil {
		return override
	}

	return base
}

func firstNonEmpty(override, base string) string {
	if override != "" {
		return override
	}

	return base
}

func firstNonEmptySlice(override, base []string) []string {
	if len(override) > 0 {
		return override
	}

	return base
}

// PolicySettings mirrors github.RepoSettingsPatch: unset (nil) fields are left
// alone on every repo, present fields are enforced.
type PolicySettings struct {
	AllowMergeCommit    *bool `yaml:"allow_merge_commit"`
	AllowSquashMerge    *bool `yaml:"allow_squash_merge"`
	AllowRebaseMerge    *bool `yaml:"allow_rebase_merge"`
	AllowAutoMerge      *bool `yaml:"allow_auto_merge"`
	AllowUpdateBranch   *bool `yaml:"allow_update_branch"`
	DeleteBranchOnMerge *bool `yaml:"delete_branch_on_merge"`
	UseSquashPRTitle    *bool `yaml:"use_squash_pr_title_as_default"`
	HasIssues           *bool `yaml:"has_issues"`
	HasProjects         *bool `yaml:"has_projects"`
	HasWiki             *bool `yaml:"has_wiki"`
	HasDiscussions      *bool `yaml:"has_discussions"`
	AllowForking        *bool `yaml:"allow_forking"`
	WebCommitSignoff    *bool `yaml:"web_commit_signoff_required"`
}

// PolicySecurity declares desired security_and_analysis feature states.
// Values are "enabled" or "disabled"; an empty string leaves that feature unmanaged.
type PolicySecurity struct {
	SecretScanning               string `yaml:"secret_scanning"`
	SecretScanningPushProtection string `yaml:"secret_scanning_push_protection"`
	DependabotSecurityUpdates    string `yaml:"dependabot_security_updates"`
}

// PolicyReleases declares the desired release-immutability state.
type PolicyReleases struct {
	Immutable *bool `yaml:"immutable"`
}

// PolicyProtection declares a desired branch-protection baseline applied to
// each repo's default branch. A nil *PolicyProtection.RequiredReviews (etc.)
// field leaves that rule unmanaged rather than clearing it.
type PolicyProtection struct {
	RequiredReviews         *int     `yaml:"required_reviews"`
	RequireCodeOwnerReviews *bool    `yaml:"require_code_owner_reviews"`
	RequireStatusChecks     []string `yaml:"require_status_checks"`
	EnforceAdmins           *bool    `yaml:"enforce_admins"`
	RequireLinearHistory    *bool    `yaml:"require_linear_history"`
	AllowForcePushes        *bool    `yaml:"allow_force_pushes"`
	AllowDeletions          *bool    `yaml:"allow_deletions"`
}

// Managed reports whether any branch-protection rule is declared.
func (p PolicyProtection) Managed() bool {
	return p.RequiredReviews != nil || p.RequireCodeOwnerReviews != nil ||
		p.RequireStatusChecks != nil || p.EnforceAdmins != nil ||
		p.RequireLinearHistory != nil || p.AllowForcePushes != nil ||
		p.AllowDeletions != nil
}

// PolicyRuleset declares a desired repository ruleset, managed by name. A
// ruleset expresses rules classic branch protection cannot, notably requiring
// a pull request with zero required approvals. An empty Name leaves rulesets
// unmanaged; rules the policy does not declare keep their live value.
type PolicyRuleset struct {
	Name                 string             `yaml:"name"`
	Enforcement          string             `yaml:"enforcement"`
	IncludeRefs          []string           `yaml:"include_refs"`
	ExcludeRefs          []string           `yaml:"exclude_refs"`
	BlockDeletion        *bool              `yaml:"block_deletion"`
	BlockForcePush       *bool              `yaml:"block_force_push"`
	RequireLinearHistory *bool              `yaml:"require_linear_history"`
	RequireStatusChecks  []string           `yaml:"require_status_checks"`
	PullRequest          *PolicyPullRequest `yaml:"pull_request"`
}

// PolicyPullRequest declares a ruleset's pull_request rule. Declaring the
// block at all requires a PR; RequiredApprovals of 0 requires no approval on
// it. Omitting the block leaves the live rule alone rather than removing it.
type PolicyPullRequest struct {
	RequiredApprovals              *int     `yaml:"required_approvals"`
	RequireCodeOwnerReview         *bool    `yaml:"require_code_owner_review"`
	RequireLastPushApproval        *bool    `yaml:"require_last_push_approval"`
	DismissStaleReviewsOnPush      *bool    `yaml:"dismiss_stale_reviews_on_push"`
	RequiredReviewThreadResolution *bool    `yaml:"required_review_thread_resolution"`
	AllowedMergeMethods            []string `yaml:"allowed_merge_methods"`
}

// Managed reports whether a ruleset is declared. Name is the identity gh-sweep
// matches on, so a policy without one manages no ruleset.
func (p PolicyRuleset) Managed() bool {
	return p.Name != ""
}

// QualifiedRepos returns Repositories with bare names prefixed by DefaultOrg.
func (p *PolicyConfig) QualifiedRepos() []string {
	repos := make([]string, 0, len(p.Repositories))
	for _, repo := range p.Repositories {
		if !strings.Contains(repo, "/") && p.DefaultOrg != "" {
			repo = p.DefaultOrg + "/" + repo
		}
		repos = append(repos, repo)
	}

	return repos
}

// policyPaths are checked in order; the first that exists wins, matching Load's
// project-then-home precedence.
var policyPaths = []string{
	".gh-sweep-policy.yaml",
	os.Getenv("HOME") + "/.gh-sweep-policy.yaml",
}

// LoadPolicy reads a PolicyConfig from path, or the first of the default
// policy-file locations if path is empty. Returns an error if none is found;
// unlike Load, there is no sensible zero-value default for desired state.
func LoadPolicy(path string) (*PolicyConfig, error) {
	paths := policyPaths
	if path != "" {
		paths = []string{path}
	}

	var data []byte
	var err error
	var foundPath string

	for _, candidate := range paths {
		data, err = os.ReadFile(candidate)
		if err == nil {
			foundPath = candidate
			break
		}
	}

	if foundPath == "" {
		return nil, fmt.Errorf("%w (looked in %s)", ErrPolicyFileNotFound, strings.Join(paths, ", "))
	}

	var cfg PolicyConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse policy from %s: %w", foundPath, err)
	}

	return &cfg, nil
}

// SavePolicy writes the policy to path as YAML.
func (p *PolicyConfig) SavePolicy(path string) error {
	data, err := yaml.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write policy: %w", err)
	}

	return nil
}

// PolicyBranches declares which leftover branches should not exist. Branches
// from merged or closed PRs carry no grace period, because the PR records the
// work and GitHub restores the branch from it on request; a branch with no PR
// has no such record, so NoPRGraceDays applies before it is pruned.
type PolicyBranches struct {
	PruneMerged     *bool    `yaml:"prune_merged"`
	PruneClosed     *bool    `yaml:"prune_closed"`
	PruneNoPR       *bool    `yaml:"prune_no_pr"`
	NoPRGraceDays   *int     `yaml:"no_pr_grace_days"`
	ExcludePatterns []string `yaml:"exclude_patterns"`
}

// Managed reports whether the policy declares any branch pruning at all.
func (p PolicyBranches) Managed() bool {
	return enabled(p.PruneMerged) || enabled(p.PruneClosed) || enabled(p.PruneNoPR)
}

func enabled(flag *bool) bool {
	return flag != nil && *flag
}
