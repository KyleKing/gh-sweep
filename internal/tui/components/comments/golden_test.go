//go:build golden

package comments

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/golden"

	"github.com/KyleKing/gh-sweep/internal/github"
)

func TestGoldenLoadedView(t *testing.T) {
	t.Parallel()

	m := NewModel("acme/widgets")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(threadsLoadedMsg{threads: []github.ReviewThread{
		{
			Repository: "acme/widgets",
			PRNumber:   42,
			PRTitle:    "Add login flow",
			Path:       "internal/auth/session.go",
			Comments: []github.ReviewComment{
				{
					Author:    "alice",
					Body:      "Consider extracting this into a helper so both handlers share it.",
					CreatedAt: time.Now().Add(-3*time.Hour - 10*time.Minute),
					URL:       "https://github.com/acme/widgets/pull/42#discussion_r1",
				},
			},
		},
		{
			Repository: "acme/widgets",
			PRNumber:   42,
			PRTitle:    "Add login flow",
			Path:       "internal/auth/token.go",
			IsResolved: true,
			Comments: []github.ReviewComment{
				{
					Author:    "bob",
					Body:      "Nit: rename to refreshToken.",
					CreatedAt: time.Now().Add(-49 * time.Hour),
					URL:       "https://github.com/acme/widgets/pull/42#discussion_r2",
				},
			},
		},
	}})

	golden.RequireEqual(t, []byte(m.View()))
}
