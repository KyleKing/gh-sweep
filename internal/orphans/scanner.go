package orphans

import (
	"context"
	"fmt"
	"sync"

	"github.com/KyleKing/gh-sweep/internal/github"
)

// NamespaceScanner scans every non-archived repository in a GitHub user or
// organization namespace for orphaned branches, concurrently up to its
// ScanOptions.Concurrency limit.
type NamespaceScanner struct {
	client  *github.Client
	options ScanOptions
}

// NewNamespaceScanner returns a NamespaceScanner that scans repositories via
// client using options.
func NewNamespaceScanner(client *github.Client, options ScanOptions) *NamespaceScanner {
	return &NamespaceScanner{
		client:  client,
		options: options,
	}
}

// ScanProgress reports a NamespaceScanner's progress as it works through a
// namespace's repositories.
type ScanProgress struct {
	Current     int
	Total       int
	CurrentRepo string
	Orphans     int
}

// ScanNamespace scans every repository in namespace and returns the
// aggregated result.
func (s *NamespaceScanner) ScanNamespace(
	ctx context.Context,
	namespace string,
) (*NamespaceScanResult, error) {
	return s.ScanNamespaceWithProgress(ctx, namespace, nil)
}

// ScanNamespaceWithProgress scans every repository in namespace concurrently,
// sending a ScanProgress update after each repository completes when
// progressCh is non-nil. Updates are dropped rather than blocking if the
// channel isn't ready to receive.
func (s *NamespaceScanner) ScanNamespaceWithProgress(
	ctx context.Context,
	namespace string,
	progressCh chan<- ScanProgress,
) (*NamespaceScanResult, error) {
	repos, isOrg, err := s.client.ListNamespaceRepositories(namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to list namespace repositories: %w", err)
	}

	var nonArchivedRepos []github.Repository
	for _, repo := range repos {
		if !repo.Archived {
			nonArchivedRepos = append(nonArchivedRepos, repo)
		}
	}

	result := &NamespaceScanResult{
		Namespace:  namespace,
		IsOrg:      isOrg,
		TotalRepos: len(nonArchivedRepos),
	}

	if len(nonArchivedRepos) == 0 {
		return result, nil
	}

	resultsCh := make(chan ScanResult, len(nonArchivedRepos))
	semaphore := make(chan struct{}, s.options.Concurrency)

	var wg sync.WaitGroup
	var progressMu sync.Mutex
	scannedCount := 0
	totalOrphans := 0

	for _, repo := range nonArchivedRepos {
		wg.Add(1)
		go func(repo github.Repository) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			}

			scanResult := s.ScanRepo(ctx, repo)
			resultsCh <- scanResult

			if progressCh != nil {
				progressMu.Lock()
				scannedCount++
				totalOrphans += len(scanResult.Orphans)
				progress := ScanProgress{
					Current:     scannedCount,
					Total:       len(nonArchivedRepos),
					CurrentRepo: repo.FullName,
					Orphans:     totalOrphans,
				}
				progressMu.Unlock()

				select {
				case progressCh <- progress:
				default:
				}
			}
		}(repo)
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	for scanResult := range resultsCh {
		result.Results = append(result.Results, scanResult)
		result.TotalOrphans += len(scanResult.Orphans)
	}

	return result, nil
}

// ScanRepo scans repo's non-default branches and returns the ScanResult,
// stopping early (with whatever orphans were already found) if ctx is
// canceled mid-scan.
func (s *NamespaceScanner) ScanRepo(ctx context.Context, repo github.Repository) ScanResult {
	result := ScanResult{
		Repository:    repo,
		DefaultBranch: repo.DefaultBranch,
	}

	branches, err := s.client.ListBranches(repo.Owner, repo.Name)
	if err != nil {
		result.Error = err
		return result
	}

	prs, err := s.client.ListPullRequests(repo.Owner, repo.Name, "all")
	if err != nil {
		result.Error = err
		return result
	}

	detector := NewDetector(s.options)

	for _, branch := range branches {
		if branch.Name == repo.DefaultBranch {
			continue
		}

		select {
		case <-ctx.Done():
			return result
		default:
		}

		if orphan := detector.ClassifyBranch(repo, branch, prs); orphan != nil {
			result.Orphans = append(result.Orphans, *orphan)
		}
	}

	return result
}
