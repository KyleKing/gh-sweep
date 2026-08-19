package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/KyleKing/gh-sweep/internal/github"
)

func TestBuildThreadFilter(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}
	cmd.Flags().String("author", "", "")
	cmd.Flags().String("since", "", "")
	cmd.Flags().String("search", "", "")

	if err := cmd.Flags().Set("author", "octocat"); err != nil {
		t.Fatalf("failed to set author flag: %v", err)
	}
	if err := cmd.Flags().Set("search", "TODO"); err != nil {
		t.Fatalf("failed to set search flag: %v", err)
	}
	if err := cmd.Flags().Set("since", "2026-01-01"); err != nil {
		t.Fatalf("failed to set since flag: %v", err)
	}

	filter, err := buildThreadFilter(cmd)
	if err != nil {
		t.Fatalf("buildThreadFilter() error = %v", err)
	}

	if filter.Author != "octocat" || filter.Search != "TODO" {
		t.Errorf("filter = %+v", filter)
	}
	if filter.Since == nil || !filter.Since.Equal(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("filter.Since = %v", filter.Since)
	}
}

func TestBuildThreadFilterInvalidSince(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}
	cmd.Flags().String("author", "", "")
	cmd.Flags().String("since", "", "")
	cmd.Flags().String("search", "", "")

	if err := cmd.Flags().Set("since", "not-a-date"); err != nil {
		t.Fatalf("failed to set since flag: %v", err)
	}

	if _, err := buildThreadFilter(cmd); err == nil {
		t.Error("expected an error for an invalid --since date")
	}
}

func TestSummarize(t *testing.T) {
	t.Parallel()

	if got := summarize("short body"); got != "short body" {
		t.Errorf("summarize(short) = %q", got)
	}

	long := strings.Repeat("word ", 30)
	got := summarize(long)
	if !strings.HasSuffix(got, "...") || len(got) != 103 {
		t.Errorf("summarize(long) = %q (len %d)", got, len(got))
	}
}

//nolint:paralleltest // captureStdout swaps the process-wide os.Stdout.
func TestPrintThreads(t *testing.T) {
	threads := []github.ReviewThread{{
		PRNumber: 7,
		PRTitle:  "Add login",
		Path:     "auth/login.go",
		Comments: []github.ReviewComment{
			{Author: "octocat", Body: "please add a test", CreatedAt: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)},
			{Author: "octocat", Body: "reply"},
		},
	}}

	output := captureStdout(t, func() {
		printThreads("acme/widgets", threads)
	})

	for _, want := range []string{
		"Unresolved review threads: acme/widgets",
		"PR #7: Add login",
		"auth/login.go",
		"@octocat (2026-01-05): please add a test",
		"1 replies",
		"Total: 1 unresolved threads",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q, got:\n%s", want, output)
		}
	}

	empty := captureStdout(t, func() {
		printThreads("acme/widgets", nil)
	})
	if !strings.Contains(empty, "No unresolved review threads found.") {
		t.Errorf("expected the empty-state message, got:\n%s", empty)
	}
}
