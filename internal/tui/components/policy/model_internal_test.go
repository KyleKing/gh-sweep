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
	m := NewModel(&config.PolicyConfig{Repositories: []string{"acme/widgets", "acme/gadgets"}})
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
	for _, want := range []string{"Policy Drift", "acme/widgets: in sync", "acme/gadgets: 1 field(s) drifted"} {
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

	m := NewModel(&config.PolicyConfig{})
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
