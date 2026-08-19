package github

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func recordingTransport(body string, paths *[]string) roundTripFunc {
	return func(req *http.Request) (*http.Response, error) {
		if paths != nil {
			*paths = append(*paths, req.Method+" "+req.URL.Path)
		}

		return jsonResponse(http.StatusOK, body), nil
	}
}

func TestNewClientWithTransportServesFakes(t *testing.T) {
	t.Parallel()

	var paths []string
	client, err := NewClientWithTransport(
		context.Background(),
		recordingTransport(`{"login":"tester"}`, &paths),
	)
	if err != nil {
		t.Fatalf("NewClientWithTransport() error = %v", err)
	}

	login, err := client.GetAuthenticatedUser()
	if err != nil {
		t.Fatalf("GetAuthenticatedUser() error = %v", err)
	}

	if login != "tester" {
		t.Errorf("login = %q, want %q", login, "tester")
	}

	if len(paths) != 1 || paths[0] != "GET /user" {
		t.Errorf("requests = %v, want [GET /user]", paths)
	}
}

//nolint:paralleltest // mutates the shared global test transport
func TestSetTestTransportRoutesNewClient(t *testing.T) {
	var paths []string
	restore := SetTestTransport(recordingTransport(`{"default_branch":"main"}`, &paths))
	defer restore()

	client, err := NewClient(context.Background())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	branch, err := client.GetDefaultBranch("acme", "widgets")
	if err != nil {
		t.Fatalf("GetDefaultBranch() error = %v", err)
	}

	if branch != "main" {
		t.Errorf("branch = %q, want %q", branch, "main")
	}

	if len(paths) != 1 || paths[0] != "GET /repos/acme/widgets" {
		t.Errorf("requests = %v, want [GET /repos/acme/widgets]", paths)
	}
}

func TestSetTestTransportRestore(t *testing.T) { //nolint:paralleltest // mutates the shared global test transport
	rt := recordingTransport(`{}`, nil)

	restore := SetTestTransport(rt)
	if currentTestTransport() == nil {
		t.Fatal("expected transport to be registered")
	}

	restore()

	if currentTestTransport() != nil {
		t.Error("expected transport to be restored to nil")
	}
}

func TestSafetyTransportPanicsOnMutation(t *testing.T) {
	t.Parallel()

	mutating := []string{http.MethodDelete, http.MethodPatch, http.MethodPost, http.MethodPut}
	for _, method := range mutating {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			guard := safetyTransport{base: recordingTransport(`{}`, nil)}
			req, err := http.NewRequestWithContext(
				context.Background(), method, "https://api.github.com/repos/acme/widgets/git/refs/heads/x", http.NoBody,
			)
			if err != nil {
				t.Fatalf("NewRequestWithContext() error = %v", err)
			}

			defer func() {
				if recover() == nil {
					t.Errorf("expected panic for %s request", method)
				}
			}()

			//nolint:bodyclose // the guard panics before a response exists
			resp, err := guard.RoundTrip(req)
			t.Fatalf("RoundTrip returned without panic: resp=%v err=%v", resp, err)
		})
	}
}

func TestSafetyTransportAllowsReads(t *testing.T) {
	t.Parallel()

	guard := safetyTransport{base: recordingTransport(`{"ok":true}`, nil)}
	req, err := http.NewRequestWithContext(
		context.Background(), http.MethodGet, "https://api.github.com/user", http.NoBody,
	)
	if err != nil {
		t.Fatalf("NewRequestWithContext() error = %v", err)
	}

	resp, err := guard.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Errorf("Close() error = %v", closeErr)
		}
	})

	var payload map[string]bool
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !payload["ok"] {
		t.Error("expected fake body to pass through the guard")
	}
}

func TestClientMutationsGoThroughFakeTransport(t *testing.T) {
	t.Parallel()

	var paths []string
	client, err := NewClientWithTransport(context.Background(), recordingTransport(`{}`, &paths))
	if err != nil {
		t.Fatalf("NewClientWithTransport() error = %v", err)
	}

	if err := client.DeleteBranch("acme", "widgets", "feature-x"); err != nil {
		t.Fatalf("DeleteBranch() error = %v", err)
	}

	want := "DELETE /repos/acme/widgets/git/refs/heads/feature-x"
	if len(paths) != 1 || paths[0] != want {
		t.Errorf("requests = %v, want [%s]", paths, want)
	}
}
