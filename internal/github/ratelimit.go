package github

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	headerRemaining  = "X-RateLimit-Remaining"
	headerReset      = "X-RateLimit-Reset"
	headerLimit      = "X-RateLimit-Limit"
	headerResource   = "X-RateLimit-Resource"
	headerRetryAfter = "Retry-After"
)

// RateLimitError reports an exhausted GitHub rate limit. Retrying before
// RetryAt burns nothing but still fails, so callers should surface the time
// rather than offering an immediate retry.
type RateLimitError struct {
	RetryAt  time.Time
	Resource string
	Limit    int
}

func (e *RateLimitError) Error() string {
	resource := e.Resource
	if resource == "" {
		resource = "api"
	}

	wait := time.Until(e.RetryAt).Round(time.Second)
	if wait < 0 {
		wait = 0
	}

	//nolint:gosmopolitan // a wall-clock time a person reads is only useful in their own zone
	return fmt.Sprintf("GitHub %s rate limit exhausted (%d/hour); resets at %s, in %s",
		resource, e.Limit, e.RetryAt.Local().Format(time.Kitchen), wait)
}

// rateLimitTransport turns an exhausted-quota response into a RateLimitError
// and short-circuits later requests until the window resets, so a retry loop
// cannot spend a request per attempt learning the same thing.
type rateLimitTransport struct {
	base    http.RoundTripper
	now     func() time.Time
	blocked map[string]*RateLimitError
	mu      sync.Mutex
}

func newRateLimitTransport(base http.RoundTripper) *rateLimitTransport {
	if base == nil {
		base = http.DefaultTransport
	}

	return &rateLimitTransport{base: base, now: time.Now, blocked: map[string]*RateLimitError{}}
}

// pool names the allowance a request spends. GitHub's pools are separate
// budgets rather than slices of one, so a tool out of core can still make
// GraphQL calls and must not be blocked from trying.
func pool(req *http.Request) string {
	if strings.HasSuffix(req.URL.Path, "/graphql") {
		return "graphql"
	}

	if strings.HasPrefix(req.URL.Path, "/search/") {
		return "search"
	}

	return "core"
}

func (t *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resource := pool(req)
	if blocked := t.current(resource); blocked != nil {
		return nil, blocked
	}

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err //nolint:wrapcheck // a transport must return the base error unchanged
	}

	if limitErr := exhausted(resp, t.now()); limitErr != nil {
		if limitErr.Resource == "" {
			limitErr.Resource = resource
		}
		t.block(limitErr)

		if err := drain(resp); err != nil {
			return nil, errors.Join(limitErr, err)
		}

		return nil, limitErr
	}

	return resp, nil
}

func (t *rateLimitTransport) current(resource string) *RateLimitError {
	t.mu.Lock()
	defer t.mu.Unlock()

	blocked, found := t.blocked[resource]
	if !found {
		return nil
	}

	if !t.now().Before(blocked.RetryAt) {
		delete(t.blocked, resource)

		return nil
	}

	return blocked
}

func (t *rateLimitTransport) block(limitErr *RateLimitError) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.blocked[limitErr.Resource] = limitErr
}

// exhausted returns an error only when the response says the quota is spent.
// A 403 for permissions carries no remaining-count header and must pass through
// as an ordinary error so it is not mistaken for a rate limit.
func exhausted(resp *http.Response, now time.Time) *RateLimitError {
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusTooManyRequests {
		return nil
	}

	limitErr := &RateLimitError{
		Resource: resp.Header.Get(headerResource),
		Limit:    intHeader(resp.Header.Get(headerLimit)),
	}

	if after := intHeader(resp.Header.Get(headerRetryAfter)); after > 0 {
		limitErr.RetryAt = now.Add(time.Duration(after) * time.Second)

		return limitErr
	}

	remaining := resp.Header.Get(headerRemaining)
	if remaining != "0" {
		return nil
	}

	limitErr.RetryAt = time.Unix(int64(intHeader(resp.Header.Get(headerReset))), 0)

	return limitErr
}

func intHeader(value string) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}

	return parsed
}

func drain(resp *http.Response) error {
	if resp.Body == nil {
		return nil
	}

	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return errors.Join(err, resp.Body.Close())
	}

	return resp.Body.Close() //nolint:wrapcheck // an io error on an abandoned body needs no context
}
