package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func configuredMainModel() MainModel {
	return NewMainModel(MainModelOptions{
		Baseline: "acme/widgets",
		Org:      "acme",
		Repo:     "acme/widgets",
		Repos:    []string{"acme/widgets", "acme/gadgets"},
	})
}

func updateMain(t *testing.T, m MainModel, msg tea.Msg) (MainModel, tea.Cmd) {
	t.Helper()

	updated, cmd := m.Update(msg)

	model, ok := updated.(MainModel)
	if !ok {
		t.Fatalf("Update returned %T, want MainModel", updated)
	}

	return model, cmd
}

func keyPress(key string) tea.KeyPressMsg {
	if key == "esc" {
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	}

	return tea.KeyPressMsg{Code: rune(key[0]), Text: key}
}

func TestHomeNavigation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		key      string
		wantMode ViewMode
		wantCmd  bool
	}{
		{"0", ViewWatching, true},
		{"1", ViewBranches, true},
		{"2", ViewProtection, true},
		{"3", ViewComments, true},
		{"4", ViewAnalytics, true},
		{"5", ViewSettings, true},
		{"6", ViewWebhooks, true},
		{"7", ViewCollaborators, true},
		{"8", ViewSecrets, true},
		{"9", ViewReleases, true},
		{"o", ViewOrphans, true},
		{"p", ViewGHAPerf, true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			t.Parallel()

			m, cmd := updateMain(t, configuredMainModel(), keyPress(tt.key))

			if m.mode != tt.wantMode {
				t.Errorf("mode = %d, want %d", m.mode, tt.wantMode)
			}

			if (cmd != nil) != tt.wantCmd {
				t.Errorf("cmd presence = %v, want %v", cmd != nil, tt.wantCmd)
			}
		})
	}
}

func TestHomeNavigationUnconfigured(t *testing.T) {
	t.Parallel()

	tests := []struct {
		key      string
		wantMode ViewMode
	}{
		{"1", ViewBranches},
		{"2", ViewProtection},
		{"3", ViewComments},
		{"4", ViewAnalytics},
		{"5", ViewSettings},
		{"6", ViewWebhooks},
		{"7", ViewCollaborators},
		{"8", ViewSecrets},
		{"9", ViewReleases},
		{"p", ViewGHAPerf},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			t.Parallel()

			m, cmd := updateMain(t, NewMainModel(MainModelOptions{}), keyPress(tt.key))

			if m.mode != tt.wantMode {
				t.Errorf("mode = %d, want %d", m.mode, tt.wantMode)
			}

			if cmd != nil {
				t.Error("expected no load command without repo configuration")
			}
		})
	}
}

func TestEscReturnsHome(t *testing.T) {
	t.Parallel()

	views := []ViewMode{
		ViewBranches, ViewProtection, ViewComments, ViewAnalytics, ViewGHAPerf,
		ViewSettings, ViewWebhooks, ViewCollaborators, ViewSecrets, ViewReleases,
		ViewWatching, ViewOrphans,
	}

	for _, view := range views {
		m := configuredMainModel()
		m.mode = view

		m, cmd := updateMain(t, m, keyPress("esc"))

		if m.mode != ViewHome {
			t.Errorf("mode after esc from %d = %d, want ViewHome", view, m.mode)
		}

		if cmd != nil {
			t.Errorf("esc from %d returned a command, want nil", view)
		}
	}
}

func TestQuitFromHome(t *testing.T) {
	t.Parallel()

	for _, key := range []string{"q", "ctrl+c"} {
		msg := keyPress("q")
		if key == "ctrl+c" {
			msg = tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
		}

		_, cmd := updateMain(t, configuredMainModel(), msg)

		if cmd == nil {
			t.Fatalf("%s: expected quit command", key)
		}

		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("%s: cmd() = %T, want tea.QuitMsg", key, cmd())
		}
	}
}

func TestWindowSizeSetsReady(t *testing.T) {
	t.Parallel()

	m := configuredMainModel()

	if got := m.renderContent(); got != "Initializing..." {
		t.Errorf("renderContent() before resize = %q, want Initializing...", got)
	}

	m, cmd := updateMain(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})

	if !m.ready {
		t.Error("expected ready after WindowSizeMsg")
	}

	if cmd != nil {
		t.Error("WindowSizeMsg returned a command, want nil")
	}

	if m.width != 100 || m.height != 30 {
		t.Errorf("size = %dx%d, want 100x30", m.width, m.height)
	}
}

func TestHomeViewListsAllMenuEntries(t *testing.T) {
	t.Parallel()

	m, _ := updateMain(t, configuredMainModel(), tea.WindowSizeMsg{Width: 120, Height: 40})

	content := m.renderContent()
	for _, want := range []string{
		"Watch Status", "Orphan Branches", "Branch Management", "Branch Protection",
		"PR Comments", "Analytics", "GHA Performance", "Settings Comparison",
		"Webhooks", "Collaborators", "Secrets Audit", "Releases",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("home view missing menu entry %q", want)
		}
	}
}

func TestUnknownKeyOnHomeKeepsMode(t *testing.T) {
	t.Parallel()

	m, cmd := updateMain(t, configuredMainModel(), keyPress("z"))

	if m.mode != ViewHome {
		t.Errorf("mode = %d, want ViewHome", m.mode)
	}

	if cmd != nil {
		t.Error("unknown key returned a command, want nil")
	}
}

func TestNewMainModel(t *testing.T) {
	t.Parallel()

	opts := MainModelOptions{
		Baseline: "acme/baseline",
		Org:      "acme",
		Repo:     "acme/repo1",
		Repos:    []string{"acme/repo1", "acme/repo2"},
	}

	m := NewMainModel(opts)

	if m.repo != "acme/repo1" {
		t.Errorf("Expected repo 'acme/repo1', got '%s'", m.repo)
	}

	if len(m.repos) != 2 {
		t.Fatalf("Expected 2 repos, got %d", len(m.repos))
	}

	if m.org != "acme" {
		t.Errorf("Expected org 'acme', got '%s'", m.org)
	}

	if m.baseline != "acme/baseline" {
		t.Errorf("Expected baseline 'acme/baseline', got '%s'", m.baseline)
	}

	if m.mode != ViewHome {
		t.Errorf("Expected initial mode ViewHome, got %d", m.mode)
	}
}
