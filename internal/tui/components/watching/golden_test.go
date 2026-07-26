//go:build golden

package watching

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/golden"

	"github.com/KyleKing/gh-sweep/internal/github"
)

func TestGoldenLoadedView(t *testing.T) {
	t.Parallel()

	pushedAt := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)

	m := NewModel()
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(dataLoadedMsg{
		username: "tester",
		repos: []github.RepoWatchInfo{
			{
				RepoBasic:          github.RepoBasic{Name: "widgets", FullName: "acme/widgets", Owner: "acme"},
				State:              github.WatchStateSubscribed,
				StargazerCount:     4,
				WatcherCount:       2,
				PushedAt:           pushedAt,
				ViewerCanSubscribe: true,
			},
			{
				RepoBasic:          github.RepoBasic{Name: "gadgets", FullName: "acme/gadgets", Owner: "acme", Private: true},
				State:              github.WatchStateDefault,
				StargazerCount:     0,
				WatcherCount:       1,
				PushedAt:           pushedAt,
				ViewerCanSubscribe: true,
			},
			{
				RepoBasic:          github.RepoBasic{Name: "dotfiles", FullName: "tester/dotfiles", Owner: "tester"},
				State:              github.WatchStateDefault,
				StargazerCount:     1,
				WatcherCount:       1,
				PushedAt:           pushedAt,
				ViewerCanSubscribe: true,
			},
		},
	})

	golden.RequireEqual(t, []byte(m.View()))
}
