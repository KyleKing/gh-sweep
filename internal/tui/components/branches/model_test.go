package branches

import (
	"testing"

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
