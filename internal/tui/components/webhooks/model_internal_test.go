package webhooks

import (
	"errors"
	"fmt"
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

func TestListScrollsWhenTallerThanViewport(t *testing.T) {
	t.Parallel()

	repos := make([]string, 0, 30)
	webhooks := make(map[string][]github.Webhook, 30)

	for i := range 30 {
		repo := fmt.Sprintf("acme/repo-%02d", i)
		repos = append(repos, repo)
		webhooks[repo] = []github.Webhook{{ID: i, URL: "https://ci.example.com/hook"}}
	}

	m := NewModel(repos)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 15})
	m, _ = m.Update(webhooksLoadedMsg{webhooks: webhooks})

	top := m.View()
	if !strings.Contains(top, "repo-00") {
		t.Errorf("top-of-list view missing first item, got %q", top)
	}
	if strings.Contains(top, "repo-29") {
		t.Errorf("top-of-list view should not show the last item yet, got %q", top)
	}
	if !strings.Contains(top, "more below") {
		t.Errorf("top-of-list view missing a below-fold hint, got %q", top)
	}

	for range 25 {
		m, _ = m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	}

	bottom := m.View()
	if !strings.Contains(bottom, "repo-25") {
		t.Errorf("scrolled view missing the cursor row, got %q", bottom)
	}
	if !strings.Contains(bottom, "more above") {
		t.Errorf("scrolled view missing an above-fold hint, got %q", bottom)
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
