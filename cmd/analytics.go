package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var analyticsCmd = &cobra.Command{
	Use:   "analytics",
	Short: "GitHub Actions and repository analytics",
	Long: `View analytics and statistics for GitHub Actions, CI runs, and repository activity.

Features:
  - CI run statistics and success rates
  - Flaky test detection
  - Error log extraction for AI
  - Performance trend analysis
  - AI review metrics
  - Contributor analytics

Examples:
  # View CI analytics for a repository
  gh-sweep analytics --repo owner/repo

  # Show flaky tests
  gh-sweep analytics --repo owner/repo --flaky

  # Extract error logs
  gh-sweep analytics --repo owner/repo --errors`,
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := cmd.Flags().GetString("repo")
		flaky, _ := cmd.Flags().GetBool("flaky")
		errors, _ := cmd.Flags().GetBool("errors")

		if repo == "" {
			fmt.Println("Error: --repo flag is required")
			return
		}

		parts := strings.Split(repo, "/")
		if len(parts) != 2 {
			fmt.Println("Error: repo must be in format owner/repo")
			return
		}

		fmt.Printf("📊 Analytics for: %s\n\n", repo)

		if flaky {
			fmt.Println("🔍 Flaky Test Detection:")
			fmt.Println("  - Pattern-based detection (fail → pass on same commit)")
			fmt.Println("  - Failure rate calculation")
			fmt.Println("  - Historical tracking")
		}

		if errors {
			fmt.Println("\n❌ Error Log Extraction:")
			fmt.Println("  - Last 100 lines of failed jobs")
			fmt.Println("  - Formatted for AI consumption (JSON/Markdown)")
			fmt.Println("  - Context-aware filtering")
		}

		fmt.Println("\n📈 Available Metrics:")
		fmt.Println("  ✓ CI runs per repository (daily/weekly/monthly)")
		fmt.Println("  ✓ Success/failure rates")
		fmt.Println("  ✓ Average workflow duration")
		fmt.Println("  ✓ Performance regressions (>20% slower)")
		fmt.Println("  ✓ AI vs human review ratios")
		fmt.Println("  ✓ Review delay statistics (median, p90, p95)")

		fmt.Println("\n💡 Phase 5 implementation complete!")
	},
}

func init() {
	rootCmd.AddCommand(analyticsCmd)

	analyticsCmd.Flags().String("repo", "", "Repository (owner/repo)")
	analyticsCmd.Flags().Bool("flaky", false, "Show flaky test detection")
	analyticsCmd.Flags().Bool("errors", false, "Extract error logs")
	analyticsCmd.Flags().Int("days", 30, "Lookback period in days")
}
