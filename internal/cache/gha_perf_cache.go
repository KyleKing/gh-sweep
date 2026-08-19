// Package cache persists GitHub Actions run timings and other API responses
// to disk and memory so repeated TUI and CLI invocations avoid refetching.
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/KyleKing/gh-sweep/internal/github"
)

const (
	cacheDirPerm  = 0o750
	cacheFilePerm = 0o600
)

// GHAPerfCache holds a repository's cached GitHub Actions run timings.
type GHAPerfCache struct {
	UpdatedAt time.Time          `json:"updated_at"`
	Repo      string             `json:"repo"`
	Runs      []github.RunTiming `json:"runs"`
}

// GHAPerfCacheManager reads and writes GHAPerfCache files on disk, one JSON
// file per repository under its cache directory.
type GHAPerfCacheManager struct {
	cacheDir string
}

// NewGHAPerfCacheManager creates the cache directory (defaulting to
// ~/.cache/gh-sweep/gha-perf when cacheDir is empty) and returns a manager
// for it.
func NewGHAPerfCacheManager(cacheDir string) (*GHAPerfCacheManager, error) {
	if cacheDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		cacheDir = filepath.Join(homeDir, ".cache", "gh-sweep", "gha-perf")
	}

	if err := os.MkdirAll(cacheDir, cacheDirPerm); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &GHAPerfCacheManager{cacheDir: cacheDir}, nil
}

func (m *GHAPerfCacheManager) cacheFilePath(owner, repo string) string {
	safeRepo := fmt.Sprintf("%s_%s.json", owner, repo)
	return filepath.Join(m.cacheDir, safeRepo)
}

// Load reads owner/repo's cache file, returning an empty GHAPerfCache when
// none exists yet.
func (m *GHAPerfCacheManager) Load(owner, repo string) (*GHAPerfCache, error) {
	path := m.cacheFilePath(owner, repo)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &GHAPerfCache{
				Repo: fmt.Sprintf("%s/%s", owner, repo),
				Runs: []github.RunTiming{},
			}, nil
		}

		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	var cache GHAPerfCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("failed to parse cache file: %w", err)
	}

	for i := range cache.Runs {
		cache.Runs[i].Duration = time.Duration(cache.Runs[i].DurationSeconds * float64(time.Second))
		for j := range cache.Runs[i].Jobs {
			cache.Runs[i].Jobs[j].Duration = time.Duration(
				cache.Runs[i].Jobs[j].DurationSeconds * float64(time.Second))
			for k := range cache.Runs[i].Jobs[j].Steps {
				cache.Runs[i].Jobs[j].Steps[k].Duration = time.Duration(
					cache.Runs[i].Jobs[j].Steps[k].DurationSeconds * float64(time.Second))
			}
		}
	}

	return &cache, nil
}

// Save writes cache to owner/repo's cache file, stamping its UpdatedAt and
// Repo fields first.
func (m *GHAPerfCacheManager) Save(owner, repo string, cache *GHAPerfCache) error {
	cache.UpdatedAt = time.Now()
	cache.Repo = fmt.Sprintf("%s/%s", owner, repo)

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	path := m.cacheFilePath(owner, repo)
	if err := os.WriteFile(path, data, cacheFilePerm); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// MergeRuns combines existing and newRuns by RunID (newRuns wins on
// conflict) and returns them sorted oldest-first.
func MergeRuns(existing, newRuns []github.RunTiming) []github.RunTiming {
	byID := make(map[int]*github.RunTiming, len(existing)+len(newRuns))

	for i := range existing {
		byID[existing[i].RunID] = &existing[i]
	}

	for i := range newRuns {
		byID[newRuns[i].RunID] = &newRuns[i]
	}

	merged := make([]github.RunTiming, 0, len(byID))
	for _, r := range byID {
		merged = append(merged, *r)
	}

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].CreatedAt.Before(merged[j].CreatedAt)
	})

	return merged
}

// GetCachedRunIDs returns the set of run IDs currently cached for
// owner/repo.
func (m *GHAPerfCacheManager) GetCachedRunIDs(owner, repo string) (map[int]bool, error) {
	cache, err := m.Load(owner, repo)
	if err != nil {
		return nil, err
	}

	ids := make(map[int]bool)
	for i := range cache.Runs {
		ids[cache.Runs[i].RunID] = true
	}

	return ids, nil
}

// Stats returns the cached run count and last-updated time for owner/repo.
func (m *GHAPerfCacheManager) Stats(owner, repo string) (int, time.Time, error) {
	cache, err := m.Load(owner, repo)
	if err != nil {
		return 0, time.Time{}, err
	}

	return len(cache.Runs), cache.UpdatedAt, nil
}

// Clear removes owner/repo's cache file, treating a missing file as
// success.
func (m *GHAPerfCacheManager) Clear(owner, repo string) error {
	path := m.cacheFilePath(owner, repo)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache file: %w", err)
	}

	return nil
}

// ClearAll removes every cache file in the manager's cache directory,
// treating a missing directory as success.
func (m *GHAPerfCacheManager) ClearAll() error {
	entries, err := os.ReadDir(m.cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			path := filepath.Join(m.cacheDir, entry.Name())
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to remove cache file %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}

// ListCaches returns the owner_repo identifiers of every cache file in the
// manager's cache directory.
func (m *GHAPerfCacheManager) ListCaches() ([]string, error) {
	entries, err := os.ReadDir(m.cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to read cache directory: %w", err)
	}

	var repos []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			name := entry.Name()
			name = name[:len(name)-5]
			repos = append(repos, name)
		}
	}

	return repos, nil
}

// FilterRunsByCommit returns the runs matching commitSHA, exactly or by a
// shared 7-character prefix. An empty commitSHA returns runs unchanged.
func FilterRunsByCommit(runs []github.RunTiming, commitSHA string) []github.RunTiming {
	if commitSHA == "" {
		return runs
	}
	var filtered []github.RunTiming
	for i := range runs {
		r := &runs[i]
		if r.HeadSHA == commitSHA || (len(commitSHA) >= 7 && len(r.HeadSHA) >= 7 &&
			r.HeadSHA[:7] == commitSHA[:7]) {
			filtered = append(filtered, *r)
		}
	}

	return filtered
}

// FilterRunsByConclusion returns the runs with the given conclusion. An
// empty conclusion returns runs unchanged.
func FilterRunsByConclusion(runs []github.RunTiming, conclusion string) []github.RunTiming {
	if conclusion == "" {
		return runs
	}
	var filtered []github.RunTiming
	for i := range runs {
		if runs[i].Conclusion == conclusion {
			filtered = append(filtered, runs[i])
		}
	}

	return filtered
}

// GetRunsInDateRange returns the runs created within [since, until]. A zero
// since or until leaves that bound unenforced.
func GetRunsInDateRange(runs []github.RunTiming, since, until time.Time) []github.RunTiming {
	var filtered []github.RunTiming
	for i := range runs {
		r := &runs[i]
		if !since.IsZero() && r.CreatedAt.Before(since) {
			continue
		}
		if !until.IsZero() && r.CreatedAt.After(until) {
			continue
		}
		filtered = append(filtered, *r)
	}

	return filtered
}

// GetLatestRunPerWorkflow returns each workflow's most recently created
// run, sorted by workflow name.
func GetLatestRunPerWorkflow(runs []github.RunTiming) []github.RunTiming {
	latest := make(map[string]*github.RunTiming)
	for i := range runs {
		r := &runs[i]
		if existing, ok := latest[r.Workflow]; !ok || r.CreatedAt.After(existing.CreatedAt) {
			latest[r.Workflow] = r
		}
	}

	result := make([]github.RunTiming, 0, len(latest))
	for _, r := range latest {
		result = append(result, *r)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Workflow < result[j].Workflow
	})

	return result
}
