package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/KyleKing/gh-sweep/internal/config"
)

func TestLoadPolicy(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	policyPath := filepath.Join(tmpDir, ".gh-sweep-policy.yaml")

	content := `
default_org: acme
repositories:
  - widgets
  - other/gadgets
settings:
  delete_branch_on_merge: true
security:
  secret_scanning: enabled
releases:
  immutable: true
protection:
  required_reviews: 1
`
	if err := os.WriteFile(policyPath, []byte(content), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	cfg, err := config.LoadPolicy(policyPath)
	if err != nil {
		t.Fatalf("config.LoadPolicy() error = %v", err)
	}

	if cfg.DefaultOrg != "acme" {
		t.Errorf("DefaultOrg = %q, want acme", cfg.DefaultOrg)
	}

	if cfg.Settings.DeleteBranchOnMerge == nil || !*cfg.Settings.DeleteBranchOnMerge {
		t.Errorf("Settings.DeleteBranchOnMerge = %v, want true", cfg.Settings.DeleteBranchOnMerge)
	}

	if cfg.Security.SecretScanning != "enabled" {
		t.Errorf("Security.SecretScanning = %q, want enabled", cfg.Security.SecretScanning)
	}

	if cfg.Releases.Immutable == nil || !*cfg.Releases.Immutable {
		t.Errorf("Releases.Immutable = %v, want true", cfg.Releases.Immutable)
	}

	if !cfg.Protection.Managed() {
		t.Error("Protection.Managed() = false, want true")
	}

	want := []string{"acme/widgets", "other/gadgets"}
	got := cfg.QualifiedRepos()
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("QualifiedRepos() = %v, want %v", got, want)
	}
}

func TestSavePolicyRoundTrip(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), ".gh-sweep-policy.yaml")

	deleteOnMerge := true
	cfg := &config.PolicyConfig{
		DefaultOrg:   "acme",
		Repositories: []string{"widgets"},
		Settings:     config.PolicySettings{DeleteBranchOnMerge: &deleteOnMerge},
	}

	if err := cfg.SavePolicy(path); err != nil {
		t.Fatalf("SavePolicy() error = %v", err)
	}

	loaded, err := config.LoadPolicy(path)
	if err != nil {
		t.Fatalf("config.LoadPolicy() error = %v", err)
	}

	if loaded.DefaultOrg != "acme" {
		t.Errorf("DefaultOrg = %q, want acme", loaded.DefaultOrg)
	}
	if loaded.Settings.DeleteBranchOnMerge == nil || !*loaded.Settings.DeleteBranchOnMerge {
		t.Errorf("Settings.DeleteBranchOnMerge = %v, want true", loaded.Settings.DeleteBranchOnMerge)
	}
}

func TestSavePolicyInvalidPath(t *testing.T) {
	t.Parallel()

	cfg := &config.PolicyConfig{DefaultOrg: "acme"}

	if err := cfg.SavePolicy(filepath.Join(t.TempDir(), "does-not-exist", "policy.yaml")); err == nil {
		t.Error("SavePolicy() error = nil, want an error for a missing parent directory")
	}
}

func TestLoadPolicyMissing(t *testing.T) {
	t.Parallel()

	if _, err := config.LoadPolicy(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Error("config.LoadPolicy() error = nil, want error for missing file")
	}
}

func TestPolicyProtectionUnmanaged(t *testing.T) {
	t.Parallel()

	if (config.PolicyProtection{}).Managed() {
		t.Error("Managed() = true for zero-value PolicyProtection, want false")
	}
}

func TestForRepo(t *testing.T) {
	t.Parallel()

	baseReviews := 2
	overrideReviews := 0
	baseEnforce := true

	base := &config.PolicyConfig{
		DefaultOrg: "acme",
		Protection: config.PolicyProtection{
			RequiredReviews: &baseReviews,
			EnforceAdmins:   &baseEnforce,
		},
		Security: config.PolicySecurity{SecretScanning: "enabled"},
		Ruleset:  config.PolicyRuleset{Name: "main", Enforcement: "active"},
		Overrides: map[string]config.PolicyOverride{
			"acme/widgets": {
				Protection: config.PolicyProtection{RequiredReviews: &overrideReviews},
				Ruleset:    config.PolicyRuleset{Name: "main", Enforcement: "evaluate"},
			},
			"gadgets": {
				Security: config.PolicySecurity{SecretScanning: "disabled"},
			},
		},
	}

	t.Run("qualified override merges field by field", func(t *testing.T) {
		t.Parallel()

		resolved := base.ForRepo("acme/widgets")

		if resolved.Protection.RequiredReviews == nil || *resolved.Protection.RequiredReviews != 0 {
			t.Errorf("RequiredReviews = %v, want 0 (from override)", resolved.Protection.RequiredReviews)
		}
		if resolved.Protection.EnforceAdmins != &baseEnforce {
			t.Errorf(
				"EnforceAdmins = %v, want base's pointer (untouched by override)",
				resolved.Protection.EnforceAdmins,
			)
		}
		if resolved.Ruleset.Enforcement != "evaluate" {
			t.Errorf(
				"Ruleset.Enforcement = %q, want evaluate (ruleset replaces wholesale)",
				resolved.Ruleset.Enforcement,
			)
		}
	})

	t.Run("bare-name override resolves against DefaultOrg", func(t *testing.T) {
		t.Parallel()

		resolved := base.ForRepo("acme/gadgets")

		if resolved.Security.SecretScanning != "disabled" {
			t.Errorf("SecretScanning = %q, want disabled (from bare-keyed override)", resolved.Security.SecretScanning)
		}
	})

	t.Run("repo with no override returns base unchanged", func(t *testing.T) {
		t.Parallel()

		resolved := base.ForRepo("acme/untouched")

		if resolved.Protection.RequiredReviews != &baseReviews {
			t.Errorf("RequiredReviews = %v, want base's pointer", resolved.Protection.RequiredReviews)
		}
	})
}
