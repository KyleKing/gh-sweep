package github_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/KyleKing/gh-sweep/internal/github"
)

const rulesetListBody = `[{"id":42,"name":"main","target":"branch","enforcement":"active"}]`

const rulesetGetBody = `{
	"id":42,"name":"main","target":"branch","enforcement":"active",
	"conditions":{"ref_name":{"include":["~DEFAULT_BRANCH"],"exclude":[]}},
	"bypass_actors":[{"actor_id":5,"actor_type":"Team","bypass_mode":"always"}],
	"rules":[
		{"type":"deletion"},
		{"type":"non_fast_forward"},
		{"type":"tag_name_pattern","parameters":{"operator":"starts_with","pattern":"v"}},
		{"type":"pull_request","parameters":{
			"required_approving_review_count":0,"require_code_owner_review":false,
			"allowed_merge_methods":["squash","rebase"]}},
		{"type":"required_status_checks","parameters":{
			"strict_required_status_checks_policy":false,
			"required_status_checks":[{"context":"ci"}]}}
	]
}`

// rulesetFake serves the list/get pair above and records the body of any write,
// so a test can assert what gh-sweep would send to GitHub.
func rulesetFake(sent *string) roundTripFunc {
	return func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet && req.Body != nil {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, fmt.Errorf("reading request body: %w", err)
			}
			*sent = string(body)
		}

		switch {
		case req.URL.Path == "/repos/acme/widgets/rulesets" && req.Method == http.MethodGet:
			return okJSON(req, rulesetListBody), nil
		case req.URL.Path == "/repos/acme/widgets/rulesets/42":
			return okJSON(req, rulesetGetBody), nil
		case req.URL.Path == "/repos/acme/widgets/rulesets":
			return okJSON(req, rulesetGetBody), nil
		case req.URL.Path == "/repos/acme/empty/rulesets":
			return okJSON(req, `[]`), nil
		default:
			return notFoundJSON(req), nil
		}
	}
}

func newRulesetClient(t *testing.T, sent *string) *github.Client {
	t.Helper()

	client, err := github.NewClientWithTransport(context.Background(), rulesetFake(sent))
	if err != nil {
		t.Fatalf("NewClientWithTransport() error = %v", err)
	}

	return client
}

func TestFindRulesetByName(t *testing.T) {
	t.Parallel()

	var sent string
	client := newRulesetClient(t, &sent)

	found, err := client.FindRulesetByName("acme", "widgets", "main")
	if err != nil {
		t.Fatalf("FindRulesetByName() error = %v", err)
	}

	if found.ID != 42 || !found.BlockDeletion || !found.BlockForcePush {
		t.Errorf("flattened ruleset = %+v, want id 42 with deletion and force-push blocked", found)
	}

	if found.PullRequest == nil || found.PullRequest.RequiredApprovals != 0 {
		t.Fatalf("PullRequest = %+v, want a rule with 0 required approvals", found.PullRequest)
	}

	if len(found.RequiredStatusChecks) != 1 || found.RequiredStatusChecks[0] != "ci" {
		t.Errorf("RequiredStatusChecks = %v, want [ci]", found.RequiredStatusChecks)
	}

	// tag_name_pattern has no field on Ruleset, so it must survive as raw JSON.
	if len(found.Unmanaged) != 1 {
		t.Errorf("Unmanaged = %v, want the tag_name_pattern rule preserved", found.Unmanaged)
	}

	if _, err := client.FindRulesetByName("acme", "empty", "main"); !errors.Is(err, github.ErrRulesetNotFound) {
		t.Errorf("FindRulesetByName() on a repo with no rulesets error = %v, want ErrRulesetNotFound", err)
	}
}

// TestUpdateRulesetPreservesUnmodeledState is the invariant that makes a
// full-replacement PUT safe: changing one rule must not drop bypass actors or
// rule types gh-sweep has no field for.
func TestUpdateRulesetPreservesUnmodeledState(t *testing.T) {
	t.Parallel()

	var sent string
	client := newRulesetClient(t, &sent)

	live, err := client.FindRulesetByName("acme", "widgets", "main")
	if err != nil {
		t.Fatalf("FindRulesetByName() error = %v", err)
	}

	live.PullRequest.RequiredApprovals = 2
	live.PullRequest.AllowedMergeMethods = []string{"squash"}

	if err := client.UpdateRuleset("acme", "widgets", live.ID, *live); err != nil {
		t.Fatalf("UpdateRuleset() error = %v", err)
	}

	var body struct {
		Name         string            `json:"name"`
		Enforcement  string            `json:"enforcement"`
		BypassActors []json.RawMessage `json:"bypass_actors"`
		Rules        []struct {
			Type       string          `json:"type"`
			Parameters json.RawMessage `json:"parameters"`
		} `json:"rules"`
	}
	if err := json.Unmarshal([]byte(sent), &body); err != nil {
		t.Fatalf("unmarshalling sent body %q: %v", sent, err)
	}

	types := make([]string, 0, len(body.Rules))
	for _, rule := range body.Rules {
		types = append(types, rule.Type)
	}
	joined := strings.Join(types, ",")

	for _, want := range []string{"deletion", "non_fast_forward", "tag_name_pattern", "pull_request"} {
		if !strings.Contains(joined, want) {
			t.Errorf("sent rules = %s, want %s preserved", joined, want)
		}
	}

	if len(body.BypassActors) != 1 {
		t.Errorf("sent bypass_actors = %v, want the live actor preserved", body.BypassActors)
	}

	if !strings.Contains(sent, `"required_approving_review_count":2`) {
		t.Errorf("sent body = %s, want the updated approval count", sent)
	}
}

func TestCreateRulesetDefaultsToDefaultBranch(t *testing.T) {
	t.Parallel()

	var sent string
	client := newRulesetClient(t, &sent)

	desired := github.Ruleset{
		Name:           "main",
		Enforcement:    "active",
		BlockDeletion:  true,
		BlockForcePush: true,
		PullRequest:    &github.PullRequestRule{AllowedMergeMethods: []string{"squash"}},
	}

	if err := client.CreateRuleset("acme", "widgets", desired); err != nil {
		t.Fatalf("CreateRuleset() error = %v", err)
	}

	for _, want := range []string{`"target":"branch"`, `"~DEFAULT_BRANCH"`, `"type":"pull_request"`} {
		if !strings.Contains(sent, want) {
			t.Errorf("sent body = %s, want it to contain %s", sent, want)
		}
	}
}
