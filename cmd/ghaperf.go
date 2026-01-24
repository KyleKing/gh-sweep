package cmd

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/KyleKing/gh-sweep/internal/cache"
	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/spf13/cobra"
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
	ghaPerfCmd.Flags().IntP("limit", "l", 30, "Number of runs to fetch")
	ghaPerfCmd.Flags().Int("days", 30, "Lookback period in days")
	ghaPerfCmd.Flags().StringP("compare", "c", "", "Compare current runs against another branch")
	ghaPerfCmd.Flags().String("base-branch", "main", "Base branch for comparisons")
	ghaPerfCmd.Flags().String("csv", "", "Export detailed data to CSV file")
	ghaPerfCmd.Flags().StringP("job", "j", "", "Show step breakdown for specific job name")
	ghaPerfCmd.Flags().Bool("by-branch", false, "Group runs by branch and compare against base")
	ghaPerfCmd.Flags().Bool("cache-only", false, "Use cached data only, do not fetch new runs")
	ghaPerfCmd.Flags().Bool("no-cache", false, "Do not use or update the cache")
	ghaPerfCmd.Flags().Bool("list-workflows", false, "List available workflows and exit")
}

func runGHAPerf(cmd *cobra.Command, _ []string) {
	repo, _ := cmd.Flags().GetString("repo")
	workflow, _ := cmd.Flags().GetString("workflow")
	branch, _ := cmd.Flags().GetString("branch")
	limit, _ := cmd.Flags().GetInt("limit")
	days, _ := cmd.Flags().GetInt("days")
	compare, _ := cmd.Flags().GetString("compare")
	baseBranch, _ := cmd.Flags().GetString("base-branch")
	csvPath, _ := cmd.Flags().GetString("csv")
	jobFilter, _ := cmd.Flags().GetString("job")
	byBranch, _ := cmd.Flags().GetBool("by-branch")
	cacheOnly, _ := cmd.Flags().GetBool("cache-only")
	noCache, _ := cmd.Flags().GetBool("no-cache")
	listWorkflows, _ := cmd.Flags().GetBool("list-workflows")

	if repo == "" {
		fmt.Println("Error: --repo flag is required")
		return
	}

	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		fmt.Println("Error: repo must be in format owner/repo")
		return
	}
	owner, repoName := parts[0], parts[1]

	ctx := context.Background()
	client, err := github.NewClient(ctx)
	if err != nil {
		fmt.Printf("Error: failed to create GitHub client: %v\n", err)
		return
	}

	if listWorkflows {
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
		return
	}

	cacheManager, err := cache.NewGHAPerfCacheManager("")
	if err != nil {
		fmt.Printf("Error: failed to create cache manager: %v\n", err)
		return
	}

	var allRuns []github.RunTiming
	var cachedCount, newCount int

	if !noCache {
		cachedData, err := cacheManager.Load(owner, repoName)
		if err != nil {
			fmt.Printf("Warning: failed to load cache: %v\n", err)
		} else {
			cachedCount = len(cachedData.Runs)
			allRuns = cachedData.Runs
		}
	}

	if !cacheOnly {
		cachedIDs := make(map[int]bool)
		for _, r := range allRuns {
			cachedIDs[r.RunID] = true
		}

		since := time.Now().AddDate(0, 0, -days)
		opts := github.FetchWorkflowRunsOptions{
			WorkflowFile: workflow,
			Branch:       branch,
			Limit:        limit,
			CreatedAfter: since,
		}

		if compare != "" {
			fmt.Printf("Fetching runs for comparison...\n")
			opts.Branch = ""
		}

		fmt.Printf("Fetching workflow runs for %s...\n", repo)
		newRuns, err := client.FetchWorkflowRunsWithDetails(owner, repoName, opts)
		if err != nil {
			if cachedCount > 0 {
				fmt.Printf("Warning: failed to fetch new runs, using cache: %v\n", err)
			} else {
				fmt.Printf("Error: failed to fetch workflow runs: %v\n", err)
				return
			}
		} else {
			var actuallyNew []github.RunTiming
			for _, r := range newRuns {
				if !cachedIDs[r.RunID] {
					actuallyNew = append(actuallyNew, r)
				}
			}
			newCount = len(actuallyNew)

			allRuns = cacheManager.MergeRuns(allRuns, newRuns)

			if !noCache && newCount > 0 {
				cachedData := &cache.GHAPerfCache{Runs: allRuns}
				if err := cacheManager.Save(owner, repoName, cachedData); err != nil {
					fmt.Printf("Warning: failed to save cache: %v\n", err)
				} else {
					fmt.Printf("Cache saved: %d runs\n", len(allRuns))
				}
			}
		}
	}

	fmt.Printf("\nTotal: %d runs (%d cached, %d new)\n", len(allRuns), cachedCount, newCount)

	if len(allRuns) == 0 {
		fmt.Println("No runs found")
		return
	}

	since := time.Now().AddDate(0, 0, -days)
	allRuns = github.FilterRunsByTimeRange(allRuns, since, time.Time{})

	if branch != "" && compare == "" {
		allRuns = github.FilterRunsByBranch(allRuns, branch)
	}

	if csvPath != "" {
		if err := exportCSV(allRuns, csvPath); err != nil {
			fmt.Printf("Error: failed to export CSV: %v\n", err)
		} else {
			fmt.Printf("Exported to %s\n", csvPath)
		}
	}

	if compare != "" {
		currentRuns := github.FilterRunsByBranch(allRuns, compare)
		baseRuns := github.FilterRunsByBranch(allRuns, baseBranch)
		printComparison(currentRuns, baseRuns, compare, baseBranch)
		return
	}

	if byBranch {
		printByBranch(allRuns, baseBranch)
		return
	}

	printSummary(allRuns)
	printJobSummary(allRuns, jobFilter)
}

func exportCSV(runs []github.RunTiming, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{
		"run_id", "workflow", "branch", "conclusion", "created_at",
		"run_duration_s", "job_name", "job_duration_s", "step_name", "step_duration_s",
	}
	if err := w.Write(header); err != nil {
		return err
	}

	for _, r := range runs {
		for _, j := range r.Jobs {
			for _, s := range j.Steps {
				row := []string{
					fmt.Sprintf("%d", r.RunID),
					r.Workflow,
					r.Branch,
					r.Conclusion,
					r.CreatedAt.Format(time.RFC3339),
					fmt.Sprintf("%.1f", r.DurationSeconds),
					j.Name,
					fmt.Sprintf("%.1f", j.DurationSeconds),
					s.Name,
					fmt.Sprintf("%.1f", s.DurationSeconds),
				}
				if err := w.Write(row); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func printSummary(runs []github.RunTiming) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("WORKFLOW PERFORMANCE SUMMARY")
	fmt.Println(strings.Repeat("=", 60))

	stats := github.ComputeWorkflowStats(runs)

	var workflows []*github.WorkflowStats
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
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("JOB PERFORMANCE SUMMARY (Top 10 by avg duration)")
	fmt.Println(strings.Repeat("=", 60))

	stats := github.ComputeJobStats(runs)
	topJobs := github.GetTopJobsByDuration(stats, 10)

	for _, s := range topJobs {
		fmt.Printf("  %s: %s avg (%d runs)\n",
			truncate(s.WorkflowJob, 50),
			github.FormatDuration(s.AvgDuration),
			s.TotalRuns)
	}

	if jobFilter != "" {
		fmt.Println()
		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("STEP BREAKDOWN FOR: %s\n", jobFilter)
		fmt.Println(strings.Repeat("-", 60))

		stepStats := make(map[string][]time.Duration)
		for _, r := range runs {
			for _, j := range r.Jobs {
				if j.Name != jobFilter {
					continue
				}
				for _, s := range j.Steps {
					stepStats[s.Name] = append(stepStats[s.Name], s.Duration)
				}
			}
		}

		type stepAvg struct {
			name string
			avg  time.Duration
			runs int
		}
		var steps []stepAvg
		for name, durations := range stepStats {
			var total time.Duration
			for _, d := range durations {
				total += d
			}
			avg := total / time.Duration(len(durations))
			steps = append(steps, stepAvg{name, avg, len(durations)})
		}

		sort.Slice(steps, func(i, j int) bool {
			return steps[i].avg > steps[j].avg
		})

		for _, s := range steps {
			fmt.Printf("  %s: %s avg (%d runs)\n",
				truncate(s.name, 40),
				github.FormatDuration(s.avg),
				s.runs)
		}
	}
}

func printByBranch(runs []github.RunTiming, baseBranch string) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("PERFORMANCE BY BRANCH")
	fmt.Println(strings.Repeat("=", 70))

	stats := github.ComputeBranchStats(runs, baseBranch)

	var branches []*github.BranchStats
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

	for _, s := range branches {
		isBase := s.Branch == baseBranch
		label := ""
		if isBase {
			label = "[BASE] "
		}

		fmt.Printf("\n%s%s (%d runs)\n", label, s.Branch, s.TotalRuns)
		fmt.Println(strings.Repeat("-", 50))

		for wf, ws := range s.WorkflowStats {
			delta := ""
			if !isBase && stats[baseBranch] != nil {
				if baseWS, ok := stats[baseBranch].WorkflowStats[wf]; ok {
					diff := ws.AvgDuration - baseWS.AvgDuration
					pct := float64(diff) / float64(baseWS.AvgDuration) * 100
					sign := "+"
					if pct < 0 {
						sign = ""
					}
					delta = fmt.Sprintf(" (%s%.0f%% vs %s)", sign, pct, baseBranch)
				}
			}

			fmt.Printf("  %s: %s avg (%d runs)%s\n",
				wf,
				github.FormatDuration(ws.AvgDuration),
				ws.TotalRuns,
				delta)
		}
	}
}

func printComparison(runsA, runsB []github.RunTiming, labelA, labelB string) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("BRANCH COMPARISON: %s vs %s\n", labelA, labelB)
	fmt.Println(strings.Repeat("=", 70))

	statsA := github.ComputeWorkflowStats(runsA)
	statsB := github.ComputeWorkflowStats(runsB)

	allWorkflows := make(map[string]bool)
	for wf := range statsA {
		allWorkflows[wf] = true
	}
	for wf := range statsB {
		allWorkflows[wf] = true
	}

	var workflows []string
	for wf := range allWorkflows {
		workflows = append(workflows, wf)
	}
	sort.Strings(workflows)

	for _, wf := range workflows {
		fmt.Printf("\n%s:\n", wf)

		sA, okA := statsA[wf]
		sB, okB := statsB[wf]

		if okA && okB {
			diff := sA.AvgDuration - sB.AvgDuration
			pct := float64(diff) / float64(sB.AvgDuration) * 100

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
		} else if okA {
			fmt.Printf("  %s: %s avg (%d runs)\n", labelA, github.FormatDuration(sA.AvgDuration), sA.TotalRuns)
			fmt.Printf("  %s: No data\n", labelB)
		} else {
			fmt.Printf("  %s: No data\n", labelA)
			fmt.Printf("  %s: %s avg (%d runs)\n", labelB, github.FormatDuration(sB.AvgDuration), sB.TotalRuns)
		}
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
