package settings

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
)

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

func TestSettingsLoadError(t *testing.T) {
	t.Parallel()

	m := NewModel([]string{"acme/widgets"}, "")
	m, _ = m.Update(settingsLoadedMsg{
		settings: map[string]*github.RepoSettings{},
		diffs:    map[string][]github.SettingsDiff{},
		err:      errors.New("forbidden"),
	})

	if !strings.Contains(m.View(), "forbidden") {
		t.Errorf("view = %q", m.View())
	}
}
