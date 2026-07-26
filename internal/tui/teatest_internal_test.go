package tui

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/teatest/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
)

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

// routeFakeGitHub serves the canned responses component load commands need so
// teatest programs never reach the network. Unknown paths return 404, which
// components surface as skipped repos or error views.
func routeFakeGitHub(req *http.Request) (*http.Response, error) {
	path := req.URL.Path

	switch {
	case path == "/user":
		return jsonResponse(req, http.StatusOK, `{"login":"tester"}`), nil
	case strings.HasPrefix(path, "/user/repos"):
		return jsonResponse(req, http.StatusOK, `[]`), nil
	case path == "/repos/acme/widgets":
		return jsonResponse(req, http.StatusOK, `{"default_branch":"main"}`), nil
	case path == "/repos/acme/widgets/branches":
		return jsonResponse(req, http.StatusOK, `[
			{"name":"main","protected":true,"commit":{"sha":"abc123","commit":{"author":{"date":"2026-01-10T12:00:00Z"}}}},
			{"name":"feature/login","commit":{"sha":"def456","commit":{"author":{"date":"2026-01-12T12:00:00Z"}}}}
		]`), nil
	case strings.HasPrefix(path, "/repos/acme/widgets/compare/"):
		return jsonResponse(req, http.StatusOK, `{"ahead_by":2,"behind_by":1}`), nil
	case strings.HasPrefix(path, "/repos/acme/widgets/pulls"):
		return jsonResponse(req, http.StatusOK, `[]`), nil
	case strings.HasSuffix(path, "/releases"):
		return jsonResponse(req, http.StatusOK, `[]`), nil
	case path == "/graphql":
		return jsonResponse(
			req,
			http.StatusOK,
			`{"data":{"viewer":{"login":"tester","repositories":{
			"pageInfo":{"hasNextPage":false,"endCursor":""},
			"nodes":[{"name":"widgets","nameWithOwner":"acme/widgets","owner":{"login":"acme"},
				"isPrivate":false,"isArchived":false,"isFork":false,"viewerSubscription":"SUBSCRIBED",
				"viewerCanSubscribe":true,"stargazerCount":0,"pushedAt":"2026-01-10T12:00:00Z",
				"updatedAt":"2026-01-10T12:00:00Z","watchers":{"totalCount":1}}]
		}}}}`,
		), nil
	default:
		return jsonResponse(req, http.StatusNotFound, `{"message":"Not Found"}`), nil
	}
}

func TestMain(m *testing.M) {
	restore := github.SetTestTransport(roundTripFunc(routeFakeGitHub))
	code := m.Run()
	restore()
	os.Exit(code)
}

func newTeatestModel() MainModel {
	return NewMainModel(MainModelOptions{
		Baseline: "acme/widgets",
		Org:      "acme",
		Repo:     "acme/widgets",
		Repos:    []string{"acme/widgets", "acme/gadgets"},
	})
}

func waitForOutput(t *testing.T, tm *teatest.TestModel, want string) {
	t.Helper()

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte(want))
	}, teatest.WithDuration(3*time.Second))
}

func pressKey(tm *teatest.TestModel, code rune) {
	tm.Send(tea.KeyPressMsg{Code: code, Text: string(code)})
}

func TestTUIBootAndQuit(t *testing.T) {
	t.Parallel()

	tm := teatest.NewTestModel(t, newTeatestModel(), teatest.WithInitialTermSize(120, 40))

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("gh-sweep")) &&
			bytes.Contains(bts, []byte("Namespace Audit"))
	}, teatest.WithDuration(3*time.Second))

	pressKey(tm, 'q')
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestTUITinyTerminal(t *testing.T) {
	t.Parallel()

	tm := teatest.NewTestModel(t, newTeatestModel(), teatest.WithInitialTermSize(40, 10))

	waitForOutput(t, tm, "gh-sweep")

	pressKey(tm, 'q')
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestTUINavigateViewsAndBack(t *testing.T) {
	t.Parallel()

	tm := teatest.NewTestModel(t, newTeatestModel(), teatest.WithInitialTermSize(120, 40))

	waitForOutput(t, tm, "Namespace Audit")

	pressKey(tm, '0')
	waitForOutput(t, tm, "Watch Status Audit")
	tm.Send(tea.KeyPressMsg{Code: tea.KeyEscape})

	pressKey(tm, '1')
	waitForOutput(t, tm, "Branches for acme/widgets")
	tm.Send(tea.KeyPressMsg{Code: tea.KeyEscape})

	pressKey(tm, '2')
	waitForOutput(t, tm, "Branch Protection Rules")
	tm.Send(tea.KeyPressMsg{Code: tea.KeyEscape})

	pressKey(tm, '9')
	waitForOutput(t, tm, "Release Overview")
	tm.Send(tea.KeyPressMsg{Code: tea.KeyEscape})

	pressKey(tm, 'q')
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	final, ok := tm.FinalModel(t).(MainModel)
	if !ok {
		t.Fatalf("FinalModel returned %T, want MainModel", tm.FinalModel(t))
	}

	if final.mode != ViewHome {
		t.Errorf("final mode = %d, want ViewHome", final.mode)
	}
}
