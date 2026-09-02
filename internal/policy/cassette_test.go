package policy_test

import (
	"context"
	"io"
	"net/http"
	"testing"

	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"

	"github.com/KyleKing/gh-sweep/internal/config"
	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/policy"
)

// requestOnlyMatcher ignores headers (the cassette's are redacted, and Time-Zone
// varies by machine) and matches purely on method, URL, and body, replaying
// interactions strictly in recorded order.
func requestOnlyMatcher(r *http.Request, i cassette.Request) bool {
	if r.Method != i.Method || r.URL.String() != i.URL {
		return false
	}

	if r.Body == nil {
		return i.Body == ""
	}

	buf := make([]byte, r.ContentLength)
	n, err := io.ReadFull(r.Body, buf)
	if err != nil {
		return false
	}
	r.Body = http.NoBody

	return string(buf[:n]) == i.Body || i.Body == ""
}

func cassetteClient(t *testing.T, name string) *github.Client {
	t.Helper()

	rec, err := recorder.New("testdata/cassettes/"+name,
		recorder.WithMode(recorder.ModeReplayOnly),
		recorder.WithMatcher(requestOnlyMatcher),
		recorder.WithSkipRequestLatency(true),
	)
	if err != nil {
		t.Fatalf("recorder.New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := rec.Stop(); err != nil {
			t.Errorf("recorder.Stop() error = %v", err)
		}
	})

	client, err := github.NewClientWithTransport(context.Background(), rec)
	if err != nil {
		t.Fatalf("NewClientWithTransport() error = %v", err)
	}

	return client
}

// TestPolicyAgainstRecordedSettingsAndSecurity replays a cassette recorded
// against the real GitHub API (see scripts/record-cassette) through the exact
// Evaluate/Apply sequence used to produce it: toggle has_wiki on and back off,
// then secret_scanning_push_protection off and back on, then confirm an
// unprotected branch diffs against a zero-value baseline instead of erroring.
// Byte-real response shapes catch drift the hand-authored fakes in
// client_endpoints_test.go would not.
func TestPolicyAgainstRecordedSettingsAndSecurity(t *testing.T) {
	t.Parallel()

	client := cassetteClient(t, "settings-and-security")
	const repo = "KyleKing/gh-sweep"

	boolPtr := func(b bool) *bool { return &b }

	assertNoDrift := func(t *testing.T, cfg *config.PolicyConfig) {
		t.Helper()

		report := policy.Evaluate(client, cfg)
		if len(report.Repos) != 1 || report.Repos[0].Err != nil {
			t.Fatalf("Evaluate() = %+v", report.Repos)
		}
		if len(report.Repos[0].Diffs) != 0 {
			t.Errorf("diffs = %+v, want none", report.Repos[0].Diffs)
		}
	}

	apply := func(t *testing.T, cfg *config.PolicyConfig, domain policy.Domain, field string) {
		t.Helper()

		drift := policy.RepoDrift{Repository: repo, Diffs: []policy.Diff{{Domain: domain, Field: field}}}
		if result := policy.Apply(client, cfg, drift, policy.ApplyOptions{}); result.Err != nil {
			t.Fatalf("Apply() error = %v", result.Err)
		}
	}

	assertNoDrift(t, &config.PolicyConfig{
		Repositories: []string{repo}, Settings: config.PolicySettings{HasWiki: boolPtr(false)},
	})

	apply(t,
		&config.PolicyConfig{Settings: config.PolicySettings{HasWiki: boolPtr(true)}},
		policy.DomainSettings, "has_wiki",
	)
	assertNoDrift(t, &config.PolicyConfig{
		Repositories: []string{repo}, Settings: config.PolicySettings{HasWiki: boolPtr(true)},
	})

	apply(t,
		&config.PolicyConfig{Settings: config.PolicySettings{HasWiki: boolPtr(false)}},
		policy.DomainSettings, "has_wiki",
	)
	assertNoDrift(t, &config.PolicyConfig{
		Repositories: []string{repo}, Settings: config.PolicySettings{HasWiki: boolPtr(false)},
	})

	apply(t, &config.PolicyConfig{Security: config.PolicySecurity{SecretScanningPushProtection: "disabled"}},
		policy.DomainSecurity, "secret_scanning_push_protection")
	assertNoDrift(t, &config.PolicyConfig{
		Repositories: []string{repo}, Security: config.PolicySecurity{SecretScanningPushProtection: "disabled"},
	})

	apply(t, &config.PolicyConfig{Security: config.PolicySecurity{SecretScanningPushProtection: "enabled"}},
		policy.DomainSecurity, "secret_scanning_push_protection")
	assertNoDrift(t, &config.PolicyConfig{
		Repositories: []string{repo}, Security: config.PolicySecurity{SecretScanningPushProtection: "enabled"},
	})

	report := policy.Evaluate(client, &config.PolicyConfig{
		Repositories: []string{repo},
		Protection:   config.PolicyProtection{EnforceAdmins: boolPtr(true)},
	})
	if len(report.Repos) != 1 || report.Repos[0].Err != nil {
		t.Fatalf("Evaluate() on unprotected branch = %+v", report.Repos)
	}
	if len(report.Repos[0].Diffs) != 1 || report.Repos[0].Diffs[0].Field != "enforce_admins" {
		t.Errorf("diffs = %+v, want one enforce_admins drift against the zero-value baseline", report.Repos[0].Diffs)
	}
}
