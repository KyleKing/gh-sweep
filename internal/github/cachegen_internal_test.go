package github

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type cacheGenTransport func(*http.Request) (*http.Response, error)

func (f cacheGenTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestDeleteBranch_PrunesOnlyThatReposBranchList(t *testing.T) {
	t.Parallel()

	var urls []string

	rt := cacheGenTransport(func(req *http.Request) (*http.Response, error) {
		urls = append(urls, req.Method+" "+req.URL.RequestURI())

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader("[]")),
			Request:    req,
		}, nil
	})

	client, err := NewClientWithTransport(context.Background(), rt)
	if err != nil {
		t.Fatalf("NewClientWithTransport() error = %v", err)
	}

	client.cacheDir = t.TempDir()

	for _, repo := range []string{"widgets", "gadgets"} {
		if _, err := client.ListBranches("acme", repo); err != nil {
			t.Fatalf("ListBranches(%s) error = %v", repo, err)
		}
	}

	if err := client.DeleteBranch("acme", "widgets", "old"); err != nil {
		t.Fatalf("DeleteBranch() error = %v", err)
	}

	urls = nil

	for _, repo := range []string{"widgets", "gadgets"} {
		if _, err := client.ListBranches("acme", repo); err != nil {
			t.Fatalf("ListBranches(%s) error = %v", repo, err)
		}
	}

	if len(urls) != 2 || !strings.Contains(urls[0], "generation=") || strings.Contains(urls[1], "generation=") {
		t.Errorf("listing URLs after delete = %v, want a generation only on widgets", urls)
	}
}
