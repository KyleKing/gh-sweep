package watching

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
)

func loadedWatchingModel() Model {
	m := NewModel()
	m, _ = m.Update(dataLoadedMsg{
		username: "tester",
		repos: []github.RepoWatchInfo{
			{
				RepoBasic: github.RepoBasic{
					Name:     "widgets",
					FullName: "acme/widgets",
					Owner:    "acme",
				},
				State: github.WatchStateSubscribed,
			},
			{
				RepoBasic: github.RepoBasic{
					Name:     "gadgets",
					FullName: "acme/gadgets",
					Owner:    "acme",
				},
				State: github.WatchStateDefault,
			},
		},
	})

	return m
}

func TestUnwatchedViewDefault(t *testing.T) {
	t.Parallel()

	view := loadedWatchingModel().View()
	for _, want := range []string{"Watch Status Audit", "User: tester", "acme/gadgets"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q", want)
		}
	}

	if strings.Contains(view, "acme/widgets") {
		t.Error("watched repo listed in unwatched view")
	}
}

func TestInvertSelection(t *testing.T) {
	t.Parallel()

	m := loadedWatchingModel()
	m, _ = m.Update(tea.KeyPressMsg{Code: '4', Text: "4"}) // "all" view: both repos
	m, _ = m.Update(tea.KeyPressMsg{Code: ' ', Text: " "}) // select index 0 (cursor starts at 0)

	m, _ = m.Update(tea.KeyPressMsg{Code: 'I', Text: "I"})

	if m.selected[0] {
		t.Error("expected the originally-selected repo deselected after invert")
	}
	if !m.selected[1] {
		t.Error("expected the other filtered repo selected after invert")
	}
}

func TestJumpTopBottom(t *testing.T) {
	t.Parallel()

	m := loadedWatchingModel()
	m, _ = m.Update(tea.KeyPressMsg{Code: '4', Text: "4"})

	m, _ = m.Update(tea.KeyPressMsg{Code: 'G', Text: "G"})
	if want := len(m.getFilteredRepos()) - 1; m.cursor != want {
		t.Errorf("cursor after G = %d, want %d", m.cursor, want)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	if m.cursor != 0 {
		t.Errorf("cursor after g = %d, want 0", m.cursor)
	}
}

func TestSearchFiltersByName(t *testing.T) {
	t.Parallel()

	m := loadedWatchingModel()
	m, _ = m.Update(tea.KeyPressMsg{Code: '4', Text: "4"})
	m, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	for _, r := range "gadgets" {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}

	filtered := m.getFilteredRepos()
	if len(filtered) != 1 || filtered[0].FullName != "acme/gadgets" {
		t.Fatalf("filtered = %+v, want only acme/gadgets", filtered)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.searching || m.searchQuery != "" {
		t.Errorf("expected search cleared after esc, got searching=%v query=%q", m.searching, m.searchQuery)
	}
	if len(m.getFilteredRepos()) != 2 {
		t.Error("expected full list restored after canceling search")
	}
}

func TestHelpToggle(t *testing.T) {
	t.Parallel()

	m := loadedWatchingModel()
	m, _ = m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	if !m.showHelp {
		t.Fatal("expected showHelp true after ?")
	}
	if !strings.Contains(m.View(), "Keybindings") {
		t.Error("help view missing Keybindings title")
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.showHelp {
		t.Error("expected showHelp false after esc")
	}
}

func TestWatchingViewModeSwitches(t *testing.T) {
	t.Parallel()

	m := loadedWatchingModel()

	m, _ = m.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	if m.viewMode != "watched" || !strings.Contains(m.View(), "acme/widgets") {
		t.Errorf("viewMode = %q after 2, view = %q", m.viewMode, m.View())
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: '4', Text: "4"})
	if m.viewMode != "all" {
		t.Errorf("viewMode = %q after 4", m.viewMode)
	}

	view := m.View()
	if !strings.Contains(view, "acme/widgets") || !strings.Contains(view, "acme/gadgets") {
		t.Errorf("all view = %q", view)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: '1', Text: "1"})
	if m.viewMode != "unwatched" {
		t.Errorf("viewMode = %q after 1", m.viewMode)
	}
}

func TestWatchResultUpdatesStatus(t *testing.T) {
	t.Parallel()

	m := loadedWatchingModel()

	m, _ = m.Update(watchResultMsg{repo: "acme/gadgets"})
	if !strings.Contains(m.statusMsg, "acme/gadgets") {
		t.Errorf("statusMsg = %q", m.statusMsg)
	}

	m, _ = m.Update(unwatchResultMsg{repo: "acme/widgets", err: errors.New("boom")})
	if !strings.Contains(m.statusMsg, "boom") && !strings.Contains(m.statusMsg, "Failed") {
		t.Errorf("statusMsg = %q", m.statusMsg)
	}

	m, _ = m.Update(openResultMsg{repo: "acme/gadgets"})
	if !strings.Contains(m.statusMsg, "Opened acme/gadgets") {
		t.Errorf("statusMsg = %q", m.statusMsg)
	}

	m, _ = m.Update(openResultMsg{repo: "acme/widgets", err: errors.New("boom")})
	if !strings.Contains(m.statusMsg, "boom") && !strings.Contains(m.statusMsg, "Failed") {
		t.Errorf("statusMsg = %q", m.statusMsg)
	}
}

func TestIgnoreResultUpdatesStatus(t *testing.T) {
	t.Parallel()

	m := loadedWatchingModel()

	m, _ = m.Update(ignoreResultMsg{repo: "acme/gadgets"})
	if !strings.Contains(m.statusMsg, "acme/gadgets") {
		t.Errorf("statusMsg = %q", m.statusMsg)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: '3', Text: "3"})
	if m.viewMode != "ignored" || !strings.Contains(m.View(), "acme/gadgets") {
		t.Errorf("ignored view = %q", m.View())
	}
}

func TestWatchingLoadError(t *testing.T) {
	t.Parallel()

	m := NewModel()
	m, _ = m.Update(dataLoadedMsg{err: errors.New("unauthorized")})

	if !strings.Contains(m.View(), "unauthorized") {
		t.Errorf("view = %q", m.View())
	}
}
