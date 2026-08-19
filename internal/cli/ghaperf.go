package cli

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/KyleKing/gh-sweep/internal/cache"
	"github.com/KyleKing/gh-sweep/internal/config"
	"github.com/KyleKing/gh-sweep/internal/github"
)

const (
	defaultGHAPerfLimit    = 30
	defaultGHAPerfDays     = 30
	sectionRuleWidth       = 60
	branchSectionRuleWidth = 70
	branchDividerWidth     = 50
	topJobsCount           = 10
	jobNameMaxLen          = 50
	stepNameMaxLen         = 40
	percentScale           = 100.0
)

var ghaPerfCmd = &cobra.Command{
	Use:   "gha-perf",
	Short: "Analyze GitHub Actions workflow performance",
	Long: `Analyze GitHub Actions workflow performance over time.

Fetches workflow run data with detailed timing for jobs and steps.
Supports caching for incremental analysis and filtering by branch,
workflow, time range, and more.

Examples:
  # Analyze all workflows
  gh-sweep gha-perf --repo owner/repo

  # Analyze specific workflow
  gh-sweep gha-perf --repo owner/repo --workflow ci.yml

  # Filter by branch and time
  gh-sweep gha-perf --repo owner/repo --branch main --days 14

  # Compare branches
  gh-sweep gha-perf --repo owner/repo --compare main

  # Export to CSV
  gh-sweep gha-perf --repo owner/repo --csv output.csv

  # Use cached data only
  gh-sweep gha-perf --repo owner/repo --cache-only`,
	Run: runGHAPerf,
}

func init() {
	rootCmd.AddCommand(ghaPerfCmd)

	ghaPerfCmd.Flags().String("repo", "", "Repository (owner/repo)")
	ghaPerfCmd.Flags().StringP("workflow", "w", "", "Workflow file to analyze")
	ghaPerfCmd.Flags().StringP("branch", "b", "", "Filter by branch name")
	ghaPerfCmd.Flags().IntP("limit", "l", defaultGHAPerfLimit, "Number of runs to fetch")
	ghaPerfCmd.Flags().Int("days", defaultGHAPerfDays, "Lookback period in days")
	ghaPerfCmd.Flags().StringP("compare", "c", "", "Compare current runs against another branch")
	ghaPerfCmd.Flags().String("base-branch", "main", "Base branch for comparisons")
	ghaPerfCmd.Flags().String("csv", "", "Export detailed data to CSV file")
	ghaPerfCmd.Flags().StringP("job", "j", "", "Show step breakdown for specific job name")
	ghaPerfCmd.Flags().Bool("by-branch", false, "Group runs by branch and compare against base")
	ghaPerfCmd.Flags().Bool("cache-only", false, "Use cached data only, do not fetch new runs")
	ghaPerfCmd.Flags().Bool("no-cache", false, "Do not use or update the cache")
	ghaPerfCmd.Flags().Bool("list-workflows", false, "List available workflows and exit")
}

type ghaPerfFlags struct {
	repo, workflow, branch, compare, baseBranch, csvPath, jobFilter string
	limit, days                                                     int
	byBranch, cacheOnly, noCache, listWorkflows                     bool
}

func parseGHAPerfFlags(cmd *cobra.Command, cfg *config.Config) ghaPerfFlags {
	f := ghaPerfFlags{
		repo:          stringFlag(cmd, "repo"),
		workflow:      stringFlag(cmd, "workflow"),
		branch:        stringFlag(cmd, "branch"),
		limit:         intFlag(cmd, "limit"),
		days:          intFlag(cmd, "days"),
		compare:       stringFlag(cmd, "compare"),
		baseBranch:    stringFlag(cmd, "base-branch"),
		csvPath:       stringFlag(cmd, "csv"),
		jobFilter:     stringFlag(cmd, "job"),
		byBranch:      boolFlag(cmd, "by-branch"),
		cacheOnly:     boolFlag(cmd, "cache-only"),
		noCache:       boolFlag(cmd, "no-cache"),
		listWorkflows: boolFlag(cmd, "list-workflows"),
	}

	if f.repo == "" {
		if qualified := cfg.QualifiedRepos(); len(qualified) > 0 {
			f.repo = qualified[0]
		}
	}
	if !cmd.Flags().Changed("days") && cfg.GHAPerf.DefaultLookbackDays > 0 {
		f.days = cfg.GHAPerf.DefaultLookbackDays
	}
	if !cmd.Flags().Changed("base-branch") && cfg.GHAPerf.BaseBranch != "" {
		f.baseBranch = cfg.GHAPerf.BaseBranch
	}

	return f
}

func runGHAPerf(cmd *cobra.Command, _ []string) {
	cfg := loadConfig()
	f := parseGHAPerfFlags(cmd, cfg)

	if f.repo == "" {
		fmt.Println("Error: --repo flag is required or set repositories in .gh-sweep.yaml")
		return
	}

	owner, repoName, ok := splitRepo(f.repo)
	if !ok {
		fmt.Println("Error: repo must be in format owner/repo")
		return
	}

	client, err := github.NewClient(context.Background())
	if err != nil {
		fmt.Printf("Error: failed to create GitHub client: %v\n", err)
		return
	}

	if f.listWorkflows {
		printGHAPerfWorkflows(client, f.repo, owner, repoName)
		return
	}

	cacheManager, err := cache.NewGHAPerfCacheManager(cfg.GHAPerf.CachePath)
	if err != nil {
		fmt.Printf("Error: failed to create cache manager: %v\n", err)
		return
	}

	allRuns, cachedCount, newCount, err := loadGHAPerfRuns(client, cacheManager, owner, repoName, f)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\nTotal: %d runs (%d cached, %d new)\n", len(allRuns), cachedCount, newCount)

	if len(allRuns) == 0 {
		fmt.Println("No runs found")
		return
	}

	reportGHAPerfRuns(allRuns, f)
}

func printGHAPerfWorkflows(client *github.Client, repo, owner, repoName string) {
	workflows, err := client.ListWorkflows(owner, repoName)
	if err != nil {
		fmt.Printf("Error: failed to list workflows: %v\n", err)
		return
	}

	fmt.Printf("Workflows for %s:\n\n", repo)
	for _, w := range workflows {
		state := ""
		if w.State != "active" {
			state = fmt.Sprintf(" (%s)", w.State)
		}
		fmt.Printf("  %s%s\n", w.Path, state)
	}
}

func loadGHAPerfRuns(
	client *github.Client,
	cacheManager *cache.GHAPerfCacheManager,
	owner, repoName string,
	f ghaPerfFlags,
) ([]github.RunTiming, int, int, error) {
	var allRuns []github.RunTiming
	var cachedCount int

	if !f.noCache {
		cachedData, err := cacheManager.Load(owner, repoName)
		if err != nil {
			fmt.Printf("Warning: failed to load cache: %v\n", err)
		} else {
			cachedCount = len(cachedData.Runs)
			allRuns = cachedData.Runs
		}
	}

	if f.cacheOnly {
		return allRuns, cachedCount, 0, nil
	}

	allRuns, newCount, err := fetchNewGHAPerfRuns(client, cacheManager, owner, repoName, f, allRuns, cachedCount)
	if err != nil {
		return nil, 0, 0, err
	}

	return allRuns, cachedCount, newCount, nil
}

func fetchNewGHAPerfRuns(
	client *github.Client,
	cacheManager *cache.GHAPerfCacheManager,
	owner, repoName string,
	f ghaPerfFlags,
	allRuns []github.RunTiming,
	cachedCount int,
) ([]github.RunTiming, int, error) {
	cachedIDs := make(map[int]bool, len(allRuns))
	for i := range allRuns {
		cachedIDs[allRuns[i].RunID] = true
	}

	since := time.Now().AddDate(0, 0, -f.days)
	opts := github.FetchWorkflowRunsOptions{
		WorkflowFile: f.workflow,
		Branch:       f.branch,
		Limit:        f.limit,
		CreatedAfter: since,
	}

	if f.compare != "" {
		fmt.Println("Fetching runs for comparison...")
		opts.Branch = ""
	}

	fmt.Printf("Fetching workflow runs for %s...\n", f.repo)
	newRuns, err := client.FetchWorkflowRunsWithDetails(owner, repoName, opts)
	if err != nil {
		if cachedCount == 0 {
			return nil, 0, fmt.Errorf("failed to fetch workflow runs: %w", err)
		}
		fmt.Printf("Warning: failed to fetch new runs, using cache: %v\n", err)

		return allRuns, 0, nil
	}

	newCount := 0
	for i := range newRuns {
		if !cachedIDs[newRuns[i].RunID] {
			newCount++
		}
	}

	allRuns = cache.MergeRuns(allRuns, newRuns)

	if !f.noCache && newCount > 0 {
		cachedData := &cache.GHAPerfCache{Runs: allRuns}
		if err := cacheManager.Save(owner, repoName, cachedData); err != nil {
			fmt.Printf("Warning: failed to save cache: %v\n", err)
		} else {
			fmt.Printf("Cache saved: %d runs\n", len(allRuns))
		}
	}

	return allRuns, newCount, nil
}

func reportGHAPerfRuns(allRuns []github.RunTiming, f ghaPerfFlags) {
	since := time.Now().AddDate(0, 0, -f.days)
	allRuns = github.FilterRunsByTimeRange(allRuns, since, time.Time{})

	if f.branch != "" && f.compare == "" {
		allRuns = github.FilterRunsByBranch(allRuns, f.branch)
	}

	if f.csvPath != "" {
		if err := exportCSV(allRuns, f.csvPath); err != nil {
			fmt.Printf("Error: failed to export CSV: %v\n", err)
		} else {
			fmt.Printf("Exported to %s\n", f.csvPath)
		}
	}

	if f.compare != "" {
		currentRuns := github.FilterRunsByBranch(allRuns, f.compare)
		baseRuns := github.FilterRunsByBranch(allRuns, f.baseBranch)
		printComparison(currentRuns, baseRuns, f.compare, f.baseBranch)

		return
	}

	if f.byBranch {
		printByBranch(allRuns, f.baseBranch)
		return
	}

	printSummary(allRuns)
	printJobSummary(allRuns, f.jobFilter)
}

func exportCSV(runs []github.RunTiming, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", path, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close %s: %v\n", path, closeErr)
		}
	}()

	w := csv.NewWriter(file)
	defer w.Flush()

	header := []string{
		"run_id", "workflow", "branch", "conclusion", "created_at",
		"run_duration_s", "job_name", "job_duration_s", "step_name", "step_duration_s",
	}
	if err := w.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	for i := range runs {
		r := &runs[i]
		for j := range r.Jobs {
			job := &r.Jobs[j]
			for _, s := range job.Steps {
				row := []string{
					strconv.Itoa(r.RunID),
					r.Workflow,
					r.Branch,
					r.Conclusion,
					r.CreatedAt.Format(time.RFC3339),
					fmt.Sprintf("%.1f", r.DurationSeconds),
					job.Name,
					fmt.Sprintf("%.1f", job.DurationSeconds),
					s.Name,
					fmt.Sprintf("%.1f", s.DurationSeconds),
				}
				if err := w.Write(row); err != nil {
					return fmt.Errorf("failed to write CSV row: %w", err)
				}
			}
		}
	}

	return nil
}

func printSummary(runs []github.RunTiming) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", sectionRuleWidth))
	fmt.Println("WORKFLOW PERFORMANCE SUMMARY")
	fmt.Println(strings.Repeat("=", sectionRuleWidth))

	stats := github.ComputeWorkflowStats(runs)

	workflows := make([]*github.WorkflowStats, 0, len(stats))
	for _, s := range stats {
		workflows = append(workflows, s)
	}
	sort.Slice(workflows, func(i, j int) bool {
		return workflows[i].Workflow < workflows[j].Workflow
	})

	for _, s := range workflows {
		fmt.Printf("\n%s:\n", s.Workflow)
		fmt.Printf("  Runs: %d\n", s.TotalRuns)
		fmt.Printf("  Avg:  %s\n", github.FormatDuration(s.AvgDuration))
		fmt.Printf("  Min:  %s\n", github.FormatDuration(s.MinDuration))
		fmt.Printf("  Max:  %s\n", github.FormatDuration(s.MaxDuration))
		fmt.Printf("  Success: %.0f%%\n", s.SuccessRate)
	}
}

func printJobSummary(runs []github.RunTiming, jobFilter string) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", sectionRuleWidth))
	fmt.Printf("JOB PERFORMANCE SUMMARY (Top %d by avg duration)\n", topJobsCount)
	fmt.Println(strings.Repeat("=", sectionRuleWidth))

	stats := github.ComputeJobStats(runs)
	topJobs := github.GetTopJobsByDuration(stats, topJobsCount)

	for _, s := range topJobs {
		fmt.Printf("  %s: %s avg (%d runs)\n",
			truncate(s.WorkflowJob, jobNameMaxLen),
			github.FormatDuration(s.AvgDuration),
			s.TotalRuns)
	}

	if jobFilter != "" {
		printStepBreakdown(runs, jobFilter)
	}
}

func printStepBreakdown(runs []github.RunTiming, jobFilter string) {
	fmt.Println()
	fmt.Println(strings.Repeat("-", sectionRuleWidth))
	fmt.Printf("STEP BREAKDOWN FOR: %s\n", jobFilter)
	fmt.Println(strings.Repeat("-", sectionRuleWidth))

	stepStats := make(map[string][]time.Duration)
	for i := range runs {
		r := &runs[i]
		for j := range r.Jobs {
			job := &r.Jobs[j]
			if job.Name != jobFilter {
				continue
			}
			for _, s := range job.Steps {
				stepStats[s.Name] = append(stepStats[s.Name], s.Duration)
			}
		}
	}

	type stepAvg struct {
		name string
		avg  time.Duration
		runs int
	}

	steps := make([]stepAvg, 0, len(stepStats))
	for name, durations := range stepStats {
		var total time.Duration
		for _, d := range durations {
			total += d
		}
		steps = append(steps, stepAvg{name, total / time.Duration(len(durations)), len(durations)})
	}

	sort.Slice(steps, func(i, j int) bool {
		return steps[i].avg > steps[j].avg
	})

	for _, s := range steps {
		fmt.Printf("  %s: %s avg (%d runs)\n",
			truncate(s.name, stepNameMaxLen),
			github.FormatDuration(s.avg),
			s.runs)
	}
}

func printByBranch(runs []github.RunTiming, baseBranch string) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", branchSectionRuleWidth))
	fmt.Println("PERFORMANCE BY BRANCH")
	fmt.Println(strings.Repeat("=", branchSectionRuleWidth))

	stats := github.ComputeBranchStats(runs, baseBranch)

	for _, s := range sortedBranchStats(stats, baseBranch) {
		printBranchStats(s, baseBranch, stats)
	}
}

func sortedBranchStats(stats map[string]*github.BranchStats, baseBranch string) []*github.BranchStats {
	branches := make([]*github.BranchStats, 0, len(stats))
	for _, s := range stats {
		branches = append(branches, s)
	}

	sort.Slice(branches, func(i, j int) bool {
		if branches[i].Branch == baseBranch {
			return true
		}
		if branches[j].Branch == baseBranch {
			return false
		}

		return branches[i].Branch < branches[j].Branch
	})

	return branches
}

func printBranchStats(s *github.BranchStats, baseBranch string, stats map[string]*github.BranchStats) {
	isBase := s.Branch == baseBranch
	label := ""
	if isBase {
		label = "[BASE] "
	}

	fmt.Printf("\n%s%s (%d runs)\n", label, s.Branch, s.TotalRuns)
	fmt.Println(strings.Repeat("-", branchDividerWidth))

	for wf, ws := range s.WorkflowStats {
		fmt.Printf("  %s: %s avg (%d runs)%s\n",
			wf,
			github.FormatDuration(ws.AvgDuration),
			ws.TotalRuns,
			branchDelta(isBase, wf, ws, baseBranch, stats))
	}
}

func branchDelta(
	isBase bool,
	wf string,
	ws *github.WorkflowStats,
	baseBranch string,
	stats map[string]*github.BranchStats,
) string {
	if isBase || stats[baseBranch] == nil {
		return ""
	}

	baseWS, ok := stats[baseBranch].WorkflowStats[wf]
	if !ok {
		return ""
	}

	diff := ws.AvgDuration - baseWS.AvgDuration
	pct := float64(diff) / float64(baseWS.AvgDuration) * percentScale
	sign := "+"
	if pct < 0 {
		sign = ""
	}

	return fmt.Sprintf(" (%s%.0f%% vs %s)", sign, pct, baseBranch)
}

func printComparison(runsA, runsB []github.RunTiming, labelA, labelB string) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", branchSectionRuleWidth))
	fmt.Printf("BRANCH COMPARISON: %s vs %s\n", labelA, labelB)
	fmt.Println(strings.Repeat("=", branchSectionRuleWidth))

	statsA := github.ComputeWorkflowStats(runsA)
	statsB := github.ComputeWorkflowStats(runsB)

	allWorkflows := make(map[string]bool)
	for wf := range statsA {
		allWorkflows[wf] = true
	}
	for wf := range statsB {
		allWorkflows[wf] = true
	}

	workflows := make([]string, 0, len(allWorkflows))
	for wf := range allWorkflows {
		workflows = append(workflows, wf)
	}
	sort.Strings(workflows)

	for _, wf := range workflows {
		fmt.Printf("\n%s:\n", wf)
		printWorkflowComparison(statsA[wf], statsB[wf], labelA, labelB)
	}
}

func printWorkflowComparison(sA, sB *github.WorkflowStats, labelA, labelB string) {
	okA, okB := sA != nil, sB != nil

	switch {
	case okA && okB:
		diff := sA.AvgDuration - sB.AvgDuration
		pct := float64(diff) / float64(sB.AvgDuration) * percentScale

		indicator := "SAME"
		if diff < 0 {
			indicator = "FASTER"
		} else if diff > 0 {
			indicator = "SLOWER"
		}

		sign := "+"
		if pct < 0 {
			sign = ""
		}

		fmt.Printf("  %s: %s avg (%d runs)\n", labelA, github.FormatDuration(sA.AvgDuration), sA.TotalRuns)
		fmt.Printf("  %s: %s avg (%d runs)\n", labelB, github.FormatDuration(sB.AvgDuration), sB.TotalRuns)
		fmt.Printf("  Delta: %s%s (%s%.1f%%) - %s\n",
			sign, github.FormatDuration(abs(diff)), sign, pct, indicator)

	case okA:
		fmt.Printf("  %s: %s avg (%d runs)\n", labelA, github.FormatDuration(sA.AvgDuration), sA.TotalRuns)
		fmt.Printf("  %s: No data\n", labelB)

	default:
		fmt.Printf("  %s: No data\n", labelA)
		fmt.Printf("  %s: %s avg (%d runs)\n", labelB, github.FormatDuration(sB.AvgDuration), sB.TotalRuns)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen-3] + "..."
}

func abs(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}

	return d
}
