package github

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

const userURL = "https://api.github.com/user"

type stubTransport struct {
	respond func(int) *http.Response
	calls   int
}

func (s *stubTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	s.calls++

	return s.respond(s.calls), nil
}

// headerOf canonicalizes keys the way a parsed response does, which a literal
// http.Header map does not.
func headerOf(pairs map[string]string) http.Header {
	header := http.Header{}
	for key, value := range pairs {
		header.Set(key, value)
	}

	return header
}

func response(status int, header http.Header) *http.Response {
	if header == nil {
		header = http.Header{}
	}

	return &http.Response{
		StatusCode: status,
		Header:     header,
		Body:       http.NoBody,
	}
}

func TestRateLimitTransport(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.September, 2, 3, 6, 0, 0, time.UTC)
	reset := now.Add(11 * time.Minute)

	exhaustedHeader := headerOf(map[string]string{
		headerRemaining: "0",
		headerReset:     strconv.FormatInt(reset.Unix(), 10),
		headerLimit:     "5000",
		headerResource:  "core",
	})

	tests := []struct {
		header    http.Header
		name      string
		status    int
		wantLimit bool
	}{
		{
			name:      "exhausted core quota",
			status:    http.StatusForbidden,
			header:    exhaustedHeader,
			wantLimit: true,
		},
		{
			name:      "secondary limit via Retry-After",
			status:    http.StatusTooManyRequests,
			header:    headerOf(map[string]string{headerRetryAfter: "60"}),
			wantLimit: true,
		},
		{
			name:   "forbidden for permissions, not quota",
			status: http.StatusForbidden,
			header: headerOf(map[string]string{headerRemaining: "4999"}),
		},
		{name: "ordinary success", status: http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			base := &stubTransport{respond: func(int) *http.Response { return response(tc.status, tc.header) }}
			rt := newRateLimitTransport(base)
			rt.now = func() time.Time { return now }

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, userURL, http.NoBody)
			if err != nil {
				t.Fatalf("building request: %v", err)
			}

			_, err = rt.RoundTrip(req) //nolint:bodyclose // the stub serves http.NoBody

			var limitErr *RateLimitError
			if got := errors.As(err, &limitErr); got != tc.wantLimit {
				t.Fatalf("RoundTrip() rate-limit error = %v (err %v), want %v", got, err, tc.wantLimit)
			}

			if !tc.wantLimit {
				return
			}

			if !limitErr.RetryAt.After(now) {
				t.Errorf("RetryAt = %v, want a time after %v", limitErr.RetryAt, now)
			}

			// A second call must not spend a request to relearn the same window.
			//nolint:bodyclose // the stub serves http.NoBody
			if _, err := rt.RoundTrip(req); !errors.As(err, &limitErr) {
				t.Fatalf("second RoundTrip() error = %v, want the recorded RateLimitError", err)
			}

			if base.calls != 1 {
				t.Errorf("base transport calls = %d, want 1", base.calls)
			}
		})
	}
}

func TestRateLimitTransportRetriesOnceTheWindowResets(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.September, 2, 3, 6, 0, 0, time.UTC)
	reset := now.Add(time.Minute)

	base := &stubTransport{respond: func(calls int) *http.Response {
		if calls == 1 {
			return response(http.StatusForbidden, headerOf(map[string]string{
				headerRemaining: "0",
				headerReset:     strconv.FormatInt(reset.Unix(), 10),
			}))
		}

		return response(http.StatusOK, nil)
	}}

	rt := newRateLimitTransport(base)
	clock := now
	rt.now = func() time.Time { return clock }

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, userURL, http.NoBody)
	if err != nil {
		t.Fatalf("building request: %v", err)
	}

	//nolint:bodyclose // the stub serves http.NoBody
	if _, err := rt.RoundTrip(req); err == nil {
		t.Fatal("RoundTrip() error = nil, want a RateLimitError")
	}

	clock = reset.Add(time.Second)

	resp, err := rt.RoundTrip(req) //nolint:bodyclose // the stub serves http.NoBody
	if err != nil {
		t.Fatalf("RoundTrip() after reset error = %v, want success", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestRateLimitErrorNamesTheResetTime(t *testing.T) {
	t.Parallel()

	limitErr := &RateLimitError{
		Resource: "core",
		Limit:    5000,
		RetryAt:  time.Now().Add(11 * time.Minute),
	}

	for _, want := range []string{"core", "5000", "resets at"} {
		if !strings.Contains(limitErr.Error(), want) {
			t.Errorf("Error() = %q, want it to mention %s", limitErr.Error(), want)
		}
	}
}
