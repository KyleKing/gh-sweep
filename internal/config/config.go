// Package config loads gh-sweep's flag-default config (.gh-sweep.yaml) and
// its declarative policy config (.gh-sweep-policy.yaml).
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const defaultBranchName = "main"

// Config represents the application configuration.
type Config struct {
	Baseline     string        `yaml:"baseline"`
	DefaultOrg   string        `yaml:"default_org"`
	Repositories []string      `yaml:"repositories"`
	Cache        CacheConfig   `yaml:"cache"`
	GitHub       GitHubConfig  `yaml:"github"`
	Filters      FilterConfig  `yaml:"filters"`
	Branches     BranchConfig  `yaml:"branches"`
	Comments     CommentConfig `yaml:"comments"`
	GHAPerf      GHAPerfConfig `yaml:"gha_perf"`
	Orphans      OrphansConfig `yaml:"orphans"`
	Pages        PagesConfig   `yaml:"pages"`
	UI           UIConfig      `yaml:"ui"`
}

// CacheConfig represents cache settings.
type CacheConfig struct {
	TTL  string `yaml:"ttl"`
	Path string `yaml:"path"`
}

// GitHubConfig represents GitHub API settings.
type GitHubConfig struct {
	Token  string `yaml:"token"`
	APIURL string `yaml:"api_url"`
}

// FilterConfig represents filter settings.
type FilterConfig struct {
	ExcludeUsers []string `yaml:"exclude_users"`
	ExcludeRepos []string `yaml:"exclude_repos"`
}

// BranchConfig represents branch management settings.
type BranchConfig struct {
	DefaultBranch     string   `yaml:"default_branch"`
	ProtectedPatterns []string `yaml:"protected_patterns"`
}

// CommentConfig represents comment review settings.
type CommentConfig struct {
	DefaultSinceDays int     `yaml:"default_since_days"`
	FuzzyThreshold   float64 `yaml:"fuzzy_threshold"`
}

// GHAPerfConfig represents GHA performance analysis settings.
type GHAPerfConfig struct {
	DefaultLookbackDays int      `yaml:"default_lookback_days"`
	BaseBranch          string   `yaml:"base_branch"`
	DefaultWorkflows    []string `yaml:"default_workflows"`
	CachePath           string   `yaml:"cache_path"`
	RegressionThreshold float64  `yaml:"regression_threshold"`
}

// OrphansConfig represents orphan branch detection settings.
type OrphansConfig struct {
	StaleDaysThreshold int      `yaml:"stale_days_threshold"`
	ExcludePatterns    []string `yaml:"exclude_patterns"`
	DefaultConcurrency int      `yaml:"default_concurrency"`
}

// PagesConfig represents GitHub Pages domain audit settings.
type PagesConfig struct {
	// Domains reverse-checks these DNS-configured subdomains against the
	// scanned repos' Pages configuration: each should have a live Pages
	// site backing it.
	Domains []string `yaml:"domains"`
}

// UIConfig represents UI preferences.
type UIConfig struct {
	Theme   string `yaml:"theme"`
	Icons   bool   `yaml:"icons"`
	Compact bool   `yaml:"compact"`
}

// DefaultConfig returns a configuration with sensible defaults.
//
//nolint:mnd // every literal here is a self-documenting default value, named by its field
func DefaultConfig() *Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	return &Config{
		Cache: CacheConfig{
			TTL:  "1h",
			Path: filepath.Join(homeDir, ".cache", "gh-sweep"),
		},
		Filters: FilterConfig{
			ExcludeUsers: []string{
				"dependabot[bot]",
				"renovate[bot]",
				"github-actions[bot]",
			},
		},
		Branches: BranchConfig{
			DefaultBranch: defaultBranchName,
			ProtectedPatterns: []string{
				defaultBranchName,
				"master",
				"develop",
			},
		},
		Comments: CommentConfig{
			DefaultSinceDays: 30,
			FuzzyThreshold:   0.7,
		},
		GHAPerf: GHAPerfConfig{
			DefaultLookbackDays: 30,
			BaseBranch:          defaultBranchName,
			DefaultWorkflows:    []string{},
			CachePath:           filepath.Join(homeDir, ".cache", "gh-sweep", "gha-perf"),
			RegressionThreshold: 20.0,
		},
		Orphans: OrphansConfig{
			StaleDaysThreshold: 30,
			ExcludePatterns: []string{
				defaultBranchName,
				"master",
				"develop",
				"release/*",
				"hotfix/*",
			},
			DefaultConcurrency: 5,
		},
		UI: UIConfig{
			Theme:   "auto",
			Icons:   true,
			Compact: false,
		},
	}
}

// QualifiedRepos returns Repositories with bare names prefixed by DefaultOrg.
func (c *Config) QualifiedRepos() []string {
	repos := make([]string, 0, len(c.Repositories))
	for _, repo := range c.Repositories {
		if !strings.Contains(repo, "/") && c.DefaultOrg != "" {
			repo = c.DefaultOrg + "/" + repo
		}
		repos = append(repos, repo)
	}

	return repos
}

// Load loads configuration from the first of the searched paths that exists,
// falling back to defaults.
func Load() (*Config, error) {
	return LoadFrom("")
}

// LoadFrom loads configuration from path. An empty path searches
// ./.gh-sweep.yaml, ~/.gh-sweep.yaml, then ~/.config/gh-sweep/config.yaml and
// falls back to defaults; an explicit path that cannot be read is an error, so
// a typo never silently audits with the wrong settings.
func LoadFrom(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path != "" {
		configData, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading config %s: %w", path, err)
		}

		return parseConfig(cfg, configData, path)
	}

	configPaths := []string{
		".gh-sweep.yaml",
		filepath.Join(os.Getenv("HOME"), ".gh-sweep.yaml"),
		filepath.Join(os.Getenv("HOME"), ".config", "gh-sweep", "config.yaml"),
	}

	var configData []byte
	var foundPath string

	for _, candidate := range configPaths {
		data, err := os.ReadFile(candidate)
		if err == nil {
			configData = data
			foundPath = candidate

			break
		}
	}

	if foundPath == "" {
		return cfg, nil
	}

	return parseConfig(cfg, configData, foundPath)
}

func parseConfig(cfg *Config, configData []byte, foundPath string) (*Config, error) {
	if err := yaml.Unmarshal(configData, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config from %s: %w", foundPath, err)
	}

	if cfg.Cache.Path == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = "."
		}
		cfg.Cache.Path = filepath.Join(homeDir, ".cache", "gh-sweep")
	}

	return cfg, nil
}

// Save saves the configuration to a file.
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create directory if needed. The config can hold GitHub.Token, so keep
	// both the directory and the file owner-only.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
