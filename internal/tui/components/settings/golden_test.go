//go:build golden

package settings

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/golden"

	"github.com/KyleKing/gh-sweep/internal/github"
)

func TestGoldenLoadedView(t *testing.T) {
	t.Parallel()

	m := NewModel([]string{"acme/widgets", "acme/gadgets"}, "acme/widgets")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(settingsLoadedMsg{
		settings: map[string]*github.RepoSettings{
			"acme/widgets": {
				Repository:          "acme/widgets",
				DefaultBranch:       "main",
				AllowSquashMerge:    true,
				DeleteBranchOnMerge: true,
				HasIssues:           true,
			},
			"acme/gadgets": {
				Repository:       "acme/gadgets",
				DefaultBranch:    "master",
				AllowMergeCommit: true,
				HasIssues:        true,
				HasWiki:          true,
			},
		},
		diffs: map[string][]github.SettingsDiff{
			"acme/gadgets": {
				{Field: "DefaultBranch", Baseline: "main", Current: "master", Severity: "warning"},
				{Field: "DeleteBranchOnMerge", Baseline: true, Current: false, Severity: "info"},
			},
		},
	})

	golden.RequireEqual(t, []byte(m.View()))
}
