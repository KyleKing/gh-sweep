package github_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/KyleKing/gh-sweep/internal/github"
)

// endpointRoutes maps "METHOD path" to the fixture body served for exact-match
// requests in the endpointFake router. Paginated and prefix-matched paths are
// handled separately in endpointFake before this table is consulted.
//
//nolint:gosec // fixture JSON: secret names like "DEPLOY_KEY" are fake test data, not credentials
var endpointRoutes = map[string]string{
	"GET /repos/acme/widgets": `{
		"name":"widgets","default_branch":"main","allow_squash_merge":true,
		"allow_merge_commit":false,"allow_rebase_merge":true,
		"delete_branch_on_merge":true,"has_issues":true,"has_projects":false,"has_wiki":false
	}`,
	"GET /repos/acme/widgets/branches": `[
		{"name":"main","protected":true,
		 "commit":{"sha":"abc123","commit":{"author":{"date":"2026-01-10T12:00:00Z"}}}},
		{"name":"feature",
		 "commit":{"sha":"def456","commit":{"author":{"date":"2026-01-12T12:00:00Z"}}}}
	]`,
	"POST /repos/acme/widgets/pulls": `{"number":8}`,
	"GET /repos/acme/widgets/collaborators": `[
		{"login":"alice","permissions":{"admin":true,"push":true,"pull":true}},
		{"login":"bob","permissions":{"push":true,"pull":true}},
		{"login":"carol","permissions":{"pull":true}}
	]`,
	"PUT /repos/acme/widgets/collaborators/dave":    `{}`,
	"DELETE /repos/acme/widgets/collaborators/dave": `{}`,
	"GET /repos/acme/widgets/releases": `[
		{"id":2,"tag_name":"v1.2.0","name":"v1.2.0","author":{"login":"alice"},
		 "published_at":"2026-01-10T12:00:00Z"},
		{"id":1,"tag_name":"v1.1.0","name":"v1.1.0","author":{"login":"alice"},
		 "published_at":"2025-11-02T12:00:00Z"}
	]`,
	"GET /repos/acme/widgets/releases/latest": `{"id":2,"tag_name":"v1.2.0","name":"v1.2.0","author":{"login":"alice"},
		"published_at":"2026-01-10T12:00:00Z"}`,
	"GET /orgs/acme/actions/secrets": `{"secrets":[{"name":"DEPLOY_KEY","created_at":"a","updated_at":"b"}]}`,
	"GET /repos/acme/widgets/actions/secrets": `{
		"secrets":[{"name":"CODECOV_TOKEN","created_at":"a","updated_at":"b"}]
	}`,
	"GET /repos/acme/widgets/pages": `{"cname":"docs.example.com","html_url":"https://docs.example.com",
		 "https_enforced":true,"status":"built","protected_domain_state":"verified"}`,
	"GET /repos/acme/widgets/contents/CNAME": `{"content":"ZG9jcy5leGFtcGxlLmNvbQo=","encoding":"base64"}`,
	"GET /repos/acme/widgets/branches/main/protection": `{
		"required_pull_request_reviews":{"required_approving_review_count":2,"require_code_owner_reviews":true},
		"required_status_checks":{"contexts":["ci"]},
		"enforce_admins":{"enabled":true},
		"required_linear_history":{"enabled":true},
		"allow_force_pushes":{"enabled":false},
		"allow_deletions":{"enabled":false}
	}`,
	"PUT /repos/acme/widgets/branches/main/protection": `{}`,
	"PATCH /repos/acme/widgets":                        `{"name":"widgets","default_branch":"main"}`,
	"GET /repos/acme/widgets/immutable-releases":       `{"enabled":false,"enforced_by_owner":false}`,
	"PUT /repos/acme/widgets/immutable-releases":       `{}`,
	"DELETE /repos/acme/widgets/immutable-releases":    `{}`,
	"GET /repos/acme/widgets/subscription":             `{"subscribed":true,"ignored":false,"reason":""}`,
	"PUT /repos/acme/widgets/subscription":             `{"subscribed":true,"ignored":false}`,
	"DELETE /repos/acme/widgets/subscription":          `{}`,
	"GET /repos/acme/widgets/hooks": `[
		{"id":7,"config":{"url":"https://ci.example.com/hook"},"events":["push"],"active":true}
	]`,
	"GET /repos/acme/widgets/hooks/7/deliveries": `[
		{"id":1,"event":"push","status_code":200,"duration":100,"delivered_at":"2026-01-10T12:00:00Z"},
		{"id":2,"event":"push","status_code":502,"duration":300,"delivered_at":"2026-01-11T12:00:00Z"}
	]`,
	"GET /repos/acme/widgets/actions/workflows": `{
		"workflows":[{"id":11,"name":"ci","path":".github/workflows/ci.yml","state":"active"}]
	}`,
	"GET /repos/acme/widgets/actions/runs": `{"workflow_runs":[
		{"id":101,"name":"ci","status":"completed","conclusion":"success",
		 "head_branch":"main","head_sha":"abc123",
		 "created_at":"2026-01-10T12:00:00Z","updated_at":"2026-01-10T12:05:00Z"}
	]}`,
	"GET /repos/acme/widgets/contents/.github/workflows/ci.yml": `{
		"content":"YWNtZTogJHt7IHNlY3JldHMuREVQTE9ZX0tFWSB9fQo=","encoding":"base64"
	}`,
}

// endpointNotFoundRoutes are exact-match paths the fake router serves as 404,
// distinct from the catch-all default so intent stays explicit at the call site.
var endpointNotFoundRoutes = map[string]bool{
	"GET /repos/acme/nopages/pages":                        true,
	"GET /repos/acme/nopages/contents/CNAME":               true,
	"GET /repos/acme/unprotected/branches/main/protection": true,
}

// endpointPaginatedRoute serves a first page of body and an empty page thereafter.
func endpointPaginatedRoute(req *http.Request, body string) *http.Response {
	if req.URL.Query().Get("page") != "1" {
		return okJSON(req, `[]`)
	}

	return okJSON(req, body)
}

func endpointFake() roundTripFunc {
	return func(req *http.Request) (*http.Response, error) {
		path := req.URL.Path
		method := req.Method
		key := method + " " + path

		switch {
		case method == http.MethodGet && strings.HasPrefix(path, "/repos/acme/widgets/compare/"):
			return okJSON(req, `{"ahead_by":3,"behind_by":1}`), nil
		case key == "GET /repos/acme/widgets/pulls":
			return endpointPaginatedRoute(req, `[
				{"number":7,"title":"Add login","state":"open",
				 "head":{"ref":"feature","sha":"def456","repo":{"full_name":"acme/widgets"}},
				 "base":{"ref":"main","sha":"abc123","repo":{"full_name":"acme/widgets"}}}
			]`), nil
		case key == "GET /orgs/acme/repos":
			return endpointPaginatedRoute(req, `[
				{"name":"widgets","full_name":"acme/widgets","owner":{"login":"acme"},"default_branch":"main"},
				{"name":"gadgets","full_name":"acme/gadgets","owner":{"login":"acme"},
				 "private":true,"archived":true,"default_branch":"main"}
			]`), nil
		case key == "GET /users/tester/repos":
			return endpointPaginatedRoute(req, `[{"name":"dotfiles","full_name":"tester/dotfiles",
				  "owner":{"login":"tester"},"default_branch":"main"}]`), nil
		case endpointNotFoundRoutes[key]:
			return notFoundJSON(req), nil
		default:
			if body, ok := endpointRoutes[key]; ok {
				return okJSON(req, body), nil
			}

			return notFoundJSON(req), nil
		}
	}
}

func okJSON(req *http.Request, body string) *http.Response {
	resp := jsonResponse(http.StatusOK, body)
	resp.Request = req

	return resp
}

func notFoundJSON(req *http.Request) *http.Response {
	resp := jsonResponse(http.StatusNotFound, `{"message":"Not Found"}`)
	resp.Request = req

	return resp
}

func newEndpointClient(t *testing.T) *github.Client {
	t.Helper()

	client, err := github.NewClientWithTransport(context.Background(), endpointFake())
	if err != nil {
		t.Fatalf("NewClientWithTransport() error = %v", err)
	}

	return client
}

func TestClientContext(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client, err := github.NewClientWithTransport(ctx, endpointFake())
	if err != nil {
		t.Fatalf("NewClientWithTransport() error = %v", err)
	}

	if client.Context() != ctx {
		t.Error("Context() did not return the context the client was created with")
	}
}

func TestGetPullRequestsForBranch(t *testing.T) {
	t.Parallel()

	prs, err := newEndpointClient(t).GetPullRequestsForBranch("acme", "widgets", "feature")
	if err != nil {
		t.Fatalf("GetPullRequestsForBranch() error = %v", err)
	}

	if len(prs) != 1 || prs[0].Number != 7 {
		t.Errorf("prs = %+v, want just #7", prs)
	}
}

func TestCreatePullRequest(t *testing.T) {
	t.Parallel()

	number, err := newEndpointClient(
		t,
	).CreatePullRequest("acme", "widgets", "title", "body", "feature", "main")
	if err != nil {
		t.Fatalf("CreatePullRequest() error = %v", err)
	}

	if number != 8 {
		t.Errorf("number = %d, want 8", number)
	}
}

func TestListCollaborators(t *testing.T) {
	t.Parallel()

	collaborators, err := newEndpointClient(t).ListCollaborators("acme", "widgets")
	if err != nil {
		t.Fatalf("ListCollaborators() error = %v", err)
	}

	want := map[string]string{"alice": "admin", "bob": "write", "carol": "read"}
	if len(collaborators) != len(want) {
		t.Fatalf("collaborators = %d, want %d", len(collaborators), len(want))
	}

	for _, collaborator := range collaborators {
		if want[collaborator.Login] != collaborator.Permission {
			t.Errorf(
				"%s permission = %q, want %q",
				collaborator.Login,
				collaborator.Permission,
				want[collaborator.Login],
			)
		}
	}
}

func TestCollaboratorMutations(t *testing.T) {
	t.Parallel()

	client := newEndpointClient(t)

	if err := client.AddCollaborator("acme", "widgets", "dave", "push"); err != nil {
		t.Errorf("AddCollaborator() error = %v", err)
	}

	if err := client.RemoveCollaborator("acme", "widgets", "dave"); err != nil {
		t.Errorf("RemoveCollaborator() error = %v", err)
	}
}

func TestListReleasesAndLatest(t *testing.T) {
	t.Parallel()

	client := newEndpointClient(t)

	releases, err := client.ListReleases("acme", "widgets")
	if err != nil {
		t.Fatalf("ListReleases() error = %v", err)
	}

	if len(releases) != 2 || releases[0].TagName != "v1.2.0" || releases[0].Author != "alice" {
		t.Errorf("releases = %+v", releases)
	}

	latest, err := client.GetLatestRelease("acme", "widgets")
	if err != nil {
		t.Fatalf("GetLatestRelease() error = %v", err)
	}

	if latest.TagName != "v1.2.0" {
		t.Errorf("latest = %+v", latest)
	}
}

func TestListSecrets(t *testing.T) {
	t.Parallel()

	client := newEndpointClient(t)

	orgSecrets, err := client.ListOrgSecrets("acme")
	if err != nil {
		t.Fatalf("ListOrgSecrets() error = %v", err)
	}

	if len(orgSecrets) != 1 || orgSecrets[0].Name != "DEPLOY_KEY" || orgSecrets[0].Scope != "org" {
		t.Errorf("org secrets = %+v", orgSecrets)
	}

	repoSecrets, err := client.ListRepoSecrets("acme", "widgets")
	if err != nil {
		t.Fatalf("ListRepoSecrets() error = %v", err)
	}

	if len(repoSecrets) != 1 || repoSecrets[0].Name != "CODECOV_TOKEN" ||
		repoSecrets[0].Scope != "repo" {
		t.Errorf("repo secrets = %+v", repoSecrets)
	}
}

func TestGetRepoSettings(t *testing.T) {
	t.Parallel()

	settings, err := newEndpointClient(t).GetRepoSettings("acme", "widgets")
	if err != nil {
		t.Fatalf("GetRepoSettings() error = %v", err)
	}

	if settings.Repository != "acme/widgets" || settings.DefaultBranch != "main" {
		t.Errorf("settings identity = %+v", settings)
	}

	if !settings.AllowSquashMerge || settings.AllowMergeCommit || !settings.DeleteBranchOnMerge {
		t.Errorf("settings merge flags = %+v", settings)
	}
}

func TestGetDefaultBranchProtection(t *testing.T) {
	t.Parallel()

	rule, err := newEndpointClient(t).GetDefaultBranchProtection("acme", "widgets")
	if err != nil {
		t.Fatalf("GetDefaultBranchProtection() error = %v", err)
	}

	if rule.Branch != "main" || rule.RequiredReviews != 2 || !rule.RequireCodeOwnerReviews {
		t.Errorf("rule = %+v", rule)
	}

	if len(rule.RequireStatusChecks) != 1 || rule.RequireStatusChecks[0] != "ci" {
		t.Errorf("status checks = %v", rule.RequireStatusChecks)
	}

	if !rule.EnforceAdmins || !rule.RequireLinearHistory || rule.AllowForcePushes ||
		rule.AllowDeletions {
		t.Errorf("rule flags = %+v", rule)
	}
}

func TestGetBranchProtectionMissing(t *testing.T) {
	t.Parallel()

	_, err := newEndpointClient(t).GetBranchProtection("acme", "unprotected", "main")
	if !errors.Is(err, github.ErrBranchNotProtected) {
		t.Fatalf("GetBranchProtection() error = %v, want ErrBranchNotProtected", err)
	}
}

func TestGetPagesInfo(t *testing.T) {
	t.Parallel()

	client := newEndpointClient(t)

	info, err := client.GetPagesInfo("acme", "widgets")
	if err != nil {
		t.Fatalf("GetPagesInfo() error = %v", err)
	}

	if info.CNAME != "docs.example.com" || !info.HTTPSEnforced || !info.DomainVerified {
		t.Errorf("info = %+v", info)
	}

	info, err = client.GetPagesInfo("acme", "nopages")
	if !errors.Is(err, github.ErrPagesNotFound) {
		t.Fatalf("GetPagesInfo() error = %v, want ErrPagesNotFound", err)
	}

	if info != nil {
		t.Errorf("expected nil info for a repo without Pages, got %+v", info)
	}
}

func TestGetCNAMEFile(t *testing.T) {
	t.Parallel()

	client := newEndpointClient(t)

	cname, err := client.GetCNAMEFile("acme", "widgets")
	if err != nil {
		t.Fatalf("GetCNAMEFile() error = %v", err)
	}

	if cname != "docs.example.com" {
		t.Errorf("cname = %q, want docs.example.com", cname)
	}

	cname, err = client.GetCNAMEFile("acme", "nopages")
	if err != nil {
		t.Fatalf("GetCNAMEFile() error = %v", err)
	}

	if cname != "" {
		t.Errorf("cname = %q, want empty for a repo without a CNAME file", cname)
	}
}

func TestSubscriptionLifecycle(t *testing.T) {
	t.Parallel()

	client := newEndpointClient(t)

	set, err := client.SetRepoSubscription("acme", "widgets", true, false)
	if err != nil {
		t.Fatalf("SetRepoSubscription() error = %v", err)
	}

	if !set.Subscribed || set.State != github.WatchStateSubscribed {
		t.Errorf("set subscription = %+v", set)
	}

	if err := client.DeleteRepoSubscription("acme", "widgets"); err != nil {
		t.Errorf("DeleteRepoSubscription() error = %v", err)
	}
}

func TestListNamespaceRepositories(t *testing.T) {
	t.Parallel()

	client := newEndpointClient(t)

	repos, isOrg, err := client.ListNamespaceRepositories("acme")
	if err != nil {
		t.Fatalf("ListNamespaceRepositories(acme) error = %v", err)
	}

	if !isOrg || len(repos) != 2 {
		t.Errorf("acme: isOrg=%v repos=%d, want org with 2 repos", isOrg, len(repos))
	}

	repos, isOrg, err = client.ListNamespaceRepositories("tester")
	if err != nil {
		t.Fatalf("ListNamespaceRepositories(tester) error = %v", err)
	}

	if isOrg || len(repos) != 1 || repos[0].FullName != "tester/dotfiles" {
		t.Errorf("tester: isOrg=%v repos=%+v, want user fallback", isOrg, repos)
	}
}

func TestListWebhooksAndDeliveries(t *testing.T) {
	t.Parallel()

	client := newEndpointClient(t)

	webhooks, err := client.ListWebhooks("acme", "widgets")
	if err != nil {
		t.Fatalf("ListWebhooks() error = %v", err)
	}

	if len(webhooks) != 1 || webhooks[0].URL != "https://ci.example.com/hook" ||
		!webhooks[0].Active {
		t.Errorf("webhooks = %+v", webhooks)
	}

	deliveries, err := client.ListWebhookDeliveries("acme", "widgets", webhooks[0].ID)
	if err != nil {
		t.Fatalf("ListWebhookDeliveries() error = %v", err)
	}

	if len(deliveries) != 2 || deliveries[1].Status != 502 {
		t.Errorf("deliveries = %+v", deliveries)
	}
}

func TestUpdateRepoSettings(t *testing.T) {
	t.Parallel()

	enabled := true
	err := newEndpointClient(t).
		UpdateRepoSettings("acme", "widgets", github.RepoSettingsPatch{DeleteBranchOnMerge: &enabled})
	if err != nil {
		t.Errorf("UpdateRepoSettings() error = %v", err)
	}
}

func TestUpdateBranchProtection(t *testing.T) {
	t.Parallel()

	err := newEndpointClient(t).UpdateBranchProtection("acme", "widgets", "main", github.ProtectionRule{
		RequiredReviews: 1,
	})
	if err != nil {
		t.Errorf("UpdateBranchProtection() error = %v", err)
	}
}

func TestImmutableReleasesLifecycle(t *testing.T) {
	t.Parallel()

	client := newEndpointClient(t)

	settings, err := client.GetImmutableReleases("acme", "widgets")
	if err != nil {
		t.Fatalf("GetImmutableReleases() error = %v", err)
	}

	if settings.Enabled || settings.EnforcedByOwner {
		t.Errorf("settings = %+v, want disabled", settings)
	}

	if err := client.SetImmutableReleases("acme", "widgets", true); err != nil {
		t.Errorf("SetImmutableReleases(true) error = %v", err)
	}

	if err := client.SetImmutableReleases("acme", "widgets", false); err != nil {
		t.Errorf("SetImmutableReleases(false) error = %v", err)
	}
}

func TestListWorkflowsAndRuns(t *testing.T) {
	t.Parallel()

	client := newEndpointClient(t)

	workflows, err := client.ListWorkflows("acme", "widgets")
	if err != nil {
		t.Fatalf("ListWorkflows() error = %v", err)
	}

	if len(workflows) != 1 || workflows[0].Name != "ci" {
		t.Errorf("workflows = %+v", workflows)
	}

	runs, err := client.FetchWorkflowRuns("acme", "widgets", github.FetchWorkflowRunsOptions{})
	if err != nil {
		t.Fatalf("FetchWorkflowRuns() error = %v", err)
	}

	if len(runs) != 1 || runs[0].Conclusion != "success" {
		t.Errorf("runs = %+v", runs)
	}

	if runs[0].Duration != 5*time.Minute {
		t.Errorf("run duration = %v, want 5m", runs[0].Duration)
	}
}
