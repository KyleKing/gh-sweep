package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPolicy(t *testing.T) {
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
	if err := os.WriteFile(policyPath, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	cfg, err := LoadPolicy(policyPath)
	if err != nil {
		t.Fatalf("LoadPolicy() error = %v", err)
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

func TestLoadPolicyMissing(t *testing.T) {
	if _, err := LoadPolicy(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Error("LoadPolicy() error = nil, want error for missing file")
	}
}

func TestPolicyProtectionUnmanaged(t *testing.T) {
	if (PolicyProtection{}).Managed() {
		t.Error("Managed() = true for zero-value PolicyProtection, want false")
	}
}
