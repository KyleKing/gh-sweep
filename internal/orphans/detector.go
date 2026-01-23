package orphans

import (
	"path/filepath"
	"time"

	"github.com/KyleKing/gh-sweep/internal/github"
)

type Detector struct {
	options ScanOptions
}

func NewDetector(options ScanOptions) *Detector {
	return &Detector{options: options}
}

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

	daysSince := int(time.Since(branch.LastCommitDate).Hours() / 24)

	var mergedPR, closedPR, openPR *github.PullRequest
	for i := range prs {
		pr := &prs[i]
		if pr.Head.Ref == branch.Name {
			switch {
			case pr.MergedAt != nil:
				mergedPR = pr
			case pr.State == "closed":
				closedPR = pr
			case pr.State == "open":
				openPR = pr
			}
		}
	}

	if openPR != nil {
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
