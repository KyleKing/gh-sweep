package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Cache.TTL != "1h" {
		t.Errorf("Expected TTL to be '1h', got '%s'", cfg.Cache.TTL)
	}

	if cfg.Branches.DefaultBranch != "main" {
		t.Errorf("Expected default branch to be 'main', got '%s'", cfg.Branches.DefaultBranch)
	}

	if len(cfg.Filters.ExcludeUsers) == 0 {
		t.Error("Expected default exclude users to be populated")
	}
}

func TestLoadConfig(t *testing.T) {
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

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Change to temp directory
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	// Load config
	cfg, err := Load()
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

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := DefaultConfig()
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
	if len(content) == 0 {
		t.Error("Saved config is empty")
	}
}
