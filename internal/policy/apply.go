package policy

import (
	"errors"
	"fmt"

	"github.com/KyleKing/gh-sweep/internal/config"
	"github.com/KyleKing/gh-sweep/internal/github"
)

// ApplyResult reports what happened when applying a policy to one repo.
type ApplyResult struct {
	Repository string
	Applied    []Domain
	Err        error
}

// Apply pushes every managed field in cfg to the repo, for each domain that
// had a diff in drift. It re-sends the whole managed subset of that domain
// rather than only the changed fields, so re-applying an already-converged
// policy is a safe no-op instead of silently skipping fields nobody diffed
// this run.
func Apply(client *github.Client, cfg *config.PolicyConfig, drift RepoDrift) ApplyResult {
	result := ApplyResult{Repository: drift.Repository}

	owner, repo, ok := splitRepo(drift.Repository)
	if !ok {
		result.Err = fmt.Errorf("%w: %q", ErrInvalidRepo, drift.Repository)
		return result
	}

	domains := domainsWithDrift(drift.Diffs)

	if domains[DomainSettings] {
		if err := client.UpdateRepoSettings(owner, repo, settingsPatch(&cfg.Settings)); err != nil {
			result.Err = fmt.Errorf("applying settings: %w", err)
			return result
		}
		result.Applied = append(result.Applied, DomainSettings)
	}

	if domains[DomainSecurity] {
		if err := applySecurity(client, owner, repo, &cfg.Security); err != nil {
			result.Err = fmt.Errorf("applying security: %w", err)
			return result
		}
		result.Applied = append(result.Applied, DomainSecurity)
	}

	if domains[DomainReleases] {
		if err := client.SetImmutableReleases(owner, repo, *cfg.Releases.Immutable); err != nil {
			result.Err = fmt.Errorf("applying release immutability: %w", err)
			return result
		}
		result.Applied = append(result.Applied, DomainReleases)
	}

	if domains[DomainProtection] {
		if err := applyProtection(client, owner, repo, &cfg.Protection); err != nil {
			result.Err = fmt.Errorf("applying branch protection: %w", err)
			return result
		}
		result.Applied = append(result.Applied, DomainProtection)
	}

	return result
}

func domainsWithDrift(diffs []Diff) map[Domain]bool {
	domains := make(map[Domain]bool, 4)
	for _, d := range diffs {
		domains[d.Domain] = true
	}

	return domains
}

func settingsPatch(want *config.PolicySettings) github.RepoSettingsPatch {
	return github.RepoSettingsPatch{
		AllowMergeCommit:    want.AllowMergeCommit,
		AllowSquashMerge:    want.AllowSquashMerge,
		AllowRebaseMerge:    want.AllowRebaseMerge,
		AllowAutoMerge:      want.AllowAutoMerge,
		AllowUpdateBranch:   want.AllowUpdateBranch,
		DeleteBranchOnMerge: want.DeleteBranchOnMerge,
		UseSquashPRTitle:    want.UseSquashPRTitle,
		HasIssues:           want.HasIssues,
		HasProjects:         want.HasProjects,
		HasWiki:             want.HasWiki,
		HasDiscussions:      want.HasDiscussions,
		AllowForking:        want.AllowForking,
		WebCommitSignoff:    want.WebCommitSignoff,
	}
}

func applySecurity(client *github.Client, owner, repo string, want *config.PolicySecurity) error {
	features := map[string]string{
		"secret_scanning":                 want.SecretScanning,
		"secret_scanning_push_protection": want.SecretScanningPushProtection,
		"dependabot_security_updates":     want.DependabotSecurityUpdates,
	}

	for feature, status := range features {
		if status == "" {
			continue
		}
		if err := client.UpdateSecurityAndAnalysis(owner, repo, feature, status); err != nil {
			return fmt.Errorf("updating %s: %w", feature, err)
		}
	}

	return nil
}

// applyProtection merges declared overrides onto the repo's current protection
// rule so fields the policy doesn't manage keep their live value rather than
// being reset by GitHub's full-replacement PUT semantics.
func applyProtection(client *github.Client, owner, repo string, want *config.PolicyProtection) error {
	branch := "main"
	if settings, err := client.GetRepoSettings(owner, repo); err == nil && settings.DefaultBranch != "" {
		branch = settings.DefaultBranch
	}

	current, err := client.GetBranchProtection(owner, repo, branch)
	if err != nil {
		if !errors.Is(err, github.ErrBranchNotProtected) {
			return fmt.Errorf("fetching current protection: %w", err)
		}
		current = &github.ProtectionRule{Repository: fmt.Sprintf("%s/%s", owner, repo), Branch: branch}
	}

	desired := *current

	if want.RequiredReviews != nil {
		desired.RequiredReviews = *want.RequiredReviews
	}
	if want.RequireCodeOwnerReviews != nil {
		desired.RequireCodeOwnerReviews = *want.RequireCodeOwnerReviews
	}
	if want.RequireStatusChecks != nil {
		desired.RequireStatusChecks = want.RequireStatusChecks
	}
	if want.EnforceAdmins != nil {
		desired.EnforceAdmins = *want.EnforceAdmins
	}
	if want.RequireLinearHistory != nil {
		desired.RequireLinearHistory = *want.RequireLinearHistory
	}
	if want.AllowForcePushes != nil {
		desired.AllowForcePushes = *want.AllowForcePushes
	}
	if want.AllowDeletions != nil {
		desired.AllowDeletions = *want.AllowDeletions
	}

	if err := client.UpdateBranchProtection(owner, repo, branch, desired); err != nil {
		return fmt.Errorf("updating branch protection: %w", err)
	}

	return nil
}
