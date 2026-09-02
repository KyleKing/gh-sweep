// Package cli implements gh-sweep's Cobra command tree: flag parsing, config
// resolution, and dispatch into the TUI or CLI list mode for each subcommand.
package cli

import (
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/KyleKing/gh-sweep/internal/config"
	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/tui"
	"github.com/KyleKing/gh-sweep/internal/tui/theme"
)

var rootCmd = &cobra.Command{
	Use:   "gh-sweep",
	Short: "TUI for sweeping GitHub repositories",
	// Every subcommand builds its own clients, so the cache is installed here
	// rather than in loadConfig, which only some of them call.
	PersistentPreRun: func(_ *cobra.Command, _ []string) { installCache(loadConfig()) },
	Long: `gh-sweep is a Terminal User Interface (TUI) for managing multiple GitHub repositories.

It provides interactive tools for:
  - Branch management with dependency visualization
  - Branch protection rule comparison and sync
  - Unresolved PR comment review and filtering
  - Cross-repo settings comparison
  - GitHub Actions analytics
  - And much more...

Use 'gh-sweep <command> --help' for more information about a command.`,
	Run: func(cmd *cobra.Command, _ []string) {
		cfg := loadConfig()
		opts := resolveMainOptions(
			cfg,
			stringFlag(cmd, "repo"),
			stringFlag(cmd, "org"),
			stringSliceFlag(cmd, "repos"),
		)

		theme.Init(theme.Detect())

		m := tui.NewMainModel(opts)
		p := tea.NewProgram(m)

		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
			os.Exit(1)
		}
	},
}

func loadConfig() *config.Config {
	cfg, err := config.LoadFrom(configPath)
	if err != nil {
		if configPath != "" {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "Warning: failed to load config, using defaults: %v\n", err)

		return config.DefaultConfig()
	}

	return cfg
}

// installCache wires the config's cache settings into every client built
// afterward. Cached responses cost no rate-limit quota, which is what keeps a
// cross-repo sweep inside the hourly budget.
func installCache(cfg *config.Config) {
	ttl, err := time.ParseDuration(cfg.Cache.TTL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: ignoring unparsable cache.ttl %q: %v\n", cfg.Cache.TTL, err)

		return
	}

	github.SetDefaultCache(cfg.Cache.Path, ttl)
}

func resolveMainOptions(cfg *config.Config, repo, org string, repos []string) tui.MainModelOptions {
	if org == "" {
		org = cfg.DefaultOrg
	}
	if len(repos) == 0 {
		repos = cfg.QualifiedRepos()
	}
	if repo == "" && len(repos) > 0 {
		repo = repos[0]
	}

	return tui.MainModelOptions{
		Baseline:            cfg.Baseline,
		Org:                 org,
		Repo:                repo,
		Repos:               repos,
		RegressionThreshold: cfg.GHAPerf.RegressionThreshold,
	}
}

// Execute runs the root command with the given build version string.
func Execute(version string) {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

//nolint:gochecknoglobals // bound to a persistent flag, read by loadConfig for every subcommand
var configPath string

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "",
		"Path to the config file (default: search ./.gh-sweep.yaml, ~/.gh-sweep.yaml, ~/.config/gh-sweep/)")
	rootCmd.Flags().String("repo", "", "Repository (owner/repo)")
	rootCmd.PersistentFlags().
		String("org", "", "GitHub organization (overrides config default_org)")
	rootCmd.PersistentFlags().
		StringSlice("repos", nil, "Repositories to manage, owner/repo (overrides config repositories)")
}
