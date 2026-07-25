package tui

import (
	"testing"
)

func TestNewMainModel(t *testing.T) {
	opts := MainModelOptions{
		Baseline: "acme/baseline",
		Org:      "acme",
		Repo:     "acme/repo1",
		Repos:    []string{"acme/repo1", "acme/repo2"},
	}

	m := NewMainModel(opts)

	if m.repo != "acme/repo1" {
		t.Errorf("Expected repo 'acme/repo1', got '%s'", m.repo)
	}

	if len(m.repos) != 2 {
		t.Fatalf("Expected 2 repos, got %d", len(m.repos))
	}

	if m.org != "acme" {
		t.Errorf("Expected org 'acme', got '%s'", m.org)
	}

	if m.baseline != "acme/baseline" {
		t.Errorf("Expected baseline 'acme/baseline', got '%s'", m.baseline)
	}

	if m.mode != ViewHome {
		t.Errorf("Expected initial mode ViewHome, got %d", m.mode)
	}
}
