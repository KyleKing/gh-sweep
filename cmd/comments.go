package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var commentsCmd = &cobra.Command{
	Use:   "comments",
	Short: "Review unresolved PR comments",
	Long: `Search, filter, and review unresolved GitHub PR comments.

Features:
  - List unresolved comments across repositories
  - Advanced filtering (author, date, fuzzy search)
  - Code context preview
  - Navigate to comment in browser
  - Caching for offline browsing

Examples:
  # Search unresolved comments
  gh-sweep comments --repo owner/repo

  # Filter by author
  gh-sweep comments --author username

  # Filter by date range
  gh-sweep comments --since 2024-01-01

  # Fuzzy search in comment text
  gh-sweep comments --search "TODO|FIXME"`,
	Run: func(cmd *cobra.Command, args []string) {
		repo, _ := cmd.Flags().GetString("repo")
		author, _ := cmd.Flags().GetString("author")
		since, _ := cmd.Flags().GetString("since")
		search, _ := cmd.Flags().GetString("search")

		fmt.Printf("Unresolved comment review for: %s\n", repo)
		fmt.Printf("Author: %s, Since: %s, Search: %s\n", author, since, search)
		fmt.Println("\n🚧 Coming in Phase 1!")
	},
}

func init() {
	rootCmd.AddCommand(commentsCmd)

	commentsCmd.Flags().String("repo", "", "Repository (owner/repo)")
	commentsCmd.Flags().String("author", "", "Filter by comment author")
	commentsCmd.Flags().String("since", "", "Filter by date (YYYY-MM-DD)")
	commentsCmd.Flags().String("search", "", "Fuzzy search in comment text")
	commentsCmd.Flags().Bool("refresh", false, "Force refresh cache")
}
