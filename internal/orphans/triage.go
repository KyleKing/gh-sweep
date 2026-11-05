package orphans

import (
	"fmt"

	"github.com/KyleKing/gh-sweep/internal/github"
)

// Comparer reports what a branch carries beyond a base branch.
type Comparer interface {
	CompareBranch(owner, repo, base, head string) (github.Divergence, error)
}

// Triage separates stale no-PR branches that are safe to delete from those
// holding changes found nowhere else. A stale branch whose tree differs from
// base (or whose comparison fails) becomes OrphanTypeUnmergedWork: with no PR to
// restore it from, deleting it loses the work. Other types pass through, and
// unmerged work gains its commit and file sample.
func Triage(c Comparer, owner, repo, base string, orphan *OrphanedBranch) {
	if orphan.Type != OrphanTypeStale && orphan.Type != OrphanTypeUnmergedWork {
		return
	}

	divergence, err := c.CompareBranch(owner, repo, base, orphan.SHA)

	switch {
	case err != nil:
		orphan.Type = OrphanTypeUnmergedWork
		orphan.Review = &Review{Note: fmt.Sprintf("comparison with %s failed: %v", base, err)}
	case divergence.FileCount == 0 && orphan.Type == OrphanTypeStale:
		return
	default:
		orphan.Type = OrphanTypeUnmergedWork
		review := &Review{
			UniqueCommits:  divergence.AheadBy,
			CommitSubjects: divergence.Commits,
			ChangedFiles:   divergence.Files,
		}

		if orphan.Review != nil {
			review.Note = orphan.Review.Note
		}

		orphan.Review = review
	}
}

var _ Comparer = (*github.Client)(nil)
