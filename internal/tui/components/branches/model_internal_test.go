package branches

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
)

var (
	errBoom        = errors.New("boom")
	errNetworkDown = errors.New("network down")
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
	t.Parallel()

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
			t.Parallel()

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
	t.Parallel()

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
			t.Parallel()

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

func TestUpdateWindowSizeAndBranchesLoaded(t *testing.T) {
	t.Parallel()

	m := NewModel("acme/widgets", "")

	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if m.width != 100 || m.height != 30 {
		t.Errorf("width/height = %d/%d, want 100/30", m.width, m.height)
	}

	m, _ = m.Update(branchesLoadedMsg{branches: branchStatuses()})
	if m.loading {
		t.Error("expected loading false after branchesLoadedMsg")
	}
	if len(m.branches) != len(branchStatuses()) {
		t.Errorf("branches = %d, want %d", len(m.branches), len(branchStatuses()))
	}

	m, _ = m.Update(branchesLoadedMsg{err: errBoom})
	if m.err == nil {
		t.Error("expected err set after a failed load")
	}
}

func TestHandleDeleteFlowSelectsEligibleAndBlocksOthers(t *testing.T) {
	t.Parallel()

	m := Model{repo: "acme/widgets", branches: branchStatuses(), selected: map[string]bool{
		"main":      true, // default: blocked
		"feature-a": true, // eligible
	}}

	m, _ = m.handleListKeys(tea.KeyPressMsg{Code: 'd', Text: "d"})

	if !m.confirmDelete {
		t.Fatal("expected confirmDelete true when at least one target is eligible")
	}
	if len(m.deleteTargets) != 1 || m.deleteTargets[0].Name != "feature-a" {
		t.Errorf("deleteTargets = %+v, want just feature-a", m.deleteTargets)
	}
	if !strings.Contains(m.statusMsg, "Blocked: main") {
		t.Errorf("statusMsg = %q, want it to mention the blocked default branch", m.statusMsg)
	}
}

func TestHandleDeleteFlowNoneEligible(t *testing.T) {
	t.Parallel()

	m := Model{repo: "acme/widgets", branches: branchStatuses(), selected: map[string]bool{"main": true}}

	m, _ = m.handleListKeys(tea.KeyPressMsg{Code: 'd', Text: "d"})

	if m.confirmDelete {
		t.Error("expected confirmDelete false when nothing is eligible")
	}
	if !strings.Contains(m.statusMsg, "Blocked: main") {
		t.Errorf("statusMsg = %q", m.statusMsg)
	}
}

func TestHandleDeleteFlowNoSelectionUsesCursor(t *testing.T) {
	t.Parallel()

	m := Model{repo: "acme/widgets", branches: branchStatuses(), selected: map[string]bool{}, cursor: 1}

	m, _ = m.handleListKeys(tea.KeyPressMsg{Code: 'd', Text: "d"})

	if !m.confirmDelete || len(m.deleteTargets) != 1 || m.deleteTargets[0].Name != "feature-a" {
		t.Errorf("expected the cursor row (feature-a) as the sole target, got %+v", m.deleteTargets)
	}
}

func TestHandleConfirmKeysCancel(t *testing.T) {
	t.Parallel()

	m := Model{
		confirmDelete: true,
		deleteTargets: []github.BranchStatus{{Branch: github.Branch{Name: "feature-a"}}},
	}

	m, cmd := m.handleConfirmKeys(tea.KeyPressMsg{Code: 'n', Text: "n"})

	if m.confirmDelete || m.deleteTargets != nil {
		t.Errorf("expected delete canceled, got confirmDelete=%v targets=%v", m.confirmDelete, m.deleteTargets)
	}
	if cmd != nil {
		t.Error("expected no command on cancel")
	}
	if m.statusMsg != "Delete canceled" {
		t.Errorf("statusMsg = %q", m.statusMsg)
	}
}

//nolint:paralleltest // mutates the shared global test transport
func TestExecuteDeleteAndBranchRemoval(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodDelete {
			return &http.Response{
				StatusCode: http.StatusNoContent,
				Header:     http.Header{},
				Body:       io.NopCloser(strings.NewReader("")),
				Request:    req,
			}, nil
		}

		return &http.Response{
			StatusCode: http.StatusNotFound,
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader(`{"message":"Not Found"}`)),
			Request:    req,
		}, nil
	})

	restore := github.SetTestTransport(transport)
	defer restore()

	m := Model{
		repo:          "acme/widgets",
		branches:      branchStatuses(),
		selected:      map[string]bool{"feature-a": true},
		confirmDelete: true,
		deleteTargets: []github.BranchStatus{{Branch: github.Branch{Name: "feature-a"}}},
	}

	m, cmd := m.handleConfirmKeys(tea.KeyPressMsg{Code: 'y', Text: "y"})
	if cmd == nil {
		t.Fatal("expected executeDelete to return a command")
	}
	if m.confirmDelete {
		t.Error("expected confirmDelete cleared once delete is in flight")
	}

	// tea.Batch collapses a single command to that command directly (no BatchMsg wrapper).
	m, _ = m.Update(cmd())

	if m.statusMsg != "Deleted: feature-a" {
		t.Errorf("statusMsg = %q", m.statusMsg)
	}
	if m.selected["feature-a"] {
		t.Error("expected feature-a deselected after delete")
	}
	for _, b := range m.branches {
		if b.Name == "feature-a" {
			t.Error("expected feature-a removed from branches after delete")
		}
	}
}

func TestExecuteDeleteInvalidRepo(t *testing.T) {
	t.Parallel()

	m := Model{
		repo:          "not-a-valid-repo",
		confirmDelete: true,
		deleteTargets: []github.BranchStatus{{Branch: github.Branch{Name: "feature-a"}}},
	}

	m, cmd := m.executeDelete()

	if cmd != nil {
		t.Error("expected no command for an invalid repo")
	}
	if m.confirmDelete || m.deleteTargets != nil {
		t.Error("expected delete state cleared for an invalid repo")
	}
	if !strings.Contains(m.statusMsg, "Invalid repository") {
		t.Errorf("statusMsg = %q", m.statusMsg)
	}
}

func TestDescribeBlocked(t *testing.T) {
	t.Parallel()

	blocked := []github.BranchStatus{
		{Branch: github.Branch{Name: "main"}, IsDefault: true},
		{Branch: github.Branch{Name: "release", Protected: true}},
	}

	got := describeBlocked(blocked)

	if !strings.Contains(got, "main (") || !strings.Contains(got, "release (") {
		t.Errorf("describeBlocked() = %q, want both branch names with their reasons", got)
	}
}

func TestViewLoadingErrorAndConfirmDialog(t *testing.T) {
	t.Parallel()

	loading := Model{loading: true}
	if !strings.Contains(loading.View(), "Loading branches") {
		t.Errorf("loading view = %q", loading.View())
	}

	failed := Model{err: errNetworkDown}
	if !strings.Contains(failed.View(), "network down") {
		t.Errorf("error view = %q", failed.View())
	}

	confirm := Model{
		repo:          "acme/widgets",
		confirmDelete: true,
		deleteTargets: []github.BranchStatus{{Branch: github.Branch{Name: "feature-a"}}},
	}
	view := confirm.View()
	if !strings.Contains(view, "Confirm Delete") || !strings.Contains(view, "feature-a") {
		t.Errorf("confirm dialog view = %q", view)
	}
}

func TestBranchAnnotations(t *testing.T) {
	t.Parallel()

	branch := github.BranchStatus{
		Branch: github.Branch{
			Name:           "feature-a",
			Protected:      true,
			LastCommitDate: time.Now().Add(-48 * time.Hour),
		},
		IsDefault: true,
		PR:        &github.PullRequest{Number: 5, State: "open"},
	}

	got := branchAnnotations(branch)

	for _, want := range []string{"[default]", "[protected]", "PR #5 (open)", "2d"} {
		if !strings.Contains(got, want) {
			t.Errorf("branchAnnotations() = %q, missing %q", got, want)
		}
	}

	if branchAnnotations(github.BranchStatus{Branch: github.Branch{Name: "plain"}}) != "" {
		t.Error("expected no annotations for a plain branch")
	}
}

func TestGetLocalBranches(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	for _, args := range [][]string{
		{"init"},
		{"config", "user.name", "Test User"},
		{"config", "user.email", "test@example.com"},
		{"commit", "--allow-empty", "-m", "initial"},
	} {
		//nolint:gosec // no shell; git rejects malformed ref names
		cmd := exec.CommandContext(context.Background(), "git", args...)
		cmd.Dir = tmpDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	branches, err := GetLocalBranches(tmpDir)
	if err != nil {
		t.Fatalf("GetLocalBranches() error = %v", err)
	}
	if len(branches) != 1 {
		t.Errorf("branches = %+v, want 1", branches)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func manyBranchesFixture(count int) []github.BranchStatus {
	branches := make([]github.BranchStatus, count)
	for i := range branches {
		branches[i] = github.BranchStatus{Branch: github.Branch{Name: fmt.Sprintf("branch-%02d", i)}}
	}

	return branches
}

func press(m Model, key string) (Model, tea.Cmd) {
	return m.handleListKeys(tea.KeyPressMsg{Code: rune(key[0]), Text: key})
}

func TestListScrollsWhenTallerThanViewport(t *testing.T) {
	t.Parallel()

	m := NewModel("acme/widgets", "")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 15})
	m, _ = m.Update(branchesLoadedMsg{branches: manyBranchesFixture(50)})

	top := m.View()
	if !strings.Contains(top, "branch-00") {
		t.Errorf("top-of-list view missing first item, got %q", top)
	}
	if strings.Contains(top, "branch-49") {
		t.Errorf("top-of-list view should not show the last item yet, got %q", top)
	}
	if !strings.Contains(top, "more below") {
		t.Errorf("top-of-list view missing a below-fold hint, got %q", top)
	}

	for range 40 {
		m, _ = press(m, "down")
	}

	bottom := m.View()
	if !strings.Contains(bottom, "branch-40") {
		t.Errorf("scrolled view missing the cursor row, got %q", bottom)
	}
	if !strings.Contains(bottom, "more above") {
		t.Errorf("scrolled view missing an above-fold hint, got %q", bottom)
	}
}
