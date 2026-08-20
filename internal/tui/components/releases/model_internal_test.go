package releases

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
)

var errForbidden = errors.New("forbidden")

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

func TestListScrollsWhenTallerThanViewport(t *testing.T) {
	t.Parallel()

	repos := make([]string, 40)
	latest := make(map[string]*github.Release)

	for i := range repos {
		repos[i] = fmt.Sprintf("acme/repo-%02d", i)
		latest[repos[i]] = &github.Release{
			ID:          i,
			TagName:     "v1.0.0",
			Author:      "alice",
			PublishedAt: time.Now(),
		}
	}

	m := NewModel(repos)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 15})
	m, _ = m.Update(releasesLoadedMsg{
		releases: map[string][]github.Release{},
		latest:   latest,
	})

	top := m.View()
	if !strings.Contains(top, "repo-00") {
		t.Errorf("top-of-list view missing first item, got %q", top)
	}
	if strings.Contains(top, "repo-39") {
		t.Errorf("top-of-list view should not show the last item yet, got %q", top)
	}
	if !strings.Contains(top, "more below") {
		t.Errorf("top-of-list view missing a below-fold hint, got %q", top)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: 'G', Text: "G"})

	bottom := m.View()
	if !strings.Contains(bottom, "repo-39") {
		t.Errorf("scrolled view missing the cursor row, got %q", bottom)
	}
	if !strings.Contains(bottom, "more above") {
		t.Errorf("scrolled view missing an above-fold hint, got %q", bottom)
	}
}

func TestReleasesLoadError(t *testing.T) {
	t.Parallel()

	m := NewModel([]string{"acme/widgets"})
	m, _ = m.Update(releasesLoadedMsg{
		releases: map[string][]github.Release{},
		latest:   map[string]*github.Release{},
		err:      errForbidden,
	})

	if !strings.Contains(m.View(), "forbidden") {
		t.Errorf("view = %q", m.View())
	}
}
