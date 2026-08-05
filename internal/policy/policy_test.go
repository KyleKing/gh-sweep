package policy_test

import (
	"testing"

	"github.com/KyleKing/gh-sweep/internal/config"
	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/policy"
)

func boolPtr(b bool) *bool { return &b }
func intPtr(n int) *int    { return &n }

func TestDiffSettings(t *testing.T) {
	t.Parallel()

	have := &github.RepoSettings{DeleteBranchOnMerge: false, HasWiki: true}

	tests := []struct {
		name string
		want config.PolicySettings
		n    int
	}{
		{name: "unmanaged field never diffs", want: config.PolicySettings{}, n: 0},
		{
			name: "single mismatch",
			want: config.PolicySettings{DeleteBranchOnMerge: boolPtr(true)},
			n:    1,
		},
		{
			name: "managed but matching does not diff",
			want: config.PolicySettings{HasWiki: boolPtr(true)},
			n:    0,
		},
		{
			name: "multiple mismatches",
			want: config.PolicySettings{DeleteBranchOnMerge: boolPtr(true), HasWiki: boolPtr(false)},
			n:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			diffs := policy.DiffSettings(&tt.want, have)
			if len(diffs) != tt.n {
				t.Errorf("DiffSettings() = %d diffs, want %d: %+v", len(diffs), tt.n, diffs)
			}
			for _, d := range diffs {
				if d.Domain != policy.DomainSettings {
					t.Errorf("diff domain = %q, want %q", d.Domain, policy.DomainSettings)
				}
			}
		})
	}
}

func TestDiffSecurity(t *testing.T) {
	t.Parallel()

	have := &github.SecurityAndAnalysis{SecretScanning: "disabled"}

	diffs := policy.DiffSecurity(&config.PolicySecurity{SecretScanning: "enabled"}, have)
	if len(diffs) != 1 || diffs[0].Field != "secret_scanning" {
		t.Fatalf("diffs = %+v, want one secret_scanning diff", diffs)
	}

	diffs = policy.DiffSecurity(&config.PolicySecurity{}, have)
	if len(diffs) != 0 {
		t.Errorf("diffs = %+v, want none for unmanaged policy", diffs)
	}
}

func TestDiffReleases(t *testing.T) {
	t.Parallel()

	if diffs := policy.DiffReleases(true, true); len(diffs) != 0 {
		t.Errorf("DiffReleases(true, true) = %+v, want none", diffs)
	}

	diffs := policy.DiffReleases(true, false)
	if len(diffs) != 1 || diffs[0].Domain != policy.DomainReleases {
		t.Errorf("DiffReleases(true, false) = %+v", diffs)
	}
}

func TestDiffProtection(t *testing.T) {
	t.Parallel()

	have := &github.ProtectionRule{RequiredReviews: 0, RequireStatusChecks: []string{"ci"}}

	diffs := policy.DiffProtection(&config.PolicyProtection{RequiredReviews: intPtr(2)}, have)
	if len(diffs) != 1 || diffs[0].Field != "required_reviews" {
		t.Fatalf("diffs = %+v", diffs)
	}

	diffs = policy.DiffProtection(&config.PolicyProtection{RequireStatusChecks: []string{"ci", "lint"}}, have)
	if len(diffs) != 1 || diffs[0].Field != "require_status_checks" {
		t.Fatalf("diffs = %+v", diffs)
	}

	diffs = policy.DiffProtection(&config.PolicyProtection{RequireStatusChecks: []string{"ci"}}, have)
	if len(diffs) != 0 {
		t.Errorf("reordered-but-equal status checks diffed: %+v", diffs)
	}
}

func TestReportHasDrift(t *testing.T) {
	t.Parallel()

	clean := &policy.Report{Repos: []policy.RepoDrift{{Repository: "acme/widgets"}}}
	if clean.HasDrift() {
		t.Error("HasDrift() = true for a repo with no diffs")
	}

	dirty := &policy.Report{Repos: []policy.RepoDrift{
		{Repository: "acme/widgets"},
		{
			Repository: "acme/gadgets",
			Diffs:      []policy.Diff{{Domain: policy.DomainSettings, Field: "has_wiki"}},
		},
	}}
	if !dirty.HasDrift() {
		t.Error("HasDrift() = false for a repo with a diff")
	}
}
