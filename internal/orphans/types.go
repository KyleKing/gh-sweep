package orphans

import (
	"time"

	"github.com/KyleKing/gh-sweep/internal/github"
)

type OrphanType string

const (
	OrphanTypeMergedPR   OrphanType = "merged_pr"
	OrphanTypeClosedPR   OrphanType = "closed_pr"
	OrphanTypeStale      OrphanType = "stale"
	OrphanTypeRecentNoPR OrphanType = "recent_no_pr"
)

func (t OrphanType) Label() string {
	switch t {
	case OrphanTypeMergedPR:
		return "Merged PR"
	case OrphanTypeClosedPR:
		return "Closed PR"
	case OrphanTypeStale:
		return "Stale"
	case OrphanTypeRecentNoPR:
		return "Recent (no PR)"
	default:
		return string(t)
	}
}

type OrphanedBranch struct {
	Repository        string
	BranchName        string
	SHA               string
	LastCommitDate    time.Time
	Type              OrphanType
	PRNumber          *int
	PRTitle           *string
	DaysSinceActivity int
	Protected         bool
}

func (o OrphanedBranch) Key() string {
	return o.Repository + "/" + o.BranchName
}

type ScanResult struct {
	Repository    github.Repository
	Orphans       []OrphanedBranch
	DefaultBranch string
	Error         error
}

type NamespaceScanResult struct {
	Namespace   string
	IsOrg       bool
	Results     []ScanResult
	TotalRepos  int
	TotalOrphans int
}

func (r *NamespaceScanResult) AllOrphans() []OrphanedBranch {
	var all []OrphanedBranch
	for _, result := range r.Results {
		all = append(all, result.Orphans...)
	}
	return all
}

func (r *NamespaceScanResult) OrphansByType(t OrphanType) []OrphanedBranch {
	var filtered []OrphanedBranch
	for _, orphan := range r.AllOrphans() {
		if orphan.Type == t {
			filtered = append(filtered, orphan)
		}
	}
	return filtered
}

type ScanOptions struct {
	StaleDaysThreshold int
	IncludeRecentNoPR  bool
	ExcludePatterns    []string
	IncludeProtected   bool
	Concurrency        int
}

func DefaultScanOptions() ScanOptions {
	return ScanOptions{
		StaleDaysThreshold: 7,
		IncludeRecentNoPR:  false,
		ExcludePatterns: []string{
			"main",
			"master",
			"develop",
			"release/*",
			"hotfix/*",
		},
		IncludeProtected: false,
		Concurrency:      5,
	}
}
