package orphans

import (
	"math"
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
	dated := !branch.LastCommitDate.IsZero()
	active := dated && daysSince < d.options.ActiveDaysThreshold

	pulls := pullRequestsByState(branch.Name, prs)
	if pulls.open != nil {
		return nil
	}

	orphan := OrphanedBranch{
		Repository:        repo.FullName,
		BranchName:        branch.Name,
		SHA:               branch.SHA,
		LastCommitDate:    branch.LastCommitDate,
		DaysSinceActivity: int32(min(daysSince, math.MaxInt32)), //nolint:gosec // clamped above
		Protected:         branch.Protected,
	}

	switch {
	case active && (pulls.merged != nil || pulls.closed != nil):
		return nil

	case pulls.merged != nil:
		return closedPRBranch(orphan, OrphanTypeMergedPR, pulls.merged, branch.SHA)

	case pulls.closed != nil:
		return closedPRBranch(orphan, OrphanTypeClosedPR, pulls.closed, branch.SHA)
	}

	orphan.Type = d.noPRType(!active && dated && daysSince >= d.options.StaleDaysThreshold)
	if orphan.Type == "" {
		return nil
	}

	return &orphan
}

func (d *Detector) noPRType(stale bool) OrphanType {
	switch {
	case stale:
		return OrphanTypeStale
	case d.options.IncludeRecentNoPR:
		return OrphanTypeRecentNoPR
	default:
		return ""
	}
}

type branchPulls struct {
	merged, closed, open *github.PullRequest
}

func pullRequestsByState(branch string, prs []github.PullRequest) branchPulls {
	var pulls branchPulls

	for i := range prs {
		pr := &prs[i]
		if pr.Head.Ref != branch {
			continue
		}

		switch {
		case pr.MergedAt != nil:
			pulls.merged = pr
		case pr.State == prStateClosed:
			pulls.closed = pr
		case pr.State == "open":
			pulls.open = pr
		}
	}

	return pulls
}

// closedPRBranch classifies a branch whose PR ended. A tip that differs from
// the PR's head was pushed after the PR closed, so the PR does not record that
// work.
func closedPRBranch(
	orphan OrphanedBranch,
	orphanType OrphanType,
	pr *github.PullRequest,
	branchSHA string,
) *OrphanedBranch {
	orphan.Type = orphanType
	orphan.PRNumber = &pr.Number
	orphan.PRTitle = &pr.Title

	if pr.Head.SHA != "" && pr.Head.SHA != branchSHA {
		orphan.Type = OrphanTypeUnmergedWork
		orphan.Review = &Review{Note: "pushed after its pull request closed"}
	}

	return &orphan
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
