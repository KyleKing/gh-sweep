// Package orphans classifies stale git branches by whether their pull
// request was merged, closed, or never opened, so gh-sweep can flag them for
// cleanup.
package orphans

import (
	"time"

	"github.com/KyleKing/gh-sweep/internal/github"
)

// prStateClosed is the GitHub pull request state value for a closed
// (unmerged) PR.
const prStateClosed = "closed"

// closedPRLabel is OrphanTypeClosedPR's human-readable Label, also used in
// tests asserting that label.
const closedPRLabel = "Closed PR"

// defaultExcludedBranch is the branch name every ScanOptions excludes by
// default, since it is the repository's default branch in most projects.
const defaultExcludedBranch = "main"

// OrphanType categorizes why a branch has no active pull request.
type OrphanType string

// The set of reasons a branch can be classified as orphaned.
const (
	OrphanTypeMergedPR   OrphanType = "merged_pr"
	OrphanTypeClosedPR   OrphanType = "closed_pr"
	OrphanTypeStale      OrphanType = "stale"
	OrphanTypeRecentNoPR OrphanType = "recent_no_pr"
)

// Label returns the human-readable name for t.
func (t OrphanType) Label() string {
	switch t {
	case OrphanTypeMergedPR:
		return "Merged PR"
	case OrphanTypeClosedPR:
		return closedPRLabel
	case OrphanTypeStale:
		return "Stale"
	case OrphanTypeRecentNoPR:
		return "Recent (no PR)"
	default:
		return string(t)
	}
}

// OrphanedBranch is a branch classified as safe to clean up, along with the
// pull request context that justified the classification.
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

// Key uniquely identifies the branch within a scan by repository and name.
func (o OrphanedBranch) Key() string {
	return o.Repository + "/" + o.BranchName
}

// ScanResult holds the orphaned branches found in a single repository, or
// the error encountered scanning it.
type ScanResult struct {
	Repository    github.Repository `json:"repository"`
	Orphans       []OrphanedBranch  `json:"orphans"`
	DefaultBranch string            `json:"default_branch"`
	Error         error             `json:"error,omitempty"`
}

// NamespaceScanResult aggregates the ScanResult for every repository in a
// user or organization namespace.
type NamespaceScanResult struct {
	Namespace    string       `json:"namespace"`
	IsOrg        bool         `json:"is_org"`
	Results      []ScanResult `json:"results"`
	TotalRepos   int          `json:"total_repos"`
	TotalOrphans int          `json:"total_orphans"`
}

// AllOrphans flattens every repository's orphaned branches into one slice.
func (r *NamespaceScanResult) AllOrphans() []OrphanedBranch {
	var all []OrphanedBranch
	for i := range r.Results {
		all = append(all, r.Results[i].Orphans...)
	}

	return all
}

// OrphansByType returns AllOrphans filtered to the given OrphanType.
func (r *NamespaceScanResult) OrphansByType(t OrphanType) []OrphanedBranch {
	var filtered []OrphanedBranch
	for _, orphan := range r.AllOrphans() {
		if orphan.Type == t {
			filtered = append(filtered, orphan)
		}
	}

	return filtered
}

// ScanOptions configures how a Detector or NamespaceScanner classifies and
// scans branches.
type ScanOptions struct {
	// StaleDaysThreshold is the grace period for a branch with no PR at all.
	// Branches from merged or closed PRs carry no grace period: the PR is the
	// record of the work, and GitHub can restore the branch from it.
	StaleDaysThreshold int
	IncludeRecentNoPR  bool
	// OnlyRepos restricts a namespace scan to these repositories, named either
	// "owner/repo" or bare. Empty scans the whole namespace.
	OnlyRepos        []string
	ExcludePatterns  []string
	IncludeProtected bool
	Concurrency      int
}

const (
	defaultStaleDaysThreshold = 21
	defaultConcurrency        = 5
)

// DefaultScanOptions returns the ScanOptions gh-sweep uses when the caller
// hasn't customized any scan settings.
func DefaultScanOptions() ScanOptions {
	return ScanOptions{
		StaleDaysThreshold: defaultStaleDaysThreshold,
		IncludeRecentNoPR:  false,
		ExcludePatterns: []string{
			defaultExcludedBranch,
			"master",
			"develop",
			"release/*",
			"hotfix/*",
		},
		IncludeProtected: false,
		Concurrency:      defaultConcurrency,
	}
}
