//go:build golden

package watching

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/golden"

	"github.com/KyleKing/gh-sweep/internal/github"
)

func TestGoldenLoadedView(t *testing.T) {
	t.Parallel()

	m := NewModel()
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(dataLoadedMsg{
		username: "tester",
		userRepos: []github.RepoBasic{
			{Name: "widgets", FullName: "acme/widgets", Owner: "acme"},
			{Name: "gadgets", FullName: "acme/gadgets", Owner: "acme", Private: true},
			{Name: "dotfiles", FullName: "tester/dotfiles", Owner: "tester"},
		},
		subscriptions: map[string]*github.Subscription{
			"acme/widgets": {
				Repository: "acme/widgets",
				Subscribed: true,
				State:      github.WatchStateSubscribed,
			},
			"acme/gadgets": {
				Repository: "acme/gadgets",
				State:      github.WatchStateDefault,
			},
			"tester/dotfiles": {
				Repository: "tester/dotfiles",
				State:      github.WatchStateDefault,
			},
		},
	})

	golden.RequireEqual(t, []byte(m.View()))
}
