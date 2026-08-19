package cli

import (
	"testing"

	"github.com/KyleKing/gh-sweep/internal/config"
)

func TestRootCmd(t *testing.T) {
	if rootCmd == nil {
		t.Fatal("rootCmd is nil")
	}

	if rootCmd.Use != "gh-sweep" {
		t.Errorf("Expected Use to be 'gh-sweep', got '%s'", rootCmd.Use)
	}
}

func TestResolveMainOptionsFromConfig(t *testing.T) {
	cfg := &config.Config{
		Baseline:     "acme/baseline",
		DefaultOrg:   "acme",
		Repositories: []string{"acme/repo1", "repo2"},
		GHAPerf:      config.GHAPerfConfig{RegressionThreshold: 25.0},
	}

	opts := resolveMainOptions(cfg, "", "", nil)

	if opts.RegressionThreshold != 25.0 {
		t.Errorf("Expected regression threshold from config, got %v", opts.RegressionThreshold)
	}

	if opts.Org != "acme" {
		t.Errorf("Expected org from config, got '%s'", opts.Org)
	}

	if len(opts.Repos) != 2 || opts.Repos[1] != "acme/repo2" {
		t.Errorf("Expected qualified repos from config, got %v", opts.Repos)
	}

	if opts.Repo != "acme/repo1" {
		t.Errorf("Expected repo to default to first config repo, got '%s'", opts.Repo)
	}

	if opts.Baseline != "acme/baseline" {
		t.Errorf("Expected baseline from config, got '%s'", opts.Baseline)
	}
}

func TestResolveMainOptionsFlagsOverrideConfig(t *testing.T) {
	cfg := &config.Config{
		DefaultOrg:   "config-org",
		Repositories: []string{"config-org/repo1"},
	}

	opts := resolveMainOptions(
		cfg,
		"flag-org/flag-repo",
		"flag-org",
		[]string{"flag-org/flag-repo"},
	)

	if opts.Org != "flag-org" {
		t.Errorf("Expected flag org to win, got '%s'", opts.Org)
	}

	if len(opts.Repos) != 1 || opts.Repos[0] != "flag-org/flag-repo" {
		t.Errorf("Expected flag repos to win, got %v", opts.Repos)
	}

	if opts.Repo != "flag-org/flag-repo" {
		t.Errorf("Expected flag repo to win, got '%s'", opts.Repo)
	}
}
