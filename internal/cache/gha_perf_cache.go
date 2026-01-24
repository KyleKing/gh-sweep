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

type GHAPerfCache struct {
	UpdatedAt time.Time            `json:"updated_at"`
	Repo      string               `json:"repo"`
	Runs      []github.RunTiming   `json:"runs"`
}

type GHAPerfCacheManager struct {
	cacheDir string
}

func NewGHAPerfCacheManager(cacheDir string) (*GHAPerfCacheManager, error) {
	if cacheDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		cacheDir = filepath.Join(homeDir, ".cache", "gh-sweep", "gha-perf")
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &GHAPerfCacheManager{cacheDir: cacheDir}, nil
}

func (m *GHAPerfCacheManager) cacheFilePath(owner, repo string) string {
	safeRepo := fmt.Sprintf("%s_%s.json", owner, repo)
	return filepath.Join(m.cacheDir, safeRepo)
}

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

func (m *GHAPerfCacheManager) Save(owner, repo string, cache *GHAPerfCache) error {
	cache.UpdatedAt = time.Now()
	cache.Repo = fmt.Sprintf("%s/%s", owner, repo)

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	path := m.cacheFilePath(owner, repo)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

func (m *GHAPerfCacheManager) MergeRuns(existing, newRuns []github.RunTiming) []github.RunTiming {
	byID := make(map[int]github.RunTiming)

	for _, r := range existing {
		byID[r.RunID] = r
	}

	for _, r := range newRuns {
		byID[r.RunID] = r
	}

	merged := make([]github.RunTiming, 0, len(byID))
	for _, r := range byID {
		merged = append(merged, r)
	}

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].CreatedAt.Before(merged[j].CreatedAt)
	})

	return merged
}

func (m *GHAPerfCacheManager) GetCachedRunIDs(owner, repo string) (map[int]bool, error) {
	cache, err := m.Load(owner, repo)
	if err != nil {
		return nil, err
	}

	ids := make(map[int]bool)
	for _, r := range cache.Runs {
		ids[r.RunID] = true
	}

	return ids, nil
}

func (m *GHAPerfCacheManager) Stats(owner, repo string) (int, time.Time, error) {
	cache, err := m.Load(owner, repo)
	if err != nil {
		return 0, time.Time{}, err
	}

	return len(cache.Runs), cache.UpdatedAt, nil
}

func (m *GHAPerfCacheManager) Clear(owner, repo string) error {
	path := m.cacheFilePath(owner, repo)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache file: %w", err)
	}
	return nil
}

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

func FilterRunsByCommit(runs []github.RunTiming, commitSHA string) []github.RunTiming {
	if commitSHA == "" {
		return runs
	}
	var filtered []github.RunTiming
	for _, r := range runs {
		if r.HeadSHA == commitSHA || (len(commitSHA) >= 7 && len(r.HeadSHA) >= 7 &&
			r.HeadSHA[:7] == commitSHA[:7]) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func FilterRunsByConclusion(runs []github.RunTiming, conclusion string) []github.RunTiming {
	if conclusion == "" {
		return runs
	}
	var filtered []github.RunTiming
	for _, r := range runs {
		if r.Conclusion == conclusion {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func GetRunsInDateRange(runs []github.RunTiming, since, until time.Time) []github.RunTiming {
	var filtered []github.RunTiming
	for _, r := range runs {
		if !since.IsZero() && r.CreatedAt.Before(since) {
			continue
		}
		if !until.IsZero() && r.CreatedAt.After(until) {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

func GetLatestRunPerWorkflow(runs []github.RunTiming) []github.RunTiming {
	latest := make(map[string]github.RunTiming)
	for _, r := range runs {
		if existing, ok := latest[r.Workflow]; !ok || r.CreatedAt.After(existing.CreatedAt) {
			latest[r.Workflow] = r
		}
	}

	result := make([]github.RunTiming, 0, len(latest))
	for _, r := range latest {
		result = append(result, r)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Workflow < result[j].Workflow
	})

	return result
}
