package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/KyleKing/gh-sweep/internal/github"
)

func TestTruncate(t *testing.T) {
	t.Parallel()

	if got := truncate("short", 10); got != "short" {
		t.Errorf("truncate(short) = %q", got)
	}
	if got := truncate("a very long workflow job name", 10); got != "a very ..." {
		t.Errorf("truncate(long) = %q", got)
	}
}

func TestAbs(t *testing.T) {
	t.Parallel()

	if got := abs(-5 * time.Second); got != 5*time.Second {
		t.Errorf("abs(-5s) = %v", got)
	}
	if got := abs(5 * time.Second); got != 5*time.Second {
		t.Errorf("abs(5s) = %v", got)
	}
}

func ghaPerfFixtureRuns() []github.RunTiming {
	return []github.RunTiming{
		{
			RunID: 1, Workflow: "ci", Branch: "main", CreatedAt: time.Now(),
			DurationSeconds: 60, Duration: time.Minute,
			Jobs: []github.JobTiming{{
				Name: "build", DurationSeconds: 30, Duration: 30 * time.Second,
				Steps: []github.StepTiming{{Name: "checkout", DurationSeconds: 5, Duration: 5 * time.Second}},
			}},
		},
		{
			RunID: 2, Workflow: "ci", Branch: "feature", CreatedAt: time.Now(),
			DurationSeconds: 90, Duration: 90 * time.Second,
			Jobs: []github.JobTiming{{
				Name: "build", DurationSeconds: 45, Duration: 45 * time.Second,
				Steps: []github.StepTiming{{Name: "checkout", DurationSeconds: 6, Duration: 6 * time.Second}},
			}},
		},
	}
}

func TestExportCSV(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "runs.csv")

	if err := exportCSV(ghaPerfFixtureRuns(), path); err != nil {
		t.Fatalf("exportCSV() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read exported CSV: %v", err)
	}

	if !strings.Contains(string(data), "run_id,workflow,branch") {
		t.Errorf("CSV missing header, got:\n%s", data)
	}
	if !strings.Contains(string(data), "checkout") {
		t.Errorf("CSV missing step row, got:\n%s", data)
	}
}

//nolint:paralleltest // captureStdout swaps the process-wide os.Stdout.
func TestPrintSummary(t *testing.T) {
	output := captureStdout(t, func() {
		printSummary(ghaPerfFixtureRuns())
	})

	if !strings.Contains(output, "WORKFLOW PERFORMANCE SUMMARY") || !strings.Contains(output, "ci:") {
		t.Errorf("output missing summary content, got:\n%s", output)
	}
}

//nolint:paralleltest // captureStdout swaps the process-wide os.Stdout.
func TestPrintJobSummary(t *testing.T) {
	output := captureStdout(t, func() {
		printJobSummary(ghaPerfFixtureRuns(), "build")
	})

	if !strings.Contains(output, "JOB PERFORMANCE SUMMARY") || !strings.Contains(output, "build") {
		t.Errorf("output missing job summary, got:\n%s", output)
	}
	if !strings.Contains(output, "STEP BREAKDOWN FOR: build") || !strings.Contains(output, "checkout") {
		t.Errorf("output missing step breakdown, got:\n%s", output)
	}
}

//nolint:paralleltest // captureStdout swaps the process-wide os.Stdout.
func TestPrintByBranch(t *testing.T) {
	output := captureStdout(t, func() {
		printByBranch(ghaPerfFixtureRuns(), "main")
	})

	if !strings.Contains(output, "PERFORMANCE BY BRANCH") || !strings.Contains(output, "[BASE] main") {
		t.Errorf("output missing base branch section, got:\n%s", output)
	}
	if !strings.Contains(output, "feature") || !strings.Contains(output, "vs main") {
		t.Errorf("output missing comparison branch delta, got:\n%s", output)
	}
}

//nolint:paralleltest // captureStdout swaps the process-wide os.Stdout.
func TestPrintComparison(t *testing.T) {
	runsA := ghaPerfFixtureRuns()[:1]
	runsB := ghaPerfFixtureRuns()[1:]

	output := captureStdout(t, func() {
		printComparison(runsA, runsB, "main", "feature")
	})

	if !strings.Contains(output, "BRANCH COMPARISON: main vs feature") {
		t.Errorf("output missing comparison header, got:\n%s", output)
	}
	if !strings.Contains(output, "Delta:") {
		t.Errorf("output missing delta line, got:\n%s", output)
	}
}
