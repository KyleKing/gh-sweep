package cli

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/KyleKing/gh-sweep/internal/tui"
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
		repo := stringFlag(cmd, "repo")

		// Launch full interactive TUI
		m := tui.NewMainModel(repo)
		p := tea.NewProgram(m, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
			os.Exit(1)
		}
	},
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
}
