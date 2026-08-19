package github

import (
	"fmt"
	"time"
)

// Terminal conclusion values reported by the GitHub Actions API.
const (
	ConclusionSuccess = "success"
	ConclusionFailure = "failure"
	ConclusionSkipped = "skipped"
)

// WorkflowRun represents a GitHub Actions workflow run.
type WorkflowRun struct {
	ID         int
	Name       string
	Status     string
	Conclusion string
	Branch     string
	HeadSHA    string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Duration   time.Duration
}

type workflowRunsResponse struct {
	WorkflowRuns []struct {
		ID         int       `json:"id"`
		Name       string    `json:"name"`
		Status     string    `json:"status"`
		Conclusion string    `json:"conclusion"`
		HeadBranch string    `json:"head_branch"`
		HeadSHA    string    `json:"head_sha"`
		CreatedAt  time.Time `json:"created_at"`
		UpdatedAt  time.Time `json:"updated_at"`
	} `json:"workflow_runs"`
}

// ListWorkflowRuns lists workflow runs for a repository.
func (c *Client) ListWorkflowRuns(owner, repo string) ([]WorkflowRun, error) {
	var response workflowRunsResponse
	path := fmt.Sprintf("repos/%s/%s/actions/runs", owner, repo)

	if err := c.Get(path, &response); err != nil {
		return nil, fmt.Errorf("failed to list workflow runs: %w", err)
	}

	runs := make([]WorkflowRun, len(response.WorkflowRuns))
	for i := range response.WorkflowRuns {
		r := &response.WorkflowRuns[i]
		runs[i] = WorkflowRun{
			ID:         r.ID,
			Name:       r.Name,
			Status:     r.Status,
			Conclusion: r.Conclusion,
			Branch:     r.HeadBranch,
			HeadSHA:    r.HeadSHA,
			CreatedAt:  r.CreatedAt,
			UpdatedAt:  r.UpdatedAt,
			Duration:   r.UpdatedAt.Sub(r.CreatedAt),
		}
	}

	return runs, nil
}

// WorkflowRunsToTestRuns adapts workflow runs for flaky detection, treating
// each workflow as a test keyed by its name. Runs without a terminal
// success/failure/skipped conclusion are dropped.
func WorkflowRunsToTestRuns(repo string, runs []WorkflowRun) []TestRun {
	testRuns := make([]TestRun, 0, len(runs))

	for i := range runs {
		run := &runs[i]
		switch run.Conclusion {
		case ConclusionSuccess, ConclusionFailure, ConclusionSkipped:
		default:
			continue
		}

		testRuns = append(testRuns, TestRun{
			Name:       run.Name,
			Status:     run.Conclusion,
			CommitSHA:  run.HeadSHA,
			Timestamp:  run.CreatedAt,
			Duration:   run.Duration,
			Repository: repo,
			WorkflowID: run.ID,
		})
	}

	return testRuns
}

// WorkflowRunStats represents statistics about workflow runs.
type WorkflowRunStats struct {
	TotalRuns    int
	SuccessRate  float64
	FailureCount int
	AvgDuration  time.Duration
	Runs         []WorkflowRun
}

// AnalyzeWorkflowRuns analyzes workflow runs and returns statistics.
func AnalyzeWorkflowRuns(runs []WorkflowRun) WorkflowRunStats {
	stats := WorkflowRunStats{
		TotalRuns: len(runs),
		Runs:      runs,
	}

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
