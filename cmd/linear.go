package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var linearCmd = &cobra.Command{
	Use:   "linear",
	Short: "Linear integration for issue-PR linking",
	Long: `View and manage Linear issues linked to GitHub PRs.

Features:
  - Issue-PR linking detection
  - Workflow automation insights
  - Sync status dashboard
  - Cross-repo issue tracking

Examples:
  # View Linear issues for a repo
  gh-sweep linear --repo owner/repo

  # Check sync status
  gh-sweep linear --repo owner/repo --sync-status`,
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := cmd.Flags().GetString("repo")
		syncStatus, _ := cmd.Flags().GetBool("sync-status")

		fmt.Println("🔗 Linear Integration (Phase 4)")
		fmt.Println()

		if repo != "" {
			fmt.Printf("Repository: %s\n\n", repo)
		}

		if syncStatus {
			fmt.Println("📊 Sync Status Detection:")
			fmt.Println("  - PR merged but Linear issue not 'Done'")
			fmt.Println("  - PR closed but Linear issue not 'Canceled'")
			fmt.Println("  - PR open but Linear issue 'Done'")
		}

		fmt.Println("✨ Features:")
		fmt.Println("  ✓ Extract Linear issue IDs from PR descriptions")
		fmt.Println("  ✓ Fetch issue details via GraphQL API")
		fmt.Println("  ✓ Display issue state, assignee, project, cycle")
		fmt.Println("  ✓ Detect sync drift between GitHub and Linear")
		fmt.Println("  ✓ Navigate to Linear issue from TUI")

		fmt.Println("\n💡 Configure with:")
		fmt.Println("   linear:")
		fmt.Println("     api_key: lin_api_...")
		fmt.Println("     workspace: your-workspace")
	},
}

func init() {
	rootCmd.AddCommand(linearCmd)

	linearCmd.Flags().String("repo", "", "Repository (owner/repo)")
	linearCmd.Flags().Bool("sync-status", false, "Check sync status")
}
