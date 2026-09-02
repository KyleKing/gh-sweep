// Package policy diffs a config.PolicyConfig's declared repository state
// against live GitHub repos and applies it. It is the shared engine behind
// the `gh-sweep policy` CLI command and TUI view.
package policy

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/KyleKing/gh-sweep/internal/config"
	"github.com/KyleKing/gh-sweep/internal/github"
)

// ErrInvalidRepo means a policy referenced a repo not in "owner/repo" form.
var ErrInvalidRepo = errors.New("invalid repo (want owner/repo)")

const repoNameParts = 2

// Domain identifies which GitHub API surface a Diff belongs to.
type Domain string

// The domains a policy can declare and diff against.
const (
	DomainSettings   Domain = "settings"
	DomainSecurity   Domain = "security"
	DomainReleases   Domain = "releases"
	DomainProtection Domain = "protection"
	DomainRulesets   Domain = "rulesets"
)

// Diff is one field where a repo's live value doesn't match the policy.
type Diff struct {
	Domain  Domain `json:"domain"`
	Field   string `json:"field"`
	Desired string `json:"desired"`
	Current string `json:"current"`
}

// RepoDrift holds every diff found for one repository, or the error that
// stopped evaluation partway through (a later domain's fetch failing doesn't
// discard diffs already found in an earlier one).
type RepoDrift struct {
	Repository string `json:"repository"`
	Diffs      []Diff `json:"diffs"`
	Err        error  `json:"error,omitempty"`
}

// Report is the result of evaluating a policy against a set of repos.
type Report struct {
	Repos []RepoDrift `json:"repos"`
}

// HasDrift reports whether any repo has at least one diff.
func (r *Report) HasDrift() bool {
	for _, repo := range r.Repos {
		if len(repo.Diffs) > 0 {
			return true
		}
	}

	return false
}

// HasErrors reports whether any repo failed to evaluate. A run that fetched
// nothing must not read as converged, so callers gating on drift check this
// too.
func (r *Report) HasErrors() bool {
	for _, repo := range r.Repos {
		if repo.Err != nil {
			return true
		}
	}

	return false
}

// Evaluate fetches live state for every repo in cfg and diffs it against the
// declared policy. A per-repo fetch error is recorded on that RepoDrift rather
// than aborting the run, so one unreachable repo doesn't hide drift on the rest.
func Evaluate(client *github.Client, cfg *config.PolicyConfig) *Report {
	report := &Report{}

	for _, fullName := range cfg.QualifiedRepos() {
		report.Repos = append(report.Repos, evaluateRepo(client, cfg, fullName))
	}

	return report
}

func evaluateRepo(client *github.Client, cfg *config.PolicyConfig, fullName string) RepoDrift {
	owner, repo, ok := splitRepo(fullName)
	if !ok {
		return RepoDrift{Repository: fullName, Err: fmt.Errorf("%w: %q", ErrInvalidRepo, fullName)}
	}

	drift := RepoDrift{Repository: fullName}

	settings, err := client.GetRepoSettings(owner, repo)
	if err != nil {
		drift.Err = fmt.Errorf("fetching settings: %w", err)
		return drift
	}

	drift.Diffs = append(drift.Diffs, DiffSettings(&cfg.Settings, settings)...)
	drift.Diffs = append(drift.Diffs, DiffSecurity(&cfg.Security, &settings.SecurityAndAnalysis)...)

	if cfg.Releases.Immutable != nil {
		releases, err := client.GetImmutableReleases(owner, repo)
		if err != nil {
			drift.Err = fmt.Errorf("fetching immutable releases: %w", err)
			return drift
		}
		drift.Diffs = append(drift.Diffs, DiffReleases(*cfg.Releases.Immutable, releases.Enabled)...)
	}

	if cfg.Protection.Managed() {
		branch := settings.DefaultBranch
		if branch == "" {
			branch = "main"
		}

		rule, err := client.GetBranchProtection(owner, repo, branch)
		if err != nil {
			if !errors.Is(err, github.ErrBranchNotProtected) {
				drift.Err = fmt.Errorf("fetching branch protection: %w", err)
				return drift
			}
			rule = &github.ProtectionRule{Repository: fmt.Sprintf("%s/%s", owner, repo), Branch: branch}
		}
		drift.Diffs = append(drift.Diffs, DiffProtection(&cfg.Protection, rule)...)
	}

	if cfg.Ruleset.Managed() {
		live, err := client.FindRulesetByName(owner, repo, cfg.Ruleset.Name)
		if err != nil {
			if !errors.Is(err, github.ErrRulesetNotFound) {
				drift.Err = fmt.Errorf("fetching rulesets: %w", err)
				return drift
			}
			live = nil
		}
		drift.Diffs = append(drift.Diffs, DiffRuleset(&cfg.Ruleset, live)...)
	}

	return drift
}

// DiffRuleset compares a policy's declared ruleset against the repo's live one.
// A nil have means no ruleset by that name exists, which is reported as a single
// "absent" diff rather than one per field, since applying creates the whole thing.
func DiffRuleset(want *config.PolicyRuleset, have *github.Ruleset) []Diff {
	if have == nil {
		return []Diff{{
			Domain: DomainRulesets, Field: "ruleset", Desired: want.Name, Current: "absent",
		}}
	}

	diffs := []Diff{}

	addBool := func(field string, wantVal *bool, haveVal bool) {
		if wantVal != nil && *wantVal != haveVal {
			diffs = append(diffs, Diff{
				Domain: DomainRulesets, Field: field,
				Desired: strconv.FormatBool(*wantVal), Current: strconv.FormatBool(haveVal),
			})
		}
	}

	if want.Enforcement != "" && want.Enforcement != have.Enforcement {
		diffs = append(diffs, Diff{
			Domain: DomainRulesets, Field: "enforcement",
			Desired: want.Enforcement, Current: have.Enforcement,
		})
	}

	if want.IncludeRefs != nil && !equalStringSets(want.IncludeRefs, have.IncludeRefs) {
		diffs = append(diffs, Diff{
			Domain: DomainRulesets, Field: "include_refs",
			Desired: strings.Join(want.IncludeRefs, ","), Current: strings.Join(have.IncludeRefs, ","),
		})
	}

	addBool("block_deletion", want.BlockDeletion, have.BlockDeletion)
	addBool("block_force_push", want.BlockForcePush, have.BlockForcePush)
	addBool("require_linear_history", want.RequireLinearHistory, have.RequireLinearHistory)

	if want.RequireStatusChecks != nil && !equalStringSets(want.RequireStatusChecks, have.RequiredStatusChecks) {
		diffs = append(diffs, Diff{
			Domain: DomainRulesets, Field: "require_status_checks",
			Desired: strings.Join(want.RequireStatusChecks, ","),
			Current: strings.Join(have.RequiredStatusChecks, ","),
		})
	}

	return append(diffs, diffPullRequest(want.PullRequest, have.PullRequest)...)
}

func diffPullRequest(want *config.PolicyPullRequest, have *github.PullRequestRule) []Diff {
	if want == nil {
		return nil
	}

	if have == nil {
		return []Diff{{
			Domain: DomainRulesets, Field: "pull_request", Desired: "required", Current: "absent",
		}}
	}

	diffs := []Diff{}

	addBool := func(field string, wantVal *bool, haveVal bool) {
		if wantVal != nil && *wantVal != haveVal {
			diffs = append(diffs, Diff{
				Domain: DomainRulesets, Field: field,
				Desired: strconv.FormatBool(*wantVal), Current: strconv.FormatBool(haveVal),
			})
		}
	}

	if want.RequiredApprovals != nil && *want.RequiredApprovals != have.RequiredApprovals {
		diffs = append(diffs, Diff{
			Domain: DomainRulesets, Field: "required_approvals",
			Desired: strconv.Itoa(*want.RequiredApprovals), Current: strconv.Itoa(have.RequiredApprovals),
		})
	}

	addBool("require_code_owner_review", want.RequireCodeOwnerReview, have.RequireCodeOwnerReview)
	addBool("require_last_push_approval", want.RequireLastPushApproval, have.RequireLastPushApproval)
	addBool("dismiss_stale_reviews_on_push", want.DismissStaleReviewsOnPush, have.DismissStaleReviewsOnPush)
	addBool("required_review_thread_resolution",
		want.RequiredReviewThreadResolution, have.RequiredReviewThreadResolution)

	if want.AllowedMergeMethods != nil && !equalStringSets(want.AllowedMergeMethods, have.AllowedMergeMethods) {
		diffs = append(diffs, Diff{
			Domain: DomainRulesets, Field: "allowed_merge_methods",
			Desired: strings.Join(want.AllowedMergeMethods, ","),
			Current: strings.Join(have.AllowedMergeMethods, ","),
		})
	}

	return diffs
}

// DiffSettings compares a policy's declared repo settings against live values.
// Fields left nil in want are never diffed.
func DiffSettings(want *config.PolicySettings, have *github.RepoSettings) []Diff {
	diffs := []Diff{}

	add := func(field string, wantVal *bool, haveVal bool) {
		if wantVal != nil && *wantVal != haveVal {
			diffs = append(diffs, Diff{
				Domain: DomainSettings, Field: field,
				Desired: strconv.FormatBool(*wantVal), Current: strconv.FormatBool(haveVal),
			})
		}
	}

	add("allow_merge_commit", want.AllowMergeCommit, have.AllowMergeCommit)
	add("allow_squash_merge", want.AllowSquashMerge, have.AllowSquashMerge)
	add("allow_rebase_merge", want.AllowRebaseMerge, have.AllowRebaseMerge)
	add("allow_auto_merge", want.AllowAutoMerge, have.AllowAutoMerge)
	add("allow_update_branch", want.AllowUpdateBranch, have.AllowUpdateBranch)
	add("delete_branch_on_merge", want.DeleteBranchOnMerge, have.DeleteBranchOnMerge)
	add("use_squash_pr_title_as_default", want.UseSquashPRTitle, have.UseSquashPRTitle)
	add("has_issues", want.HasIssues, have.HasIssues)
	add("has_projects", want.HasProjects, have.HasProjects)
	add("has_wiki", want.HasWiki, have.HasWiki)
	add("has_discussions", want.HasDiscussions, have.HasDiscussions)
	add("allow_forking", want.AllowForking, have.AllowForking)
	add("web_commit_signoff_required", want.WebCommitSignoff, have.WebCommitSignoff)

	return diffs
}

// DiffSecurity compares a policy's declared security_and_analysis features
// against live values. Fields left empty in want are never diffed.
func DiffSecurity(want *config.PolicySecurity, have *github.SecurityAndAnalysis) []Diff {
	diffs := []Diff{}

	add := func(field, wantVal, haveVal string) {
		if wantVal != "" && wantVal != haveVal {
			diffs = append(diffs, Diff{Domain: DomainSecurity, Field: field, Desired: wantVal, Current: haveVal})
		}
	}

	add("secret_scanning", want.SecretScanning, have.SecretScanning)
	add("secret_scanning_push_protection", want.SecretScanningPushProtection, have.SecretScanningPushProtection)
	add("dependabot_security_updates", want.DependabotSecurityUpdates, have.DependabotSecurityUpdates)

	return diffs
}

// DiffReleases compares a policy's declared release-immutability state against live.
func DiffReleases(want, have bool) []Diff {
	if want == have {
		return nil
	}

	return []Diff{{
		Domain: DomainReleases, Field: "immutable",
		Desired: strconv.FormatBool(want), Current: strconv.FormatBool(have),
	}}
}

// DiffProtection compares a policy's declared branch-protection baseline
// against a repo's live rule. Fields left nil in want are never diffed.
func DiffProtection(want *config.PolicyProtection, have *github.ProtectionRule) []Diff {
	diffs := []Diff{}

	if want.RequiredReviews != nil && *want.RequiredReviews != have.RequiredReviews {
		diffs = append(diffs, Diff{
			Domain: DomainProtection, Field: "required_reviews",
			Desired: strconv.Itoa(*want.RequiredReviews), Current: strconv.Itoa(have.RequiredReviews),
		})
	}

	if want.RequireCodeOwnerReviews != nil && *want.RequireCodeOwnerReviews != have.RequireCodeOwnerReviews {
		diffs = append(diffs, Diff{
			Domain: DomainProtection, Field: "require_code_owner_reviews",
			Desired: strconv.FormatBool(*want.RequireCodeOwnerReviews),
			Current: strconv.FormatBool(have.RequireCodeOwnerReviews),
		})
	}

	if want.RequireStatusChecks != nil && !equalStringSets(want.RequireStatusChecks, have.RequireStatusChecks) {
		diffs = append(diffs, Diff{
			Domain: DomainProtection, Field: "require_status_checks",
			Desired: strings.Join(want.RequireStatusChecks, ","),
			Current: strings.Join(have.RequireStatusChecks, ","),
		})
	}

	if want.EnforceAdmins != nil && *want.EnforceAdmins != have.EnforceAdmins {
		diffs = append(diffs, Diff{
			Domain: DomainProtection, Field: "enforce_admins",
			Desired: strconv.FormatBool(*want.EnforceAdmins), Current: strconv.FormatBool(have.EnforceAdmins),
		})
	}

	if want.RequireLinearHistory != nil && *want.RequireLinearHistory != have.RequireLinearHistory {
		diffs = append(diffs, Diff{
			Domain: DomainProtection, Field: "require_linear_history",
			Desired: strconv.FormatBool(*want.RequireLinearHistory),
			Current: strconv.FormatBool(have.RequireLinearHistory),
		})
	}

	if want.AllowForcePushes != nil && *want.AllowForcePushes != have.AllowForcePushes {
		diffs = append(diffs, Diff{
			Domain: DomainProtection, Field: "allow_force_pushes",
			Desired: strconv.FormatBool(*want.AllowForcePushes), Current: strconv.FormatBool(have.AllowForcePushes),
		})
	}

	if want.AllowDeletions != nil && *want.AllowDeletions != have.AllowDeletions {
		diffs = append(diffs, Diff{
			Domain: DomainProtection, Field: "allow_deletions",
			Desired: strconv.FormatBool(*want.AllowDeletions), Current: strconv.FormatBool(have.AllowDeletions),
		})
	}

	return diffs
}

func equalStringSets(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	counts := make(map[string]int, len(a))
	for _, v := range a {
		counts[v]++
	}
	for _, v := range b {
		counts[v]--
	}
	for _, n := range counts {
		if n != 0 {
			return false
		}
	}

	return true
}

func splitRepo(fullName string) (string, string, bool) {
	parts := strings.SplitN(fullName, "/", repoNameParts)
	if len(parts) != repoNameParts || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}

	return parts[0], parts[1], true
}
