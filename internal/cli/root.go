package cli

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/KyleKing/gh-sweep/internal/config"
	"github.com/KyleKing/gh-sweep/internal/tui"
	"github.com/KyleKing/gh-sweep/internal/tui/theme"
)

var rootCmd = &cobra.Command{
	Use:   "gh-sweep",
	Short: "TUI for sweeping GitHub repositories",
	Long: `gh-sweep is a Terminal User Interface (TUI) for managing multiple GitHub repositories.

It provides interactive tools for:
  - Branch management with dependency visualization
  - Branch protection rule comparison and sync
  - Unresolved PR comment review and filtering
  - Cross-repo settings comparison
  - GitHub Actions analytics
  - And much more...

Use 'gh-sweep <command> --help' for more information about a command.`,
	Run: func(cmd *cobra.Command, args []string) {
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
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config, using defaults: %v\n", err)
		return config.DefaultConfig()
	}

	return cfg
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

func init() {
	rootCmd.Flags().String("repo", "", "Repository (owner/repo)")
	rootCmd.PersistentFlags().
		String("org", "", "GitHub organization (overrides config default_org)")
	rootCmd.PersistentFlags().
		StringSlice("repos", nil, "Repositories to manage, owner/repo (overrides config repositories)")
}
