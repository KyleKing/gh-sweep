// Package dns resolves domains to check GitHub Pages custom-domain
// configuration against what's actually live.
package dns

import (
	"context"
	"fmt"
	"net"
	"strings"
)

// Resolver looks up DNS records for a hostname. Implemented by the system
// resolver in production and faked in tests.
type Resolver interface {
	LookupCNAME(ctx context.Context, host string) (string, error)
	LookupHost(ctx context.Context, host string) ([]string, error)
}

type netResolver struct {
	resolver *net.Resolver
}

// NewResolver returns a Resolver backed by the system's default DNS resolver.
func NewResolver() Resolver { //nolint:ireturn // constructor for the Resolver seam; no concrete type is exported
	return netResolver{resolver: net.DefaultResolver}
}

func (n netResolver) LookupCNAME(ctx context.Context, host string) (string, error) {
	cname, err := n.resolver.LookupCNAME(ctx, host)
	if err != nil {
		return "", fmt.Errorf("looking up CNAME for %s: %w", host, err)
	}

	return cname, nil
}

func (n netResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	addrs, err := n.resolver.LookupHost(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("looking up host %s: %w", host, err)
	}

	return addrs, nil
}

// githubPagesIPs are the apex A records GitHub Pages documents for custom
// domains without a CNAME (https://docs.github.com/pages/configuring-a-custom-domain-for-your-github-pages-site).
var githubPagesIPs = map[string]bool{
	"185.199.108.153": true,
	"185.199.109.153": true,
	"185.199.110.153": true,
	"185.199.111.153": true,
}

// Resolution is what a domain currently resolves to.
type Resolution struct {
	// Resolves is true when the domain has any CNAME or A/AAAA record.
	Resolves bool
	// PointsAtPages is true when the domain's CNAME targets *.github.io or
	// an A record matches a documented GitHub Pages IP.
	PointsAtPages bool
}

// Resolve looks up host and reports whether it currently points at GitHub
// Pages infrastructure. A host with neither a CNAME nor an A/AAAA record
// (NXDOMAIN) reports Resolves: false.
func Resolve(ctx context.Context, r Resolver, host string) Resolution {
	cname, cnameErr := r.LookupCNAME(ctx, host)
	addrs, hostErr := r.LookupHost(ctx, host)

	if cnameErr != nil && hostErr != nil {
		return Resolution{}
	}

	if cnameErr == nil && strings.HasSuffix(strings.TrimSuffix(cname, "."), ".github.io") {
		return Resolution{Resolves: true, PointsAtPages: true}
	}

	for _, addr := range addrs {
		if githubPagesIPs[addr] {
			return Resolution{Resolves: true, PointsAtPages: true}
		}
	}

	return Resolution{Resolves: true}
}
