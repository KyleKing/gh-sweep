//go:build golden

package webhooks

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/golden"

	"github.com/KyleKing/gh-sweep/internal/github"
)

func TestGoldenLoadedView(t *testing.T) {
	t.Parallel()

	m := NewModel([]string{"acme/widgets"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(webhooksLoadedMsg{
		webhooks: map[string][]github.Webhook{
			"acme/widgets": {
				{
					ID:         7,
					Repository: "acme/widgets",
					URL:        "https://ci.example.com/hooks/github",
					Events:     []string{"push", "pull_request"},
					Active:     true,
				},
			},
		},
		health: map[string]map[int]github.WebhookHealth{
			"acme/widgets": {
				7: {WebhookID: 7, SuccessRate: 98.5, TotalDeliveries: 200, Failures: 3, AvgDuration: 120},
			},
		},
	})

	golden.RequireEqual(t, []byte(m.View()))
}
