package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/KyleKing/gh-sweep/internal/dns"
	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/pages"
)

var pagesCmd = &cobra.Command{
	Use:   "pages",
	Short: "Audit GitHub Pages custom domains against live DNS",
	Long: `Cross-check GitHub Pages custom domains against DNS in both directions:
flag repos whose domain no longer resolves to GitHub Pages, repos where DNS
still points at GitHub Pages while Pages itself is disabled (a subdomain-
takeover risk, since anyone can claim the subdomain on their own Pages site),
and repos with an unverified custom domain.

Set the pages.domains config key to reverse-check DNS-configured subdomains
that should have a live Pages site backing them.

A domain proxied through Cloudflare or another CDN resolves to the proxy's
IPs, not GitHub's, so this audit cannot see past the proxy and will report a
false "dangling" finding for it. Verify those by hand before acting on them.

Examples:
  gh-sweep pages --org my-org
  gh-sweep pages --namespace my-user --format json`,
	Run: runPages,
}

func init() {
	rootCmd.AddCommand(pagesCmd)

	pagesCmd.Flags().String("org", "", "Organization to scan")
	pagesCmd.Flags().String("namespace", "", "Namespace (org or user) to scan")
	pagesCmd.Flags().StringP("output", "o", "", "Output file path")
	pagesCmd.Flags().String("format", "table", "Output format: table, json, markdown")
}

func runPages(cmd *cobra.Command, _ []string) {
	ctx := context.Background()

	client, err := github.NewClient(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create GitHub client: %v\n", err)
		os.Exit(1)
	}

	outputPath := stringFlag(cmd, "output")
	format := stringFlag(cmd, "format")

	cfg := loadConfig()

	namespace, err := resolveNamespace(cmd, client, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Scanning namespace: %s\n", namespace)

	scanner := pages.NewScanner(client, dns.NewResolver())
	result, err := scanner.ScanNamespace(ctx, namespace, cfg.Pages.Domains)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to scan namespace: %v\n", err)
		os.Exit(1)
	}

	if outputPath != "" || format == formatJSON || format == formatMarkdown {
		outputPagesResult(result, outputPath, format)
		return
	}

	printPagesTable(result)
}

func outputPagesResult(result *pages.NamespaceAuditResult, outputPath, format string) {
	writeOutput(outputPath, format,
		func() ([]byte, error) { return json.MarshalIndent(result, "", "  ") },
		func() string { return formatPagesMarkdown(result) },
		func(b *strings.Builder) { printPagesTableTo(b, result) },
	)
}

func formatPagesMarkdown(result *pages.NamespaceAuditResult) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# GitHub Pages Domain Audit: %s\n\n", result.Namespace)
	fmt.Fprintf(&b, "**Total Repositories:** %d\n\n", result.TotalRepos)

	findings := result.AllFindings()
	if len(findings) == 0 {
		b.WriteString("No findings.\n")
		return b.String()
	}

	b.WriteString("## Findings\n\n")
	b.WriteString("| Repository | Domain | Type | Detail |\n")
	b.WriteString("|------------|--------|------|--------|\n")

	for _, finding := range findings {
		repo := finding.Repository
		if repo == "" {
			repo = "-"
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", repo, finding.Domain, finding.Type.Label(), finding.Detail)
	}

	return b.String()
}

func printPagesTable(result *pages.NamespaceAuditResult) {
	var b strings.Builder
	printPagesTableTo(&b, result)
	fmt.Print(b.String())
}

func printPagesTableTo(b *strings.Builder, result *pages.NamespaceAuditResult) {
	fmt.Fprintf(b, "GitHub Pages Domain Audit: %s\n\n", result.Namespace)
	fmt.Fprintf(b, "Total Repositories: %d\n\n", result.TotalRepos)

	findings := result.AllFindings()
	if len(findings) == 0 {
		b.WriteString("No findings.\n")
		return
	}

	for _, finding := range findings {
		repo := finding.Repository
		if repo == "" {
			repo = "(reverse check)"
		}
		fmt.Fprintf(b, "  [%s] %s %s: %s\n", finding.Type.Label(), repo, finding.Domain, finding.Detail)
	}

	fmt.Fprintf(b, "\nTotal: %d finding(s)\n", len(findings))
}
