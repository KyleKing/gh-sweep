package cmd

import (
	"context"
	"fmt"

	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/spf13/cobra"
)

var watchingCmd = &cobra.Command{
	Use:   "watching",
	Short: "Audit and manage repository watch status",
	Long: `Audit repositories in your namespace to identify unwatched repos and enable batch watching.

Examples:
  # Launch interactive TUI
  gh-sweep watching

  # List unwatched repos
  gh-sweep watching --unwatched

  # Watch all repos in namespace
  gh-sweep watching --watch-all`,
	Run: func(cmd *cobra.Command, args []string) {
		unwatched, _ := cmd.Flags().GetBool("unwatched")
		watchAll, _ := cmd.Flags().GetBool("watch-all")

		ctx := context.Background()
		client, err := github.NewClient(ctx)
		if err != nil {
			fmt.Printf("Error: failed to create GitHub client: %v\n", err)
			return
		}

		username, err := client.GetAuthenticatedUser()
		if err != nil {
			fmt.Printf("Error: failed to get authenticated user: %v\n", err)
			return
		}

		repos, err := client.ListUserRepos()
		if err != nil {
			fmt.Printf("Error: failed to list user repos: %v\n", err)
			return
		}

		var unwatchedRepos []github.RepoBasic
		for _, repo := range repos {
			sub, err := client.GetRepoSubscription(repo.Owner, repo.Name)
			if err != nil {
				continue
			}
			if sub.State == github.WatchStateNotWatching {
				unwatchedRepos = append(unwatchedRepos, repo)
			}
		}

		if unwatched {
			fmt.Printf("Unwatched repositories for %s:\n\n", username)
			if len(unwatchedRepos) == 0 {
				fmt.Println("All repositories are being watched.")
				return
			}
			for _, repo := range unwatchedRepos {
				fmt.Printf("  - %s\n", repo.FullName)
			}
			fmt.Printf("\nTotal: %d unwatched repositories\n", len(unwatchedRepos))
			return
		}

		if watchAll {
			if len(unwatchedRepos) == 0 {
				fmt.Println("All repositories are already being watched.")
				return
			}
			fmt.Printf("Watching %d repositories...\n\n", len(unwatchedRepos))
			for _, repo := range unwatchedRepos {
				_, err := client.SetRepoSubscription(repo.Owner, repo.Name, true, false)
				if err != nil {
					fmt.Printf("  Failed to watch %s: %v\n", repo.FullName, err)
					continue
				}
				fmt.Printf("  Watching %s\n", repo.FullName)
			}
			fmt.Println("\nDone.")
			return
		}

		fmt.Printf("Watch Status Audit for: %s\n\n", username)
		fmt.Printf("Total repositories: %d\n", len(repos))
		fmt.Printf("Unwatched repositories: %d\n\n", len(unwatchedRepos))
		fmt.Println("Use --unwatched to list unwatched repos")
		fmt.Println("Use --watch-all to watch all unwatched repos")
		fmt.Println("\nOr launch the full TUI with: gh-sweep (then press 0)")
	},
}

func init() {
	rootCmd.AddCommand(watchingCmd)

	watchingCmd.Flags().Bool("unwatched", false, "List unwatched repositories")
	watchingCmd.Flags().Bool("watch-all", false, "Watch all unwatched repositories")
}
