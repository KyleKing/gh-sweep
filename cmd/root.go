package cmd

import (
	"fmt"
	"os"

	"github.com/KyleKing/gh-sweep/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
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
		repo, _ := cmd.Flags().GetString("repo")

		// Launch full interactive TUI
		m := tui.NewMainModel(repo)
		p := tea.NewProgram(m, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
			os.Exit(1)
		}
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
	rootCmd.Flags().String("repo", "", "Repository (owner/repo)")
}
