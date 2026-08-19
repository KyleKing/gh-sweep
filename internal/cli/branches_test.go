package cli

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/KyleKing/gh-sweep/internal/github"
)

func TestFormatCommitAge(t *testing.T) {
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		commit time.Time
		want   string
	}{
		{name: "zero time", commit: time.Time{}, want: "-"},
		{name: "same day", commit: now.Add(-2 * time.Hour), want: "<1d"},
		{name: "days", commit: now.AddDate(0, 0, -3), want: "3d"},
		{name: "just under a year", commit: now.AddDate(0, 0, -364), want: "364d"},
		{name: "years", commit: now.AddDate(-2, 0, -1), want: "2y"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatCommitAge(tt.commit, now); got != tt.want {
				t.Errorf("formatCommitAge() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatBranchPR(t *testing.T) {
	tests := []struct {
		name string
		pr   *github.PullRequest
		want string
	}{
		{name: "no PR", pr: nil, want: "-"},
		{name: "open PR", pr: &github.PullRequest{Number: 12, State: "open"}, want: "#12 (open)"},
		{
			name: "closed PR",
			pr:   &github.PullRequest{Number: 4, State: "closed"},
			want: "#4 (closed)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatBranchPR(tt.pr); got != tt.want {
				t.Errorf("formatBranchPR() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatBool(t *testing.T) {
	t.Parallel()

	if got := formatBool(true); got != "yes" {
		t.Errorf("formatBool(true) = %q, want yes", got)
	}
	if got := formatBool(false); got != "no" {
		t.Errorf("formatBool(false) = %q, want no", got)
	}
}

func TestRenderBranchTable(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	branches := []github.BranchStatus{
		{Branch: github.Branch{Name: "main", LastCommitDate: now.Add(-24 * time.Hour)}, IsDefault: true},
		{
			Branch:    github.Branch{Name: "feature", Protected: true, LastCommitDate: now.Add(-2 * time.Hour)},
			PR:        &github.PullRequest{Number: 5, State: "open"},
			IsDefault: false,
		},
	}

	table := renderBranchTable(branches, now)

	for _, want := range []string{"BRANCH", "main (default)", "feature", "#5 (open)", "yes", "no"} {
		if !strings.Contains(table, want) {
			t.Errorf("table missing %q, got:\n%s", want, table)
		}
	}
}

func TestListBranches(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		path := req.URL.Path

		switch {
		case path == "/repos/acme/widgets":
			return jsonResponse(req, http.StatusOK, `{"default_branch":"main"}`), nil
		case path == "/repos/acme/widgets/branches":
			return jsonResponse(req, http.StatusOK, `[
				{"name":"main","protected":true,
				 "commit":{"sha":"abc","commit":{"author":{"date":"2026-01-10T12:00:00Z"}}}},
				{"name":"feature",
				 "commit":{"sha":"def","commit":{"author":{"date":"2026-01-12T12:00:00Z"}}}}
			]`), nil
		case strings.HasPrefix(path, "/repos/acme/widgets/compare/"):
			return jsonResponse(req, http.StatusOK, `{"ahead_by":1,"behind_by":0}`), nil
		case path == "/repos/acme/widgets/pulls":
			return jsonResponse(req, http.StatusOK, `[]`), nil
		default:
			return jsonResponse(req, http.StatusNotFound, `{"message":"Not Found"}`), nil
		}
	})

	restore := github.SetTestTransport(transport)
	defer restore()

	output := captureStdout(t, func() {
		listBranches("acme", "widgets", "")
	})

	if !strings.Contains(output, "main (default)") || !strings.Contains(output, "feature") {
		t.Errorf("output missing branch rows, got:\n%s", output)
	}
}

func TestListBranchesNoBranches(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		path := req.URL.Path

		switch path {
		case "/repos/acme/empty":
			return jsonResponse(req, http.StatusOK, `{"default_branch":"main"}`), nil
		case "/repos/acme/empty/branches":
			return jsonResponse(req, http.StatusOK, `[]`), nil
		case "/repos/acme/empty/pulls":
			return jsonResponse(req, http.StatusOK, `[]`), nil
		default:
			return jsonResponse(req, http.StatusNotFound, `{"message":"Not Found"}`), nil
		}
	})

	restore := github.SetTestTransport(transport)
	defer restore()

	output := captureStdout(t, func() {
		listBranches("acme", "empty", "")
	})

	if !strings.Contains(output, "No branches found.") {
		t.Errorf("expected the empty-state message, got:\n%s", output)
	}
}
