package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/KyleKing/gh-sweep/internal/github"
	commentstui "github.com/KyleKing/gh-sweep/internal/tui/components/comments"
	"github.com/KyleKing/gh-sweep/internal/tui/theme"
)

var commentsCmd = &cobra.Command{
	Use:   "comments",
	Short: "Review unresolved PR review threads",
	Long: fmt.Sprintf(`Review unresolved GitHub PR review threads for a repository.

By default scans the newest %d open pull requests; use --pr to target a
single pull request. Unresolved means the review thread has not been
marked resolved on GitHub.

Filters:
  --author  match threads with any comment by this user (case-insensitive)
  --since   keep threads with activity on or after this date (YYYY-MM-DD)
  --search  case-insensitive substring match on file path or comment body

Examples:
  # Launch interactive TUI
  gh-sweep comments --repo owner/repo

  # Single pull request
  gh-sweep comments --repo owner/repo --pr 42

  # List unresolved threads (no TUI)
  gh-sweep comments --repo owner/repo --list

  # Filter by author and date
  gh-sweep comments --repo owner/repo --list --author octocat --since 2026-01-01`, github.DefaultOpenPRCap),
	Run: runComments,
}

func init() {
	rootCmd.AddCommand(commentsCmd)

	commentsCmd.Flags().String("repo", "", "Repository (owner/repo)")
	commentsCmd.Flags().Int("pr", 0, "Pull request number (default: newest open PRs)")
	commentsCmd.Flags().Bool("list", false, "CLI list mode (no TUI)")
	commentsCmd.Flags().String("author", "", "Filter by comment author")
	commentsCmd.Flags().String("since", "", "Filter by activity date (YYYY-MM-DD)")
	commentsCmd.Flags().
		String("search", "", "Case-insensitive substring search in path or comment text")
}

type commentsProgram struct {
	model commentstui.Model
}

func (p commentsProgram) Init() tea.Cmd {
	return p.model.Init()
}

func (p commentsProgram) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := p.model.Update(msg)
	p.model = model

	return p, cmd
}

func (p commentsProgram) View() tea.View {
	v := tea.NewView(p.model.View())
	v.AltScreen = true

	return v
}

func runComments(cmd *cobra.Command, args []string) {
	repo := stringFlag(cmd, "repo")
	prNumber := intFlag(cmd, "pr")
	listMode := boolFlag(cmd, "list")

	filter, err := buildThreadFilter(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if repo == "" {
		fmt.Fprintln(os.Stderr, "Error: --repo is required (owner/repo)")
		os.Exit(1)
	}
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		fmt.Fprintf(os.Stderr, "Error: invalid repo %q, expected owner/repo\n", repo)
		os.Exit(1)
	}
	owner, name := parts[0], parts[1]

	if !listMode {
		theme.Init(theme.Detect())

		m := commentsProgram{model: commentstui.NewModelWithOptions(repo, prNumber, filter)}
		p := tea.NewProgram(m)

		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
			os.Exit(1)
		}

		return
	}

	client, err := github.NewClient(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create GitHub client: %v\n", err)
		os.Exit(1)
	}

	gql, err := github.NewGQLClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	threads, err := gql.ListRepoReviewThreads(
		client,
		owner,
		name,
		prNumber,
		github.DefaultOpenPRCap,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	unresolved := filter.Apply(github.FilterUnresolvedThreads(threads))
	printThreads(repo, unresolved)
}

func buildThreadFilter(cmd *cobra.Command) (github.ThreadFilter, error) {
	filter := github.ThreadFilter{
		Author: stringFlag(cmd, "author"),
		Search: stringFlag(cmd, "search"),
	}

	if since := stringFlag(cmd, "since"); since != "" {
		parsed, err := github.ParseSinceDate(since)
		if err != nil {
			return github.ThreadFilter{}, err
		}
		filter.Since = &parsed
	}

	return filter, nil
}

func printThreads(repo string, threads []github.ReviewThread) {
	fmt.Printf("Unresolved review threads: %s\n\n", repo)

	if len(threads) == 0 {
		fmt.Println("No unresolved review threads found.")
		return
	}

	lastPR := 0
	for _, thread := range threads {
		if thread.PRNumber != lastPR {
			fmt.Printf("PR #%d: %s\n", thread.PRNumber, thread.PRTitle)
			lastPR = thread.PRNumber
		}

		outdated := ""
		if thread.IsOutdated {
			outdated = " [outdated]"
		}
		fmt.Printf("  %s%s\n", thread.Path, outdated)

		if first, ok := thread.FirstComment(); ok {
			fmt.Printf(
				"    @%s (%s): %s\n",
				first.Author,
				first.CreatedAt.Format("2006-01-02"),
				summarize(first.Body),
			)
			if replies := len(thread.Comments) - 1; replies > 0 {
				fmt.Printf("    %d replies\n", replies)
			}
			if first.URL != "" {
				fmt.Printf("    %s\n", first.URL)
			}
		}
		fmt.Println()
	}

	fmt.Printf("Total: %d unresolved threads\n", len(threads))
}

func summarize(body string) string {
	flat := strings.Join(strings.Fields(body), " ")
	if len(flat) > 100 {
		return flat[:100] + "..."
	}

	return flat
}
