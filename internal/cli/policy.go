package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/KyleKing/gh-sweep/internal/config"
	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/policy"
	policytui "github.com/KyleKing/gh-sweep/internal/tui/components/policy"
	"github.com/KyleKing/gh-sweep/internal/tui/theme"
)

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Diff and sync repository settings against a declared policy file",
	Long: `Diff live repository settings, security & analysis, release immutability,
branch protection, and repository rulesets against a .gh-sweep-policy.yaml file,
and optionally sync drift back to GitHub. A field left out of the policy is never
reported or changed; only what you declare is managed.

See .gh-sweep-policy.yaml.example for the schema.

Examples:
  # Launch interactive TUI
  gh-sweep policy

  # List drift (no TUI); exits 1 if any repo has drift or fails to load, for CI use
  gh-sweep policy --list

  # Apply the policy, confirming each repo
  gh-sweep policy --apply

  # Apply without prompting (CI / scripted use)
  gh-sweep policy --apply --yes

  # Machine-readable drift report
  gh-sweep policy --list --format json`,
	Run: runPolicy,
}

func init() {
	rootCmd.AddCommand(policyCmd)

	policyCmd.Flags().String("policy", "", "Path to the policy file (default: .gh-sweep-policy.yaml)")
	policyCmd.Flags().Bool("list", false, "CLI list mode (no TUI); exits 1 if drift is found or a repo fails to load")
	policyCmd.Flags().Bool("apply", false, "Sync drifted repos toward the policy")
	policyCmd.Flags().Bool(confirmYes, false, "Skip the per-repo confirmation prompt when applying")
	policyCmd.Flags().String("format", "table", "Output format: table, json, markdown")
}

func runPolicy(cmd *cobra.Command, _ []string) {
	policyPath := stringFlag(cmd, "policy")
	listMode := boolFlag(cmd, "list")
	apply := boolFlag(cmd, "apply")
	yes := boolFlag(cmd, confirmYes)
	format := stringFlag(cmd, "format")

	cfg, err := config.LoadPolicy(policyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !listMode && !apply {
		theme.Init(theme.Detect())

		m := policyProgram{model: policytui.NewModel(cfg)}
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

	report := policy.Evaluate(client, cfg)

	if apply {
		runApply(client, cfg, report, yes)
		return
	}

	printReport(report, format)

	if report.HasDrift() || report.HasErrors() {
		os.Exit(1)
	}
}

type policyProgram struct {
	model policytui.Model
}

func (p policyProgram) Init() tea.Cmd {
	return p.model.Init()
}

func (p policyProgram) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := p.model.Update(msg)
	p.model = model

	return p, cmd
}

func (p policyProgram) View() tea.View {
	v := tea.NewView(p.model.View())
	v.AltScreen = true

	return v
}

func runApply(client *github.Client, cfg *config.PolicyConfig, report *policy.Report, yes bool) {
	if !report.HasDrift() {
		fmt.Println("No drift found; nothing to apply.")
		return
	}

	reader := bufio.NewReader(os.Stdin)
	applied, failed, skipped := 0, 0, 0

	for _, drift := range report.Repos {
		if drift.Err != nil || len(drift.Diffs) == 0 {
			continue
		}

		fmt.Printf("%s: %d field(s) drifted\n", drift.Repository, len(drift.Diffs))
		for _, d := range drift.Diffs {
			fmt.Printf("  [%s] %s: %s -> %s\n", d.Domain, d.Field, d.Current, d.Desired)
		}

		if !yes && !confirm(reader, fmt.Sprintf("Apply to %s?", drift.Repository)) {
			fmt.Println("  skipped")
			skipped++

			continue
		}

		result := policy.Apply(client, cfg, drift)
		if result.Err != nil {
			fmt.Printf("  [FAILED] %v\n", result.Err)
			failed++

			continue
		}

		fmt.Printf("  [APPLIED] %v\n", result.Applied)
		applied++
	}

	fmt.Printf("\nTotal: %d applied, %d skipped, %d failed\n", applied, skipped, failed)

	if failed > 0 {
		os.Exit(1)
	}
}

func confirm(reader *bufio.Reader, prompt string) bool {
	fmt.Printf("%s [y/N] ", prompt)

	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	return strings.EqualFold(strings.TrimSpace(line), "y")
}

func printReport(report *policy.Report, format string) {
	switch format {
	case formatJSON:
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to marshal JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))

	case formatMarkdown:
		fmt.Print(formatPolicyMarkdown(report))

	default:
		fmt.Print(formatPolicyTable(report))
	}
}

func formatPolicyTable(report *policy.Report) string {
	var b strings.Builder

	if !report.HasDrift() {
		hasErrors := false
		for _, r := range report.Repos {
			if r.Err != nil {
				hasErrors = true
			}
		}
		if !hasErrors {
			b.WriteString("No drift found.\n")
			return b.String()
		}
	}

	for _, drift := range report.Repos {
		if drift.Err != nil {
			fmt.Fprintf(&b, "%s: ERROR: %v\n", drift.Repository, drift.Err)
			continue
		}

		if len(drift.Diffs) == 0 {
			continue
		}

		fmt.Fprintf(&b, "%s (%d drifted field(s))\n", drift.Repository, len(drift.Diffs))
		for _, d := range drift.Diffs {
			fmt.Fprintf(&b, "  [%s] %s: current=%s desired=%s\n", d.Domain, d.Field, d.Current, d.Desired)
		}
	}

	return b.String()
}

func formatPolicyMarkdown(report *policy.Report) string {
	var b strings.Builder

	b.WriteString("# Policy Drift Report\n\n")
	b.WriteString("| Repository | Domain | Field | Current | Desired |\n")
	b.WriteString("|------------|--------|-------|---------|---------|\n")

	for _, drift := range report.Repos {
		if drift.Err != nil {
			fmt.Fprintf(&b, "| %s | - | error | %v | - |\n", drift.Repository, drift.Err)
			continue
		}

		for _, d := range drift.Diffs {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
				drift.Repository, d.Domain, d.Field, d.Current, d.Desired)
		}
	}

	return b.String()
}
