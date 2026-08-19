package cli

import (
	"bufio"
	"errors"
	"strings"
	"testing"

	"github.com/KyleKing/gh-sweep/internal/policy"
)

func testPolicyReport() *policy.Report {
	return &policy.Report{Repos: []policy.RepoDrift{
		{
			Repository: "acme/widgets",
			Diffs: []policy.Diff{{
				Domain: policy.DomainSettings, Field: "has_issues", Current: "false", Desired: "true",
			}},
		},
		{Repository: "acme/broken", Err: errors.New("fetch failed")},
	}}
}

func TestFormatPolicyTable(t *testing.T) {
	t.Parallel()

	table := formatPolicyTable(testPolicyReport())

	for _, want := range []string{
		"acme/widgets (1 drifted field(s))",
		"[settings] has_issues: current=false desired=true",
		"acme/broken: ERROR: fetch failed",
	} {
		if !strings.Contains(table, want) {
			t.Errorf("table missing %q, got:\n%s", want, table)
		}
	}

	clean := formatPolicyTable(&policy.Report{Repos: []policy.RepoDrift{{Repository: "acme/clean"}}})
	if !strings.Contains(clean, "No drift found.") {
		t.Errorf("expected the no-drift message, got:\n%s", clean)
	}
}

func TestFormatPolicyMarkdown(t *testing.T) {
	t.Parallel()

	md := formatPolicyMarkdown(testPolicyReport())

	for _, want := range []string{
		"# Policy Drift Report",
		"| acme/widgets | settings | has_issues | false | true |",
		"| acme/broken | - | error | fetch failed | - |",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q, got:\n%s", want, md)
		}
	}
}

func TestPrintReport(t *testing.T) {
	report := testPolicyReport()

	jsonOut := captureStdout(t, func() { printReport(report, "json") })
	if !strings.Contains(jsonOut, `"Repository": "acme/widgets"`) {
		t.Errorf("json output missing repository field, got:\n%s", jsonOut)
	}

	mdOut := captureStdout(t, func() { printReport(report, "markdown") })
	if !strings.Contains(mdOut, "# Policy Drift Report") {
		t.Errorf("markdown output missing header, got:\n%s", mdOut)
	}

	tableOut := captureStdout(t, func() { printReport(report, "table") })
	if !strings.Contains(tableOut, "acme/widgets (1 drifted field(s))") {
		t.Errorf("table output missing drift line, got:\n%s", tableOut)
	}
}

func TestConfirm(t *testing.T) {
	t.Parallel()

	if !confirm(bufio.NewReader(strings.NewReader("y\n")), "Apply?") {
		t.Error("confirm(y) = false, want true")
	}
	if !confirm(bufio.NewReader(strings.NewReader("Y\n")), "Apply?") {
		t.Error("confirm(Y) = false, want true")
	}
	if confirm(bufio.NewReader(strings.NewReader("n\n")), "Apply?") {
		t.Error("confirm(n) = true, want false")
	}
	if confirm(bufio.NewReader(strings.NewReader("")), "Apply?") {
		t.Error("confirm(EOF) = true, want false")
	}
}
