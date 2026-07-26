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
				RepoBasic: github.RepoBasic{Name: "widgets", FullName: "acme/widgets", Owner: "acme"},
				State:     github.WatchStateSubscribed,
			},
			{
				RepoBasic: github.RepoBasic{Name: "gadgets", FullName: "acme/gadgets", Owner: "acme"},
				State:     github.WatchStateDefault,
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
