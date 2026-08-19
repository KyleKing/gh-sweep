package cli

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/orphans"
)

type deleteCountingTransport struct {
	deletes int32
}

func (t *deleteCountingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method == http.MethodDelete {
		atomic.AddInt32(&t.deletes, 1)
	}

	return &http.Response{
		StatusCode: http.StatusNoContent,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    req,
	}, nil
}

func testCleanupResult() *orphans.NamespaceScanResult {
	return &orphans.NamespaceScanResult{
		Namespace: "acme",
		Results: []orphans.ScanResult{{
			Repository: github.Repository{FullName: "acme/widgets"},
			Orphans: []orphans.OrphanedBranch{
				{Repository: "acme/widgets", BranchName: "merged-feature", Type: orphans.OrphanTypeMergedPR},
				{Repository: "acme/widgets", BranchName: "abandoned-feature", Type: orphans.OrphanTypeClosedPR},
				{Repository: "acme/widgets", BranchName: "stale-branch", Type: orphans.OrphanTypeStale},
			},
		}},
	}
}

func withStdin(t *testing.T, input string) func() {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdin pipe: %v", err)
	}

	if _, err := w.WriteString(input); err != nil {
		t.Fatalf("failed to write stdin input: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close stdin pipe writer: %v", err)
	}

	original := os.Stdin
	os.Stdin = r

	return func() {
		os.Stdin = original
		if err := r.Close(); err != nil {
			t.Errorf("failed to close stdin pipe reader: %v", err)
		}
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}

	original := os.Stdout
	os.Stdout = w

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close stdout pipe writer: %v", err)
	}
	os.Stdout = original

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read captured stdout: %v", err)
	}

	return string(out)
}

func TestRunCleanupExcludesClosedPRByDefault(t *testing.T) {
	transport := &deleteCountingTransport{}
	client, err := github.NewClientWithTransport(context.Background(), transport)
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	output := captureStdout(t, func() {
		runCleanup(context.Background(), client, testCleanupResult(), true, false, false)
	})

	if strings.Contains(output, "abandoned-feature") {
		t.Errorf("expected closed-PR branch to be excluded from the dry-run list, got:\n%s", output)
	}
	if !strings.Contains(output, "merged-feature") || !strings.Contains(output, "stale-branch") {
		t.Errorf("expected merged and stale branches in the dry-run list, got:\n%s", output)
	}
	if !strings.Contains(output, "Skipping 1 closed-PR branch") {
		t.Errorf("expected closed-PR skip notice, got:\n%s", output)
	}
	if atomic.LoadInt32(&transport.deletes) != 0 {
		t.Error("dry-run must not delete any branches")
	}
}

func TestRunCleanupIncludeClosedPR(t *testing.T) {
	transport := &deleteCountingTransport{}
	client, err := github.NewClientWithTransport(context.Background(), transport)
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	output := captureStdout(t, func() {
		runCleanup(context.Background(), client, testCleanupResult(), true, false, true)
	})

	if !strings.Contains(output, "abandoned-feature") {
		t.Errorf("expected closed-PR branch to be included, got:\n%s", output)
	}
	if !strings.Contains(output, "Total: 3 would be deleted") {
		t.Errorf("expected all three orphans counted, got:\n%s", output)
	}
}

func TestRunCleanupAbortsWithoutTypedYes(t *testing.T) {
	transport := &deleteCountingTransport{}
	client, err := github.NewClientWithTransport(context.Background(), transport)
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	restoreStdin := withStdin(t, "no\n")
	defer restoreStdin()

	output := captureStdout(t, func() {
		runCleanup(context.Background(), client, testCleanupResult(), false, false, false)
	})

	if !strings.Contains(output, "Aborted") {
		t.Errorf("expected abort message, got:\n%s", output)
	}
	if atomic.LoadInt32(&transport.deletes) != 0 {
		t.Error("a declined confirmation must not delete any branches")
	}
}

func TestRunCleanupProceedsWithYesFlag(t *testing.T) {
	transport := &deleteCountingTransport{}
	client, err := github.NewClientWithTransport(context.Background(), transport)
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	output := captureStdout(t, func() {
		runCleanup(context.Background(), client, testCleanupResult(), false, true, false)
	})

	if !strings.Contains(output, "Total: 2 deleted, 0 failed") {
		t.Errorf("expected the two non-closed-PR branches deleted, got:\n%s", output)
	}
	if atomic.LoadInt32(&transport.deletes) != 2 {
		t.Errorf("expected 2 DELETE calls, got %d", transport.deletes)
	}
}

func TestRunCleanupTypedYesConfirms(t *testing.T) {
	transport := &deleteCountingTransport{}
	client, err := github.NewClientWithTransport(context.Background(), transport)
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	restoreStdin := withStdin(t, "yes\n")
	defer restoreStdin()

	output := captureStdout(t, func() {
		runCleanup(context.Background(), client, testCleanupResult(), false, false, false)
	})

	if !strings.Contains(output, "Total: 2 deleted, 0 failed") {
		t.Errorf("expected the two non-closed-PR branches deleted, got:\n%s", output)
	}
	if atomic.LoadInt32(&transport.deletes) != 2 {
		t.Errorf("expected 2 DELETE calls, got %d", transport.deletes)
	}
}

func testScanResult() *orphans.NamespaceScanResult {
	result := &orphans.NamespaceScanResult{
		Namespace:  "acme",
		TotalRepos: 1,
		Results: []orphans.ScanResult{{
			Repository: github.Repository{FullName: "acme/widgets"},
			Orphans: []orphans.OrphanedBranch{
				{Repository: "acme/widgets", BranchName: "merged-feature", Type: orphans.OrphanTypeMergedPR},
				{Repository: "acme/widgets", BranchName: "abandoned-feature", Type: orphans.OrphanTypeClosedPR},
				{Repository: "acme/widgets", BranchName: "stale-branch", Type: orphans.OrphanTypeStale},
			},
		}},
	}
	result.TotalOrphans = len(result.AllOrphans())

	return result
}

func TestFormatMarkdown(t *testing.T) {
	t.Parallel()

	md := formatMarkdown(testScanResult())

	for _, want := range []string{
		"# Orphaned Branches Report: acme",
		"| Merged PR | 1 |",
		"| Closed PR | 1 |",
		"| Stale | 1 |",
		"| acme/widgets | merged-feature | Merged PR | 0 | - |",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q, got:\n%s", want, md)
		}
	}
}

func TestPrintTableTo(t *testing.T) {
	t.Parallel()

	var b strings.Builder
	printTableTo(&b, testScanResult())
	table := b.String()

	for _, want := range []string{
		"Orphaned Branches Report: acme",
		"Total Orphaned Branches: 3",
		"acme/widgets (3 orphans)",
		"merged-feature [Merged PR, 0 days]",
	} {
		if !strings.Contains(table, want) {
			t.Errorf("table missing %q, got:\n%s", want, table)
		}
	}

	var empty strings.Builder
	printTableTo(&empty, &orphans.NamespaceScanResult{Namespace: "acme"})
	if !strings.Contains(empty.String(), "No orphaned branches found.") {
		t.Errorf("expected the empty-state message, got:\n%s", empty.String())
	}
}

func TestPrintTable(t *testing.T) {
	output := captureStdout(t, func() {
		printTable(testScanResult())
	})

	if !strings.Contains(output, "Orphaned Branches Report: acme") {
		t.Errorf("output missing report header, got:\n%s", output)
	}
}

func TestOutputResult(t *testing.T) {
	result := testScanResult()

	jsonOut := captureStdout(t, func() { outputResult(result, "", "json") })
	if !strings.Contains(jsonOut, `"namespace": "acme"`) {
		t.Errorf("json output missing namespace field, got:\n%s", jsonOut)
	}

	mdOut := captureStdout(t, func() { outputResult(result, "", "markdown") })
	if !strings.Contains(mdOut, "# Orphaned Branches Report") {
		t.Errorf("markdown output missing header, got:\n%s", mdOut)
	}

	path := filepath.Join(t.TempDir(), "orphans.json")
	captureStdout(t, func() { outputResult(result, path, "json") })

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if !strings.Contains(string(data), `"namespace": "acme"`) {
		t.Errorf("output file missing namespace field, got:\n%s", data)
	}
}
