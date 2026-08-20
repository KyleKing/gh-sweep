package watching

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
)

var (
	errBoom         = errors.New("boom")
	errUnauthorized = errors.New("unauthorized")
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

	m, _ = m.Update(unwatchResultMsg{repo: "acme/widgets", err: errBoom})
	if !strings.Contains(m.statusMsg, "boom") && !strings.Contains(m.statusMsg, "Failed") {
		t.Errorf("statusMsg = %q", m.statusMsg)
	}

	m, _ = m.Update(openResultMsg{repo: "acme/gadgets"})
	if !strings.Contains(m.statusMsg, "Opened acme/gadgets") {
		t.Errorf("statusMsg = %q", m.statusMsg)
	}

	m, _ = m.Update(openResultMsg{repo: "acme/widgets", err: errBoom})
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

func press(m Model, key string) (Model, tea.Cmd) {
	return m.Update(tea.KeyPressMsg{Code: rune(key[0]), Text: key})
}

func manyReposFixture(count int) []github.RepoWatchInfo {
	repos := make([]github.RepoWatchInfo, count)
	for i := range repos {
		repos[i] = github.RepoWatchInfo{
			RepoBasic: github.RepoBasic{
				Name:     fmt.Sprintf("repo-%02d", i),
				FullName: fmt.Sprintf("acme/repo-%02d", i),
				Owner:    "acme",
			},
			State: github.WatchStateSubscribed,
		}
	}

	return repos
}

func TestListScrollsWhenTallerThanViewport(t *testing.T) {
	t.Parallel()

	m := NewModel()
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 15})
	m, _ = m.Update(dataLoadedMsg{username: "tester", repos: manyReposFixture(50)})
	m, _ = m.Update(tea.KeyPressMsg{Code: '2', Text: "2"}) // watched view: all fixture repos

	top := m.View()
	if !strings.Contains(top, "repo-00") {
		t.Errorf("top-of-list view missing first item, got %q", top)
	}
	if strings.Contains(top, "repo-49") {
		t.Errorf("top-of-list view should not show the last item yet, got %q", top)
	}
	if !strings.Contains(top, "more below") {
		t.Errorf("top-of-list view missing a below-fold hint, got %q", top)
	}

	for range 40 {
		m, _ = press(m, "down")
	}

	bottom := m.View()
	if !strings.Contains(bottom, "repo-40") {
		t.Errorf("scrolled view missing the cursor row, got %q", bottom)
	}
	if !strings.Contains(bottom, "more above") {
		t.Errorf("scrolled view missing an above-fold hint, got %q", bottom)
	}
}

func TestWatchingLoadError(t *testing.T) {
	t.Parallel()

	m := NewModel()
	m, _ = m.Update(dataLoadedMsg{err: errUnauthorized})

	if !strings.Contains(m.View(), "unauthorized") {
		t.Errorf("view = %q", m.View())
	}
}
