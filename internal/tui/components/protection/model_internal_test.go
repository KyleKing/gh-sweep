package protection

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
)

func loadedProtectionModel() Model {
	m := NewModel([]string{"acme/widgets", "acme/gadgets"}, "acme/widgets")
	m, _ = m.Update(rulesLoadedMsg{
		rules: map[string]*github.ProtectionRule{
			"acme/widgets": {
				Repository:          "acme/widgets",
				Branch:              "main",
				RequiredReviews:     2,
				RequireStatusChecks: []string{"ci"},
				EnforceAdmins:       true,
			},
		},
		diffs: map[string][]string{
			"RequiredReviews": {"acme/gadgets: 0 (baseline: 2)"},
		},
	})

	return m
}

func TestLoadedRulesView(t *testing.T) {
	t.Parallel()

	view := loadedProtectionModel().View()
	for _, want := range []string{
		"Branch Protection Rules",
		"Baseline: acme/widgets",
		"Reviews: 2",
		"acme/gadgets: No protection",
		"Differences from baseline",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q", want)
		}
	}
}

func TestCursorNavigation(t *testing.T) {
	t.Parallel()

	m := loadedProtectionModel()

	m, _ = m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	if m.cursor != 1 {
		t.Errorf("cursor after j = %d, want 1", m.cursor)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	if m.cursor != 1 {
		t.Errorf("cursor clamped = %d, want 1", m.cursor)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	if m.cursor != 0 {
		t.Errorf("cursor after k = %d, want 0", m.cursor)
	}
}

func TestJumpTopBottom(t *testing.T) {
	t.Parallel()

	m := loadedProtectionModel()
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

	m := loadedProtectionModel()
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

func TestRulesLoadError(t *testing.T) {
	t.Parallel()

	m := NewModel([]string{"acme/widgets"}, "")
	m, _ = m.Update(rulesLoadedMsg{
		rules: map[string]*github.ProtectionRule{},
		diffs: map[string][]string{},
		err:   errors.New("forbidden"),
	})

	if !strings.Contains(m.View(), "forbidden") {
		t.Errorf("view = %q", m.View())
	}
}
