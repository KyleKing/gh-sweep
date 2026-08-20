package secrets

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
)

var errForbidden = errors.New("forbidden")

func press(m Model, key string) (Model, tea.Cmd) {
	return m.Update(tea.KeyPressMsg{Code: rune(key[0]), Text: key})
}

func manyOrgSecretsFixture(count int) []github.Secret {
	secrets := make([]github.Secret, count)
	for i := range secrets {
		secrets[i] = github.Secret{Name: fmt.Sprintf("secret-%02d", i)}
	}

	return secrets
}

func TestOrgSecretsListScrollsWhenTallerThanViewport(t *testing.T) {
	t.Parallel()

	m := NewModel("acme", nil)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 15})
	m, _ = m.Update(secretsLoadedMsg{
		orgSecrets:  manyOrgSecretsFixture(50),
		repoSecrets: make(map[string][]github.Secret),
	})

	top := m.View()
	if !strings.Contains(top, "secret-00") {
		t.Errorf("top-of-list view missing first item, got %q", top)
	}
	if strings.Contains(top, "secret-49") {
		t.Errorf("top-of-list view should not show the last item yet, got %q", top)
	}
	if !strings.Contains(top, "more below") {
		t.Errorf("top-of-list view missing a below-fold hint, got %q", top)
	}

	for range 40 {
		m, _ = press(m, "down")
	}

	bottom := m.View()
	if !strings.Contains(bottom, "secret-40") {
		t.Errorf("scrolled view missing the cursor row, got %q", bottom)
	}
	if !strings.Contains(bottom, "more above") {
		t.Errorf("scrolled view missing an above-fold hint, got %q", bottom)
	}
}

func loadedSecretsModel() Model {
	m := NewModel("acme", []string{"acme/widgets"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = m.Update(secretsLoadedMsg{
		orgSecrets: []github.Secret{
			{Name: "DEPLOY_KEY", Scope: "org", UpdatedAt: "2026-01-05T00:00:00Z"},
		},
		repoSecrets: map[string][]github.Secret{
			"acme/widgets": {
				{Name: "CODECOV_TOKEN", Scope: "repo", Repository: "acme/widgets"},
			},
		},
		unusedSecrets: []github.SecretUsage{
			{Name: "CODECOV_TOKEN", Scope: "repo", Repository: "acme/widgets", Unused: true},
		},
	})

	return m
}

func TestOrgSecretsViewDefault(t *testing.T) {
	t.Parallel()

	view := loadedSecretsModel().View()
	for _, want := range []string{"Secrets Audit", "Organization Secrets: acme", "DEPLOY_KEY"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q", want)
		}
	}
}

func TestSecretsViewModeSwitches(t *testing.T) {
	t.Parallel()

	m := loadedSecretsModel()

	m, _ = m.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	if m.viewMode != "repo" || !strings.Contains(m.View(), "CODECOV_TOKEN") {
		t.Errorf("viewMode = %q after 2, view = %q", m.viewMode, m.View())
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: '3', Text: "3"})
	if m.viewMode != "unused" || !strings.Contains(m.View(), "CODECOV_TOKEN (repo, acme/widgets)") {
		t.Errorf("viewMode = %q after 3, view = %q", m.viewMode, m.View())
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: '1', Text: "1"})
	if m.viewMode != "org" {
		t.Errorf("viewMode = %q after 1", m.viewMode)
	}
}

func TestJumpTopBottom(t *testing.T) {
	t.Parallel()

	m := loadedSecretsModel()

	m, _ = m.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	if m.cursor != 0 {
		t.Errorf("cursor after g = %d, want 0", m.cursor)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: 'G', Text: "G"})
	want := len(m.orgSecrets) - 1
	if m.cursor != want {
		t.Errorf("cursor after G = %d, want %d", m.cursor, want)
	}
}

func TestHelpToggle(t *testing.T) {
	t.Parallel()

	m := loadedSecretsModel()
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

func TestSecretsLoadError(t *testing.T) {
	t.Parallel()

	m := NewModel("acme", []string{"acme/widgets"})
	m, _ = m.Update(secretsLoadedMsg{err: errForbidden})

	if !strings.Contains(m.View(), "forbidden") {
		t.Errorf("view = %q", m.View())
	}
}
