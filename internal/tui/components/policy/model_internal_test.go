package policy

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/KyleKing/gh-sweep/internal/config"
	"github.com/KyleKing/gh-sweep/internal/policy"
)

var errNoPolicyFile = errors.New("no policy file found")

func loadedPolicyModel() Model {
	m := NewModel(
		&config.PolicyConfig{Repositories: []string{"acme/widgets", "acme/gadgets"}},
		"/tmp/.gh-sweep-policy.yaml",
		policy.ApplyOptions{},
	)
	m, _ = m.Update(reportLoadedMsg{report: &policy.Report{Repos: []policy.RepoDrift{
		{Repository: "acme/widgets"},
		{Repository: "acme/gadgets", Diffs: []policy.Diff{
			{Domain: policy.DomainSettings, Field: "has_wiki", Desired: "false", Current: "true"},
		}},
	}}})

	return m
}

func TestPolicyViewShowsDriftAndSync(t *testing.T) {
	t.Parallel()

	view := loadedPolicyModel().View()
	for _, want := range []string{"Policy:", "acme/widgets: in sync", "acme/gadgets: 1 field(s) drifted"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q:\n%s", want, view)
		}
	}
}

func TestJumpTopBottom(t *testing.T) {
	t.Parallel()

	m := loadedPolicyModel()

	m, _ = m.Update(tea.KeyPressMsg{Code: 'G', Text: "G"})
	if want := len(m.report.Repos) - 1; m.cursor != want {
		t.Errorf("cursor after G = %d, want %d", m.cursor, want)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	if m.cursor != 0 {
		t.Errorf("cursor after g = %d, want 0", m.cursor)
	}
}

func TestHelpToggle(t *testing.T) {
	t.Parallel()

	m := loadedPolicyModel()
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

func TestPolicyApplyRequiresConfirmation(t *testing.T) {
	t.Parallel()

	m := loadedPolicyModel()
	m.cursor = 1 // acme/gadgets has drift

	m, _ = m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	if !m.confirmApply || m.applyTarget != "acme/gadgets" {
		t.Fatalf("confirmApply = %v, applyTarget = %q", m.confirmApply, m.applyTarget)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	if m.confirmApply {
		t.Error("confirmApply should clear on cancel")
	}
}

func TestPolicyApplyOnCleanRepoNoOps(t *testing.T) {
	t.Parallel()

	m := loadedPolicyModel() // cursor defaults to 0: acme/widgets, no drift

	m, _ = m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	if m.confirmApply {
		t.Error("confirmApply = true for a repo with no drift")
	}
}

func TestPolicyApplyResultClearsDiffs(t *testing.T) {
	t.Parallel()

	m := loadedPolicyModel()
	m, _ = m.Update(applyResultMsg{result: policy.ApplyResult{
		Repository: "acme/gadgets",
		Applied:    []policy.Domain{policy.DomainSettings},
	}})

	if len(m.report.Repos[1].Diffs) != 0 {
		t.Errorf("diffs not cleared after apply: %+v", m.report.Repos[1].Diffs)
	}
	if !strings.Contains(m.statusMsg, "Applied acme/gadgets") {
		t.Errorf("statusMsg = %q", m.statusMsg)
	}
}

func manyReposFixture(count int) *policy.Report {
	repos := make([]policy.RepoDrift, count)
	for i := range repos {
		repos[i] = policy.RepoDrift{Repository: fmt.Sprintf("acme/repo-%02d", i)}
	}

	return &policy.Report{Repos: repos}
}

func TestListScrollsWhenTallerThanViewport(t *testing.T) {
	t.Parallel()

	m := NewModel(&config.PolicyConfig{}, "/tmp/.gh-sweep-policy.yaml", policy.ApplyOptions{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 15})
	m, _ = m.Update(reportLoadedMsg{report: manyReposFixture(50)})

	top := m.View()
	if !strings.Contains(top, "acme/repo-00") {
		t.Errorf("top-of-list view missing first item, got %q", top)
	}
	if strings.Contains(top, "acme/repo-49") {
		t.Errorf("top-of-list view should not show the last item yet, got %q", top)
	}
	if !strings.Contains(top, "more below") {
		t.Errorf("top-of-list view missing a below-fold hint, got %q", top)
	}

	for range 40 {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}

	bottom := m.View()
	if !strings.Contains(bottom, "acme/repo-40") {
		t.Errorf("scrolled view missing the cursor row, got %q", bottom)
	}
	if !strings.Contains(bottom, "more above") {
		t.Errorf("scrolled view missing an above-fold hint, got %q", bottom)
	}
}

func TestDiffFocusTogglesBoolFieldImmediately(t *testing.T) {
	t.Parallel()

	m := loadedPolicyModel()
	m.cursor = 1 // acme/gadgets: has_wiki desired=false

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !m.diffFocus {
		t.Fatal("expected diffFocus true after enter on a drifted repo")
	}

	m, cmd := m.Update(tea.KeyPressMsg{Code: 'e', Text: "e"})
	if m.editing {
		t.Error("bool field should toggle immediately, not open a text prompt")
	}
	if m.cfg.Settings.HasWiki == nil || !*m.cfg.Settings.HasWiki {
		t.Fatalf("HasWiki = %v, want true after toggling desired=false", m.cfg.Settings.HasWiki)
	}
	if cmd == nil {
		t.Error("expected a reload command after committing an edit")
	}
}

func TestDiffFocusEsc(t *testing.T) {
	t.Parallel()

	m := loadedPolicyModel()
	m.cursor = 1

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !m.diffFocus {
		t.Fatal("expected diffFocus true")
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.diffFocus {
		t.Error("expected diffFocus false after esc")
	}
}

func TestEditTextFieldPromptAndCommit(t *testing.T) {
	t.Parallel()

	m := NewModel(
		&config.PolicyConfig{Protection: config.PolicyProtection{RequiredReviews: intPtr(1)}},
		"/tmp/.gh-sweep-policy.yaml",
		policy.ApplyOptions{},
	)
	m, _ = m.Update(reportLoadedMsg{report: &policy.Report{Repos: []policy.RepoDrift{
		{Repository: "acme/widgets", Diffs: []policy.Diff{
			{Domain: policy.DomainProtection, Field: "required_reviews", Desired: "1", Current: "0"},
		}},
	}}})

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'e', Text: "e"})
	if !m.editing || m.editBuffer != "1" {
		t.Fatalf("editing = %v, editBuffer = %q, want editing seeded with the declared value", m.editing, m.editBuffer)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m, _ = m.Update(tea.KeyPressMsg{Code: '3', Text: "3"})
	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if m.editing {
		t.Error("expected editing false after confirming")
	}
	if m.cfg.Protection.RequiredReviews == nil || *m.cfg.Protection.RequiredReviews != 3 {
		t.Errorf("RequiredReviews = %v, want 3", m.cfg.Protection.RequiredReviews)
	}
	if cmd == nil {
		t.Error("expected a reload command after committing an edit")
	}
}

func TestEditTextFieldCancel(t *testing.T) {
	t.Parallel()

	m := NewModel(
		&config.PolicyConfig{Protection: config.PolicyProtection{RequiredReviews: intPtr(1)}},
		"/tmp/.gh-sweep-policy.yaml",
		policy.ApplyOptions{},
	)
	m, _ = m.Update(reportLoadedMsg{report: &policy.Report{Repos: []policy.RepoDrift{
		{Repository: "acme/widgets", Diffs: []policy.Diff{
			{Domain: policy.DomainProtection, Field: "required_reviews", Desired: "1", Current: "0"},
		}},
	}}})

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'e', Text: "e"})
	m, _ = m.Update(tea.KeyPressMsg{Code: '9', Text: "9"})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})

	if m.editing {
		t.Error("expected editing false after esc")
	}
	if *m.cfg.Protection.RequiredReviews != 1 {
		t.Errorf("RequiredReviews = %d, want unchanged 1 after cancel", *m.cfg.Protection.RequiredReviews)
	}
}

func TestSaveWritesPolicyFile(t *testing.T) {
	t.Parallel()

	path := t.TempDir() + "/.gh-sweep-policy.yaml"
	m := NewModel(&config.PolicyConfig{DefaultOrg: "acme"}, path, policy.ApplyOptions{})

	m, cmd := m.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	if cmd == nil {
		t.Fatal("expected a save command")
	}

	m, _ = m.Update(cmd())
	if !strings.Contains(m.statusMsg, "Saved") {
		t.Errorf("statusMsg = %q, want a Saved confirmation", m.statusMsg)
	}

	if _, err := config.LoadPolicy(path); err != nil {
		t.Errorf("LoadPolicy() after save error = %v", err)
	}
}

func TestPolicyConfigLoadError(t *testing.T) {
	t.Parallel()

	m := NewModelWithConfigError(errNoPolicyFile)
	if m.Init() != nil {
		t.Error("Init() should not fetch when constructed with a config error")
	}

	if !strings.Contains(m.View(), "no policy file found") {
		t.Errorf("view = %q", m.View())
	}
}
