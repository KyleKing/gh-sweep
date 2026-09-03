package config_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/KyleKing/gh-sweep/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()

	if cfg.Cache.TTL != "1h" {
		t.Errorf("Expected TTL to be '1h', got '%s'", cfg.Cache.TTL)
	}

	if len(cfg.Filters.ExcludeUsers) == 0 {
		t.Error("Expected default exclude users to be populated")
	}
}

func TestLoadConfig(t *testing.T) { //nolint:paralleltest // t.Chdir/t.Setenv forbid t.Parallel()
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gh-sweep.yaml")

	configContent := `
default_org: test-org
repositories:
  - owner/repo1
  - owner/repo2
cache:
  ttl: 2h
  path: /tmp/cache
`

	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	t.Chdir(tmpDir)

	// Load config
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.DefaultOrg != "test-org" {
		t.Errorf("Expected default_org to be 'test-org', got '%s'", cfg.DefaultOrg)
	}

	if len(cfg.Repositories) != 2 {
		t.Errorf("Expected 2 repositories, got %d", len(cfg.Repositories))
	}

	if cfg.Cache.TTL != "2h" {
		t.Errorf("Expected TTL to be '2h', got '%s'", cfg.Cache.TTL)
	}
}

func TestLoadMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	t.Setenv("HOME", tmpDir)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Expected no error for missing config file, got: %v", err)
	}

	if cfg.Cache.TTL != "1h" {
		t.Errorf("Expected default TTL '1h', got '%s'", cfg.Cache.TTL)
	}

	if cfg.DefaultOrg != "" {
		t.Errorf("Expected empty default_org, got '%s'", cfg.DefaultOrg)
	}
}

func TestLoadProjectOverridesHome(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()

	projectConfig := "default_org: project-org\n"
	if err := os.WriteFile(
		filepath.Join(projectDir, ".gh-sweep.yaml"),
		[]byte(projectConfig),
		0o600,
	); err != nil {
		t.Fatalf("Failed to write project config: %v", err)
	}

	homeConfig := "default_org: home-org\n"
	if err := os.WriteFile(
		filepath.Join(homeDir, ".gh-sweep.yaml"),
		[]byte(homeConfig),
		0o600,
	); err != nil {
		t.Fatalf("Failed to write home config: %v", err)
	}

	t.Chdir(projectDir)
	t.Setenv("HOME", homeDir)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.DefaultOrg != "project-org" {
		t.Errorf("Expected project config to win, got '%s'", cfg.DefaultOrg)
	}
}

func TestLoadHomeFallback(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()

	homeConfig := "default_org: home-org\n"
	if err := os.WriteFile(
		filepath.Join(homeDir, ".gh-sweep.yaml"),
		[]byte(homeConfig),
		0o600,
	); err != nil {
		t.Fatalf("Failed to write home config: %v", err)
	}

	t.Chdir(projectDir)
	t.Setenv("HOME", homeDir)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.DefaultOrg != "home-org" {
		t.Errorf("Expected home config to be used, got '%s'", cfg.DefaultOrg)
	}
}

func TestExampleFileMatchesStruct(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join("..", "..", ".gh-sweep.yaml.example"))
	if err != nil {
		t.Fatalf("Failed to read example config: %v", err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	var cfg config.Config
	if err := decoder.Decode(&cfg); err != nil {
		t.Fatalf("Example config contains fields unknown to Config struct: %v", err)
	}

	if cfg.DefaultOrg != "your-org" {
		t.Errorf("Expected default_org 'your-org', got '%s'", cfg.DefaultOrg)
	}

	if cfg.GHAPerf.DefaultLookbackDays != 30 {
		t.Errorf(
			"Expected gha_perf.default_lookback_days 30, got %d",
			cfg.GHAPerf.DefaultLookbackDays,
		)
	}

	if cfg.Orphans.StaleDaysThreshold != 21 {
		t.Errorf("Expected orphans.stale_days_threshold 21, got %d", cfg.Orphans.StaleDaysThreshold)
	}
}

func TestQualifiedRepos(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		DefaultOrg:   "acme",
		Repositories: []string{"owner/repo1", "repo2"},
	}

	repos := cfg.QualifiedRepos()

	if len(repos) != 2 {
		t.Fatalf("Expected 2 repos, got %d", len(repos))
	}

	if repos[0] != "owner/repo1" {
		t.Errorf("Expected 'owner/repo1', got '%s'", repos[0])
	}

	if repos[1] != "acme/repo2" {
		t.Errorf("Expected 'acme/repo2', got '%s'", repos[1])
	}
}

func TestSaveConfig(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := config.DefaultConfig()
	cfg.DefaultOrg = "test-org"

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Load it back
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read saved config: %v", err)
	}

	// Should contain default_org
	content := string(data)
	if content == "" {
		t.Error("Saved config is empty")
	}
}
