//go:build golden

package ghaperf

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
	m, _ = m.Update(dataLoadedMsg{
		runs: []github.RunTiming{
			{
				RunID:      101,
				Workflow:   "ci",
				Branch:     "main",
				Conclusion: "success",
				Duration:   5 * time.Minute,
			},
			{
				RunID:      102,
				Workflow:   "ci",
				Branch:     "feature/login",
				Conclusion: "failure",
				Duration:   7 * time.Minute,
			},
			{
				RunID:      103,
				Workflow:   "release",
				Branch:     "main",
				Conclusion: "success",
				Duration:   12 * time.Minute,
			},
		},
		workflows: []github.WorkflowFile{
			{ID: 1, Name: "ci", Path: ".github/workflows/ci.yml", State: "active"},
		},
		workflowStats: map[string]*github.WorkflowStats{
			"ci": {
				Workflow:    "ci",
				TotalRuns:   2,
				AvgDuration: 6 * time.Minute,
				MinDuration: 5 * time.Minute,
				MaxDuration: 7 * time.Minute,
				SuccessRate: 50,
			},
		},
		jobStats: map[string]*github.JobStats{
			"ci/test": {
				WorkflowJob: "ci/test",
				TotalRuns:   2,
				AvgDuration: 4 * time.Minute,
				MinDuration: 3 * time.Minute,
				MaxDuration: 5 * time.Minute,
			},
		},
		branchStats: map[string]*github.BranchStats{
			"main": {Branch: "main", TotalRuns: 2, AvgDuration: 8 * time.Minute},
		},
		cachedCount: 2,
		newCount:    1,
	})

	golden.RequireEqual(t, []byte(m.View()))
}
