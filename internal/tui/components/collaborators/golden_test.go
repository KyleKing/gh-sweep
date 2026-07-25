//go:build golden

package collaborators

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
	m, _ = m.Update(collaboratorsLoadedMsg{collaborators: map[string][]github.Collaborator{
		"acme/widgets": {
			{Login: "alice", Permission: "admin", Repository: "acme/widgets"},
			{Login: "bob", Permission: "write", Repository: "acme/widgets"},
			{Login: "carol", Permission: "read", Repository: "acme/widgets"},
		},
	}})

	golden.RequireEqual(t, []byte(m.View()))
}
