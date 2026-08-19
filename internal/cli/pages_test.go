package cli

import (
	"strings"
	"testing"

	"github.com/KyleKing/gh-sweep/internal/pages"
)

func testPagesResult() *pages.NamespaceAuditResult {
	return &pages.NamespaceAuditResult{
		Namespace:  "acme",
		TotalRepos: 2,
		Repos: []pages.RepoAudit{{
			Repository: "acme/widgets",
			CNAME:      "docs.example.com",
			Findings: []pages.Finding{{
				Repository: "acme/widgets",
				Domain:     "docs.example.com",
				Type:       pages.FindingDangling,
				Detail:     "domain no longer resolves to GitHub Pages",
			}},
		}},
		ReverseFindings: []pages.Finding{{
			Domain: "unclaimed.example.com",
			Type:   pages.FindingNoLiveSite,
			Detail: "no scanned repo has this domain configured as its Pages custom domain",
		}},
	}
}

func TestFormatPagesMarkdown(t *testing.T) {
	t.Parallel()

	md := formatPagesMarkdown(testPagesResult())

	for _, want := range []string{
		"# GitHub Pages Domain Audit: acme",
		"| acme/widgets | docs.example.com | Dangling |",
		"| - | unclaimed.example.com | No live site |",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q, got:\n%s", want, md)
		}
	}

	empty := formatPagesMarkdown(&pages.NamespaceAuditResult{Namespace: "acme"})
	if !strings.Contains(empty, "No findings.") {
		t.Errorf("expected the empty-state message, got:\n%s", empty)
	}
}

func TestPrintPagesTableTo(t *testing.T) {
	t.Parallel()

	testPrintTableRoundTrip(t,
		func(b *strings.Builder) { printPagesTableTo(b, testPagesResult()) },
		[]string{
			"GitHub Pages Domain Audit: acme",
			"[Dangling] acme/widgets docs.example.com: domain no longer resolves to GitHub Pages",
			"[No live site] (reverse check) unclaimed.example.com",
			"Total: 2 finding(s)",
		},
		func(b *strings.Builder) { printPagesTableTo(b, &pages.NamespaceAuditResult{Namespace: "acme"}) },
		"No findings.",
	)
}

//nolint:paralleltest // captureStdout swaps the process-wide os.Stdout.
func TestPrintPagesTable(t *testing.T) {
	output := captureStdout(t, func() {
		printPagesTable(testPagesResult())
	})

	if !strings.Contains(output, "GitHub Pages Domain Audit: acme") {
		t.Errorf("output missing report header, got:\n%s", output)
	}
}

//nolint:paralleltest // testOutputRoundTrip uses captureStdout, which swaps the process-wide os.Stdout.
func TestOutputPagesResult(t *testing.T) {
	result := testPagesResult()

	testOutputRoundTrip(t, func(outputPath, format string) {
		outputPagesResult(result, outputPath, format)
	}, "# GitHub Pages Domain Audit", "pages.json")
}
