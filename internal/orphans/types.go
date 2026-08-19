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
	Repository        string     `json:"repository"`
	BranchName        string     `json:"branch_name"`
	SHA               string     `json:"sha"`
	LastCommitDate    time.Time  `json:"last_commit_date"`
	Type              OrphanType `json:"type"`
	PRNumber          *int       `json:"pr_number,omitempty"`
	PRTitle           *string    `json:"pr_title,omitempty"`
	DaysSinceActivity int        `json:"days_since_activity"`
	Protected         bool       `json:"protected"`
}

func (o OrphanedBranch) Key() string {
	return o.Repository + "/" + o.BranchName
}

type ScanResult struct {
	Repository    github.Repository `json:"repository"`
	Orphans       []OrphanedBranch  `json:"orphans"`
	DefaultBranch string            `json:"default_branch"`
	Error         error             `json:"error,omitempty"`
}

type NamespaceScanResult struct {
	Namespace    string       `json:"namespace"`
	IsOrg        bool         `json:"is_org"`
	Results      []ScanResult `json:"results"`
	TotalRepos   int          `json:"total_repos"`
	TotalOrphans int          `json:"total_orphans"`
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
		StaleDaysThreshold: 30,
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
