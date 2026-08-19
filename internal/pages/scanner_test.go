package pages_test

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/KyleKing/gh-sweep/internal/dns"
	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/pages"
)

var errNoSuchHost = errors.New("no such host")

type fakeResolver struct {
	byHost map[string]dns.Resolution
}

func (f fakeResolver) LookupCNAME(_ context.Context, host string) (string, error) {
	if f.byHost[host].PointsAtPages {
		return host + ".", nil
	}

	return "", nil
}

func (f fakeResolver) LookupHost(_ context.Context, host string) ([]string, error) {
	res := f.byHost[host]
	if !res.Resolves {
		return nil, errNoSuchHost
	}
	if res.PointsAtPages {
		return []string{"185.199.108.153"}, nil
	}

	return []string{"93.184.216.34"}, nil
}

func testResolver() fakeResolver {
	return fakeResolver{byHost: map[string]dns.Resolution{
		"healthy.example.com":     {Resolves: true, PointsAtPages: true},
		"dangling.example.com":    {},
		"wrongtarget.example.com": {Resolves: true, PointsAtPages: false},
		"unverified.example.com":  {Resolves: true, PointsAtPages: true},
		"orphaned.example.com":    {Resolves: true, PointsAtPages: true},
		"unclaimed.example.com":   {Resolves: true, PointsAtPages: true},
	}}
}

type repoFixture struct {
	pagesBody string // "" means the Pages API 404s: Pages disabled
	cnameBody string // "" means the repo has no root CNAME file
}

func pagesFixtures() map[string]repoFixture {
	return map[string]repoFixture{
		"healthy":     {pagesBody: `{"cname":"healthy.example.com","protected_domain_state":"verified"}`},
		"dangling":    {pagesBody: `{"cname":"dangling.example.com","protected_domain_state":"verified"}`},
		"wrongtarget": {pagesBody: `{"cname":"wrongtarget.example.com","protected_domain_state":"verified"}`},
		"unverified":  {pagesBody: `{"cname":"unverified.example.com","protected_domain_state":"pending"}`},
		"disabled":    {cnameBody: `{"content":"` + encodeCNAME("orphaned.example.com") + `","encoding":"base64"}`},
		"plain":       {pagesBody: `{}`},
	}
}

func encodeCNAME(domain string) string {
	return base64.StdEncoding.EncodeToString([]byte(domain + "\n"))
}

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

func fixtureResponse(req *http.Request, body string) *http.Response {
	if body == "" {
		return jsonResponse(req, http.StatusNotFound, `{"message":"Not Found"}`)
	}

	return jsonResponse(req, http.StatusOK, body)
}

func reposListJSON(fixtures map[string]repoFixture) string {
	entries := make([]string, 0, len(fixtures))
	for name := range fixtures {
		entries = append(entries, fmt.Sprintf(`{"name":%q,"full_name":"acme/%s","owner":{"login":"acme"}}`, name, name))
	}

	return "[" + strings.Join(entries, ",") + "]"
}

func newPagesTestClient(t *testing.T, fixtures map[string]repoFixture) *github.Client {
	t.Helper()

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		path := req.URL.Path

		if strings.HasSuffix(path, "/orgs/acme/repos") {
			return jsonResponse(req, http.StatusOK, reposListJSON(fixtures)), nil
		}

		for name, fixture := range fixtures {
			if path == fmt.Sprintf("/repos/acme/%s/pages", name) {
				return fixtureResponse(req, fixture.pagesBody), nil
			}
			if path == fmt.Sprintf("/repos/acme/%s/contents/CNAME", name) {
				return fixtureResponse(req, fixture.cnameBody), nil
			}
		}

		return jsonResponse(req, http.StatusNotFound, `{"message":"Not Found"}`), nil
	})

	client, err := github.NewClientWithTransport(context.Background(), transport)
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	return client
}

func findingTypesFor(result *pages.NamespaceAuditResult, repo string) []pages.FindingType {
	var types []pages.FindingType

	for _, r := range result.Repos {
		if r.Repository != repo {
			continue
		}
		for _, f := range r.Findings {
			types = append(types, f.Type)
		}
	}

	return types
}

func TestScanNamespace(t *testing.T) {
	t.Parallel()

	fixtures := pagesFixtures()
	client := newPagesTestClient(t, fixtures)
	scanner := pages.NewScanner(client, testResolver())

	result, err := scanner.ScanNamespace(context.Background(), "acme", []string{"unclaimed.example.com"})
	if err != nil {
		t.Fatalf("ScanNamespace() error = %v", err)
	}

	if result.TotalRepos != len(fixtures) {
		t.Errorf("TotalRepos = %d, want %d", result.TotalRepos, len(fixtures))
	}

	tests := []struct {
		repo string
		want []pages.FindingType
	}{
		{repo: "acme/healthy", want: nil},
		{repo: "acme/plain", want: nil},
		{repo: "acme/dangling", want: []pages.FindingType{pages.FindingDangling}},
		{repo: "acme/wrongtarget", want: []pages.FindingType{pages.FindingDangling}},
		{repo: "acme/unverified", want: []pages.FindingType{pages.FindingUnverified}},
		{repo: "acme/disabled", want: []pages.FindingType{pages.FindingTakeoverRisk}},
	}

	for _, tt := range tests {
		got := findingTypesFor(result, tt.repo)
		if len(got) != len(tt.want) {
			t.Errorf("%s findings = %v, want %v", tt.repo, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("%s findings = %v, want %v", tt.repo, got, tt.want)
				break
			}
		}
	}

	if len(result.ReverseFindings) != 1 || result.ReverseFindings[0].Domain != "unclaimed.example.com" {
		t.Errorf("ReverseFindings = %+v, want a single unclaimed.example.com finding", result.ReverseFindings)
	}
}
