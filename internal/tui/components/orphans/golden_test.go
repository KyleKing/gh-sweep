//go:build golden

package orphans

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/golden"

	"github.com/KyleKing/gh-sweep/internal/github"
	orphanscore "github.com/KyleKing/gh-sweep/internal/orphans"
)

func goldenScanResult() *orphanscore.NamespaceScanResult {
	prNumber := 42

	return &orphanscore.NamespaceScanResult{
		Namespace:    "acme",
		IsOrg:        true,
		TotalRepos:   2,
		TotalOrphans: 3,
		Results: []orphanscore.ScanResult{
			{
				Repository:    github.Repository{Owner: "acme", Name: "widgets"},
				DefaultBranch: "main",
				Orphans: []orphanscore.OrphanedBranch{
					{
						Repository:        "acme/widgets",
						BranchName:        "feature/login",
						Type:              orphanscore.OrphanTypeMergedPR,
						PRNumber:          &prNumber,
						DaysSinceActivity: 12,
						LastCommitDate:    time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC),
					},
					{
						Repository:        "acme/widgets",
						BranchName:        "spike/cache",
						Type:              orphanscore.OrphanTypeStale,
						DaysSinceActivity: 60,
						LastCommitDate:    time.Date(2025, 11, 20, 10, 0, 0, 0, time.UTC),
					},
				},
			},
			{
				Repository:    github.Repository{Owner: "acme", Name: "gadgets"},
				DefaultBranch: "main",
				Orphans: []orphanscore.OrphanedBranch{
					{
						Repository:        "acme/gadgets",
						BranchName:        "fix/typo",
						Type:              orphanscore.OrphanTypeClosedPR,
						DaysSinceActivity: 30,
						LastCommitDate:    time.Date(2025, 12, 20, 10, 0, 0, 0, time.UTC),
					},
				},
			},
		},
	}
}

func TestGoldenLoadedView(t *testing.T) {
	t.Parallel()

	m := NewModel("acme", orphanscore.DefaultScanOptions())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(scanCompleteMsg{result: goldenScanResult()})

	golden.RequireEqual(t, []byte(m.View()))
}

func TestGoldenConfirmDeleteView(t *testing.T) {
	t.Parallel()

	m := NewModel("acme", orphanscore.DefaultScanOptions())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(scanCompleteMsg{result: goldenScanResult()})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})

	golden.RequireEqual(t, []byte(m.View()))
}
