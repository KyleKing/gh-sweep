package collaborators

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
)

func loadedCollaboratorsModel() Model {
	m := NewModel([]string{"acme/widgets"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(collaboratorsLoadedMsg{collaborators: map[string][]github.Collaborator{
		"acme/widgets": {
			{Login: "alice", Permission: "admin", Repository: "acme/widgets"},
			{Login: "bob", Permission: "write", Repository: "acme/widgets"},
		},
	}})

	return m
}

func TestLoadedByRepoView(t *testing.T) {
	t.Parallel()

	view := loadedCollaboratorsModel().View()
	for _, want := range []string{"Collaborator Management", "acme/widgets (2 collaborators)", "alice", "[admin]"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q", want)
		}
	}
}

func TestByUserViewSwitch(t *testing.T) {
	t.Parallel()

	m := loadedCollaboratorsModel()
	m, _ = m.Update(tea.KeyPressMsg{Code: '2', Text: "2"})

	if m.viewMode != "byuser" {
		t.Errorf("viewMode = %q, want byuser", m.viewMode)
	}

	if !strings.Contains(m.View(), "alice") {
		t.Errorf("by-user view missing collaborator: %q", m.View())
	}
}

func TestCollaboratorsLoadError(t *testing.T) {
	t.Parallel()

	m := NewModel([]string{"acme/widgets"})
	m, _ = m.Update(collaboratorsLoadedMsg{
		collaborators: map[string][]github.Collaborator{},
		err:           errors.New("forbidden"),
	})

	if !strings.Contains(m.View(), "forbidden") {
		t.Errorf("view = %q", m.View())
	}
}
