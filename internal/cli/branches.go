package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/KyleKing/gh-sweep/internal/github"
	branchestui "github.com/KyleKing/gh-sweep/internal/tui/components/branches"
	"github.com/KyleKing/gh-sweep/internal/tui/theme"
)

var branchesCmd = &cobra.Command{
	Use:   "branches",
	Short: "Interactive branch management",
	Long: `Manage branches for a repository.

The default mode launches an interactive TUI: navigate with j/k, multi-select
with space, and delete with 'd' (confirmation required). The default branch,
protected branches, and branches with an open PR are blocked from deletion.

Examples:
  # Launch interactive branch manager
  gh-sweep branches --repo owner/repo

  # Print a non-interactive branch table
  gh-sweep branches --repo owner/repo --list`,
	Run: runBranches,
}

func init() {
	rootCmd.AddCommand(branchesCmd)

	branchesCmd.Flags().String("repo", "", "Repository (owner/repo)")
	branchesCmd.Flags().String("base", "", "Base branch for ahead/behind comparison (default: repository default branch)")
	branchesCmd.Flags().Bool("list", false, "CLI list mode (no TUI)")
}

type branchesProgram struct {
	model branchestui.Model
}

func (p branchesProgram) Init() tea.Cmd {
	return p.model.Init()
}

func (p branchesProgram) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := p.model.Update(msg)
	p.model = model

	return p, cmd
}

func (p branchesProgram) View() tea.View {
	v := tea.NewView(p.model.View())
	v.AltScreen = true

	return v
}

func runBranches(cmd *cobra.Command, args []string) {
	repo := stringFlag(cmd, "repo")
	base := stringFlag(cmd, "base")
	listMode := boolFlag(cmd, "list")

	if repo == "" {
		fmt.Fprintln(os.Stderr, "Error: --repo flag is required")
		fmt.Fprintln(os.Stderr, "\nUsage: gh-sweep branches --repo owner/repo")
		os.Exit(1)
	}

	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		fmt.Fprintln(os.Stderr, "Error: repo must be in format owner/repo")
		os.Exit(1)
	}

	if listMode {
		listBranches(parts[0], parts[1], base)
		return
	}

	theme.Init(theme.Detect())

	m := branchesProgram{model: branchestui.NewModel(repo, base)}
	p := tea.NewProgram(m)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func listBranches(owner, repo, base string) {
	ctx := context.Background()

	client, err := github.NewClient(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create GitHub client: %v\n", err)
		os.Exit(1)
	}

	branches, err := client.ListBranchStatuses(owner, repo, base)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to list branches: %v\n", err)
		os.Exit(1)
	}

	if len(branches) == 0 {
		fmt.Println("No branches found.")
		return
	}

	fmt.Print(renderBranchTable(branches, time.Now()))
}

func renderBranchTable(branches []github.BranchStatus, now time.Time) string {
	rows := make([][]string, 0, len(branches)+1)
	rows = append(rows, []string{"BRANCH", "LAST COMMIT", "PR", "PROTECTED"})

	for _, branch := range branches {
		name := branch.Name
		if branch.IsDefault {
			name += " (default)"
		}

		rows = append(rows, []string{
			name,
			formatCommitAge(branch.LastCommitDate, now),
			formatBranchPR(branch.PR),
			formatBool(branch.Protected),
		})
	}

	widths := make([]int, len(rows[0]))
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	var b strings.Builder
	for _, row := range rows {
		for i, cell := range row {
			if i == len(row)-1 {
				b.WriteString(cell)
			} else {
				fmt.Fprintf(&b, "%-*s  ", widths[i], cell)
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

func formatCommitAge(commit, now time.Time) string {
	if commit.IsZero() {
		return "-"
	}

	days := int(now.Sub(commit).Hours() / 24)
	switch {
	case days < 1:
		return "<1d"
	case days < 365:
		return fmt.Sprintf("%dd", days)
	default:
		return fmt.Sprintf("%dy", days/365)
	}
}

func formatBranchPR(pr *github.PullRequest) string {
	if pr == nil {
		return "-"
	}

	return fmt.Sprintf("#%d (%s)", pr.Number, pr.State)
}

func formatBool(value bool) string {
	if value {
		return "yes"
	}

	return "no"
}
