package branches

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
)

func branchStatuses() []github.BranchStatus {
	return []github.BranchStatus{
		{Branch: github.Branch{Name: "main"}, IsDefault: true},
		{Branch: github.Branch{Name: "feature-a"}},
		{
			Branch: github.Branch{Name: "feature-b"},
			PR:     &github.PullRequest{Number: 9, State: "open"},
		},
		{Branch: github.Branch{Name: "release", Protected: true}},
	}
}

func names(branches []github.BranchStatus) []string {
	result := make([]string, 0, len(branches))
	for _, branch := range branches {
		result = append(result, branch.Name)
	}

	return result
}

func TestCollectDeleteTargets(t *testing.T) {
	tests := []struct {
		name         string
		selected     map[string]bool
		cursor       int
		wantEligible []string
		wantBlocked  []string
	}{
		{
			name:         "cursor on deletable branch",
			selected:     map[string]bool{},
			cursor:       1,
			wantEligible: []string{"feature-a"},
			wantBlocked:  nil,
		},
		{
			name:         "cursor on default branch is blocked",
			selected:     map[string]bool{},
			cursor:       0,
			wantEligible: nil,
			wantBlocked:  []string{"main"},
		},
		{
			name:         "cursor on branch with open PR is blocked",
			selected:     map[string]bool{},
			cursor:       2,
			wantEligible: nil,
			wantBlocked:  []string{"feature-b"},
		},
		{
			name:         "selection overrides cursor",
			selected:     map[string]bool{"feature-a": true},
			cursor:       0,
			wantEligible: []string{"feature-a"},
			wantBlocked:  nil,
		},
		{
			name:         "mixed selection partitions eligible and blocked",
			selected:     map[string]bool{"main": true, "feature-a": true, "release": true},
			cursor:       0,
			wantEligible: []string{"feature-a"},
			wantBlocked:  []string{"main", "release"},
		},
		{
			name:         "deselected entries are ignored",
			selected:     map[string]bool{"feature-a": false},
			cursor:       1,
			wantEligible: []string{"feature-a"},
			wantBlocked:  nil,
		},
		{
			name:         "cursor out of range yields nothing",
			selected:     map[string]bool{},
			cursor:       10,
			wantEligible: nil,
			wantBlocked:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eligible, blocked := collectDeleteTargets(branchStatuses(), tt.selected, tt.cursor)

			assertNames(t, "eligible", names(eligible), tt.wantEligible)
			assertNames(t, "blocked", names(blocked), tt.wantBlocked)
		})
	}
}

func assertNames(t *testing.T, label string, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("%s = %v, want %v", label, got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s[%d] = %s, want %s", label, i, got[i], want[i])
		}
	}
}

func TestInvertSelection(t *testing.T) {
	t.Parallel()

	m := Model{branches: branchStatuses(), selected: map[string]bool{"main": true}}

	m, _ = m.handleListKeys(tea.KeyPressMsg{Code: 'I', Text: "I"})

	if m.selected["main"] {
		t.Error("expected the originally-selected branch deselected after invert")
	}
	for _, name := range []string{"feature-a", "feature-b", "release"} {
		if !m.selected[name] {
			t.Errorf("expected %s selected after invert", name)
		}
	}
}

func TestJumpTopBottom(t *testing.T) {
	t.Parallel()

	m := Model{branches: branchStatuses(), selected: map[string]bool{}, cursor: 2}

	m, _ = m.handleListKeys(tea.KeyPressMsg{Code: 'g', Text: "g"})
	if m.cursor != 0 {
		t.Errorf("cursor after g = %d, want 0", m.cursor)
	}

	m, _ = m.handleListKeys(tea.KeyPressMsg{Code: 'G', Text: "G"})
	if want := len(m.branches) - 1; m.cursor != want {
		t.Errorf("cursor after G = %d, want %d", m.cursor, want)
	}
}

func TestSearchFiltersByName(t *testing.T) {
	t.Parallel()

	m := Model{branches: branchStatuses(), selected: map[string]bool{}}

	m, _ = m.handleListKeys(tea.KeyPressMsg{Code: '/', Text: "/"})
	for _, r := range "release" {
		m, _ = m.handleSearchKeys(tea.KeyPressMsg{Code: r, Text: string(r)})
	}

	visible := m.getVisibleBranches()
	if len(visible) != 1 || visible[0].Name != "release" {
		t.Fatalf("visible = %+v, want only release", visible)
	}

	m, _ = m.handleSearchKeys(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.searching || m.searchQuery != "" {
		t.Errorf("expected search cleared after esc, got searching=%v query=%q", m.searching, m.searchQuery)
	}
	if len(m.getVisibleBranches()) != len(m.branches) {
		t.Error("expected full list restored after canceling search")
	}
}

func TestHelpToggle(t *testing.T) {
	t.Parallel()

	m := Model{branches: branchStatuses(), selected: map[string]bool{}}

	m, _ = m.handleListKeys(tea.KeyPressMsg{Code: '?', Text: "?"})
	if !m.showHelp {
		t.Fatal("expected showHelp true after ?")
	}
	if !strings.Contains(m.View(), "Keybindings") {
		t.Error("help view missing Keybindings title")
	}
}

func TestSplitRepo(t *testing.T) {
	tests := []struct {
		name      string
		repo      string
		wantOwner string
		wantName  string
		wantOK    bool
	}{
		{
			name:      "valid repo",
			repo:      "owner/repo",
			wantOwner: "owner",
			wantName:  "repo",
			wantOK:    true,
		},
		{name: "missing slash", repo: "owner", wantOK: false},
		{name: "empty owner", repo: "/repo", wantOK: false},
		{name: "empty name", repo: "owner/", wantOK: false},
		{name: "empty string", repo: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, name, ok := splitRepo(tt.repo)

			if ok != tt.wantOK {
				t.Fatalf("splitRepo(%q) ok = %v, want %v", tt.repo, ok, tt.wantOK)
			}
			if owner != tt.wantOwner || name != tt.wantName {
				t.Errorf("splitRepo(%q) = (%q, %q), want (%q, %q)",
					tt.repo, owner, name, tt.wantOwner, tt.wantName)
			}
		})
	}
}
