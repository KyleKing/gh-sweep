// Package pages audits GitHub Pages custom domains against live DNS in both
// directions: repos whose configured domain no longer resolves to Pages (or
// resolves to Pages while Pages itself is disabled, a takeover risk), and
// configured domains with no live Pages site behind them.
package pages

// FindingType classifies a domain audit finding.
type FindingType string

const (
	// FindingDangling means a repo's Pages custom domain no longer resolves
	// to GitHub Pages.
	FindingDangling FindingType = "dangling"
	// FindingTakeoverRisk means DNS still points at GitHub Pages for a
	// domain whose repo has Pages disabled: anyone can claim the subdomain
	// on their own Pages site.
	FindingTakeoverRisk FindingType = "takeover_risk"
	// FindingUnverified means a repo's Pages custom domain is configured
	// but not verified.
	FindingUnverified FindingType = "unverified"
	// FindingNoLiveSite means a domain from the reverse-check list has no
	// live Pages site backing it.
	FindingNoLiveSite FindingType = "no_live_site"
)

// Label returns the finding type's display name.
func (t FindingType) Label() string {
	switch t {
	case FindingDangling:
		return "Dangling"
	case FindingTakeoverRisk:
		return "Takeover risk"
	case FindingUnverified:
		return "Unverified"
	case FindingNoLiveSite:
		return "No live site"
	default:
		return string(t)
	}
}

// Finding is one domain audit result. Repository is empty for reverse-check
// findings, which aren't tied to a specific scanned repo.
type Finding struct {
	Repository string
	Domain     string
	Type       FindingType
	Detail     string
}

// RepoAudit is the Pages configuration and findings for one repository.
type RepoAudit struct {
	Repository string
	CNAME      string
	Enabled    bool
	Findings   []Finding
}

// NamespaceAuditResult is the outcome of auditing every repo in a namespace.
type NamespaceAuditResult struct {
	Namespace       string
	TotalRepos      int
	Repos           []RepoAudit
	ReverseFindings []Finding
}

// AllFindings returns every per-repo finding plus reverse-check findings.
func (r *NamespaceAuditResult) AllFindings() []Finding {
	all := make([]Finding, 0, len(r.ReverseFindings))
	for _, repo := range r.Repos {
		all = append(all, repo.Findings...)
	}

	return append(all, r.ReverseFindings...)
}
