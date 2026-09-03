package policy

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/KyleKing/gh-sweep/internal/config"
	"github.com/KyleKing/gh-sweep/internal/policy"
)

const fieldRequireStatusChecks = "require_status_checks"

// ErrUnknownField means a Diff named a domain/field pair this editor does not
// recognize, which would mean policy.go grew a diff kind edit.go hasn't
// caught up with.
var ErrUnknownField = errors.New("unknown policy field")

// fieldKind says how to prompt for and parse a new value for a declared
// field. A diff only exists for a field the policy already declares, so
// editing always changes a value in place; it can never declare a
// previously-unmanaged field.
type fieldKind int

const (
	kindBool fieldKind = iota
	kindInt
	kindString
	kindStringSlice
	kindUnsupported
)

// fieldKindOf reports how to edit the field behind one Diff, keyed by the
// same Domain/Field pair Diff carries.
func fieldKindOf(domain policy.Domain, field string) fieldKind {
	switch domain {
	case policy.DomainSettings, policy.DomainReleases:
		return kindBool

	case policy.DomainSecurity:
		return kindString

	case policy.DomainProtection:
		switch field {
		case "required_reviews":
			return kindInt
		case fieldRequireStatusChecks:
			return kindStringSlice
		default:
			return kindBool
		}

	case policy.DomainRulesets:
		return rulesetFieldKind(field)

	case policy.DomainBranches:
		if field == "no_pr_grace_days" {
			return kindInt
		}

		return kindBool

	default:
		return kindUnsupported
	}
}

func rulesetFieldKind(field string) fieldKind {
	switch field {
	case "ruleset", "pull_request":
		// Marks the whole block absent on the live repo; there is no single
		// field to edit until it exists.
		return kindUnsupported
	case "enforcement":
		return kindString
	case "include_refs", fieldRequireStatusChecks, "allowed_merge_methods":
		return kindStringSlice
	case "required_approvals":
		return kindInt
	default:
		return kindBool
	}
}

// toggledBool computes the flipped value for a boolean Diff's currently
// declared value, as the same "true"/"false" string Diff.Desired uses.
func toggledBool(desired string) string {
	if desired == "true" {
		return "false"
	}

	return "true"
}

// applyEdit writes newValue into cfg's declared field for domain/field,
// parsed according to that field's kind. It reports an error rather than
// silently ignoring an unparsable or unsupported edit.
func applyEdit(cfg *config.PolicyConfig, domain policy.Domain, field, newValue string) error {
	switch domain {
	case policy.DomainSettings:
		return setSettingsField(&cfg.Settings, field, newValue)
	case policy.DomainSecurity:
		return setSecurityField(&cfg.Security, field, newValue)
	case policy.DomainReleases:
		return setBoolPtr(&cfg.Releases.Immutable, newValue)
	case policy.DomainProtection:
		return setProtectionField(&cfg.Protection, field, newValue)
	case policy.DomainRulesets:
		return setRulesetField(&cfg.Ruleset, field, newValue)
	case policy.DomainBranches:
		return setBranchesField(&cfg.Branches, field, newValue)
	default:
		return fmt.Errorf("%w: %s/%s", ErrUnknownField, domain, field)
	}
}

func setBoolPtr(field **bool, newValue string) error {
	parsed, err := strconv.ParseBool(newValue)
	if err != nil {
		return fmt.Errorf("parsing %q as true/false: %w", newValue, err)
	}

	*field = &parsed

	return nil
}

func setIntPtr(field **int, newValue string) error {
	parsed, err := strconv.Atoi(newValue)
	if err != nil {
		return fmt.Errorf("parsing %q as a number: %w", newValue, err)
	}

	*field = &parsed

	return nil
}

func setStringSlice(field *[]string, newValue string) {
	if strings.TrimSpace(newValue) == "" {
		*field = nil
		return
	}

	parts := strings.Split(newValue, ",")
	values := make([]string, 0, len(parts))

	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}

	*field = values
}

func setSettingsField(s *config.PolicySettings, field, newValue string) error {
	target := map[string]**bool{
		"allow_merge_commit":             &s.AllowMergeCommit,
		"allow_squash_merge":             &s.AllowSquashMerge,
		"allow_rebase_merge":             &s.AllowRebaseMerge,
		"allow_auto_merge":               &s.AllowAutoMerge,
		"allow_update_branch":            &s.AllowUpdateBranch,
		"delete_branch_on_merge":         &s.DeleteBranchOnMerge,
		"use_squash_pr_title_as_default": &s.UseSquashPRTitle,
		"has_issues":                     &s.HasIssues,
		"has_projects":                   &s.HasProjects,
		"has_wiki":                       &s.HasWiki,
		"has_discussions":                &s.HasDiscussions,
		"allow_forking":                  &s.AllowForking,
		"web_commit_signoff_required":    &s.WebCommitSignoff,
	}[field]
	if target == nil {
		return fmt.Errorf("%w: settings/%s", ErrUnknownField, field)
	}

	return setBoolPtr(target, newValue)
}

func setSecurityField(s *config.PolicySecurity, field, newValue string) error {
	target := map[string]*string{
		"secret_scanning":                 &s.SecretScanning,
		"secret_scanning_push_protection": &s.SecretScanningPushProtection,
		"dependabot_security_updates":     &s.DependabotSecurityUpdates,
	}[field]
	if target == nil {
		return fmt.Errorf("%w: security/%s", ErrUnknownField, field)
	}

	*target = newValue

	return nil
}

func setProtectionField(p *config.PolicyProtection, field, newValue string) error {
	switch field {
	case "required_reviews":
		return setIntPtr(&p.RequiredReviews, newValue)
	case "require_code_owner_reviews":
		return setBoolPtr(&p.RequireCodeOwnerReviews, newValue)
	case fieldRequireStatusChecks:
		setStringSlice(&p.RequireStatusChecks, newValue)
		return nil
	case "enforce_admins":
		return setBoolPtr(&p.EnforceAdmins, newValue)
	case "require_linear_history":
		return setBoolPtr(&p.RequireLinearHistory, newValue)
	case "allow_force_pushes":
		return setBoolPtr(&p.AllowForcePushes, newValue)
	case "allow_deletions":
		return setBoolPtr(&p.AllowDeletions, newValue)
	default:
		return fmt.Errorf("%w: protection/%s", ErrUnknownField, field)
	}
}

func setRulesetField(r *config.PolicyRuleset, field, newValue string) error {
	switch field {
	case "enforcement":
		r.Enforcement = newValue
		return nil
	case "include_refs":
		setStringSlice(&r.IncludeRefs, newValue)
		return nil
	case "block_deletion":
		return setBoolPtr(&r.BlockDeletion, newValue)
	case "block_force_push":
		return setBoolPtr(&r.BlockForcePush, newValue)
	case "require_linear_history":
		return setBoolPtr(&r.RequireLinearHistory, newValue)
	case fieldRequireStatusChecks:
		setStringSlice(&r.RequireStatusChecks, newValue)
		return nil
	default:
		return setPullRequestField(r.PullRequest, field, newValue)
	}
}

func setPullRequestField(pr *config.PolicyPullRequest, field, newValue string) error {
	if pr == nil {
		return fmt.Errorf("%w: pull_request/%s", ErrUnknownField, field)
	}

	switch field {
	case "required_approvals":
		return setIntPtr(&pr.RequiredApprovals, newValue)
	case "require_code_owner_review":
		return setBoolPtr(&pr.RequireCodeOwnerReview, newValue)
	case "require_last_push_approval":
		return setBoolPtr(&pr.RequireLastPushApproval, newValue)
	case "dismiss_stale_reviews_on_push":
		return setBoolPtr(&pr.DismissStaleReviewsOnPush, newValue)
	case "required_review_thread_resolution":
		return setBoolPtr(&pr.RequiredReviewThreadResolution, newValue)
	case "allowed_merge_methods":
		setStringSlice(&pr.AllowedMergeMethods, newValue)
		return nil
	default:
		return fmt.Errorf("%w: ruleset/%s", ErrUnknownField, field)
	}
}

func setBranchesField(b *config.PolicyBranches, field, newValue string) error {
	switch field {
	case "prune_merged":
		return setBoolPtr(&b.PruneMerged, newValue)
	case "prune_closed":
		return setBoolPtr(&b.PruneClosed, newValue)
	case "prune_no_pr":
		return setBoolPtr(&b.PruneNoPR, newValue)
	case "no_pr_grace_days":
		return setIntPtr(&b.NoPRGraceDays, newValue)
	default:
		return fmt.Errorf("%w: branches/%s", ErrUnknownField, field)
	}
}
