package analytics

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
)

func loadedAnalyticsModel() Model {
	m := NewModel("acme/widgets")
	m, _ = m.Update(analyticsLoadedMsg{
		stats: &github.WorkflowRunStats{
			TotalRuns:    20,
			SuccessRate:  85.0,
			FailureCount: 3,
			AvgDuration:  4*time.Minute + 30*time.Second,
		},
		runs: []github.WorkflowRun{
			{ID: 1, Name: "ci", Conclusion: "success"},
			{ID: 2, Name: "ci", Conclusion: "failure"},
		},
		flaky: []github.FlakyTest{
			{Name: "ci", FailureRate: 0.15, FailureCount: 3, TotalRuns: 20, Pattern: "intermittent"},
		},
	})

	return m
}

func TestLoadedOverview(t *testing.T) {
	t.Parallel()

	view := loadedAnalyticsModel().View()
	for _, want := range []string{"Analytics: acme/widgets", "Total Runs:     20", "Success Rate:   85.0%"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q", want)
		}
	}
}

func TestFlakyViewSwitch(t *testing.T) {
	t.Parallel()

	m := loadedAnalyticsModel()
	m, _ = m.Update(tea.KeyPressMsg{Code: '2', Text: "2"})

	view := m.View()
	if !strings.Contains(view, "Flaky Test Detection") || !strings.Contains(view, "intermittent") {
		t.Errorf("flaky view = %q", view)
	}
}

func TestLoadErrorRendered(t *testing.T) {
	t.Parallel()

	m := NewModel("acme/widgets")
	m, _ = m.Update(analyticsLoadedMsg{err: errors.New("boom")})

	if !strings.Contains(m.View(), "boom") {
		t.Errorf("view = %q", m.View())
	}
}

func TestSplitRepo(t *testing.T) {
	t.Parallel()

	if _, _, err := splitRepo(""); err == nil {
		t.Error("expected error for empty repo")
	}

	if _, _, err := splitRepo("acme"); err == nil {
		t.Error("expected error for missing slash")
	}

	owner, repo, err := splitRepo("acme/widgets")
	if err != nil || owner != "acme" || repo != "widgets" {
		t.Errorf("splitRepo = %q/%q err=%v", owner, repo, err)
	}
}
