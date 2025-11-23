package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "gh-sweep",
	Short: "A powerful TUI for GitHub repository management",
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
		// Launch full TUI if no subcommand specified
		fmt.Println("gh-sweep TUI - Coming soon!")
		fmt.Println("\nAvailable commands:")
		fmt.Println("  gh-sweep branches    - Interactive branch management")
		fmt.Println("  gh-sweep protection  - Branch protection rules")
		fmt.Println("  gh-sweep comments    - Unresolved PR comments")
		fmt.Println("\nUse 'gh-sweep <command> --help' for more information")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)
}
