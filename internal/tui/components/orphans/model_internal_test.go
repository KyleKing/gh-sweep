package orphans

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
	orphanscore "github.com/KyleKing/gh-sweep/internal/orphans"
)

func scanFixture() *orphanscore.NamespaceScanResult {
	prNumber := 42

	return &orphanscore.NamespaceScanResult{
		Namespace:    "acme",
		TotalRepos:   2,
		TotalOrphans: 3,
		Results: []orphanscore.ScanResult{
			{
				Repository: github.Repository{Owner: "acme", Name: "widgets"},
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
				Repository: github.Repository{Owner: "acme", Name: "gadgets"},
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

func loadedModel(t *testing.T) Model {
	t.Helper()

	m := NewModel("acme", orphanscore.DefaultScanOptions())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(scanCompleteMsg{result: scanFixture()})

	if m.loading {
		t.Fatal("expected model to leave loading state")
	}

	return m
}

func press(m Model, key string) (Model, tea.Cmd) {
	if key == "esc" {
		return m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	}

	return m.Update(tea.KeyPressMsg{Code: rune(key[0]), Text: key})
}

func TestConfirmFlowCursorTarget(t *testing.T) {
	t.Parallel()

	m := loadedModel(t)

	m, cmd := press(m, "d")

	if cmd != nil {
		t.Error("entering confirm state returned a command, want nil")
	}

	if !m.confirmDelete {
		t.Fatal("expected confirmDelete after d")
	}

	if len(m.deleteTargets) != 1 {
		t.Fatalf("deleteTargets = %d, want 1 (cursor row)", len(m.deleteTargets))
	}

	if got := m.deleteTargets[0].Key(); got != "acme/gadgets/fix/typo" {
		t.Errorf("target = %q, want first sorted orphan acme/gadgets/fix/typo", got)
	}
}

func TestConfirmFlowCancel(t *testing.T) {
	t.Parallel()

	for _, key := range []string{"n", "N", "esc"} {
		t.Run(key, func(t *testing.T) {
			t.Parallel()

			m := loadedModel(t)
			m, _ = press(m, "d")

			m, cmd := press(m, key)

			if cmd != nil {
				t.Error("cancel returned a command, want nil")
			}

			if m.confirmDelete {
				t.Error("expected confirmDelete cleared after cancel")
			}

			if m.deleteTargets != nil {
				t.Error("expected deleteTargets cleared after cancel")
			}

			if m.statusMsg != "Delete canceled" {
				t.Errorf("statusMsg = %q, want Delete canceled", m.statusMsg)
			}
		})
	}
}

func TestConfirmFlowSelectionTargets(t *testing.T) {
	t.Parallel()

	m := loadedModel(t)
	m, _ = m.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	m, _ = press(m, "j")
	m, _ = m.Update(tea.KeyPressMsg{Code: ' ', Text: " "})

	m, _ = press(m, "d")

	if !m.confirmDelete {
		t.Fatal("expected confirmDelete after d with selection")
	}

	if len(m.deleteTargets) != 2 {
		t.Fatalf("deleteTargets = %d, want 2 selected orphans", len(m.deleteTargets))
	}
}

func TestConfirmFlowExecuteReturnsBatch(t *testing.T) {
	t.Parallel()

	m := loadedModel(t)
	m, _ = press(m, "d")

	m, cmd := press(m, "y")

	if m.confirmDelete {
		t.Error("expected confirmDelete cleared after y")
	}

	if cmd == nil {
		t.Fatal("expected delete command batch after y")
	}
}

func TestDeleteResultRemovesOrphan(t *testing.T) {
	t.Parallel()

	m := loadedModel(t)

	m, _ = m.Update(deleteResultMsg{branch: "acme/widgets/spike/cache"})

	if m.result.TotalOrphans != 2 {
		t.Errorf("TotalOrphans = %d, want 2", m.result.TotalOrphans)
	}

	for _, orphan := range m.result.AllOrphans() {
		if orphan.Key() == "acme/widgets/spike/cache" {
			t.Error("expected orphan removed from result")
		}
	}

	if m.statusMsg != "Deleted: acme/widgets/spike/cache" {
		t.Errorf("statusMsg = %q", m.statusMsg)
	}
}

func TestDeleteResultError(t *testing.T) {
	t.Parallel()

	m := loadedModel(t)

	m, _ = m.Update(deleteResultMsg{branch: "acme/gadgets/fix/typo", err: errors.New("boom")})

	if m.result.TotalOrphans != 3 {
		t.Errorf("TotalOrphans = %d, want 3 (nothing removed)", m.result.TotalOrphans)
	}

	if !strings.Contains(m.statusMsg, "Failed to delete acme/gadgets/fix/typo") {
		t.Errorf("statusMsg = %q", m.statusMsg)
	}
}

func TestTypeFiltersAndViewModeCycle(t *testing.T) {
	t.Parallel()

	m := loadedModel(t)

	m, _ = press(m, "2")
	if got := len(m.getFilteredOrphans()); got != 1 {
		t.Errorf("merged filter count = %d, want 1", got)
	}

	m, _ = press(m, "3")
	if got := len(m.getFilteredOrphans()); got != 1 {
		t.Errorf("closed filter count = %d, want 1", got)
	}

	m, _ = press(m, "4")
	if got := len(m.getFilteredOrphans()); got != 1 {
		t.Errorf("stale filter count = %d, want 1", got)
	}

	m, _ = press(m, "1")
	if got := len(m.getFilteredOrphans()); got != 3 {
		t.Errorf("all filter count = %d, want 3", got)
	}

	modes := []ViewMode{ViewModeByType, ViewModeFlat, ViewModeByRepo}
	for _, want := range modes {
		m, _ = press(m, "v")
		if m.viewMode != want {
			t.Errorf("viewMode = %s, want %s", m.viewMode, want)
		}
	}
}

func TestFlatViewSortsByLastCommit(t *testing.T) {
	t.Parallel()

	m := loadedModel(t)
	m.viewMode = ViewModeFlat

	filtered := m.getFilteredOrphans()
	for i := 1; i < len(filtered); i++ {
		if filtered[i].LastCommitDate.Before(filtered[i-1].LastCommitDate) {
			t.Fatalf("flat view not sorted by last commit: %v", filtered)
		}
	}
}

func TestScanErrorRendered(t *testing.T) {
	t.Parallel()

	m := NewModel("acme", orphanscore.DefaultScanOptions())
	m, _ = m.Update(scanCompleteMsg{err: errors.New("rate limited")})

	view := m.View()
	if !strings.Contains(view, "rate limited") {
		t.Errorf("view missing scan error, got %q", view)
	}
}

func TestScanProgressRendered(t *testing.T) {
	t.Parallel()

	m := NewModel("acme", orphanscore.DefaultScanOptions())
	m, _ = m.Update(scanProgressMsg{current: 3, total: 10, currentRepo: "acme/widgets", orphans: 2})

	view := m.View()
	if !strings.Contains(view, "3/10") || !strings.Contains(view, "acme/widgets") {
		t.Errorf("progress view = %q", view)
	}
}
