// Package cli implements gh-sweep's Cobra command tree: flag parsing, config
// resolution, and dispatch into the TUI or CLI list mode for each subcommand.
package cli

import (
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/kyleking/aragonite/keyusage"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/KyleKing/gh-sweep/internal/config"
	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/policy"
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
		opts.PolicyPath = stringFlag(cmd, "policy")
		opts.ApplyOpts = policy.ApplyOptions{PruneBranches: boolFlag(cmd, "prune")}

		theme.Init(theme.Detect())

		opts.Usage = usageRecorder
		m := tui.NewMainModel(opts)
		p := tea.NewProgram(m)

		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
			usageRecorder.Close() //nolint:errcheck,gosec // a failed close loses only the last log line
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
		ScanOptions:         cfg.ScanOptions(),
		RegressionThreshold: cfg.GHAPerf.RegressionThreshold,
	}
}

//nolint:gochecknoglobals // opened once per process in Execute and read by the root command
var usageRecorder *keyusage.Recorder

// Execute runs the root command with the given build version string.
func Execute(version string) {
	rootCmd.Version = version

	usageRecorder = keyusage.Open(usageApp)
	usageRecorder.Invoke(CommandTree(), os.Args[1:])

	err := rootCmd.Execute()

	usageRecorder.Close() //nolint:errcheck,gosec // a failed close loses only the last log line

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// CommandTree is the cobra command tree as a keyusage root.
func CommandTree() keyusage.Command { return commandOf(rootCmd) }

func commandOf(cmd *cobra.Command) keyusage.Command {
	cmd.InitDefaultHelpFlag()
	cmd.InitDefaultVersionFlag()

	out := keyusage.Command{Name: cmd.Name(), What: cmd.Short}

	for _, set := range []*pflag.FlagSet{cmd.LocalFlags(), cmd.InheritedFlags()} {
		set.VisitAll(func(f *pflag.Flag) {
			if f.Hidden {
				return
			}

			out.Flags = append(out.Flags, "--"+f.Name)

			if f.Shorthand != "" {
				out.Flags = append(out.Flags, "-"+f.Shorthand)
			}
		})
	}

	for _, sub := range cmd.Commands() {
		if !sub.Hidden {
			out.Subs = append(out.Subs, commandOf(sub))
		}
	}

	return out
}

//nolint:gochecknoglobals // bound to a persistent flag, read by loadConfig for every subcommand
var configPath string

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "",
		"Path to the config file (default: search ./.gh-sweep.yaml, ~/.gh-sweep.yaml, ~/.config/gh-sweep/)")
	rootCmd.PersistentFlags().
		String("policy", "", "Path to the policy file (default: .gh-sweep-policy.yaml)")
	rootCmd.PersistentFlags().
		Bool("prune", false, "Let a policy apply delete the branches its branches block reports")
	rootCmd.Flags().String("repo", "", "Repository (owner/repo)")
	rootCmd.PersistentFlags().
		String("org", "", "GitHub organization (overrides config default_org)")
	rootCmd.PersistentFlags().
		StringSlice("repos", nil, "Repositories to manage, owner/repo (overrides config repositories)")
}
