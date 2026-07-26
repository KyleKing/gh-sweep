package webhooks

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
)

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
	for _, want := range []string{"Webhooks", "acme/widgets (1 webhooks)", "https://ci.example.com/hook", "push, pull_request"} {
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

func TestWebhooksLoadError(t *testing.T) {
	t.Parallel()

	m := NewModel([]string{"acme/widgets"})
	m, _ = m.Update(webhooksLoadedMsg{err: errors.New("forbidden")})

	if !strings.Contains(m.View(), "forbidden") {
		t.Errorf("view = %q", m.View())
	}
}
