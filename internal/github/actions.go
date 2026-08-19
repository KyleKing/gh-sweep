package github

import "time"

// Terminal conclusion values reported by the GitHub Actions API.
const (
	ConclusionSuccess = "success"
	ConclusionFailure = "failure"
	ConclusionSkipped = "skipped"
)

// RunsToTestRuns adapts workflow runs for flaky detection, treating each
// workflow as a test keyed by its name. Runs without a terminal
// success/failure/skipped conclusion are dropped.
func RunsToTestRuns(repo string, runs []RunTiming) []TestRun {
	testRuns := make([]TestRun, 0, len(runs))

	for i := range runs {
		run := &runs[i]
		switch run.Conclusion {
		case ConclusionSuccess, ConclusionFailure, ConclusionSkipped:
		default:
			continue
		}

		testRuns = append(testRuns, TestRun{
			Name:       run.Workflow,
			Status:     run.Conclusion,
			CommitSHA:  run.HeadSHA,
			Timestamp:  run.CreatedAt,
			Duration:   run.Duration,
			Repository: repo,
			WorkflowID: run.WorkflowID,
		})
	}

	return testRuns
}

// AnalyzeRuns aggregates timing and success rate across all of the given
// runs, regardless of which workflow each belongs to.
func AnalyzeRuns(runs []RunTiming) WorkflowStats {
	stats := WorkflowStats{TotalRuns: len(runs)}

	if len(runs) == 0 {
		return stats
	}

	successCount := 0
	var totalDuration time.Duration

	for i := range runs {
		run := &runs[i]
		switch run.Conclusion {
		case ConclusionSuccess:
			successCount++
		case ConclusionFailure:
			stats.FailureCount++
		}
		totalDuration += run.Duration
	}

	stats.SuccessRate = float64(successCount) / float64(len(runs)) * percentMultiplier
	stats.AvgDuration = totalDuration / time.Duration(len(runs))

	return stats
}
