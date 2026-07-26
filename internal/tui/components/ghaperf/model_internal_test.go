package ghaperf

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
)

func loadedFixture() dataLoadedMsg {
	return dataLoadedMsg{
		runs: []github.RunTiming{
			{
				RunID:      101,
				Workflow:   "ci",
				Branch:     "main",
				Conclusion: "success",
				Duration:   5 * time.Minute,
			},
			{
				RunID:      102,
				Workflow:   "ci",
				Branch:     "feature",
				Conclusion: "failure",
				Duration:   7 * time.Minute,
			},
		},
		workflowStats: map[string]*github.WorkflowStats{
			"ci": {Workflow: "ci", TotalRuns: 2},
		},
		jobStats: map[string]*github.JobStats{
			"ci/build": {WorkflowJob: "ci/build", TotalRuns: 2},
			"ci/test":  {WorkflowJob: "ci/test", TotalRuns: 2},
		},
		branchStats: map[string]*github.BranchStats{
			"main":    {Branch: "main", TotalRuns: 1},
			"feature": {Branch: "feature", TotalRuns: 1},
			"develop": {Branch: "develop", TotalRuns: 1},
		},
		cachedCount: 1,
		newCount:    1,
	}
}

func TestNewModelParsesRepo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		repo                string
		wantOwner, wantName string
	}{
		{"valid", "acme/widgets", "acme", "widgets"},
		{"missing slash", "acme", "", ""},
		{"too many parts", "a/b/c", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := NewModel(tt.repo)

			if m.owner != tt.wantOwner || m.repoName != tt.wantName {
				t.Errorf(
					"owner/name = %q/%q, want %q/%q",
					m.owner,
					m.repoName,
					tt.wantOwner,
					tt.wantName,
				)
			}
		})
	}
}

func TestOptionsApply(t *testing.T) {
	t.Parallel()

	m := NewModel("acme/widgets",
		WithBranch("develop"),
		WithDays(7),
		WithWorkflow("ci.yml"),
		WithCacheOnly(true),
		WithBaseBranch("trunk"),
	)

	if m.filterBranch != "develop" {
		t.Errorf("filterBranch = %q", m.filterBranch)
	}

	if m.filterDays != 7 {
		t.Errorf("filterDays = %d", m.filterDays)
	}

	if m.selectedWorkflow != "ci.yml" {
		t.Errorf("selectedWorkflow = %q", m.selectedWorkflow)
	}

	if !m.cacheOnly {
		t.Error("cacheOnly = false, want true")
	}

	if m.baseBranch != "trunk" {
		t.Errorf("baseBranch = %q", m.baseBranch)
	}
}

func TestGetMaxCursorPerView(t *testing.T) {
	t.Parallel()

	m := NewModel("acme/widgets")
	m, _ = m.Update(loadedFixture())

	tests := []struct {
		mode viewMode
		want int
	}{
		{viewOverview, 1},
		{viewWorkflows, 0},
		{viewJobs, 1},
		{viewBranches, 2},
	}

	for _, tt := range tests {
		m.viewMode = tt.mode
		if got := m.getMaxCursor(); got != tt.want {
			t.Errorf("getMaxCursor(%d) = %d, want %d", tt.mode, got, tt.want)
		}
	}
}

func TestTabKeysSwitchViews(t *testing.T) {
	t.Parallel()

	m := NewModel("acme/widgets")
	m, _ = m.Update(loadedFixture())
	m.cursor = 1
	m.scrollTop = 1

	tests := []struct {
		key  string
		want viewMode
	}{
		{"2", viewWorkflows},
		{"3", viewJobs},
		{"4", viewBranches},
		{"1", viewOverview},
	}

	for _, tt := range tests {
		m, _ = m.Update(tea.KeyPressMsg{Code: rune(tt.key[0]), Text: tt.key})

		if m.viewMode != tt.want {
			t.Errorf("key %s: viewMode = %d, want %d", tt.key, m.viewMode, tt.want)
		}

		if m.cursor != 0 || m.scrollTop != 0 {
			t.Errorf("key %s: cursor/scrollTop = %d/%d, want 0/0", tt.key, m.cursor, m.scrollTop)
		}
	}
}

func TestCursorNavigationClamps(t *testing.T) {
	t.Parallel()

	m := NewModel("acme/widgets")
	m, _ = m.Update(loadedFixture())

	m, _ = m.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	if m.cursor != 0 {
		t.Errorf("cursor after k at top = %d, want 0", m.cursor)
	}

	for range 5 {
		m, _ = m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	}

	if m.cursor != 1 {
		t.Errorf("cursor after j past end = %d, want 1 (clamped to runs)", m.cursor)
	}
}

func TestWindowSizeClampsMaxVisible(t *testing.T) {
	t.Parallel()

	m := NewModel("acme/widgets")

	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	if m.maxVisible != 5 {
		t.Errorf("maxVisible at height 10 = %d, want floor of 5", m.maxVisible)
	}

	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	if m.maxVisible != 28 {
		t.Errorf("maxVisible at height 40 = %d, want 28", m.maxVisible)
	}
}

func TestLoadedViewRendersSummary(t *testing.T) {
	t.Parallel()

	m := NewModel("acme/widgets")
	m, _ = m.Update(loadedFixture())

	view := m.View()
	for _, want := range []string{"GHA Performance: acme/widgets", "Total Runs", "2 runs (1 cached, 1 new)"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q", want)
		}
	}
}

func TestErrorViewRendered(t *testing.T) {
	t.Parallel()

	m := NewModel("bad-repo")
	msg := m.loadData()

	loaded, ok := msg.(dataLoadedMsg)
	if !ok {
		t.Fatalf("loadData() = %T, want dataLoadedMsg", msg)
	}

	if loaded.err == nil {
		t.Fatal("expected invalid repo error")
	}

	m, _ = m.Update(loaded)

	if !strings.Contains(m.View(), "invalid repo format") {
		t.Errorf("view = %q", m.View())
	}
}
