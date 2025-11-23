package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var protectionCmd = &cobra.Command{
	Use:   "protection",
	Short: "Branch protection rule management",
	Long: `Compare and sync branch protection rules across repositories.

Features:
  - Visual comparison of protection settings
  - Apply templates to multiple repos
  - Detect drift from baseline
  - Export/import rule configurations

Examples:
  # Compare protection rules across repos
  gh-sweep protection --repos owner/repo1,owner/repo2

  # Apply template
  gh-sweep protection --template templates/default.yaml --apply

  # Show drift from baseline
  gh-sweep protection --baseline owner/baseline-repo`,
	Run: func(cmd *cobra.Command, args []string) {
		repos, _ := cmd.Flags().GetString("repos")
		template, _ := cmd.Flags().GetString("template")
		baseline, _ := cmd.Flags().GetString("baseline")

		fmt.Printf("Protection rule management\n")
		fmt.Printf("Repos: %s\n", repos)
		fmt.Printf("Template: %s, Baseline: %s\n", template, baseline)
		fmt.Println("\n🚧 Coming in Phase 1!")
	},
}

func init() {
	rootCmd.AddCommand(protectionCmd)

	protectionCmd.Flags().String("repos", "", "Comma-separated list of repos (owner/repo1,owner/repo2)")
	protectionCmd.Flags().String("template", "", "Path to protection rule template (YAML)")
	protectionCmd.Flags().String("baseline", "", "Baseline repository to compare against")
	protectionCmd.Flags().Bool("apply", false, "Apply changes (default: dry-run)")
}
