// Package github wraps the GitHub REST and GraphQL APIs used by gh-sweep.
package github

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
)

const (
	testAPIHost   = "github.com"
	testAuthToken = "gh-sweep-test-token"
)

var (
	testTransportMu sync.RWMutex
	testTransport   http.RoundTripper
)

// SetTestTransport routes every client created afterward through rt so tests
// never reach the real GitHub API. It returns a restore function and panics
// when called outside `go test`.
func SetTestTransport(rt http.RoundTripper) func() {
	if !testing.Testing() {
		panic("github.SetTestTransport is test-only")
	}

	testTransportMu.Lock()
	previous := testTransport
	testTransport = rt
	testTransportMu.Unlock()

	return func() {
		testTransportMu.Lock()
		testTransport = previous
		testTransportMu.Unlock()
	}
}

func currentTestTransport() http.RoundTripper {
	testTransportMu.RLock()
	defer testTransportMu.RUnlock()

	return testTransport
}

// clientOptions returns default options for real clients. Under `go test` it
// pins host and token (so no keyring lookup happens) and routes requests to
// the registered fake transport, or to a mutation-guarded real transport when
// no fake is registered.
func clientOptions() *api.ClientOptions {
	if !testing.Testing() {
		return &api.ClientOptions{}
	}

	transport := currentTestTransport()
	if transport == nil {
		transport = safetyTransport{base: http.DefaultTransport}
	}

	return &api.ClientOptions{
		Host:      testAPIHost,
		AuthToken: testAuthToken,
		Transport: transport,
	}
}

// safetyTransport panics on mutating requests during tests. It mirrors the
// runtime guard in gh-lazydispatch's exec.RealExecutor: a test that forgets to
// fake the transport fails loudly before it can touch real GitHub resources.
type safetyTransport struct {
	base http.RoundTripper
}

func (s safetyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if testing.Testing() && isMutatingMethod(req.Method) {
		panic(fmt.Sprintf(
			"SAFETY VIOLATION: attempted real %s %s during a test.\n"+
				"This could mutate real GitHub resources!\n"+
				"Inject a fake via github.SetTestTransport or github.NewClientWithTransport instead.",
			req.Method, req.URL,
		))
	}

	return s.base.RoundTrip(req) //nolint:wrapcheck // transparent proxy: http.Client wraps this in *url.Error itself
}

func isMutatingMethod(method string) bool {
	switch method {
	case http.MethodDelete, http.MethodPatch, http.MethodPost, http.MethodPut:
		return true
	default:
		return false
	}
}
