package policy

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/KyleKing/gh-sweep/internal/config"
	"github.com/KyleKing/gh-sweep/internal/policy"
)

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

func TestPolicyConfigLoadError(t *testing.T) {
	t.Parallel()

	m := NewModelWithConfigError(errors.New("no policy file found"))
	if m.Init() != nil {
		t.Error("Init() should not fetch when constructed with a config error")
	}

	if !strings.Contains(m.View(), "no policy file found") {
		t.Errorf("view = %q", m.View())
	}
}
