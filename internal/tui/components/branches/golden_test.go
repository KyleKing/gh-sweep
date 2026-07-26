//go:build golden

package branches

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/golden"

	"github.com/KyleKing/gh-sweep/internal/github"
)

func TestGoldenLoadedView(t *testing.T) {
	t.Parallel()

	m := NewModel("acme/widgets", "")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(branchesLoadedMsg{branches: []github.BranchStatus{
		{
			Branch: github.Branch{
				Name:           "main",
				Protected:      true,
				LastCommitDate: time.Now().Add(-49 * time.Hour),
			},
			ComparedTo: "main",
			IsDefault:  true,
		},
		{
			Branch: github.Branch{
				Name:           "feature/login",
				Ahead:          3,
				Behind:         1,
				LastCommitDate: time.Now().Add(-73 * time.Hour),
			},
			ComparedTo: "main",
			PR:         &github.PullRequest{Number: 42, State: "open"},
		},
		{
			Branch: github.Branch{
				Name:           "chore/cleanup",
				Behind:         5,
				LastCommitDate: time.Now().Add(-241 * time.Hour),
			},
			ComparedTo: "main",
		},
	}})

	golden.RequireEqual(t, []byte(m.View()))
}
