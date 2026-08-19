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

func TestInvertSelection(t *testing.T) {
	t.Parallel()

	m := loadedModel(t)
	m, _ = m.Update(tea.KeyPressMsg{Code: ' ', Text: " "})

	m, _ = press(m, "I")

	filtered := m.getFilteredOrphans()
	if len(filtered) != 3 {
		t.Fatalf("filtered orphans = %d, want 3", len(filtered))
	}

	if m.selected[filtered[0].Key()] {
		t.Errorf("expected the originally-selected orphan to be deselected after invert")
	}
	for _, orphan := range filtered[1:] {
		if !m.selected[orphan.Key()] {
			t.Errorf("expected %s selected after invert", orphan.Key())
		}
	}
}

func TestJumpTopBottom(t *testing.T) {
	t.Parallel()

	m := loadedModel(t)
	m, _ = press(m, "j")
	m, _ = press(m, "j")

	m, _ = press(m, "g")
	if m.cursor != 0 {
		t.Errorf("cursor after g = %d, want 0", m.cursor)
	}

	m, _ = press(m, "G")
	want := len(m.getFilteredOrphans()) - 1
	if m.cursor != want {
		t.Errorf("cursor after G = %d, want %d", m.cursor, want)
	}
}

func TestSortReverse(t *testing.T) {
	t.Parallel()

	m := loadedModel(t)
	forward := m.getFilteredOrphans()

	m, _ = press(m, "R")
	reversed := m.getFilteredOrphans()

	if len(forward) != len(reversed) {
		t.Fatalf("forward = %d orphans, reversed = %d", len(forward), len(reversed))
	}
	for i := range forward {
		if forward[i].Key() != reversed[len(reversed)-1-i].Key() {
			t.Fatalf("reversed order mismatch at %d: %s vs %s", i, forward[i].Key(), reversed[len(reversed)-1-i].Key())
		}
	}
}

func TestSearchFiltersByName(t *testing.T) {
	t.Parallel()

	m := loadedModel(t)
	m, _ = press(m, "/")
	for _, r := range "cache" {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}

	filtered := m.getFilteredOrphans()
	if len(filtered) != 1 || filtered[0].BranchName != "spike/cache" {
		t.Fatalf("filtered = %+v, want only spike/cache", filtered)
	}

	m, _ = press(m, "esc")
	if m.searching || m.searchQuery != "" {
		t.Errorf("expected search cleared after esc, got searching=%v query=%q", m.searching, m.searchQuery)
	}
	if len(m.getFilteredOrphans()) == 1 {
		t.Error("expected full list restored after canceling search")
	}
}

func TestHelpToggle(t *testing.T) {
	t.Parallel()

	m := loadedModel(t)
	m, _ = press(m, "?")
	if !m.showHelp {
		t.Fatal("expected showHelp true after ?")
	}
	if !strings.Contains(m.View(), "Keybindings") {
		t.Error("help view missing Keybindings title")
	}

	m, _ = press(m, "esc")
	if m.showHelp {
		t.Error("expected showHelp false after esc")
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
