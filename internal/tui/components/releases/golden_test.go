//go:build golden

package releases

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/golden"

	"github.com/KyleKing/gh-sweep/internal/github"
)

// The golden state uses the "all" tab: the latest/outdated tabs render ages
// via time.Since, which would change the snapshot from one day to the next.
func TestGoldenLoadedView(t *testing.T) {
	t.Parallel()

	m := NewModel([]string{"acme/widgets", "acme/gadgets"})
	m.viewMode = "all"
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(releasesLoadedMsg{
		releases: map[string][]github.Release{
			"acme/widgets": {
				{
					ID:          2,
					Repository:  "acme/widgets",
					TagName:     "v1.2.0",
					Name:        "v1.2.0",
					Author:      "alice",
					PublishedAt: time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC),
				},
				{
					ID:          1,
					Repository:  "acme/widgets",
					TagName:     "v1.1.0",
					Name:        "v1.1.0",
					Author:      "alice",
					PublishedAt: time.Date(2025, 11, 2, 12, 0, 0, 0, time.UTC),
				},
			},
		},
		latest: map[string]*github.Release{
			"acme/widgets": {
				ID:          2,
				Repository:  "acme/widgets",
				TagName:     "v1.2.0",
				Name:        "v1.2.0",
				Author:      "alice",
				PublishedAt: time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC),
			},
		},
	})

	golden.RequireEqual(t, []byte(m.View()))
}
