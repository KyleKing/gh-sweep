package pages

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/KyleKing/gh-sweep/internal/dns"
	"github.com/KyleKing/gh-sweep/internal/github"
)

const defaultConcurrency = 5

// Scanner audits GitHub Pages configuration across a namespace's repos.
type Scanner struct {
	client      *github.Client
	resolver    dns.Resolver
	concurrency int
}

// NewScanner returns a Scanner that resolves domains through resolver.
func NewScanner(client *github.Client, resolver dns.Resolver) *Scanner {
	return &Scanner{client: client, resolver: resolver, concurrency: defaultConcurrency}
}

// ScanNamespace audits every non-archived repo in namespace, then reverse-
// checks reverseDomains against the scanned repos' claimed Pages CNAMEs.
func (s *Scanner) ScanNamespace(
	ctx context.Context,
	namespace string,
	reverseDomains []string,
) (*NamespaceAuditResult, error) {
	repos, _, err := s.client.ListNamespaceRepositories(namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to list namespace repos: %w", err)
	}

	var nonArchived []github.Repository
	for _, repo := range repos {
		if !repo.Archived {
			nonArchived = append(nonArchived, repo)
		}
	}

	result := &NamespaceAuditResult{Namespace: namespace, TotalRepos: len(nonArchived)}
	if len(nonArchived) == 0 {
		return result, nil
	}

	resultsCh := make(chan RepoAudit, len(nonArchived))
	semaphore := make(chan struct{}, s.concurrency)

	var wg sync.WaitGroup
	for _, repo := range nonArchived {
		wg.Add(1)
		go func(repo github.Repository) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			}

			resultsCh <- s.scanRepo(ctx, repo)
		}(repo)
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	for audit := range resultsCh {
		result.Repos = append(result.Repos, audit)
	}

	result.ReverseFindings = s.reverseCheck(ctx, result.Repos, reverseDomains)

	return result, nil
}

func (s *Scanner) scanRepo(ctx context.Context, repo github.Repository) RepoAudit {
	audit := RepoAudit{Repository: repo.FullName}

	info, err := s.client.GetPagesInfo(repo.Owner, repo.Name)
	if err != nil && !errors.Is(err, github.ErrPagesNotFound) {
		return audit
	}

	cnameFile, err := s.client.GetCNAMEFile(repo.Owner, repo.Name)
	if err != nil {
		cnameFile = ""
	}

	cname := cnameFile
	enabled := info != nil
	verified := true

	if enabled {
		verified = info.DomainVerified
		if info.CNAME != "" {
			cname = info.CNAME
		}
	}

	audit.CNAME = cname
	audit.Enabled = enabled

	if cname == "" {
		return audit
	}

	resolution := dns.Resolve(ctx, s.resolver, cname)
	audit.Findings = classifyRepo(repo.FullName, cname, enabled, verified, resolution)

	return audit
}

func (s *Scanner) reverseCheck(ctx context.Context, repos []RepoAudit, domains []string) []Finding {
	claimed := make(map[string]bool, len(repos))
	for _, repo := range repos {
		if repo.Enabled && repo.CNAME != "" {
			claimed[strings.ToLower(repo.CNAME)] = true
		}
	}

	var findings []Finding
	for _, domain := range domains {
		resolution := dns.Resolve(ctx, s.resolver, domain)

		switch {
		case !resolution.Resolves:
			findings = append(findings, Finding{
				Domain: domain,
				Type:   FindingNoLiveSite,
				Detail: "domain does not resolve",
			})
		case !resolution.PointsAtPages:
			findings = append(findings, Finding{
				Domain: domain,
				Type:   FindingNoLiveSite,
				Detail: "domain does not point at GitHub Pages",
			})
		case !claimed[strings.ToLower(domain)]:
			findings = append(findings, Finding{
				Domain: domain,
				Type:   FindingNoLiveSite,
				Detail: "no scanned repo has this domain configured as its Pages custom domain",
			})
		}
	}

	return findings
}
