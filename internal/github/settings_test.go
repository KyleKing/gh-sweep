package github_test

import (
	"testing"

	"github.com/KyleKing/gh-sweep/internal/github"
)

func compareSettingsBaseline() *github.RepoSettings {
	return &github.RepoSettings{
		Repository:          "owner/baseline",
		DefaultBranch:       "main",
		AllowMergeCommit:    false,
		AllowSquashMerge:    true,
		AllowRebaseMerge:    true,
		DeleteBranchOnMerge: true,
		HasIssues:           true,
		HasProjects:         false,
		HasWiki:             false,
	}
}

func compareSettingsCases() []struct {
	name          string
	current       *github.RepoSettings
	expectedDiffs int
	hasCritical   bool
} {
	return []struct {
		name          string
		current       *github.RepoSettings
		expectedDiffs int
		hasCritical   bool
	}{
		{
			name: "identical settings",
			current: &github.RepoSettings{
				Repository:          "owner/repo",
				DefaultBranch:       "main",
				AllowMergeCommit:    false,
				AllowSquashMerge:    true,
				AllowRebaseMerge:    true,
				DeleteBranchOnMerge: true,
				HasIssues:           true,
				HasProjects:         false,
				HasWiki:             false,
			},
			expectedDiffs: 0,
			hasCritical:   false,
		},
		{
			name: "different default branch",
			current: &github.RepoSettings{
				Repository:          "owner/repo",
				DefaultBranch:       "master", // Different
				AllowMergeCommit:    false,
				AllowSquashMerge:    true,
				AllowRebaseMerge:    true,
				DeleteBranchOnMerge: true,
				HasIssues:           true,
				HasProjects:         false,
				HasWiki:             false,
			},
			expectedDiffs: 1, // DefaultBranch
			hasCritical:   false,
		},
		{
			name: "different merge strategies",
			current: &github.RepoSettings{
				Repository:          "owner/repo",
				DefaultBranch:       "main",
				AllowMergeCommit:    true,  // Different
				AllowSquashMerge:    false, // Different
				AllowRebaseMerge:    true,
				DeleteBranchOnMerge: true,
				HasIssues:           true,
				HasProjects:         false,
				HasWiki:             false,
			},
			expectedDiffs: 1, // MergeStrategies (grouped)
			hasCritical:   false,
		},
		{
			name: "multiple differences",
			current: &github.RepoSettings{
				Repository:          "owner/repo",
				DefaultBranch:       "develop",
				AllowMergeCommit:    true,
				AllowSquashMerge:    false,
				AllowRebaseMerge:    false,
				DeleteBranchOnMerge: false, // Different
				HasIssues:           true,
				HasProjects:         false,
				HasWiki:             false,
			},
			expectedDiffs: 3, // DefaultBranch, MergeStrategies, DeleteBranchOnMerge
			hasCritical:   false,
		},
	}
}

// TestCompareSettings tests settings comparison logic.
func TestCompareSettings(t *testing.T) {
	t.Parallel()

	baseline := compareSettingsBaseline()

	for _, tt := range compareSettingsCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			diffs := github.CompareSettings(baseline, tt.current)

			if len(diffs) != tt.expectedDiffs {
				t.Errorf("Expected %d diffs, got %d: %+v",
					tt.expectedDiffs, len(diffs), diffs)
			}

			// Check severity levels
			hasCritical := false
			for _, diff := range diffs {
				if diff.Severity == "critical" {
					hasCritical = true
					break
				}
			}

			if hasCritical != tt.hasCritical {
				t.Errorf("Expected hasCritical=%v, got %v",
					tt.hasCritical, hasCritical)
			}
		})
	}
}

// TestSettingsDiffSeverity tests severity classification.
func TestSettingsDiffSeverity(t *testing.T) {
	t.Parallel()

	baseline := &github.RepoSettings{
		DefaultBranch:       "main",
		DeleteBranchOnMerge: true,
	}

	current := &github.RepoSettings{
		DefaultBranch:       "master",
		DeleteBranchOnMerge: false,
	}

	diffs := github.CompareSettings(baseline, current)

	// DefaultBranch should be warning
	defaultBranchDiff := findDiff(diffs, "DefaultBranch")
	if defaultBranchDiff == nil {
		t.Fatal("Expected DefaultBranch diff")
	}
	if defaultBranchDiff.Severity != "warning" {
		t.Errorf("Expected warning severity for DefaultBranch, got %s",
			defaultBranchDiff.Severity)
	}

	// DeleteBranchOnMerge should be info
	deleteBranchDiff := findDiff(diffs, "DeleteBranchOnMerge")
	if deleteBranchDiff == nil {
		t.Fatal("Expected DeleteBranchOnMerge diff")
	}
	if deleteBranchDiff.Severity != "info" {
		t.Errorf("Expected info severity for DeleteBranchOnMerge, got %s",
			deleteBranchDiff.Severity)
	}
}

// TestBatchCompareSettings tests comparing multiple repositories.
func TestBatchCompareSettings(t *testing.T) {
	t.Parallel()

	baseline := &github.RepoSettings{
		Repository:    "owner/baseline",
		DefaultBranch: "main",
	}

	repos := []*github.RepoSettings{
		{Repository: "owner/repo1", DefaultBranch: "main"},
		{Repository: "owner/repo2", DefaultBranch: "master"},
		{Repository: "owner/repo3", DefaultBranch: "main"},
	}

	allDiffs := make(map[string][]github.SettingsDiff)
	for _, repo := range repos {
		diffs := github.CompareSettings(baseline, repo)
		if len(diffs) > 0 {
			allDiffs[repo.Repository] = diffs
		}
	}

	// Only repo2 should have diffs
	if len(allDiffs) != 1 {
		t.Errorf("Expected 1 repo with diffs, got %d", len(allDiffs))
	}

	if _, ok := allDiffs["owner/repo2"]; !ok {
		t.Error("Expected owner/repo2 to have diffs")
	}
}

// Helper function to find a specific diff.
func findDiff(diffs []github.SettingsDiff, field string) *github.SettingsDiff {
	for i := range diffs {
		if diffs[i].Field == field {
			return &diffs[i]
		}
	}

	return nil
}
