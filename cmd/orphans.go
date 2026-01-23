package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/orphans"
	orphanstui "github.com/KyleKing/gh-sweep/internal/tui/components/orphans"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
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

func init() {
	rootCmd.AddCommand(orphansCmd)

	orphansCmd.Flags().String("org", "", "Organization to scan")
	orphansCmd.Flags().String("namespace", "", "Namespace (org or user) to scan")
	orphansCmd.Flags().StringSlice("repos", nil, "Specific repos to scan (comma-separated)")
	orphansCmd.Flags().Bool("list", false, "CLI list mode (no TUI)")
	orphansCmd.Flags().Bool("cleanup", false, "Delete orphaned branches")
	orphansCmd.Flags().Bool("dry-run", false, "Preview deletions without executing")
	orphansCmd.Flags().Int("stale-days", 7, "Days of inactivity before a branch is considered stale")
	orphansCmd.Flags().Bool("include-recent", false, "Include recent branches without PRs")
	orphansCmd.Flags().StringSlice("exclude", nil, "Branch patterns to exclude")
	orphansCmd.Flags().StringP("output", "o", "", "Output file path")
	orphansCmd.Flags().String("format", "table", "Output format: table, json, markdown")
}

func runOrphans(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	client, err := github.NewClient(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create GitHub client: %v\n", err)
		os.Exit(1)
	}

	org, _ := cmd.Flags().GetString("org")
	namespace, _ := cmd.Flags().GetString("namespace")
	listMode, _ := cmd.Flags().GetBool("list")
	cleanup, _ := cmd.Flags().GetBool("cleanup")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	staleDays, _ := cmd.Flags().GetInt("stale-days")
	includeRecent, _ := cmd.Flags().GetBool("include-recent")
	excludePatterns, _ := cmd.Flags().GetStringSlice("exclude")
	outputPath, _ := cmd.Flags().GetString("output")
	format, _ := cmd.Flags().GetString("format")

	if namespace == "" {
		namespace = org
	}
	if namespace == "" {
		username, err := client.GetAuthenticatedUser()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to get authenticated user: %v\n", err)
			os.Exit(1)
		}
		namespace = username
	}

	options := orphans.DefaultScanOptions()
	options.StaleDaysThreshold = staleDays
	options.IncludeRecentNoPR = includeRecent
	if len(excludePatterns) > 0 {
		options.ExcludePatterns = append(options.ExcludePatterns, excludePatterns...)
	}

	if !listMode && !cleanup && outputPath == "" {
		m := orphanstui.NewModel(namespace, options)
		p := tea.NewProgram(m, tea.WithAltScreen())

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
		runCleanup(ctx, client, result, dryRun)
		return
	}

	if outputPath != "" || format == "json" || format == "markdown" {
		outputResult(result, outputPath, format)
		return
	}

	printTable(result)
}

func runCleanup(ctx context.Context, client *github.Client, result *orphans.NamespaceScanResult, dryRun bool) {
	allOrphans := result.AllOrphans()

	if len(allOrphans) == 0 {
		fmt.Println("No orphaned branches to clean up.")
		return
	}

	if dryRun {
		fmt.Println("DRY RUN - Would delete the following branches:")
	} else {
		fmt.Println("Deleting orphaned branches:")
	}
	fmt.Println()

	deleted := 0
	failed := 0

	for _, orphan := range allOrphans {
		parts := strings.SplitN(orphan.Repository, "/", 2)
		if len(parts) != 2 {
			continue
		}
		owner, repo := parts[0], parts[1]

		if dryRun {
			fmt.Printf("  [DRY RUN] Would delete %s/%s\n", orphan.Repository, orphan.BranchName)
			deleted++
			continue
		}

		err := client.DeleteBranch(owner, repo, orphan.BranchName)
		if err != nil {
			fmt.Printf("  [FAILED] %s/%s: %v\n", orphan.Repository, orphan.BranchName, err)
			failed++
		} else {
			fmt.Printf("  [DELETED] %s/%s\n", orphan.Repository, orphan.BranchName)
			deleted++
		}
	}

	fmt.Printf("\nTotal: %d deleted, %d failed\n", deleted, failed)
}

func outputResult(result *orphans.NamespaceScanResult, outputPath, format string) {
	var output string

	switch format {
	case "json":
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to marshal JSON: %v\n", err)
			os.Exit(1)
		}
		output = string(data)

	case "markdown":
		output = formatMarkdown(result)

	default:
		var b strings.Builder
		printTableTo(&b, result)
		output = b.String()
	}

	if outputPath != "" {
		if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to write output file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Output written to: %s\n", outputPath)
	} else {
		fmt.Print(output)
	}
}

func formatMarkdown(result *orphans.NamespaceScanResult) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# Orphaned Branches Report: %s\n\n", result.Namespace))
	b.WriteString(fmt.Sprintf("**Total Repositories:** %d\n", result.TotalRepos))
	b.WriteString(fmt.Sprintf("**Total Orphaned Branches:** %d\n\n", result.TotalOrphans))

	b.WriteString("## Summary by Type\n\n")
	b.WriteString("| Type | Count |\n")
	b.WriteString("|------|-------|\n")
	b.WriteString(fmt.Sprintf("| Merged PR | %d |\n", len(result.OrphansByType(orphans.OrphanTypeMergedPR))))
	b.WriteString(fmt.Sprintf("| Closed PR | %d |\n", len(result.OrphansByType(orphans.OrphanTypeClosedPR))))
	b.WriteString(fmt.Sprintf("| Stale | %d |\n", len(result.OrphansByType(orphans.OrphanTypeStale))))
	b.WriteString(fmt.Sprintf("| Recent (no PR) | %d |\n\n", len(result.OrphansByType(orphans.OrphanTypeRecentNoPR))))

	b.WriteString("## Orphaned Branches\n\n")
	b.WriteString("| Repository | Branch | Type | Days Inactive | PR |\n")
	b.WriteString("|------------|--------|------|---------------|----|\n")

	for _, orphan := range result.AllOrphans() {
		prInfo := "-"
		if orphan.PRNumber != nil {
			prInfo = fmt.Sprintf("#%d", *orphan.PRNumber)
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %d | %s |\n",
			orphan.Repository, orphan.BranchName, orphan.Type.Label(),
			orphan.DaysSinceActivity, prInfo))
	}

	return b.String()
}

func printTable(result *orphans.NamespaceScanResult) {
	var b strings.Builder
	printTableTo(&b, result)
	fmt.Print(b.String())
}

func printTableTo(b *strings.Builder, result *orphans.NamespaceScanResult) {
	b.WriteString(fmt.Sprintf("Orphaned Branches Report: %s\n\n", result.Namespace))
	b.WriteString(fmt.Sprintf("Total Repositories: %d\n", result.TotalRepos))
	b.WriteString(fmt.Sprintf("Total Orphaned Branches: %d\n\n", result.TotalOrphans))

	if result.TotalOrphans == 0 {
		b.WriteString("No orphaned branches found.\n")
		return
	}

	b.WriteString("Summary by Type:\n")
	b.WriteString(fmt.Sprintf("  Merged PR:      %d\n", len(result.OrphansByType(orphans.OrphanTypeMergedPR))))
	b.WriteString(fmt.Sprintf("  Closed PR:      %d\n", len(result.OrphansByType(orphans.OrphanTypeClosedPR))))
	b.WriteString(fmt.Sprintf("  Stale:          %d\n", len(result.OrphansByType(orphans.OrphanTypeStale))))
	b.WriteString(fmt.Sprintf("  Recent (no PR): %d\n\n", len(result.OrphansByType(orphans.OrphanTypeRecentNoPR))))

	b.WriteString("Orphaned Branches:\n\n")

	for _, scanResult := range result.Results {
		if len(scanResult.Orphans) == 0 {
			continue
		}

		b.WriteString(fmt.Sprintf("  %s (%d orphans)\n", scanResult.Repository.FullName, len(scanResult.Orphans)))

		for _, orphan := range scanResult.Orphans {
			prInfo := ""
			if orphan.PRNumber != nil {
				prInfo = fmt.Sprintf(" (PR #%d)", *orphan.PRNumber)
			}
			b.WriteString(fmt.Sprintf("    - %s [%s, %d days]%s\n",
				orphan.BranchName, orphan.Type.Label(), orphan.DaysSinceActivity, prInfo))
		}
		b.WriteString("\n")
	}
}
