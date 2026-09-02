package orphans

import (
	"path/filepath"
	"time"

	"github.com/KyleKing/gh-sweep/internal/github"
)

const hoursPerDay = 24

// Detector classifies individual branches as orphaned according to a fixed
// ScanOptions.
type Detector struct {
	options ScanOptions
}

// NewDetector returns a Detector that classifies branches using options.
func NewDetector(options ScanOptions) *Detector {
	return &Detector{options: options}
}

// ClassifyBranch returns branch's OrphanedBranch classification, or nil when
// branch is excluded, protected, has an open PR, or doesn't otherwise
// qualify as orphaned.
func (d *Detector) ClassifyBranch(
	repo github.Repository,
	branch github.Branch,
	prs []github.PullRequest,
) *OrphanedBranch {
	if d.shouldExclude(branch.Name) {
		return nil
	}

	if branch.Protected && !d.options.IncludeProtected {
		return nil
	}

	daysSince := int(time.Since(branch.LastCommitDate).Hours() / hoursPerDay)

	var mergedPR, closedPR, openPR *github.PullRequest
	for i := range prs {
		pr := &prs[i]
		if pr.Head.Ref == branch.Name {
			switch {
			case pr.MergedAt != nil:
				mergedPR = pr
			case pr.State == prStateClosed:
				closedPR = pr
			case pr.State == "open":
				openPR = pr
			}
		}
	}

	if openPR != nil {
		return nil
	}

	if d.tooRecent(branch, daysSince) {
		return nil
	}

	orphan := OrphanedBranch{
		Repository:        repo.FullName,
		BranchName:        branch.Name,
		SHA:               branch.SHA,
		LastCommitDate:    branch.LastCommitDate,
		DaysSinceActivity: daysSince,
		Protected:         branch.Protected,
	}

	switch {
	case mergedPR != nil:
		orphan.Type = OrphanTypeMergedPR
		orphan.PRNumber = &mergedPR.Number
		orphan.PRTitle = &mergedPR.Title

		return &orphan

	case closedPR != nil:
		orphan.Type = OrphanTypeClosedPR
		orphan.PRNumber = &closedPR.Number
		orphan.PRTitle = &closedPR.Title

		return &orphan

	case daysSince >= d.options.StaleDaysThreshold:
		orphan.Type = OrphanTypeStale
		return &orphan

	case d.options.IncludeRecentNoPR:
		orphan.Type = OrphanTypeRecentNoPR
		return &orphan
	}

	return nil
}

// tooRecent reports whether MinAgeDays spares this branch. An unknown commit
// date reads as too recent to touch rather than infinitely old, or the guard
// spares nothing on the branches it could not date.
func (d *Detector) tooRecent(branch github.Branch, daysSince int) bool {
	if d.options.MinAgeDays <= 0 {
		return false
	}

	return branch.LastCommitDate.IsZero() || daysSince < d.options.MinAgeDays
}

func (d *Detector) shouldExclude(branchName string) bool {
	for _, pattern := range d.options.ExcludePatterns {
		matched, err := filepath.Match(pattern, branchName)
		if err == nil && matched {
			return true
		}

		if pattern == branchName {
			return true
		}
	}

	return false
}
