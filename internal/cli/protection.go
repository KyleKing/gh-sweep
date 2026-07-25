package cli

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/KyleKing/gh-sweep/internal/github"
	protectiontui "github.com/KyleKing/gh-sweep/internal/tui/components/protection"
	"github.com/KyleKing/gh-sweep/internal/tui/theme"
)

var protectionCmd = &cobra.Command{
	Use:   "protection",
	Short: "Compare branch protection rules across repositories",
	Long: `Compare branch protection rules across repositories.

Protection rules are read from each repository's default branch. When a
baseline repo is given, values that drift from the baseline are marked
with an asterisk (*).

Examples:
  # Launch interactive TUI comparison
  gh-sweep protection --repos owner/repo1,owner/repo2

  # Non-interactive table output
  gh-sweep protection --repos owner/repo1,owner/repo2 --list

  # Highlight drift from a baseline repo
  gh-sweep protection --repos owner/repo1,owner/repo2 --baseline owner/repo1 --list`,
	Run: runProtection,
}

func init() {
	rootCmd.AddCommand(protectionCmd)

	protectionCmd.Flags().StringSlice("repos", nil, "Comma-separated list of repos (owner/repo1,owner/repo2)")
	protectionCmd.Flags().String("baseline", "", "Baseline repository to compare against")
	protectionCmd.Flags().Bool("list", false, "CLI table mode (no TUI)")
}

type protectionProgram struct {
	model protectiontui.Model
}

func (p protectionProgram) Init() tea.Cmd {
	return p.model.Init()
}

func (p protectionProgram) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := p.model.Update(msg)
	p.model = model

	return p, cmd
}

func (p protectionProgram) View() tea.View {
	v := tea.NewView(p.model.View())
	v.AltScreen = true

	return v
}

func runProtection(cmd *cobra.Command, args []string) {
	repos := stringSliceFlag(cmd, "repos")
	baseline := stringFlag(cmd, "baseline")
	listMode := boolFlag(cmd, "list")

	if len(repos) == 0 {
		fmt.Fprintln(os.Stderr, "Error: --repos is required (owner/repo1,owner/repo2)")
		os.Exit(1)
	}

	if baseline != "" && !slices.Contains(repos, baseline) {
		repos = append([]string{baseline}, repos...)
	}

	if !listMode {
		theme.Init(theme.Detect())

		m := protectionProgram{model: protectiontui.NewModel(repos, baseline)}
		p := tea.NewProgram(m)

		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
			os.Exit(1)
		}

		return
	}

	ctx := context.Background()
	client, err := github.NewClient(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create GitHub client: %v\n", err)
		os.Exit(1)
	}

	rules := fetchProtectionRules(client, repos)
	fmt.Print(formatProtectionTable(repos, rules, baseline))
}

func fetchProtectionRules(client *github.Client, repos []string) map[string]*github.ProtectionRule {
	rules := make(map[string]*github.ProtectionRule)

	for _, repoStr := range repos {
		parts := strings.SplitN(repoStr, "/", 2)
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "Warning: skipping invalid repo %q (expected owner/repo)\n", repoStr)
			continue
		}

		rule, err := client.GetDefaultBranchProtection(parts[0], parts[1])
		if err != nil {
			continue
		}

		rules[repoStr] = rule
	}

	return rules
}

func formatProtectionTable(repos []string, rules map[string]*github.ProtectionRule, baseline string) string {
	var b strings.Builder

	b.WriteString("Branch Protection Comparison\n")
	if baseline != "" {
		fmt.Fprintf(&b, "Baseline: %s (drift marked with *)\n", baseline)
	}
	b.WriteString("\n")

	w := tabwriter.NewWriter(&b, 0, 4, 2, ' ', 0)
	rows := []string{"REPO\tBRANCH\tREVIEWS\tCODEOWNERS\tADMINS\tLINEAR\tFORCE-PUSH\tDELETIONS\tSTATUS CHECKS"}

	base := rules[baseline]
	for _, repo := range repos {
		rows = append(rows, protectionRow(repo, rules[repo], base))
	}

	for _, row := range rows {
		if _, err := fmt.Fprintln(w, row); err != nil {
			return b.String()
		}
	}

	if err := w.Flush(); err != nil {
		return b.String()
	}

	return b.String()
}

func protectionRow(repo string, rule, base *github.ProtectionRule) string {
	if rule == nil {
		return repo + "\tno protection\t-\t-\t-\t-\t-\t-\t-"
	}

	fields := []string{
		repo,
		rule.Branch,
		markDrift(strconv.Itoa(rule.RequiredReviews), base != nil && rule.RequiredReviews != base.RequiredReviews),
		markDrift(strconv.FormatBool(rule.RequireCodeOwnerReviews), base != nil && rule.RequireCodeOwnerReviews != base.RequireCodeOwnerReviews),
		markDrift(strconv.FormatBool(rule.EnforceAdmins), base != nil && rule.EnforceAdmins != base.EnforceAdmins),
		markDrift(strconv.FormatBool(rule.RequireLinearHistory), base != nil && rule.RequireLinearHistory != base.RequireLinearHistory),
		markDrift(strconv.FormatBool(rule.AllowForcePushes), base != nil && rule.AllowForcePushes != base.AllowForcePushes),
		markDrift(strconv.FormatBool(rule.AllowDeletions), base != nil && rule.AllowDeletions != base.AllowDeletions),
		markDrift(formatChecks(rule.RequireStatusChecks), base != nil && !equalStringSets(rule.RequireStatusChecks, base.RequireStatusChecks)),
	}

	return strings.Join(fields, "\t")
}

func markDrift(value string, drifted bool) string {
	if drifted {
		return value + "*"
	}

	return value
}

func formatChecks(checks []string) string {
	if len(checks) == 0 {
		return "-"
	}

	return strings.Join(checks, ",")
}

func equalStringSets(a, b []string) bool {
	sortedA := slices.Clone(a)
	sortedB := slices.Clone(b)
	slices.Sort(sortedA)
	slices.Sort(sortedB)

	return slices.Equal(sortedA, sortedB)
}
