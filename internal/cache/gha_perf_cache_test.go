package cache

import (
	"testing"
	"time"

	"github.com/KyleKing/gh-sweep/internal/github"
)

func TestNewGHAPerfCacheManagerCreatesDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir() + "/nested"

	m, err := NewGHAPerfCacheManager(dir)
	if err != nil {
		t.Fatalf("NewGHAPerfCacheManager() error = %v", err)
	}
	if m.cacheDir != dir {
		t.Errorf("cacheDir = %q, want %q", m.cacheDir, dir)
	}
}

func TestLoadMissingCacheReturnsEmpty(t *testing.T) {
	t.Parallel()

	m, err := NewGHAPerfCacheManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewGHAPerfCacheManager() error = %v", err)
	}

	cache, err := m.Load("acme", "widgets")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cache.Repo != "acme/widgets" || len(cache.Runs) != 0 {
		t.Errorf("cache = %+v, want empty acme/widgets", cache)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	t.Parallel()

	m, err := NewGHAPerfCacheManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewGHAPerfCacheManager() error = %v", err)
	}

	cache := &GHAPerfCache{Runs: []github.RunTiming{
		{RunID: 1, Workflow: "ci", DurationSeconds: 90, CreatedAt: time.Now()},
	}}

	if err := m.Save("acme", "widgets", cache); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := m.Load("acme", "widgets")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(loaded.Runs) != 1 || loaded.Runs[0].RunID != 1 {
		t.Fatalf("loaded runs = %+v", loaded.Runs)
	}
	if loaded.Runs[0].Duration != 90*time.Second {
		t.Errorf("Duration = %v, want 90s (derived from DurationSeconds on load)", loaded.Runs[0].Duration)
	}
	if loaded.Repo != "acme/widgets" {
		t.Errorf("Repo = %q, want acme/widgets", loaded.Repo)
	}
}

func TestMergeRuns(t *testing.T) {
	t.Parallel()

	m := &GHAPerfCacheManager{}

	older := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	existing := []github.RunTiming{{RunID: 1, CreatedAt: newer, Conclusion: "success"}}
	incoming := []github.RunTiming{
		{RunID: 1, CreatedAt: newer, Conclusion: "failure"}, // updates the existing run
		{RunID: 2, CreatedAt: older, Conclusion: "success"},
	}

	merged := m.MergeRuns(existing, incoming)

	if len(merged) != 2 {
		t.Fatalf("merged = %+v, want 2 runs", merged)
	}
	if merged[0].RunID != 2 || merged[1].RunID != 1 {
		t.Errorf("merged not sorted oldest-first: %+v", merged)
	}
	if merged[1].Conclusion != "failure" {
		t.Errorf("existing run not overwritten by incoming: %+v", merged[1])
	}
}

func TestGetCachedRunIDsAndStats(t *testing.T) {
	t.Parallel()

	m, err := NewGHAPerfCacheManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewGHAPerfCacheManager() error = %v", err)
	}

	if err := m.Save("acme", "widgets", &GHAPerfCache{
		Runs: []github.RunTiming{{RunID: 5}, {RunID: 9}},
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	ids, err := m.GetCachedRunIDs("acme", "widgets")
	if err != nil {
		t.Fatalf("GetCachedRunIDs() error = %v", err)
	}
	if !ids[5] || !ids[9] || len(ids) != 2 {
		t.Errorf("ids = %v, want {5, 9}", ids)
	}

	count, updatedAt, err := m.Stats("acme", "widgets")
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if count != 2 || updatedAt.IsZero() {
		t.Errorf("Stats() = (%d, %v)", count, updatedAt)
	}
}

func TestClearAndClearAll(t *testing.T) {
	t.Parallel()

	m, err := NewGHAPerfCacheManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewGHAPerfCacheManager() error = %v", err)
	}

	if err := m.Save("acme", "widgets", &GHAPerfCache{}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if err := m.Save("acme", "gadgets", &GHAPerfCache{}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if err := m.Clear("acme", "widgets"); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}
	if err := m.Clear("acme", "widgets"); err != nil {
		t.Errorf("Clear() on an already-missing file should be a no-op, got: %v", err)
	}

	repos, err := m.ListCaches()
	if err != nil {
		t.Fatalf("ListCaches() error = %v", err)
	}
	if len(repos) != 1 || repos[0] != "acme_gadgets" {
		t.Errorf("ListCaches() = %v, want [acme_gadgets]", repos)
	}

	if err := m.ClearAll(); err != nil {
		t.Fatalf("ClearAll() error = %v", err)
	}

	repos, err = m.ListCaches()
	if err != nil {
		t.Fatalf("ListCaches() error = %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("ListCaches() after ClearAll = %v, want empty", repos)
	}
}

func TestListCachesMissingDir(t *testing.T) {
	t.Parallel()

	m := &GHAPerfCacheManager{cacheDir: t.TempDir() + "/does-not-exist"}

	repos, err := m.ListCaches()
	if err != nil {
		t.Fatalf("ListCaches() error = %v", err)
	}
	if repos != nil {
		t.Errorf("ListCaches() = %v, want nil", repos)
	}

	if err := m.ClearAll(); err != nil {
		t.Errorf("ClearAll() on a missing dir should be a no-op, got: %v", err)
	}
}

func TestFilterRunsByCommit(t *testing.T) {
	t.Parallel()

	runs := []github.RunTiming{
		{RunID: 1, HeadSHA: "abcdef1234"},
		{RunID: 2, HeadSHA: "zzzzzzzzzz"},
	}

	if got := FilterRunsByCommit(runs, ""); len(got) != 2 {
		t.Errorf("FilterRunsByCommit(empty) = %d runs, want 2", len(got))
	}
	if got := FilterRunsByCommit(runs, "abcdef1234"); len(got) != 1 || got[0].RunID != 1 {
		t.Errorf("FilterRunsByCommit(exact) = %+v", got)
	}
	if got := FilterRunsByCommit(runs, "abcdef19999"); len(got) != 1 || got[0].RunID != 1 {
		t.Errorf("FilterRunsByCommit(matching 7-char prefix) = %+v", got)
	}
}

func TestFilterRunsByConclusion(t *testing.T) {
	t.Parallel()

	runs := []github.RunTiming{
		{RunID: 1, Conclusion: "success"},
		{RunID: 2, Conclusion: "failure"},
	}

	if got := FilterRunsByConclusion(runs, ""); len(got) != 2 {
		t.Errorf("FilterRunsByConclusion(empty) = %d runs, want 2", len(got))
	}
	if got := FilterRunsByConclusion(runs, "failure"); len(got) != 1 || got[0].RunID != 2 {
		t.Errorf("FilterRunsByConclusion(failure) = %+v", got)
	}
}

func TestGetRunsInDateRange(t *testing.T) {
	t.Parallel()

	day1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	day3 := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)
	runs := []github.RunTiming{{RunID: 1, CreatedAt: day1}, {RunID: 2, CreatedAt: day2}, {RunID: 3, CreatedAt: day3}}

	got := GetRunsInDateRange(runs, day2, time.Time{})
	if len(got) != 2 || got[0].RunID != 2 {
		t.Errorf("GetRunsInDateRange(since=day2) = %+v", got)
	}

	got = GetRunsInDateRange(runs, time.Time{}, day2)
	if len(got) != 2 || got[1].RunID != 2 {
		t.Errorf("GetRunsInDateRange(until=day2) = %+v", got)
	}
}

func TestGetLatestRunPerWorkflow(t *testing.T) {
	t.Parallel()

	older := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	runs := []github.RunTiming{
		{RunID: 1, Workflow: "ci", CreatedAt: older},
		{RunID: 2, Workflow: "ci", CreatedAt: newer},
		{RunID: 3, Workflow: "release", CreatedAt: older},
	}

	got := GetLatestRunPerWorkflow(runs)
	if len(got) != 2 {
		t.Fatalf("got %d workflows, want 2", len(got))
	}
	if got[0].Workflow != "ci" || got[0].RunID != 2 {
		t.Errorf("ci entry = %+v, want the newer run", got[0])
	}
	if got[1].Workflow != "release" || got[1].RunID != 3 {
		t.Errorf("release entry = %+v", got[1])
	}
}
