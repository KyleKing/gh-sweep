package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	DefaultOrg   string        `yaml:"default_org"`
	Repositories []string      `yaml:"repositories"`
	Cache        CacheConfig   `yaml:"cache"`
	GitHub       GitHubConfig  `yaml:"github"`
	Filters      FilterConfig  `yaml:"filters"`
	Branches     BranchConfig  `yaml:"branches"`
	Comments     CommentConfig `yaml:"comments"`
	GHAPerf      GHAPerfConfig `yaml:"gha_perf"`
	Orphans      OrphansConfig `yaml:"orphans"`
	UI           UIConfig      `yaml:"ui"`
}

// CacheConfig represents cache settings
type CacheConfig struct {
	TTL  string `yaml:"ttl"`
	Path string `yaml:"path"`
}

// GitHubConfig represents GitHub API settings
type GitHubConfig struct {
	Token  string `yaml:"token"`
	APIURL string `yaml:"api_url"`
}

// FilterConfig represents filter settings
type FilterConfig struct {
	ExcludeUsers []string `yaml:"exclude_users"`
	ExcludeRepos []string `yaml:"exclude_repos"`
}

// BranchConfig represents branch management settings
type BranchConfig struct {
	DefaultBranch     string   `yaml:"default_branch"`
	ProtectedPatterns []string `yaml:"protected_patterns"`
}

// CommentConfig represents comment review settings
type CommentConfig struct {
	DefaultSinceDays int     `yaml:"default_since_days"`
	FuzzyThreshold   float64 `yaml:"fuzzy_threshold"`
}

// GHAPerfConfig represents GHA performance analysis settings
type GHAPerfConfig struct {
	DefaultLookbackDays int      `yaml:"default_lookback_days"`
	BaseBranch          string   `yaml:"base_branch"`
	DefaultWorkflows    []string `yaml:"default_workflows"`
	CachePath           string   `yaml:"cache_path"`
	RegressionThreshold float64  `yaml:"regression_threshold"`
}

// OrphansConfig represents orphan branch detection settings
type OrphansConfig struct {
	StaleDaysThreshold int      `yaml:"stale_days_threshold"`
	ExcludePatterns    []string `yaml:"exclude_patterns"`
	DefaultConcurrency int      `yaml:"default_concurrency"`
}

// UIConfig represents UI preferences
type UIConfig struct {
	Theme   string `yaml:"theme"`
	Icons   bool   `yaml:"icons"`
	Compact bool   `yaml:"compact"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
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
			DefaultBranch: "main",
			ProtectedPatterns: []string{
				"main",
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
			BaseBranch:          "main",
			DefaultWorkflows:    []string{},
			CachePath:           filepath.Join(homeDir, ".cache", "gh-sweep", "gha-perf"),
			RegressionThreshold: 20.0,
		},
		Orphans: OrphansConfig{
			StaleDaysThreshold: 7,
			ExcludePatterns: []string{
				"main",
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

// Load loads configuration from file, falling back to defaults
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Try to load from multiple locations
	configPaths := []string{
		".gh-sweep.yaml",
		filepath.Join(os.Getenv("HOME"), ".gh-sweep.yaml"),
		filepath.Join(os.Getenv("HOME"), ".config", "gh-sweep", "config.yaml"),
	}

	var configData []byte
	var err error
	var foundPath string

	for _, path := range configPaths {
		configData, err = os.ReadFile(path)
		if err == nil {
			foundPath = path
			break
		}
	}

	// If no config file found, return defaults
	if foundPath == "" {
		return cfg, nil
	}

	// Parse YAML
	if err := yaml.Unmarshal(configData, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config from %s: %w", foundPath, err)
	}

	// Expand cache path if needed
	if cfg.Cache.Path == "" {
		homeDir, _ := os.UserHomeDir()
		cfg.Cache.Path = filepath.Join(homeDir, ".cache", "gh-sweep")
	}

	return cfg, nil
}

// Save saves the configuration to a file
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
