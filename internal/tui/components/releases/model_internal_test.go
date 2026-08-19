package releases

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
)

func loadedReleasesModel() Model {
	m := NewModel([]string{"acme/widgets", "acme/gadgets"})
	m, _ = m.Update(releasesLoadedMsg{
		releases: map[string][]github.Release{
			"acme/widgets": {
				{
					ID:          2,
					TagName:     "v1.2.0",
					Author:      "alice",
					PublishedAt: time.Now().Add(-48 * time.Hour),
				},
				{
					ID:          1,
					TagName:     "v1.1.0",
					Author:      "alice",
					PublishedAt: time.Now().Add(-30 * 24 * time.Hour),
				},
			},
		},
		latest: map[string]*github.Release{
			"acme/widgets": {
				ID:          2,
				TagName:     "v1.2.0",
				Author:      "alice",
				PublishedAt: time.Now().Add(-48 * time.Hour),
			},
		},
	})

	return m
}

func TestLatestViewDefault(t *testing.T) {
	t.Parallel()

	view := loadedReleasesModel().View()
	for _, want := range []string{"Release Overview", "Latest Releases", "v1.2.0", "acme/gadgets: No releases"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q", want)
		}
	}
}

func TestViewModeSwitches(t *testing.T) {
	t.Parallel()

	m := loadedReleasesModel()

	m, _ = m.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	if m.viewMode != "all" || !strings.Contains(m.View(), "All Releases") {
		t.Errorf("viewMode = %q after 2", m.viewMode)
	}

	if !strings.Contains(m.View(), "v1.1.0") {
		t.Error("all view missing older release")
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: '3', Text: "3"})
	if m.viewMode != "outdated" || !strings.Contains(m.View(), "Outdated Releases") {
		t.Errorf("viewMode = %q after 3", m.viewMode)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: '1', Text: "1"})
	if m.viewMode != "latest" {
		t.Errorf("viewMode = %q after 1", m.viewMode)
	}
}

func TestJumpTopBottom(t *testing.T) {
	t.Parallel()

	m := loadedReleasesModel()
	m, _ = m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})

	m, _ = m.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	if m.cursor != 0 {
		t.Errorf("cursor after g = %d, want 0", m.cursor)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: 'G', Text: "G"})
	want := len(m.repos) - 1
	if m.cursor != want {
		t.Errorf("cursor after G = %d, want %d", m.cursor, want)
	}
}

func TestHelpToggle(t *testing.T) {
	t.Parallel()

	m := loadedReleasesModel()
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

func TestReleasesLoadError(t *testing.T) {
	t.Parallel()

	m := NewModel([]string{"acme/widgets"})
	m, _ = m.Update(releasesLoadedMsg{
		releases: map[string][]github.Release{},
		latest:   map[string]*github.Release{},
		err:      errors.New("forbidden"),
	})

	if !strings.Contains(m.View(), "forbidden") {
		t.Errorf("view = %q", m.View())
	}
}
