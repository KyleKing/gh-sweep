package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/KyleKing/gh-sweep/internal/config"
	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/orphans"
	orphanstui "github.com/KyleKing/gh-sweep/internal/tui/components/orphans"
	"github.com/KyleKing/gh-sweep/internal/tui/theme"
)

var orphansCmd = &cobra.Command{
	Use:   "orphans",
	Short: "Detect and clean up orphaned branches across repositories",
	Long: `Scan repositories in a namespace (org or user) for orphaned branches.

Orphan types detected:
  - merged_pr:   Branch from a merged PR that wasn't auto-deleted
  - closed_pr:   Branch from a closed (unmerged) PR
  - stale:       No associated PR, inactive > threshold (default 7 days)

Examples:
  # Launch interactive TUI for current user
  gh-sweep orphans

  # TUI for an organization
  gh-sweep orphans --org mycompany

  # List orphaned branches (no TUI)
  gh-sweep orphans --org mycompany --list

  # Preview cleanup without executing
  gh-sweep orphans --cleanup --dry-run

  # Export to JSON
  gh-sweep orphans --format json -o orphans.json`,
	Run: runOrphans,
}

const defaultStaleDays = 21

func init() {
	rootCmd.AddCommand(orphansCmd)

	orphansCmd.Flags().String("org", "", "Organization to scan")
	orphansCmd.Flags().String("namespace", "", "Namespace (org or user) to scan")
	orphansCmd.Flags().StringSlice("repos", nil, "Specific repos to scan (comma-separated)")
	orphansCmd.Flags().Bool("list", false, "CLI list mode (no TUI)")
	orphansCmd.Flags().Bool("cleanup", false, "Delete orphaned branches")
	orphansCmd.Flags().Bool("dry-run", false, "Preview deletions without executing")
	orphansCmd.Flags().Bool(confirmYes, false, "Skip the cleanup confirmation prompt")
	orphansCmd.Flags().
		Bool("exclude-closed-pr", false, "Keep closed-PR branches out of --cleanup (deleted by default)")
	orphansCmd.Flags().
		Int("stale-days", defaultStaleDays, "Grace period in days for a branch with no PR at all")
	orphansCmd.Flags().Bool("include-recent", false, "Include recent branches without PRs")
	orphansCmd.Flags().StringSlice("exclude", nil, "Branch patterns to exclude")
	orphansCmd.Flags().StringP("output", "o", "", "Output file path")
	orphansCmd.Flags().String("format", "table", "Output format: table, json, markdown")
}

type orphansProgram struct {
	model orphanstui.Model
}

func (p orphansProgram) Init() tea.Cmd {
	return p.model.Init()
}

func (p orphansProgram) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := p.model.Update(msg)
	p.model = model

	return p, cmd
}

func (p orphansProgram) View() tea.View {
	v := tea.NewView(p.model.View())
	v.AltScreen = true

	return v
}

// intFallback prefers the config value when the flag was left at its default,
// so a config setting is not silently overridden by a flag nobody passed.
func intFallback(cmd *cobra.Command, name string, flagValue, configValue int) int {
	if !cmd.Flags().Changed(name) && configValue > 0 {
		return configValue
	}

	return flagValue
}

func scanOptions(
	cmd *cobra.Command,
	cfg *config.Config,
	staleDays int,
	includeRecent bool,
	excludePatterns []string,
) orphans.ScanOptions {
	options := cfg.ScanOptions()
	options.StaleDaysThreshold = staleDays
	options.IncludeRecentNoPR = includeRecent

	if repos := stringSliceFlag(cmd, "repos"); len(repos) > 0 {
		options.OnlyRepos = repos
	}

	if len(excludePatterns) > 0 {
		options.ExcludePatterns = append(options.ExcludePatterns, excludePatterns...)
	}

	return options
}

func runOrphans(cmd *cobra.Command, _ []string) {
	ctx := context.Background()

	client, err := github.NewClient(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create GitHub client: %v\n", err)
		os.Exit(1)
	}

	listMode := boolFlag(cmd, "list")
	cleanup := boolFlag(cmd, "cleanup")
	dryRun := boolFlag(cmd, "dry-run")
	yes := boolFlag(cmd, confirmYes)
	includeClosedPR := !boolFlag(cmd, "exclude-closed-pr")
	staleDays := intFlag(cmd, "stale-days")
	includeRecent := boolFlag(cmd, "include-recent")
	excludePatterns := stringSliceFlag(cmd, "exclude")
	outputPath := stringFlag(cmd, "output")
	format := stringFlag(cmd, "format")

	cfg := loadConfig()

	namespace, err := resolveNamespace(cmd, client, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	staleDays = intFallback(cmd, "stale-days", staleDays, cfg.Orphans.StaleDaysThreshold)

	options := scanOptions(cmd, cfg, staleDays, includeRecent, excludePatterns)

	if !listMode && !cleanup && outputPath == "" {
		theme.Init(theme.Detect())

		m := orphansProgram{model: orphanstui.NewModel(namespace, options)}
		p := tea.NewProgram(m)

		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
			os.Exit(1)
		}

		return
	}

	fmt.Printf("Scanning namespace: %s\n", namespace)
	scanner := orphans.NewNamespaceScanner(client, options)
	result, err := scanner.ScanNamespace(ctx, namespace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to scan namespace: %v\n", err)
		os.Exit(1)
	}

	if cleanup {
		runCleanup(client, result, dryRun, yes, includeClosedPR)
		return
	}

	if outputPath != "" || format == formatJSON || format == formatMarkdown {
		outputResult(result, outputPath, format)
		return
	}

	printTable(result)
}

func filterClosedPROrphans(
	allOrphans []orphans.OrphanedBranch,
	includeClosedPR bool,
) ([]orphans.OrphanedBranch, int) {
	if includeClosedPR {
		return allOrphans, 0
	}

	filtered := make([]orphans.OrphanedBranch, 0, len(allOrphans))
	skipped := 0

	for _, orphan := range allOrphans {
		if orphan.Type == orphans.OrphanTypeClosedPR {
			skipped++
			continue
		}
		filtered = append(filtered, orphan)
	}

	return filtered, skipped
}

func runCleanup(
	client *github.Client,
	result *orphans.NamespaceScanResult,
	dryRun, yes, includeClosedPR bool,
) {
	allOrphans, skippedClosedPR := filterClosedPROrphans(result.AllOrphans(), includeClosedPR)

	if len(allOrphans) == 0 {
		fmt.Println("No orphaned branches to clean up.")
		if skippedClosedPR > 0 {
			fmt.Printf(
				"Skipped %d closed-PR branch(es) per --exclude-closed-pr.\n",
				skippedClosedPR,
			)
		}

		return
	}

	if dryRun {
		fmt.Println("DRY RUN - Would delete the following branches:")
	} else {
		fmt.Println("Branches to delete:")
	}
	fmt.Println()

	for _, orphan := range allOrphans {
		fmt.Printf("  %s/%s [%s, %d days]\n", orphan.Repository, orphan.BranchName,
			orphan.Type.Label(), orphan.DaysSinceActivity)
	}
	fmt.Println()

	if skippedClosedPR > 0 {
		fmt.Printf(
			"Skipping %d closed-PR branch(es) per --exclude-closed-pr.\n\n",
			skippedClosedPR,
		)
	}

	if !dryRun && !yes {
		if !confirmTypedYes(fmt.Sprintf("Type \"yes\" to delete these %d branch(es): ", len(allOrphans))) {
			fmt.Println("Aborted; no branches deleted.")
			return
		}
		fmt.Println()
	}

	if dryRun {
		fmt.Printf("Total: %d would be deleted\n", len(allOrphans))
		return
	}

	deleted, failed := deleteOrphanedBranches(client, allOrphans)
	fmt.Printf("\nTotal: %d deleted, %d failed\n", deleted, failed)
}

func deleteOrphanedBranches(client *github.Client, allOrphans []orphans.OrphanedBranch) (int, int) {
	fmt.Println("Deleting orphaned branches:")

	deleted, failed := 0, 0

	for _, orphan := range allOrphans {
		owner, repo, ok := splitRepo(orphan.Repository)
		if !ok {
			continue
		}

		if err := client.DeleteBranch(owner, repo, orphan.BranchName); err != nil {
			fmt.Printf("  [FAILED] %s/%s: %v\n", orphan.Repository, orphan.BranchName, err)
			failed++

			continue
		}

		fmt.Printf("  [DELETED] %s/%s\n", orphan.Repository, orphan.BranchName)
		deleted++
	}

	return deleted, failed
}

func outputResult(result *orphans.NamespaceScanResult, outputPath, format string) {
	writeOutput(outputPath, format,
		func() ([]byte, error) { return json.MarshalIndent(result, "", "  ") },
		func() string { return formatOrphansMarkdown(result) },
		func(b *strings.Builder) { printTableTo(b, result) },
	)
}

func formatOrphansMarkdown(result *orphans.NamespaceScanResult) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# Orphaned Branches Report: %s\n\n", result.Namespace)
	fmt.Fprintf(&b, "**Total Repositories:** %d\n", result.TotalRepos)
	fmt.Fprintf(&b, "**Total Orphaned Branches:** %d\n\n", result.TotalOrphans)

	b.WriteString("## Summary by Type\n\n")
	b.WriteString("| Type | Count |\n")
	b.WriteString("|------|-------|\n")
	fmt.Fprintf(&b, "| Merged PR | %d |\n", len(result.OrphansByType(orphans.OrphanTypeMergedPR)))
	fmt.Fprintf(&b, "| Closed PR | %d |\n", len(result.OrphansByType(orphans.OrphanTypeClosedPR)))
	fmt.Fprintf(&b, "| Stale | %d |\n", len(result.OrphansByType(orphans.OrphanTypeStale)))
	fmt.Fprintf(
		&b,
		"| Recent (no PR) | %d |\n\n",
		len(result.OrphansByType(orphans.OrphanTypeRecentNoPR)),
	)

	b.WriteString("## Orphaned Branches\n\n")
	b.WriteString("| Repository | Branch | Type | Days Inactive | PR |\n")
	b.WriteString("|------------|--------|------|---------------|----|\n")

	for _, orphan := range result.AllOrphans() {
		prInfo := "-"
		if orphan.PRNumber != nil {
			prInfo = fmt.Sprintf("#%d", *orphan.PRNumber)
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %d | %s |\n",
			orphan.Repository, orphan.BranchName, orphan.Type.Label(),
			orphan.DaysSinceActivity, prInfo)
	}

	return b.String()
}

func printTable(result *orphans.NamespaceScanResult) {
	var b strings.Builder
	printTableTo(&b, result)
	fmt.Print(b.String())
}

func printTableTo(b *strings.Builder, result *orphans.NamespaceScanResult) {
	fmt.Fprintf(b, "Orphaned Branches Report: %s\n\n", result.Namespace)
	fmt.Fprintf(b, "Total Repositories: %d\n", result.TotalRepos)
	fmt.Fprintf(b, "Total Orphaned Branches: %d\n\n", result.TotalOrphans)

	if result.TotalOrphans == 0 {
		b.WriteString("No orphaned branches found.\n")
		return
	}

	b.WriteString("Summary by Type:\n")
	fmt.Fprintf(b, "  Merged PR:      %d\n", len(result.OrphansByType(orphans.OrphanTypeMergedPR)))
	fmt.Fprintf(b, "  Closed PR:      %d\n", len(result.OrphansByType(orphans.OrphanTypeClosedPR)))
	fmt.Fprintf(b, "  Stale:          %d\n", len(result.OrphansByType(orphans.OrphanTypeStale)))
	fmt.Fprintf(
		b,
		"  Recent (no PR): %d\n\n",
		len(result.OrphansByType(orphans.OrphanTypeRecentNoPR)),
	)

	b.WriteString("Orphaned Branches:\n\n")

	for i := range result.Results {
		scanResult := &result.Results[i]
		if len(scanResult.Orphans) == 0 {
			continue
		}

		fmt.Fprintf(
			b,
			"  %s (%d orphans)\n",
			scanResult.Repository.FullName,
			len(scanResult.Orphans),
		)

		for _, orphan := range scanResult.Orphans {
			prInfo := ""
			if orphan.PRNumber != nil {
				prInfo = fmt.Sprintf(" (PR #%d)", *orphan.PRNumber)
			}
			fmt.Fprintf(b, "    - %s [%s, %d days]%s\n",
				orphan.BranchName, orphan.Type.Label(), orphan.DaysSinceActivity, prInfo)
		}
		b.WriteString("\n")
	}
}
