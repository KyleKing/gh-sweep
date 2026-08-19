package pages

import "github.com/KyleKing/gh-sweep/internal/dns"

// classifyRepo turns one repo's Pages configuration and DNS resolution into
// findings. Cname is the domain to audit (the Pages API's cname when Pages
// is enabled, otherwise a lingering CNAME file), already resolved into
// resolution by the caller.
func classifyRepo(repository, cname string, enabled, verified bool, resolution dns.Resolution) []Finding {
	if cname == "" {
		return nil
	}

	var findings []Finding

	switch {
	case !enabled && resolution.PointsAtPages:
		findings = append(findings, Finding{
			Repository: repository,
			Domain:     cname,
			Type:       FindingTakeoverRisk,
			Detail:     "DNS still points at GitHub Pages but Pages is disabled for this repo",
		})
	case enabled && !resolution.Resolves:
		findings = append(findings, Finding{
			Repository: repository,
			Domain:     cname,
			Type:       FindingDangling,
			Detail:     "domain does not resolve",
		})
	case enabled && !resolution.PointsAtPages:
		findings = append(findings, Finding{
			Repository: repository,
			Domain:     cname,
			Type:       FindingDangling,
			Detail:     "domain no longer resolves to GitHub Pages",
		})
	}

	if enabled && !verified {
		findings = append(findings, Finding{
			Repository: repository,
			Domain:     cname,
			Type:       FindingUnverified,
			Detail:     "custom domain is not verified",
		})
	}

	return findings
}
