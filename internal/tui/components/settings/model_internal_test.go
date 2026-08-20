package settings

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
)

var errForbidden = errors.New("forbidden")

func loadedSettingsModel() Model {
	m := NewModel([]string{"acme/widgets", "acme/gadgets"}, "acme/widgets")
	m, _ = m.Update(settingsLoadedMsg{
		settings: map[string]*github.RepoSettings{
			"acme/widgets": {
				Repository:       "acme/widgets",
				DefaultBranch:    "main",
				AllowSquashMerge: true,
			},
			"acme/gadgets": {
				Repository:       "acme/gadgets",
				DefaultBranch:    "master",
				AllowMergeCommit: true,
			},
		},
		diffs: map[string][]github.SettingsDiff{
			"acme/gadgets": {
				{Field: "DefaultBranch", Baseline: "main", Current: "master", Severity: "warning"},
			},
		},
	})

	return m
}

func TestOverviewViewDefault(t *testing.T) {
	t.Parallel()

	view := loadedSettingsModel().View()
	for _, want := range []string{"Repository Settings Comparison", "Baseline: acme/widgets", "Default Branch: main"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q", want)
		}
	}
}

func TestDiffViewSwitch(t *testing.T) {
	t.Parallel()

	m := loadedSettingsModel()
	m, _ = m.Update(tea.KeyPressMsg{Code: '2', Text: "2"})

	if m.viewMode != "diff" {
		t.Fatalf("viewMode = %q, want diff", m.viewMode)
	}

	view := m.View()
	if !strings.Contains(view, "DefaultBranch") || !strings.Contains(view, "master") {
		t.Errorf("diff view = %q", view)
	}
}

func TestJumpTopBottom(t *testing.T) {
	t.Parallel()

	m := loadedSettingsModel()

	m, _ = m.Update(tea.KeyPressMsg{Code: 'G', Text: "G"})
	if want := len(m.repos) - 1; m.cursor != want {
		t.Errorf("cursor after G = %d, want %d", m.cursor, want)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	if m.cursor != 0 {
		t.Errorf("cursor after g = %d, want 0", m.cursor)
	}
}

func TestHelpToggle(t *testing.T) {
	t.Parallel()

	m := loadedSettingsModel()
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

func manySettingsFixture(count int) ([]string, map[string]*github.RepoSettings) {
	repos := make([]string, count)
	settings := make(map[string]*github.RepoSettings)

	for i := range repos {
		repo := fmt.Sprintf("acme/repo-%02d", i)
		repos[i] = repo
		settings[repo] = &github.RepoSettings{Repository: repo, DefaultBranch: "main"}
	}

	return repos, settings
}

func TestOverviewScrollsWhenTallerThanViewport(t *testing.T) {
	t.Parallel()

	repos, repoSettings := manySettingsFixture(30)

	m := NewModel(repos, "")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 15})
	m, _ = m.Update(settingsLoadedMsg{settings: repoSettings, diffs: map[string][]github.SettingsDiff{}})

	top := m.View()
	if !strings.Contains(top, "acme/repo-00") {
		t.Errorf("top-of-list view missing first item, got %q", top)
	}
	if strings.Contains(top, "acme/repo-29") {
		t.Errorf("top-of-list view should not show the last item yet, got %q", top)
	}
	if !strings.Contains(top, "more below") {
		t.Errorf("top-of-list view missing a below-fold hint, got %q", top)
	}

	for range len(repos) - 1 {
		m, _ = m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	}

	bottom := m.View()
	if !strings.Contains(bottom, "acme/repo-29") {
		t.Errorf("scrolled view missing the cursor row, got %q", bottom)
	}
	if !strings.Contains(bottom, "more above") {
		t.Errorf("scrolled view missing an above-fold hint, got %q", bottom)
	}
}

func TestSettingsLoadError(t *testing.T) {
	t.Parallel()

	m := NewModel([]string{"acme/widgets"}, "")
	m, _ = m.Update(settingsLoadedMsg{
		settings: map[string]*github.RepoSettings{},
		diffs:    map[string][]github.SettingsDiff{},
		err:      errForbidden,
	})

	if !strings.Contains(m.View(), "forbidden") {
		t.Errorf("view = %q", m.View())
	}
}
