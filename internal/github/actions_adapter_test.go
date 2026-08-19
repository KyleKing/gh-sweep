package github_test

import (
	"testing"
	"time"

	"github.com/KyleKing/gh-sweep/internal/github"
)

func TestRunsToTestRuns(t *testing.T) {
	t.Parallel()

	now := time.Now()
	runs := []github.RunTiming{
		{
			RunID:      1,
			Workflow:   "CI",
			WorkflowID: 10,
			Conclusion: "success",
			HeadSHA:    "abc",
			CreatedAt:  now,
			Duration:   time.Minute,
		},
		{
			RunID: 2, Workflow: "CI", WorkflowID: 10,
			Conclusion: "failure", HeadSHA: "abc", CreatedAt: now.Add(time.Hour),
		},
		{RunID: 3, Workflow: "Deploy", WorkflowID: 20, Conclusion: "skipped", HeadSHA: "def", CreatedAt: now},
		{RunID: 4, Workflow: "CI", WorkflowID: 10, Conclusion: "timed_out", HeadSHA: "def", CreatedAt: now},
		{RunID: 5, Workflow: "CI", WorkflowID: 10, Conclusion: "", HeadSHA: "def", CreatedAt: now},
	}

	testRuns := github.RunsToTestRuns("owner/repo", runs)

	if len(testRuns) != 3 {
		t.Fatalf("expected 3 test runs, got %d", len(testRuns))
	}

	first := testRuns[0]
	if first.Name != "CI" || first.Status != "success" || first.CommitSHA != "abc" {
		t.Errorf("unexpected first test run: %+v", first)
	}
	if first.Repository != "owner/repo" {
		t.Errorf("expected repository owner/repo, got %s", first.Repository)
	}
	if first.WorkflowID != 10 {
		t.Errorf("expected workflow ID 10, got %d", first.WorkflowID)
	}
	if first.Duration != time.Minute {
		t.Errorf("expected duration 1m, got %s", first.Duration)
	}

	if testRuns[2].Status != "skipped" {
		t.Errorf("expected skipped status preserved, got %s", testRuns[2].Status)
	}
}

func TestRunsToTestRunsEmpty(t *testing.T) {
	t.Parallel()

	testRuns := github.RunsToTestRuns("owner/repo", nil)
	if len(testRuns) != 0 {
		t.Fatalf("expected no test runs, got %d", len(testRuns))
	}
}
