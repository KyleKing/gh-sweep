package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var branchesCmd = &cobra.Command{
	Use:   "branches",
	Short: "Interactive branch management",
	Long: `Interactive branch management with dependency visualization.

Features:
  - View branch relationships as a tree
  - Create stacked PRs with automatic dependency detection
  - Batch delete branches with confirmation
  - Show ahead/behind counts
  - Multi-select interface (ranges, "all", etc.)

Examples:
  # Launch interactive branch manager
  gh-sweep branches

  # Show branch tree for specific repo
  gh-sweep branches --repo owner/repo --tree

  # Create stacked PRs
  gh-sweep branches --repo owner/repo --stacked-prs`,
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := cmd.Flags().GetString("repo")
		tree, _ := cmd.Flags().GetBool("tree")
		stackedPRs, _ := cmd.Flags().GetBool("stacked-prs")

		fmt.Printf("Branch management for: %s\n", repo)
		fmt.Printf("Tree mode: %v, Stacked PRs: %v\n", tree, stackedPRs)
		fmt.Println("\n🚧 Coming in Phase 1!")
	},
}

func init() {
	rootCmd.AddCommand(branchesCmd)

	branchesCmd.Flags().String("repo", "", "Repository (owner/repo)")
	branchesCmd.Flags().Bool("tree", false, "Show branch tree visualization")
	branchesCmd.Flags().Bool("stacked-prs", false, "Create stacked PRs from selected branches")
}
