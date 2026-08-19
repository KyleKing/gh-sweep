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
	DefaultOrg   string           `yaml:"default_org"`
	Repositories []string         `yaml:"repositories"`
	Settings     PolicySettings   `yaml:"settings"`
	Security     PolicySecurity   `yaml:"security"`
	Releases     PolicyReleases   `yaml:"releases"`
	Protection   PolicyProtection `yaml:"protection"`
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
