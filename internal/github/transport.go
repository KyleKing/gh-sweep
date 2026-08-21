// Package github wraps the GitHub REST and GraphQL APIs used by gh-sweep.
package github

import (
	"net/http"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/kyleking/aragonite/transport"
)

const (
	testAPIHost   = "github.com"
	testAuthToken = "gh-sweep-test-token"
)

// SetTestTransport routes every client created afterward through rt so tests
// never reach the real GitHub API. It returns a restore function and panics
// when called outside `go test`.
func SetTestTransport(rt http.RoundTripper) func() {
	return transport.SetTestTransport(rt)
}

// clientOptions returns default options for real clients. Under `go test` it
// pins host and token (so no keyring lookup happens) and routes requests to
// the registered fake transport, or to a mutation-guarded real transport when
// no fake is registered.
func clientOptions() *api.ClientOptions {
	if !testing.Testing() {
		return &api.ClientOptions{}
	}

	return &api.ClientOptions{
		Host:      testAPIHost,
		AuthToken: testAuthToken,
		Transport: transport.Current(http.DefaultTransport),
	}
}
