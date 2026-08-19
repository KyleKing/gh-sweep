package cli

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/KyleKing/gh-sweep/internal/github"
)

func TestFormatProtectionTableDrift(t *testing.T) {
	repos := []string{"owner/base", "owner/other", "owner/none"}
	rules := map[string]*github.ProtectionRule{
		"owner/base": {
			Repository:          "owner/base",
			Branch:              "main",
			RequiredReviews:     2,
			RequireStatusChecks: []string{"lint", "test"},
			EnforceAdmins:       true,
		},
		"owner/other": {
			Repository:          "owner/other",
			Branch:              "develop",
			RequiredReviews:     1,
			RequireStatusChecks: []string{"test", "lint"},
			EnforceAdmins:       true,
		},
	}

	output := formatProtectionTable(repos, rules, "owner/base")

	if !strings.Contains(output, "Baseline: owner/base") {
		t.Errorf("expected baseline header, got:\n%s", output)
	}
	if !strings.Contains(output, "1*") {
		t.Errorf("expected drifted review count marked, got:\n%s", output)
	}
	if strings.Contains(output, "2*") {
		t.Errorf("baseline review count must not be marked, got:\n%s", output)
	}
	if !strings.Contains(output, "owner/none") || !strings.Contains(output, "no protection") {
		t.Errorf("expected unprotected repo row, got:\n%s", output)
	}
	if strings.Contains(output, "lint,test*") || strings.Contains(output, "test,lint*") {
		t.Errorf("status checks with same set must not be marked, got:\n%s", output)
	}
}

func TestFormatProtectionTableNoBaseline(t *testing.T) {
	rules := map[string]*github.ProtectionRule{
		"owner/a": {Repository: "owner/a", Branch: "main", RequiredReviews: 1},
	}

	output := formatProtectionTable([]string{"owner/a"}, rules, "")

	if strings.Contains(output, "Baseline:") {
		t.Errorf("unexpected baseline header, got:\n%s", output)
	}
	if strings.Contains(output, "*") {
		t.Errorf("no drift markers expected without baseline, got:\n%s", output)
	}
}

func TestEqualStringSets(t *testing.T) {
	if !equalStringSets([]string{"a", "b"}, []string{"b", "a"}) {
		t.Error("expected order-insensitive equality")
	}
	if equalStringSets([]string{"a"}, []string{"a", "b"}) {
		t.Error("expected inequality for different lengths")
	}
	if !equalStringSets(nil, nil) {
		t.Error("expected nil sets to be equal")
	}
}

func TestFetchProtectionRules(t *testing.T) {
	t.Parallel()

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/repos/acme/widgets":
			return jsonResponse(req, http.StatusOK, `{"default_branch":"main"}`), nil
		case "/repos/acme/widgets/branches/main/protection":
			return jsonResponse(
				req, http.StatusOK,
				`{"required_pull_request_reviews":{"required_approving_review_count":2}}`,
			), nil
		case "/repos/acme/unprotected":
			return jsonResponse(req, http.StatusOK, `{"default_branch":"main"}`), nil
		case "/repos/acme/unprotected/branches/main/protection":
			return jsonResponse(req, http.StatusNotFound, `{"message":"Not Found"}`), nil
		default:
			return jsonResponse(req, http.StatusNotFound, `{"message":"Not Found"}`), nil
		}
	})

	client, err := github.NewClientWithTransport(context.Background(), transport)
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	rules := fetchProtectionRules(client, []string{"acme/widgets", "acme/unprotected", "invalid-repo"})

	if len(rules) != 1 {
		t.Fatalf("rules = %+v, want just acme/widgets", rules)
	}
	if rules["acme/widgets"].RequiredReviews != 2 {
		t.Errorf("acme/widgets rule = %+v", rules["acme/widgets"])
	}
}
