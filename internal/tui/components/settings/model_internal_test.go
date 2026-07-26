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
