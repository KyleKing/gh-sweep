package github

import (
	"testing"
	"time"
)

const watchInfoPageOneFixture = `{
  "viewer": {
    "login": "tester",
    "repositories": {
      "pageInfo": {"hasNextPage": true, "endCursor": "cursor-1"},
      "nodes": [
        {
          "name": "widgets",
          "nameWithOwner": "tester/widgets",
          "owner": {"login": "tester"},
          "isPrivate": false,
          "isArchived": false,
          "isFork": false,
          "viewerSubscription": "SUBSCRIBED",
          "viewerCanSubscribe": true,
          "stargazerCount": 5,
          "pushedAt": "2026-07-01T00:00:00Z",
          "updatedAt": "2026-07-02T00:00:00Z",
          "watchers": {"totalCount": 2}
        }
      ]
    }
  }
}`

const watchInfoPageTwoFixture = `{
  "viewer": {
    "login": "tester",
    "repositories": {
      "pageInfo": {"hasNextPage": false, "endCursor": "cursor-2"},
      "nodes": [
        {
          "name": "gadgets",
          "nameWithOwner": "tester/gadgets",
          "owner": {"login": "tester"},
          "isPrivate": true,
          "isArchived": true,
          "isFork": true,
          "viewerSubscription": "IGNORED",
          "viewerCanSubscribe": true,
          "stargazerCount": 0,
          "pushedAt": "2020-01-01T00:00:00Z",
          "updatedAt": "2020-01-02T00:00:00Z",
          "watchers": {"totalCount": 0}
        },
        {
          "name": "dotfiles",
          "nameWithOwner": "tester/dotfiles",
          "owner": {"login": "tester"},
          "isPrivate": false,
          "isArchived": false,
          "isFork": false,
          "viewerSubscription": "UNSUBSCRIBED",
          "viewerCanSubscribe": true,
          "stargazerCount": 1,
          "pushedAt": "2026-06-01T00:00:00Z",
          "updatedAt": "2026-06-02T00:00:00Z",
          "watchers": {"totalCount": 1}
        }
      ]
    }
  }
}`

func assertWatchInfoCalls(t *testing.T, calls []map[string]any) {
	t.Helper()

	if len(calls) != 2 {
		t.Fatalf("expected 2 GraphQL calls, got %d", len(calls))
	}
	if calls[0]["cursor"] != nil {
		t.Errorf("expected nil cursor on first call, got %v", calls[0]["cursor"])
	}
	if calls[1]["cursor"] != "cursor-1" {
		t.Errorf("expected cursor-1 on second call, got %v", calls[1]["cursor"])
	}
}

func assertWatchInfoRepos(t *testing.T, infos []RepoWatchInfo) {
	t.Helper()

	widgets := infos[0]
	if widgets.FullName != "tester/widgets" || widgets.State != WatchStateSubscribed {
		t.Errorf("widgets = %+v", widgets)
	}
	if widgets.StargazerCount != 5 || widgets.WatcherCount != 2 {
		t.Errorf("widgets metadata = %+v", widgets)
	}
	if !widgets.PushedAt.Equal(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("widgets.PushedAt = %v", widgets.PushedAt)
	}

	gadgets := infos[1]
	if !gadgets.Private || !gadgets.IsArchived || !gadgets.IsFork ||
		gadgets.State != WatchStateIgnored {
		t.Errorf("gadgets = %+v", gadgets)
	}

	dotfiles := infos[2]
	if dotfiles.State != WatchStateDefault {
		t.Errorf("dotfiles.State = %q, want default", dotfiles.State)
	}
}

func TestListViewerRepoWatchInfoPagination(t *testing.T) {
	t.Parallel()

	fake := &fakeGQLDoer{pages: []string{watchInfoPageOneFixture, watchInfoPageTwoFixture}}
	client := &GQLClient{doer: fake}

	username, infos, err := client.ListViewerRepoWatchInfo()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if username != "tester" {
		t.Errorf("username = %q, want %q", username, "tester")
	}

	if len(infos) != 3 {
		t.Fatalf("expected 3 repos across pages, got %d", len(infos))
	}

	assertWatchInfoCalls(t, fake.calls)
	assertWatchInfoRepos(t, infos)
}
