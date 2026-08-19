package orphans_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/orphans"
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

func scannerFake() roundTripFunc {
	return func(req *http.Request) (*http.Response, error) {
		path := req.URL.Path

		switch {
		case strings.HasSuffix(path, "/orgs/acme/repos"):
			return jsonResponse(req, http.StatusOK, `[
				{"name":"widgets","full_name":"acme/widgets","owner":{"login":"acme"},"default_branch":"main"},
				{"name":"archived","full_name":"acme/archived","owner":{"login":"acme"},"archived":true,"default_branch":"main"}
			]`), nil

		case path == "/repos/acme/widgets/branches":
			return jsonResponse(req, http.StatusOK, `[
				{"name":"main","protected":true,"commit":{"sha":"abc","commit":{"author":{"date":"2026-01-01T00:00:00Z"}}}},
				{"name":"merged-feature","commit":{"sha":"def","commit":{"author":{"date":"2026-01-05T00:00:00Z"}}}},
				{"name":"stale-feature","commit":{"sha":"ghi","commit":{"author":{"date":"2020-01-05T00:00:00Z"}}}}
			]`), nil

		case path == "/repos/acme/widgets/pulls":
			return jsonResponse(req, http.StatusOK, `[
				{"number":1,"title":"merged","state":"closed","merged_at":"2026-01-06T00:00:00Z",
				 "head":{"ref":"merged-feature","sha":"def","repo":{"full_name":"acme/widgets"}},
				 "base":{"ref":"main","sha":"abc","repo":{"full_name":"acme/widgets"}}}
			]`), nil

		default:
			return jsonResponse(req, http.StatusNotFound, `{"message":"Not Found"}`), nil
		}
	}
}

func TestScanNamespace(t *testing.T) {
	t.Parallel()

	client, err := github.NewClientWithTransport(context.Background(), scannerFake())
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	scanner := orphans.NewNamespaceScanner(client, orphans.DefaultScanOptions())

	result, err := scanner.ScanNamespace(context.Background(), "acme")
	if err != nil {
		t.Fatalf("ScanNamespace() error = %v", err)
	}

	if result.TotalRepos != 1 {
		t.Errorf("TotalRepos = %d, want 1 (archived repo must be skipped)", result.TotalRepos)
	}

	orphansByType := map[orphans.OrphanType]int{}
	for _, o := range result.AllOrphans() {
		orphansByType[o.Type]++
	}

	if orphansByType[orphans.OrphanTypeMergedPR] != 1 {
		t.Errorf("expected 1 merged-PR orphan, got %d", orphansByType[orphans.OrphanTypeMergedPR])
	}
	if orphansByType[orphans.OrphanTypeStale] != 1 {
		t.Errorf("expected 1 stale orphan, got %d", orphansByType[orphans.OrphanTypeStale])
	}

	if len(result.OrphansByType(orphans.OrphanTypeMergedPR)) != 1 {
		t.Errorf("OrphansByType(MergedPR) mismatch")
	}
}

func TestScanNamespaceWithProgress(t *testing.T) {
	t.Parallel()

	client, err := github.NewClientWithTransport(context.Background(), scannerFake())
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	scanner := orphans.NewNamespaceScanner(client, orphans.DefaultScanOptions())

	progressCh := make(chan orphans.ScanProgress, 8)
	result, err := scanner.ScanNamespaceWithProgress(context.Background(), "acme", progressCh)
	if err != nil {
		t.Fatalf("ScanNamespaceWithProgress() error = %v", err)
	}

	if result.TotalOrphans != 2 {
		t.Errorf("TotalOrphans = %d, want 2", result.TotalOrphans)
	}
}

func TestScanNamespaceEmpty(t *testing.T) {
	t.Parallel()

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if strings.HasSuffix(req.URL.Path, "/orgs/empty-org/repos") {
			return jsonResponse(req, http.StatusOK, `[]`), nil
		}

		return jsonResponse(req, http.StatusNotFound, `{"message":"Not Found"}`), nil
	})

	client, err := github.NewClientWithTransport(context.Background(), transport)
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	scanner := orphans.NewNamespaceScanner(client, orphans.DefaultScanOptions())

	result, err := scanner.ScanNamespace(context.Background(), "empty-org")
	if err != nil {
		t.Fatalf("ScanNamespace() error = %v", err)
	}

	if result.TotalRepos != 0 || len(result.AllOrphans()) != 0 {
		t.Errorf("expected an empty result, got %+v", result)
	}
}
