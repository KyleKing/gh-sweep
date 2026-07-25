//go:build golden

package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/golden"
)

var goldenSizes = []struct {
	name          string
	width, height int
}{
	{"tiny_40x10", 40, 10},
	{"small_80x24", 80, 24},
	{"standard_120x40", 120, 40},
	{"wide_160x50", 160, 50},
}

func resizedMainModel(t *testing.T, width, height int) MainModel {
	t.Helper()

	m := NewMainModel(MainModelOptions{
		Baseline: "acme/baseline",
		Org:      "acme",
		Repo:     "acme/widgets",
		Repos:    []string{"acme/widgets", "acme/gadgets"},
	})

	updated, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})

	resized, ok := updated.(MainModel)
	if !ok {
		t.Fatalf("Update returned %T, want MainModel", updated)
	}

	return resized
}

func TestGoldenHomeView(t *testing.T) {
	t.Parallel()

	for _, tt := range goldenSizes {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := resizedMainModel(t, tt.width, tt.height)
			golden.RequireEqual(t, []byte(m.renderContent()))
		})
	}
}

func TestGoldenHomeViewUnconfigured(t *testing.T) {
	t.Parallel()

	m := NewMainModel(MainModelOptions{})

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	resized, ok := updated.(MainModel)
	if !ok {
		t.Fatalf("Update returned %T, want MainModel", updated)
	}

	golden.RequireEqual(t, []byte(resized.renderContent()))
}
