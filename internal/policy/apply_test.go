package policy_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/KyleKing/gh-sweep/internal/config"
	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/policy"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func okBody(req *http.Request, body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

const protectionFixture = `{"enforce_admins":{"enabled":false},` +
	`"allow_force_pushes":{"enabled":false},` +
	`"allow_deletions":{"enabled":false},` +
	`"required_linear_history":{"enabled":false}}`

// routedTransport dispatches on "METHOD /path" so each test only wires up the
// calls it cares about, and falls back to an empty 200 for everything else.
func routedTransport(routes map[string]func(*http.Request)) roundTripFunc {
	return func(req *http.Request) (*http.Response, error) {
		if hook, ok := routes[req.Method+" "+req.URL.Path]; ok {
			hook(req)
		}

		if req.Method == http.MethodGet && req.URL.Path == "/repos/acme/widgets/branches/main/protection" {
			return okBody(req, protectionFixture), nil
		}
		if req.Method == http.MethodGet && req.URL.Path == "/repos/acme/widgets" {
			return okBody(req, `{"default_branch":"main"}`), nil
		}

		return okBody(req, `{}`), nil
	}
}

func TestApplyPushesOnlyDriftedDomains(t *testing.T) {
	t.Parallel()

	calls := map[string]bool{}
	mark := func(name string) func(*http.Request) { return func(*http.Request) { calls[name] = true } }

	client, err := github.NewClientWithTransport(context.Background(), routedTransport(map[string]func(*http.Request){
		http.MethodPatch + " /repos/acme/widgets":                        mark("settings"),
		http.MethodPut + " /repos/acme/widgets/immutable-releases":       mark("releases"),
		http.MethodPut + " /repos/acme/widgets/branches/main/protection": mark("protection"),
	}))
	if err != nil {
		t.Fatalf("NewClientWithTransport() error = %v", err)
	}

	cfg := &config.PolicyConfig{
		Settings:   config.PolicySettings{HasWiki: boolPtr(false)},
		Releases:   config.PolicyReleases{Immutable: boolPtr(true)},
		Protection: config.PolicyProtection{EnforceAdmins: boolPtr(true)},
	}

	drift := policy.RepoDrift{
		Repository: "acme/widgets",
		Diffs: []policy.Diff{
			{Domain: policy.DomainSettings, Field: "has_wiki"},
			{Domain: policy.DomainReleases, Field: "immutable"},
			{Domain: policy.DomainProtection, Field: "enforce_admins"},
		},
	}

	result := policy.Apply(client, cfg, drift)
	if result.Err != nil {
		t.Fatalf("Apply() error = %v", result.Err)
	}

	for _, want := range []string{"settings", "releases", "protection"} {
		if !calls[want] {
			t.Errorf("expected a call for domain %q", want)
		}
	}

	if len(result.Applied) != 3 {
		t.Errorf("Applied = %v, want 3 domains", result.Applied)
	}
}

func TestApplySkipsUndriftedDomains(t *testing.T) {
	t.Parallel()

	securityApplied := false

	client, err := github.NewClientWithTransport(context.Background(), routedTransport(map[string]func(*http.Request){
		http.MethodPatch + " /repos/acme/widgets": func(*http.Request) { securityApplied = true },
	}))
	if err != nil {
		t.Fatalf("NewClientWithTransport() error = %v", err)
	}

	cfg := &config.PolicyConfig{Security: config.PolicySecurity{SecretScanning: "enabled"}}
	result := policy.Apply(client, cfg, policy.RepoDrift{Repository: "acme/widgets"})

	if result.Err != nil {
		t.Fatalf("Apply() error = %v", result.Err)
	}
	if securityApplied {
		t.Error("security PATCH sent despite no drift for that domain")
	}
	if len(result.Applied) != 0 {
		t.Errorf("Applied = %v, want none", result.Applied)
	}
}

func TestApplyInvalidRepo(t *testing.T) {
	t.Parallel()

	client, err := github.NewClientWithTransport(
		context.Background(),
		routedTransport(nil),
	)
	if err != nil {
		t.Fatalf("NewClientWithTransport() error = %v", err)
	}

	result := policy.Apply(client, &config.PolicyConfig{}, policy.RepoDrift{Repository: "not-a-repo"})
	if result.Err == nil {
		t.Error("Apply() error = nil, want error for malformed repo name")
	}
}
