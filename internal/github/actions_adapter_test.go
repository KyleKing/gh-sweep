package github

import (
	"testing"
	"time"
)

func TestWorkflowRunsToTestRuns(t *testing.T) {
	now := time.Now()
	runs := []WorkflowRun{
		{
			ID:         1,
			Name:       "CI",
			Conclusion: "success",
			HeadSHA:    "abc",
			CreatedAt:  now,
			Duration:   time.Minute,
		},
		{ID: 2, Name: "CI", Conclusion: "failure", HeadSHA: "abc", CreatedAt: now.Add(time.Hour)},
		{ID: 3, Name: "Deploy", Conclusion: "skipped", HeadSHA: "def", CreatedAt: now},
		{ID: 4, Name: "CI", Conclusion: "timed_out", HeadSHA: "def", CreatedAt: now},
		{ID: 5, Name: "CI", Conclusion: "", HeadSHA: "def", CreatedAt: now},
	}

	testRuns := WorkflowRunsToTestRuns("owner/repo", runs)

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
	if first.WorkflowID != 1 {
		t.Errorf("expected workflow ID 1, got %d", first.WorkflowID)
	}
	if first.Duration != time.Minute {
		t.Errorf("expected duration 1m, got %s", first.Duration)
	}

	if testRuns[2].Status != "skipped" {
		t.Errorf("expected skipped status preserved, got %s", testRuns[2].Status)
	}
}

func TestWorkflowRunsToTestRunsEmpty(t *testing.T) {
	testRuns := WorkflowRunsToTestRuns("owner/repo", nil)
	if len(testRuns) != 0 {
		t.Fatalf("expected no test runs, got %d", len(testRuns))
	}
}
