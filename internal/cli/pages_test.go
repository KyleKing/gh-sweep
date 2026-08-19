package cli

import (
	"os"
	"path/filepath"
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

	var b strings.Builder
	printPagesTableTo(&b, testPagesResult())
	table := b.String()

	for _, want := range []string{
		"GitHub Pages Domain Audit: acme",
		"[Dangling] acme/widgets docs.example.com: domain no longer resolves to GitHub Pages",
		"[No live site] (reverse check) unclaimed.example.com",
		"Total: 2 finding(s)",
	} {
		if !strings.Contains(table, want) {
			t.Errorf("table missing %q, got:\n%s", want, table)
		}
	}

	var empty strings.Builder
	printPagesTableTo(&empty, &pages.NamespaceAuditResult{Namespace: "acme"})
	if !strings.Contains(empty.String(), "No findings.") {
		t.Errorf("expected the empty-state message, got:\n%s", empty.String())
	}
}

func TestPrintPagesTable(t *testing.T) {
	output := captureStdout(t, func() {
		printPagesTable(testPagesResult())
	})

	if !strings.Contains(output, "GitHub Pages Domain Audit: acme") {
		t.Errorf("output missing report header, got:\n%s", output)
	}
}

func TestOutputPagesResult(t *testing.T) {
	result := testPagesResult()

	jsonOut := captureStdout(t, func() { outputPagesResult(result, "", "json") })
	if !strings.Contains(jsonOut, `"Namespace": "acme"`) {
		t.Errorf("json output missing namespace field, got:\n%s", jsonOut)
	}

	mdOut := captureStdout(t, func() { outputPagesResult(result, "", "markdown") })
	if !strings.Contains(mdOut, "# GitHub Pages Domain Audit") {
		t.Errorf("markdown output missing header, got:\n%s", mdOut)
	}

	path := filepath.Join(t.TempDir(), "pages.json")
	captureStdout(t, func() { outputPagesResult(result, path, "json") })

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if !strings.Contains(string(data), `"Namespace": "acme"`) {
		t.Errorf("output file missing namespace field, got:\n%s", data)
	}
}
