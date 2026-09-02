package policy

import (
	"fmt"

	"github.com/KyleKing/gh-sweep/internal/config"
	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/orphans"
)

const defaultNoPRGraceDays = 21

// PrunableBranch is one branch the policy says should not exist, carried
// alongside the diff so Apply deletes exactly what Evaluate reported.
type PrunableBranch struct {
	Owner  string
	Repo   string
	Name   string
	Reason string
}

// evaluateBranches classifies a repo's leftover branches with the same
// detector the orphans view uses, so the two cannot disagree about what counts
// as prunable.
func evaluateBranches(
	client *github.Client,
	declared *config.PolicyBranches,
	owner, repo string,
) ([]Diff, []PrunableBranch, error) {
	branches, err := client.ListBranchesWithDates(owner, repo)
	if err != nil {
		return nil, nil, fmt.Errorf("listing branches: %w", err)
	}

	prs, err := client.ListPullRequests(owner, repo, "all")
	if err != nil {
		return nil, nil, fmt.Errorf("listing pull requests: %w", err)
	}

	options := orphans.DefaultScanOptions()
	options.StaleDaysThreshold = graceDays(declared)

	if len(declared.ExcludePatterns) > 0 {
		options.ExcludePatterns = append(options.ExcludePatterns, declared.ExcludePatterns...)
	}

	detector := orphans.NewDetector(options)
	repoRef := github.Repository{
		Name:     repo,
		FullName: owner + "/" + repo,
		Owner:    owner,
	}

	var (
		diffs     []Diff
		prunables []PrunableBranch
	)

	for i := range branches {
		orphan := detector.ClassifyBranch(repoRef, branches[i], prs)
		if orphan == nil || !pruned(declared, orphan.Type) {
			continue
		}

		diffs = append(diffs, Diff{
			Domain:  DomainBranches,
			Field:   orphan.BranchName,
			Current: describeOrphan(orphan),
			Desired: "deleted",
		})
		prunables = append(prunables, PrunableBranch{
			Owner:  owner,
			Repo:   repo,
			Name:   orphan.BranchName,
			Reason: orphan.Type.Label(),
		})
	}

	return diffs, prunables, nil
}

func describeOrphan(orphan *orphans.OrphanedBranch) string {
	if orphan.PRNumber != nil {
		return fmt.Sprintf("%s #%d, %dd", orphan.Type.Label(), *orphan.PRNumber, orphan.DaysSinceActivity)
	}

	return fmt.Sprintf("%s, %dd", orphan.Type.Label(), orphan.DaysSinceActivity)
}

func pruned(declared *config.PolicyBranches, orphanType orphans.OrphanType) bool {
	switch orphanType {
	case orphans.OrphanTypeMergedPR:
		return enabled(declared.PruneMerged)
	case orphans.OrphanTypeClosedPR:
		return enabled(declared.PruneClosed)
	case orphans.OrphanTypeStale:
		return enabled(declared.PruneNoPR)
	case orphans.OrphanTypeRecentNoPR:
		return false
	}

	return false
}

func graceDays(declared *config.PolicyBranches) int {
	if declared.NoPRGraceDays != nil {
		return *declared.NoPRGraceDays
	}

	return defaultNoPRGraceDays
}

func enabled(flag *bool) bool {
	return flag != nil && *flag
}
