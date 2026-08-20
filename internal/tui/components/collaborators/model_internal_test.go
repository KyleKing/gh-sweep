package collaborators

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
)

var errForbidden = errors.New("forbidden")

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

func TestJumpTopBottom(t *testing.T) {
	t.Parallel()

	m := loadedCollaboratorsModel()
	m, _ = m.Update(tea.KeyPressMsg{Code: '2', Text: "2"})

	m, _ = m.Update(tea.KeyPressMsg{Code: 'G', Text: "G"})
	if want := m.getTotalCollaborators() - 1; m.cursor != want {
		t.Errorf("cursor after G = %d, want %d", m.cursor, want)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	if m.cursor != 0 {
		t.Errorf("cursor after g = %d, want 0", m.cursor)
	}
}

func TestHelpToggle(t *testing.T) {
	t.Parallel()

	m := loadedCollaboratorsModel()
	m, _ = m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	if !m.showHelp {
		t.Fatal("expected showHelp true after ?")
	}
	if !strings.Contains(m.View(), "Keybindings") {
		t.Error("help view missing Keybindings title")
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.showHelp {
		t.Error("expected showHelp false after esc")
	}
}

func manyRepoCollaboratorsFixture(n int) ([]string, map[string][]github.Collaborator) {
	repos := make([]string, n)
	collabs := make(map[string][]github.Collaborator, n)

	for i := range n {
		repo := fmt.Sprintf("acme/repo-%02d", i)
		repos[i] = repo
		collabs[repo] = []github.Collaborator{
			{Login: fmt.Sprintf("user-%02d", i), Permission: "write", Repository: repo},
		}
	}

	return repos, collabs
}

func TestListScrollsWhenTallerThanViewport(t *testing.T) {
	t.Parallel()

	repos, collabs := manyRepoCollaboratorsFixture(50)
	m := NewModel(repos)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 15})
	m, _ = m.Update(collaboratorsLoadedMsg{collaborators: collabs})

	top := m.View()
	if !strings.Contains(top, "repo-00") {
		t.Errorf("top-of-list view missing first item, got %q", top)
	}
	if strings.Contains(top, "repo-49") {
		t.Errorf("top-of-list view should not show the last item yet, got %q", top)
	}
	if !strings.Contains(top, "more below") {
		t.Errorf("top-of-list view missing a below-fold hint, got %q", top)
	}

	for range 40 {
		m, _ = m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	}

	bottom := m.View()
	if !strings.Contains(bottom, "repo-40") {
		t.Errorf("scrolled view missing the cursor row, got %q", bottom)
	}
	if !strings.Contains(bottom, "more above") {
		t.Errorf("scrolled view missing an above-fold hint, got %q", bottom)
	}
}

func TestCollaboratorsLoadError(t *testing.T) {
	t.Parallel()

	m := NewModel([]string{"acme/widgets"})
	m, _ = m.Update(collaboratorsLoadedMsg{
		collaborators: map[string][]github.Collaborator{},
		err:           errForbidden,
	})

	if !strings.Contains(m.View(), "forbidden") {
		t.Errorf("view = %q", m.View())
	}
}
