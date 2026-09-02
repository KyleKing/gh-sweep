package policy_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/KyleKing/gh-sweep/internal/config"
	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/policy"
)

// branchFake serves a repo with one branch per orphan type plus a branch whose
// PR is still open, and records every DELETE so a test can assert what an
// apply actually removed.
func branchFake(deleted *[]string) roundTripFunc {
	return func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodDelete {
			*deleted = append(*deleted, req.URL.Path)

			return okBody(req, `{}`), nil
		}

		switch {
		case strings.HasSuffix(req.URL.Path, "/branches"):
			return okBody(req, `[
				{"name":"merged","commit":{"sha":"a","commit":{"author":{"date":"2026-08-01T00:00:00Z"}}}},
				{"name":"closed","commit":{"sha":"b","commit":{"author":{"date":"2026-08-01T00:00:00Z"}}}},
				{"name":"nopr-old","commit":{"sha":"c","commit":{"author":{"date":"2026-01-01T00:00:00Z"}}}},
				{"name":"nopr-new","commit":{"sha":"d","commit":{"author":{"date":"2026-08-30T00:00:00Z"}}}},
				{"name":"open","commit":{"sha":"e","commit":{"author":{"date":"2026-08-01T00:00:00Z"}}}}
			]`), nil
		case strings.Contains(req.URL.Path, "/pulls"):
			return okBody(req, `[
				{"number":1,"title":"m","state":"closed","merged_at":"2026-08-02T00:00:00Z","head":{"ref":"merged"}},
				{"number":2,"title":"c","state":"closed","head":{"ref":"closed"}},
				{"number":3,"title":"o","state":"open","head":{"ref":"open"}}
			]`), nil
		case strings.Contains(req.URL.Path, "/repos/acme/widgets"):
			return okBody(req, `{"name":"widgets","full_name":"acme/widgets","default_branch":"main"}`), nil
		default:
			return okBody(req, `{}`), nil
		}
	}
}

func TestBranchPolicy(t *testing.T) {
	t.Parallel()

	cfg := &config.PolicyConfig{
		Repositories: []string{"acme/widgets"},
		Branches: config.PolicyBranches{
			PruneMerged:   boolPtr(true),
			PruneClosed:   boolPtr(true),
			PruneNoPR:     boolPtr(true),
			NoPRGraceDays: intPtr(21),
		},
	}

	var deleted []string

	client, err := github.NewClientWithTransport(t.Context(), branchFake(&deleted))
	if err != nil {
		t.Fatalf("NewClientWithTransport() error = %v", err)
	}

	report := policy.Evaluate(client, cfg)
	if len(report.Repos) != 1 {
		t.Fatalf("Repos = %d, want 1", len(report.Repos))
	}

	drift := report.Repos[0]
	if drift.Err != nil {
		t.Fatalf("Evaluate() error = %v", drift.Err)
	}

	names := make([]string, 0, len(drift.Prunable))
	for _, branch := range drift.Prunable {
		names = append(names, branch.Name)
	}

	got := strings.Join(names, ",")
	// "open" has a live PR and "nopr-new" is inside the grace period.
	if got != "merged,closed,nopr-old" {
		t.Errorf("prunable = %s, want merged,closed,nopr-old", got)
	}

	// Without --prune, an apply must touch no refs at all.
	if result := policy.Apply(client, cfg, drift, policy.ApplyOptions{}); result.Err != nil {
		t.Fatalf("Apply() without prune error = %v", result.Err)
	}

	if len(deleted) != 0 {
		t.Fatalf("apply without PruneBranches deleted %v, want nothing", deleted)
	}

	if result := policy.Apply(client, cfg, drift, policy.ApplyOptions{PruneBranches: true}); result.Err != nil {
		t.Fatalf("Apply() with prune error = %v", result.Err)
	}

	if len(deleted) != 3 {
		t.Errorf("deleted = %v, want the three prunable refs", deleted)
	}
}
