//go:build golden

package protection

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/golden"

	"github.com/KyleKing/gh-sweep/internal/github"
)

func TestGoldenLoadedView(t *testing.T) {
	t.Parallel()

	m := NewModel([]string{"acme/widgets", "acme/gadgets"}, "acme/widgets")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(rulesLoadedMsg{
		rules: map[string]*github.ProtectionRule{
			"acme/widgets": {
				Repository:          "acme/widgets",
				Branch:              "main",
				RequiredReviews:     2,
				RequireStatusChecks: []string{"ci"},
				EnforceAdmins:       true,
			},
		},
		diffs: map[string][]string{
			"RequiredReviews": {"acme/gadgets: 0 (baseline: 2)"},
		},
	})

	golden.RequireEqual(t, []byte(m.View()))
}
