package cli

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/KyleKing/gh-sweep/internal/github"
)

type subscribeCountingTransport struct {
	subscribes int32
}

func (t *subscribeCountingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method == http.MethodPut {
		atomic.AddInt32(&t.subscribes, 1)
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"subscribed":true,"ignored":false}`)),
		Request:    req,
	}, nil
}

func testUnwatchedRepos() []github.RepoWatchInfo {
	return []github.RepoWatchInfo{
		{RepoBasic: github.RepoBasic{Owner: "acme", Name: "widgets", FullName: "acme/widgets"}},
		{RepoBasic: github.RepoBasic{Owner: "acme", Name: "gadgets", FullName: "acme/gadgets"}},
	}
}

func TestRunWatchAllAbortsWithoutTypedYes(t *testing.T) {
	transport := &subscribeCountingTransport{}
	client, err := github.NewClientWithTransport(context.Background(), transport)
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	restoreStdin := withStdin(t, "no\n")
	defer restoreStdin()

	output := captureStdout(t, func() {
		runWatchAll(client, testUnwatchedRepos(), false)
	})

	if !strings.Contains(output, "Aborted") {
		t.Errorf("expected abort message, got:\n%s", output)
	}
	if atomic.LoadInt32(&transport.subscribes) != 0 {
		t.Error("a declined confirmation must not watch any repositories")
	}
}

func TestRunWatchAllProceedsWithYesFlag(t *testing.T) {
	transport := &subscribeCountingTransport{}
	client, err := github.NewClientWithTransport(context.Background(), transport)
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	output := captureStdout(t, func() {
		runWatchAll(client, testUnwatchedRepos(), true)
	})

	if !strings.Contains(output, "Done.") {
		t.Errorf("expected completion message, got:\n%s", output)
	}
	if atomic.LoadInt32(&transport.subscribes) != 2 {
		t.Errorf("expected 2 subscription calls, got %d", transport.subscribes)
	}
}

func TestRunWatchAllTypedYesConfirms(t *testing.T) {
	transport := &subscribeCountingTransport{}
	client, err := github.NewClientWithTransport(context.Background(), transport)
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	restoreStdin := withStdin(t, "yes\n")
	defer restoreStdin()

	output := captureStdout(t, func() {
		runWatchAll(client, testUnwatchedRepos(), false)
	})

	if !strings.Contains(output, "Done.") {
		t.Errorf("expected completion message, got:\n%s", output)
	}
	if atomic.LoadInt32(&transport.subscribes) != 2 {
		t.Errorf("expected 2 subscription calls, got %d", transport.subscribes)
	}
}
