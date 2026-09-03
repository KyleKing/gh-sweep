package policy

import (
	"errors"
	"testing"

	"github.com/KyleKing/gh-sweep/internal/config"
	"github.com/KyleKing/gh-sweep/internal/policy"
)

func TestFieldKindOf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		domain policy.Domain
		field  string
		want   fieldKind
	}{
		{policy.DomainSettings, "has_wiki", kindBool},
		{policy.DomainSecurity, "secret_scanning", kindString},
		{policy.DomainReleases, "immutable", kindBool},
		{policy.DomainProtection, "required_reviews", kindInt},
		{policy.DomainProtection, "require_status_checks", kindStringSlice},
		{policy.DomainProtection, "enforce_admins", kindBool},
		{policy.DomainRulesets, "ruleset", kindUnsupported},
		{policy.DomainRulesets, "enforcement", kindString},
		{policy.DomainRulesets, "include_refs", kindStringSlice},
		{policy.DomainRulesets, "required_approvals", kindInt},
		{policy.DomainRulesets, "block_deletion", kindBool},
		{policy.DomainBranches, "no_pr_grace_days", kindInt},
		{policy.DomainBranches, "prune_merged", kindBool},
	}

	for _, tt := range tests {
		t.Run(string(tt.domain)+"/"+tt.field, func(t *testing.T) {
			t.Parallel()

			if got := fieldKindOf(tt.domain, tt.field); got != tt.want {
				t.Errorf("fieldKindOf(%s, %s) = %v, want %v", tt.domain, tt.field, got, tt.want)
			}
		})
	}
}

func TestToggledBool(t *testing.T) {
	t.Parallel()

	if got := toggledBool("true"); got != "false" {
		t.Errorf("toggledBool(true) = %s, want false", got)
	}
	if got := toggledBool("false"); got != "true" {
		t.Errorf("toggledBool(false) = %s, want true", got)
	}
}

func TestApplyEditSettings(t *testing.T) {
	t.Parallel()

	cfg := &config.PolicyConfig{Settings: config.PolicySettings{HasWiki: boolPtr(true)}}

	if err := applyEdit(cfg, policy.DomainSettings, "has_wiki", "false"); err != nil {
		t.Fatalf("applyEdit() error = %v", err)
	}
	if cfg.Settings.HasWiki == nil || *cfg.Settings.HasWiki {
		t.Errorf("HasWiki = %v, want false", cfg.Settings.HasWiki)
	}
}

func TestApplyEditProtectionInt(t *testing.T) {
	t.Parallel()

	cfg := &config.PolicyConfig{Protection: config.PolicyProtection{RequiredReviews: intPtr(1)}}

	if err := applyEdit(cfg, policy.DomainProtection, "required_reviews", "3"); err != nil {
		t.Fatalf("applyEdit() error = %v", err)
	}
	if cfg.Protection.RequiredReviews == nil || *cfg.Protection.RequiredReviews != 3 {
		t.Errorf("RequiredReviews = %v, want 3", cfg.Protection.RequiredReviews)
	}

	if err := applyEdit(cfg, policy.DomainProtection, "required_reviews", "not-a-number"); err == nil {
		t.Error("applyEdit() error = nil, want a parse error")
	}
}

func TestApplyEditProtectionStringSlice(t *testing.T) {
	t.Parallel()

	cfg := &config.PolicyConfig{}

	if err := applyEdit(cfg, policy.DomainProtection, "require_status_checks", "ci, lint , "); err != nil {
		t.Fatalf("applyEdit() error = %v", err)
	}

	want := []string{"ci", "lint"}
	got := cfg.Protection.RequireStatusChecks
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("RequireStatusChecks = %v, want %v", got, want)
	}
}

func TestApplyEditRulesetPullRequestField(t *testing.T) {
	t.Parallel()

	cfg := &config.PolicyConfig{
		Ruleset: config.PolicyRuleset{
			Name:        "main",
			PullRequest: &config.PolicyPullRequest{RequiredApprovals: intPtr(0)},
		},
	}

	if err := applyEdit(cfg, policy.DomainRulesets, "required_approvals", "1"); err != nil {
		t.Fatalf("applyEdit() error = %v", err)
	}
	if *cfg.Ruleset.PullRequest.RequiredApprovals != 1 {
		t.Errorf("RequiredApprovals = %d, want 1", *cfg.Ruleset.PullRequest.RequiredApprovals)
	}
}

func TestApplyEditRulesetPullRequestFieldNilBlock(t *testing.T) {
	t.Parallel()

	cfg := &config.PolicyConfig{Ruleset: config.PolicyRuleset{Name: "main"}}

	if err := applyEdit(cfg, policy.DomainRulesets, "required_approvals", "1"); !errors.Is(err, ErrUnknownField) {
		t.Errorf("applyEdit() error = %v, want ErrUnknownField for a nil pull_request block", err)
	}
}

func TestApplyEditUnknownField(t *testing.T) {
	t.Parallel()

	cfg := &config.PolicyConfig{}

	if err := applyEdit(cfg, policy.DomainSettings, "not_a_real_field", "true"); !errors.Is(err, ErrUnknownField) {
		t.Errorf("applyEdit() error = %v, want ErrUnknownField", err)
	}

	if err := applyEdit(cfg, policy.Domain("bogus"), "x", "y"); !errors.Is(err, ErrUnknownField) {
		t.Errorf("applyEdit() error = %v, want ErrUnknownField for an unknown domain", err)
	}
}

func boolPtr(b bool) *bool { return &b }
func intPtr(n int) *int    { return &n }
