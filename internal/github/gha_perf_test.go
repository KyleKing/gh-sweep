package github_test

import (
	"testing"
	"time"

	"github.com/KyleKing/gh-sweep/internal/github"
)

func perfRuns() []github.RunTiming {
	return []github.RunTiming{
		{
			RunID:      1,
			Workflow:   "ci",
			Branch:     "main",
			Conclusion: "success",
			CreatedAt:  time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC),
			Duration:   4 * time.Minute,
			Jobs: []github.JobTiming{
				{Name: "build", Duration: time.Minute},
				{Name: "test", Duration: 3 * time.Minute},
			},
		},
		{
			RunID:      2,
			Workflow:   "ci",
			Branch:     "feature",
			Conclusion: "failure",
			CreatedAt:  time.Date(2026, 1, 12, 12, 0, 0, 0, time.UTC),
			Duration:   8 * time.Minute,
			Jobs: []github.JobTiming{
				{Name: "build", Duration: 2 * time.Minute},
			},
		},
		{
			RunID:      3,
			Workflow:   "release",
			Branch:     "main",
			Conclusion: "success",
			CreatedAt:  time.Date(2026, 1, 14, 12, 0, 0, 0, time.UTC),
			Duration:   10 * time.Minute,
		},
	}
}

func TestComputeWorkflowStats(t *testing.T) {
	t.Parallel()

	stats := github.ComputeWorkflowStats(perfRuns())

	ci := stats["ci"]
	if ci == nil {
		t.Fatal("missing ci workflow stats")
	}

	if ci.TotalRuns != 2 || ci.FailureCount != 1 {
		t.Errorf("ci runs/failures = %d/%d, want 2/1", ci.TotalRuns, ci.FailureCount)
	}

	if ci.AvgDuration != 6*time.Minute {
		t.Errorf("ci avg = %v, want 6m", ci.AvgDuration)
	}

	if ci.MinDuration != 4*time.Minute || ci.MaxDuration != 8*time.Minute {
		t.Errorf("ci min/max = %v/%v", ci.MinDuration, ci.MaxDuration)
	}

	if ci.SuccessRate != 50 {
		t.Errorf("ci success rate = %v, want 50", ci.SuccessRate)
	}
}

func TestComputeJobStats(t *testing.T) {
	t.Parallel()

	stats := github.ComputeJobStats(perfRuns())

	build := stats["ci:build"]
	if build == nil {
		t.Fatal("missing ci:build job stats")
	}

	if build.TotalRuns != 2 {
		t.Errorf("build runs = %d, want 2", build.TotalRuns)
	}

	if build.AvgDuration != 90*time.Second {
		t.Errorf("build avg = %v, want 1m30s", build.AvgDuration)
	}

	if _, ok := stats["ci:test"]; !ok {
		t.Error("missing ci:test job stats")
	}
}

func TestComputeBranchStatsDelta(t *testing.T) {
	t.Parallel()

	stats := github.ComputeBranchStats(perfRuns(), "main")

	main := stats["main"]
	if main == nil || main.AvgDuration != 7*time.Minute {
		t.Fatalf("main avg = %+v, want 7m", main)
	}

	feature := stats["feature"]
	if feature == nil {
		t.Fatal("missing feature branch stats")
	}

	if feature.DeltaVsBase != 60 {
		t.Errorf("feature delta seconds = %v, want 60", feature.DeltaVsBase)
	}

	wantPct := float64(time.Minute) / float64(7*time.Minute) * 100
	if feature.DeltaVsBasePct != wantPct {
		t.Errorf("feature delta pct = %v, want %v", feature.DeltaVsBasePct, wantPct)
	}
}

func TestFilterRuns(t *testing.T) {
	t.Parallel()

	runs := perfRuns()

	if got := github.FilterRunsByBranch(runs, "main"); len(got) != 2 {
		t.Errorf("branch filter = %d runs, want 2", len(got))
	}

	if got := github.FilterRunsByBranch(runs, ""); len(got) != 3 {
		t.Errorf("empty branch filter = %d runs, want all 3", len(got))
	}

	if got := github.FilterRunsByWorkflows(runs, []string{"release"}); len(got) != 1 {
		t.Errorf("workflow filter = %d runs, want 1", len(got))
	}

	since := time.Date(2026, 1, 11, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, 1, 13, 0, 0, 0, 0, time.UTC)
	if got := github.FilterRunsByTimeRange(runs, since, until); len(got) != 1 || got[0].RunID != 2 {
		t.Errorf("time filter = %+v, want only run 2", got)
	}
}

func TestSortRunsByDate(t *testing.T) {
	t.Parallel()

	runs := perfRuns()

	github.SortRunsByDate(runs, false)
	if runs[0].RunID != 3 {
		t.Errorf("descending first run = %d, want 3", runs[0].RunID)
	}

	github.SortRunsByDate(runs, true)
	if runs[0].RunID != 1 {
		t.Errorf("ascending first run = %d, want 1", runs[0].RunID)
	}
}

func TestGetTopJobsByDuration(t *testing.T) {
	t.Parallel()

	stats := github.ComputeJobStats(perfRuns())

	jobs := github.GetTopJobsByDuration(stats, 0)
	if len(jobs) != 2 {
		t.Fatalf("jobs = %d, want 2", len(jobs))
	}

	if jobs[0].AvgDuration < jobs[1].AvgDuration {
		t.Error("jobs not sorted by avg duration descending")
	}

	if got := github.GetTopJobsByDuration(stats, 1); len(got) != 1 {
		t.Errorf("limited jobs = %d, want 1", len(got))
	}
}

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		d    time.Duration
		want string
	}{
		{45 * time.Second, "45s"},
		{90 * time.Second, "1.5m"},
		{90 * time.Minute, "1.5h"},
	}

	for _, tt := range tests {
		if got := github.FormatDuration(tt.d); got != tt.want {
			t.Errorf("FormatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestAnalyzeWebhookHealth(t *testing.T) {
	t.Parallel()

	deliveries := []github.WebhookDelivery{
		{ID: 1, Status: 200, Duration: 100},
		{ID: 2, Status: 200, Duration: 200},
		{ID: 3, Status: 500, Duration: 300},
		{ID: 4, Status: 404, Duration: 400},
	}

	health := github.AnalyzeWebhookHealth(deliveries)

	if health.TotalDeliveries != 4 {
		t.Errorf("TotalDeliveries = %d, want 4", health.TotalDeliveries)
	}

	if health.Failures != 2 {
		t.Errorf("Failures = %d, want 2", health.Failures)
	}

	if health.SuccessRate != 50 {
		t.Errorf("SuccessRate = %v, want 50", health.SuccessRate)
	}
}

func TestCompareProtectionRules(t *testing.T) {
	t.Parallel()

	rules := []*github.ProtectionRule{
		{Repository: "acme/widgets", RequiredReviews: 2, EnforceAdmins: true},
		{
			Repository:              "acme/gadgets",
			RequiredReviews:         0,
			EnforceAdmins:           true,
			RequireCodeOwnerReviews: true,
		},
	}

	diffs := github.CompareProtectionRules(rules)

	if len(diffs["RequiredReviews"]) != 1 {
		t.Errorf("RequiredReviews diffs = %v", diffs["RequiredReviews"])
	}

	if len(diffs["RequireCodeOwnerReviews"]) != 1 {
		t.Errorf("RequireCodeOwnerReviews diffs = %v", diffs["RequireCodeOwnerReviews"])
	}

	if len(diffs["EnforceAdmins"]) != 0 {
		t.Errorf("EnforceAdmins diffs = %v, want none", diffs["EnforceAdmins"])
	}

	if got := github.CompareProtectionRules(rules[:1]); len(got) != 0 {
		t.Errorf("single rule diffs = %v, want empty", got)
	}
}

func TestCompareReleases(t *testing.T) {
	t.Parallel()

	releases := map[string]*github.Release{
		"acme/current":  {TagName: "v2.0.0", PublishedAt: time.Now().Add(-24 * time.Hour)},
		"acme/outdated": {TagName: "2.0.0", PublishedAt: time.Now().AddDate(-1, 0, 0)},
		"acme/none":     nil,
	}

	comparison := github.CompareReleases(releases)

	if len(comparison.Repositories) != 3 {
		t.Errorf("Repositories = %d, want 3", len(comparison.Repositories))
	}

	if len(comparison.OutdatedRepos) != 2 {
		t.Errorf("OutdatedRepos = %v, want outdated and none", comparison.OutdatedRepos)
	}

	if len(comparison.NonSemVerRepos) != 1 || comparison.NonSemVerRepos[0] != "acme/outdated" {
		t.Errorf("NonSemVerRepos = %v, want [acme/outdated]", comparison.NonSemVerRepos)
	}
}
