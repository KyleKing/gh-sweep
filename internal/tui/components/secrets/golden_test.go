//go:build golden

package secrets

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/golden"

	"github.com/KyleKing/gh-sweep/internal/github"
)

func TestGoldenLoadedView(t *testing.T) {
	t.Parallel()

	m := NewModel("acme", []string{"acme/widgets"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(secretsLoadedMsg{
		orgSecrets: []github.Secret{
			{
				Name:      "DEPLOY_KEY",
				Scope:     "org",
				CreatedAt: "2025-06-01T00:00:00Z",
				UpdatedAt: "2026-01-05T00:00:00Z",
			},
		},
		repoSecrets: map[string][]github.Secret{
			"acme/widgets": {
				{
					Name:       "CODECOV_TOKEN",
					Scope:      "repo",
					Repository: "acme/widgets",
					CreatedAt:  "2025-08-15T00:00:00Z",
					UpdatedAt:  "2025-08-15T00:00:00Z",
				},
			},
		},
		unusedSecrets: []string{},
	})

	golden.RequireEqual(t, []byte(m.View()))
}
