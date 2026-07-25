//go:build golden

package analytics

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/golden"

	"github.com/KyleKing/gh-sweep/internal/github"
)

func TestGoldenLoadedView(t *testing.T) {
	t.Parallel()

	m := NewModel("acme/widgets")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(analyticsLoadedMsg{
		stats: &github.WorkflowRunStats{
			TotalRuns:    20,
			SuccessRate:  85.0,
			FailureCount: 3,
			AvgDuration:  4*time.Minute + 30*time.Second,
		},
		runs: []github.WorkflowRun{
			{ID: 1, Name: "ci", Conclusion: "success", Branch: "main"},
			{ID: 2, Name: "ci", Conclusion: "failure", Branch: "main"},
		},
		flaky: []github.FlakyTest{
			{
				Name:         "ci",
				FailureRate:  0.15,
				FailureCount: 3,
				TotalRuns:    20,
				FlipCount:    4,
				Pattern:      "intermittent",
				LastFlip:     time.Date(2026, 1, 12, 9, 30, 0, 0, time.UTC),
			},
		},
	})

	golden.RequireEqual(t, []byte(m.View()))
}
