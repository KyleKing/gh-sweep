package webhooks

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
)

var errForbidden = errors.New("forbidden")

func loadedWebhooksModel() Model {
	m := NewModel([]string{"acme/widgets"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(webhooksLoadedMsg{
		webhooks: map[string][]github.Webhook{
			"acme/widgets": {
				{
					ID:         7,
					Repository: "acme/widgets",
					URL:        "https://ci.example.com/hook",
					Events:     []string{"push", "pull_request"},
					Active:     true,
				},
			},
		},
		health: map[string]map[int]github.WebhookHealth{
			"acme/widgets": {
				7: {WebhookID: 7, SuccessRate: 0.95, TotalDeliveries: 20, Failures: 1},
			},
		},
	})

	return m
}

func TestLoadedWebhooksView(t *testing.T) {
	t.Parallel()

	view := loadedWebhooksModel().View()
	wants := []string{
		"Webhooks", "acme/widgets (1 webhooks)", "https://ci.example.com/hook", "push, pull_request",
	}
	for _, want := range wants {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q", want)
		}
	}
}

func TestWebhooksCursorNavigation(t *testing.T) {
	t.Parallel()

	m := NewModel([]string{"acme/a", "acme/b"})
	m, _ = m.Update(webhooksLoadedMsg{webhooks: map[string][]github.Webhook{}})

	m, _ = m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	if m.cursor != 1 {
		t.Errorf("cursor = %d after j, want 1", m.cursor)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	if m.cursor != 1 {
		t.Errorf("cursor = %d after j at end, want 1", m.cursor)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	if m.cursor != 0 {
		t.Errorf("cursor = %d after k, want 0", m.cursor)
	}
}

func TestJumpTopBottom(t *testing.T) {
	t.Parallel()

	m := NewModel([]string{"acme/a", "acme/b"})
	m, _ = m.Update(webhooksLoadedMsg{webhooks: map[string][]github.Webhook{}})
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

	m := loadedWebhooksModel()
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

func TestWebhooksLoadError(t *testing.T) {
	t.Parallel()

	m := NewModel([]string{"acme/widgets"})
	m, _ = m.Update(webhooksLoadedMsg{err: errForbidden})

	if !strings.Contains(m.View(), "forbidden") {
		t.Errorf("view = %q", m.View())
	}
}
