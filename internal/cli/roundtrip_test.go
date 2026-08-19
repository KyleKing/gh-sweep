package cli

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

// testPrintTableRoundTrip exercises the populated/empty table-rendering paths
// shared by printTableTo and printPagesTableTo against the given closures.
func testPrintTableRoundTrip(
	t *testing.T,
	printTable func(*strings.Builder),
	wants []string,
	printEmpty func(*strings.Builder),
	emptyWant string,
) {
	t.Helper()

	var b strings.Builder
	printTable(&b)
	table := b.String()

	for _, want := range wants {
		if !strings.Contains(table, want) {
			t.Errorf("table missing %q, got:\n%s", want, table)
		}
	}

	var empty strings.Builder
	printEmpty(&empty)
	if !strings.Contains(empty.String(), emptyWant) {
		t.Errorf("expected the empty-state message, got:\n%s", empty.String())
	}
}

// testOutputRoundTrip exercises the json/markdown/output-file paths shared by
// outputResult and outputPagesResult against the given run closure.
func testOutputRoundTrip(t *testing.T, run func(outputPath, format string), mdHeader, fileBase string) {
	t.Helper()

	jsonOut := captureStdout(t, func() { run("", formatJSON) })
	if !strings.Contains(jsonOut, `"namespace": "acme"`) {
		t.Errorf("json output missing namespace field, got:\n%s", jsonOut)
	}

	mdOut := captureStdout(t, func() { run("", formatMarkdown) })
	if !strings.Contains(mdOut, mdHeader) {
		t.Errorf("markdown output missing header, got:\n%s", mdOut)
	}

	path := filepath.Join(t.TempDir(), fileBase)
	captureStdout(t, func() { run(path, formatJSON) })

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if !strings.Contains(string(data), `"namespace": "acme"`) {
		t.Errorf("output file missing namespace field, got:\n%s", data)
	}
}
