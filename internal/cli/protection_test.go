package cli

import (
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
