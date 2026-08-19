package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

const singlePageFixture = `{
  "repository": {
    "pullRequest": {
      "title": "Add feature",
      "reviewThreads": {
        "pageInfo": {"hasNextPage": false, "endCursor": "abc"},
        "nodes": [
          {
            "isResolved": false,
            "isOutdated": true,
            "path": "internal/app/main.go",
            "comments": {
              "nodes": [
                {
                  "author": {"login": "octocat"},
                  "body": "Consider renaming this",
                  "createdAt": "2026-06-01T10:00:00Z",
                  "url": "https://github.com/o/r/pull/7#discussion_r1"
                },
                {
                  "author": {"login": "hubot"},
                  "body": "Agreed",
                  "createdAt": "2026-06-02T11:00:00Z",
                  "url": "https://github.com/o/r/pull/7#discussion_r2"
                }
              ]
            }
          },
          {
            "isResolved": true,
            "isOutdated": false,
            "path": "README.md",
            "comments": {
              "nodes": [
                {
                  "author": {"login": "hubot"},
                  "body": "Typo here",
                  "createdAt": "2026-06-03T09:00:00Z",
                  "url": "https://github.com/o/r/pull/7#discussion_r3"
                }
              ]
            }
          }
        ]
      }
    }
  }
}`

func TestMapReviewThreads(t *testing.T) {
	var response reviewThreadsResponse
	if err := json.Unmarshal([]byte(singlePageFixture), &response); err != nil {
		t.Fatalf("failed to unmarshal fixture: %v", err)
	}

	pr := response.Repository.PullRequest
	threads := mapReviewThreads("o", "r", 7, pr.Title, pr.ReviewThreads.Nodes)

	if len(threads) != 2 {
		t.Fatalf("expected 2 threads, got %d", len(threads))
	}

	first := threads[0]
	if first.Repository != "o/r" || first.PRNumber != 7 || first.PRTitle != "Add feature" {
		t.Errorf("unexpected thread metadata: %+v", first)
	}
	if first.IsResolved || !first.IsOutdated || first.Path != "internal/app/main.go" {
		t.Errorf("unexpected thread state: %+v", first)
	}
	if len(first.Comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(first.Comments))
	}
	if first.Comments[0].Author != "octocat" || first.Comments[0].Body != "Consider renaming this" {
		t.Errorf("unexpected first comment: %+v", first.Comments[0])
	}
	if first.Comments[0].URL != "https://github.com/o/r/pull/7#discussion_r1" {
		t.Errorf("unexpected comment URL: %s", first.Comments[0].URL)
	}
	if !first.Comments[0].CreatedAt.Equal(time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)) {
		t.Errorf("unexpected createdAt: %v", first.Comments[0].CreatedAt)
	}

	if !threads[1].IsResolved {
		t.Errorf("expected second thread resolved: %+v", threads[1])
	}
}

type fakeGQLDoer struct {
	pages []string
	calls []map[string]interface{}
}

func (f *fakeGQLDoer) Do(_ string, variables map[string]interface{}, response interface{}) error {
	f.calls = append(f.calls, variables)

	if err := json.Unmarshal([]byte(f.pages[len(f.calls)-1]), response); err != nil {
		return fmt.Errorf("failed to unmarshal fixture: %w", err)
	}

	return nil
}

const pageOneFixture = `{
  "repository": {
    "pullRequest": {
      "title": "Paginated PR",
      "reviewThreads": {
        "pageInfo": {"hasNextPage": true, "endCursor": "cursor-1"},
        "nodes": [
          {
            "isResolved": false,
            "isOutdated": false,
            "path": "a.go",
            "comments": {"nodes": [
              {"author": {"login": "octocat"}, "body": "first", "createdAt": "2026-06-01T00:00:00Z", "url": ""}
            ]}
          }
        ]
      }
    }
  }
}`

const pageTwoFixture = `{
  "repository": {
    "pullRequest": {
      "title": "Paginated PR",
      "reviewThreads": {
        "pageInfo": {"hasNextPage": false, "endCursor": "cursor-2"},
        "nodes": [
          {
            "isResolved": true,
            "isOutdated": false,
            "path": "b.go",
            "comments": {"nodes": [
              {"author": {"login": "hubot"}, "body": "second", "createdAt": "2026-06-02T00:00:00Z", "url": ""}
            ]}
          }
        ]
      }
    }
  }
}`

func TestListPRReviewThreadsPagination(t *testing.T) {
	fake := &fakeGQLDoer{pages: []string{pageOneFixture, pageTwoFixture}}
	client := &GQLClient{doer: fake}

	threads, err := client.ListPRReviewThreads("o", "r", 9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(threads) != 2 {
		t.Fatalf("expected 2 threads across pages, got %d", len(threads))
	}
	if threads[0].Path != "a.go" || threads[1].Path != "b.go" {
		t.Errorf("unexpected thread order: %+v", threads)
	}

	if len(fake.calls) != 2 {
		t.Fatalf("expected 2 GraphQL calls, got %d", len(fake.calls))
	}
	if fake.calls[0]["cursor"] != nil {
		t.Errorf("expected nil cursor on first call, got %v", fake.calls[0]["cursor"])
	}
	if fake.calls[1]["cursor"] != "cursor-1" {
		t.Errorf("expected cursor-1 on second call, got %v", fake.calls[1]["cursor"])
	}
}

func makeThread(
	pr int,
	path, author, body string,
	createdAt time.Time,
	resolved bool,
) ReviewThread {
	return ReviewThread{
		Repository: "o/r",
		PRNumber:   pr,
		PRTitle:    "title",
		Path:       path,
		IsResolved: resolved,
		Comments: []ReviewComment{
			{Author: author, Body: body, CreatedAt: createdAt},
		},
	}
}

func TestFilterUnresolvedThreads(t *testing.T) {
	now := time.Now()
	threads := []ReviewThread{
		makeThread(1, "a.go", "octocat", "open question", now, false),
		makeThread(1, "b.go", "hubot", "done", now, true),
	}

	unresolved := FilterUnresolvedThreads(threads)
	if len(unresolved) != 1 || unresolved[0].Path != "a.go" {
		t.Errorf("unexpected unresolved threads: %+v", unresolved)
	}
}

func TestThreadFilterApply(t *testing.T) {
	older := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	cutoff := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	threads := []ReviewThread{
		makeThread(1, "cmd/main.go", "octocat", "Fix the TODO here", older, false),
		makeThread(2, "internal/api.go", "hubot", "Looks good overall", newer, false),
	}

	tests := []struct {
		name   string
		filter ThreadFilter
		want   []string
	}{
		{"no filters", ThreadFilter{}, []string{"cmd/main.go", "internal/api.go"}},
		{"author case-insensitive", ThreadFilter{Author: "OctoCat"}, []string{"cmd/main.go"}},
		{"author no match", ThreadFilter{Author: "nobody"}, []string{}},
		{"since cutoff", ThreadFilter{Since: &cutoff}, []string{"internal/api.go"}},
		{"search body", ThreadFilter{Search: "todo"}, []string{"cmd/main.go"}},
		{"search path", ThreadFilter{Search: "internal"}, []string{"internal/api.go"}},
		{"combined", ThreadFilter{Author: "hubot", Search: "good"}, []string{"internal/api.go"}},
		{"combined excludes", ThreadFilter{Author: "hubot", Search: "todo"}, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.Apply(threads)
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d threads, got %d: %+v", len(tt.want), len(got), got)
			}
			for i, path := range tt.want {
				if got[i].Path != path {
					t.Errorf("expected %s at index %d, got %s", path, i, got[i].Path)
				}
			}
		})
	}
}

func TestParseSinceDate(t *testing.T) {
	parsed, err := ParseSinceDate("2026-03-01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !parsed.Equal(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("unexpected parsed date: %v", parsed)
	}

	for _, invalid := range []string{"", "03/01/2026", "2026-13-01", "yesterday"} {
		if _, err := ParseSinceDate(invalid); err == nil {
			t.Errorf("expected error for %q", invalid)
		}
	}
}

func TestReviewThreadHelpers(t *testing.T) {
	empty := ReviewThread{}
	if _, ok := empty.FirstComment(); ok {
		t.Error("expected no first comment for empty thread")
	}
	if !empty.LastActivity().IsZero() {
		t.Error("expected zero last activity for empty thread")
	}

	early := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	late := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	thread := ReviewThread{Comments: []ReviewComment{
		{Author: "octocat", CreatedAt: early},
		{Author: "hubot", CreatedAt: late},
	}}

	first, ok := thread.FirstComment()
	if !ok || first.Author != "octocat" {
		t.Errorf("unexpected first comment: %+v", first)
	}
	if !thread.LastActivity().Equal(late) {
		t.Errorf("unexpected last activity: %v", thread.LastActivity())
	}
}

const twoOpenPRsFixture = `[
	{"number":1,"title":"first","state":"open",
	 "head":{"ref":"a","sha":"1","repo":{"full_name":"acme/widgets"}},
	 "base":{"ref":"main","sha":"0","repo":{"full_name":"acme/widgets"}}},
	{"number":2,"title":"second","state":"open",
	 "head":{"ref":"b","sha":"2","repo":{"full_name":"acme/widgets"}},
	 "base":{"ref":"main","sha":"0","repo":{"full_name":"acme/widgets"}}}
]`

func openPRsTransport(body string) roundTripFunc {
	return func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/repos/acme/widgets/pulls" && req.URL.Query().Get("page") == "1" {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    req,
			}, nil
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader(`[]`)),
			Request:    req,
		}, nil
	}
}

func TestListOpenPRReviewThreads(t *testing.T) {
	transport := openPRsTransport(twoOpenPRsFixture)

	client, err := NewClientWithTransport(context.Background(), transport)
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	doer := &fakeGQLDoer{pages: []string{singlePageFixture, singlePageFixture}}
	gql := &GQLClient{doer: doer}

	threads, err := gql.ListOpenPRReviewThreads(client, "acme", "widgets", 0)
	if err != nil {
		t.Fatalf("ListOpenPRReviewThreads() error = %v", err)
	}

	if len(threads) != 4 {
		t.Fatalf("expected 4 threads (2 per PR x 2 PRs), got %d", len(threads))
	}
	if len(doer.calls) != 2 {
		t.Errorf("expected one GraphQL call per PR, got %d", len(doer.calls))
	}
}

func TestListOpenPRReviewThreadsCapped(t *testing.T) {
	transport := openPRsTransport(twoOpenPRsFixture)

	client, err := NewClientWithTransport(context.Background(), transport)
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	doer := &fakeGQLDoer{pages: []string{singlePageFixture}}
	gql := &GQLClient{doer: doer}

	threads, err := gql.ListOpenPRReviewThreads(client, "acme", "widgets", 1)
	if err != nil {
		t.Fatalf("ListOpenPRReviewThreads() error = %v", err)
	}
	if len(threads) != 2 {
		t.Fatalf("expected threads from only the first PR (capped), got %d", len(threads))
	}
	if len(doer.calls) != 1 {
		t.Errorf("expected exactly one GraphQL call under the cap, got %d", len(doer.calls))
	}
}

func TestListRepoReviewThreads(t *testing.T) {
	doer := &fakeGQLDoer{pages: []string{singlePageFixture}}
	gql := &GQLClient{doer: doer}

	threads, err := gql.ListRepoReviewThreads(nil, "acme", "widgets", 7, 0)
	if err != nil {
		t.Fatalf("ListRepoReviewThreads() error = %v", err)
	}
	if len(threads) != 2 || threads[0].PRNumber != 7 {
		t.Errorf("expected threads scoped to PR #7, got %+v", threads)
	}
}
