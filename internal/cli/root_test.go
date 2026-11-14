package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kyleking/aragonite/keyusage"

	"github.com/KyleKing/gh-sweep/internal/config"
)

func TestRootCmd(t *testing.T) {
	t.Parallel()

	if rootCmd == nil {
		t.Fatal("rootCmd is nil")
	}

	if rootCmd.Use != "gh-sweep" {
		t.Errorf("Expected Use to be 'gh-sweep', got '%s'", rootCmd.Use)
	}
}

func TestResolveMainOptionsFromConfig(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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

func TestInvocationLogsFlagNamesNotValues(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	rec := keyusage.Open(usageApp, keyusage.WithDir(dir))
	rec.Invoke(CommandTree(), []string{"protection", "--org", "secret-org", "-h"})

	if err := rec.Close(); err != nil {
		t.Fatal(err)
	}

	u, err := keyusage.Load(dir, keyusage.Keymap{keyusage.CLI: keyusage.Commands(CommandTree())})
	if err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"gh-sweep protection", "gh-sweep protection --org", "gh-sweep protection -h"} {
		if got := u.Stat(keyusage.CLI, key).Count; got != 1 {
			t.Errorf("%q logged %d times, want 1", key, got)
		}
	}

	logs, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if err != nil || len(logs) != 1 {
		t.Fatalf("log files = %v, err = %v", logs, err)
	}

	raw, err := os.ReadFile(logs[0])
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(raw), "secret-org") {
		t.Errorf("flag value was logged: %s", raw)
	}
}
