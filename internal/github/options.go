package github

import (
	"net/http"
	"sync"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

// Option configures a Client at construction.
type Option func(*api.ClientOptions)

// WithCache serves repeat GET and GraphQL responses from dir for ttl. Cached
// responses cost no rate-limit quota, which is what keeps a cross-repo sweep
// inside the hourly budget. A zero ttl leaves the cache off.
func WithCache(dir string, ttl time.Duration) Option {
	return func(opts *api.ClientOptions) {
		if ttl <= 0 {
			return
		}

		opts.EnableCache = true
		opts.CacheTTL = ttl
		opts.CacheDir = dir
	}
}

// applyOptions layers opts onto base and installs the rate-limit transport, so
// every real client fails fast with a reset time instead of a bare 403.
func applyOptions(base *api.ClientOptions, opts []Option) *api.ClientOptions {
	if dir, ttl := installedCache(); ttl > 0 {
		WithCache(dir, ttl)(base)
	}

	for _, opt := range opts {
		opt(base)
	}

	if base.Transport == nil {
		base.Transport = http.DefaultTransport
	}
	base.Transport = newRateLimitTransport(base.Transport)

	return base
}

//nolint:gochecknoglobals // a process-wide installed default, mirroring aragonite's cache.SetDiskCache
var (
	defaultCacheMu  sync.Mutex
	defaultCacheDir string
	defaultCacheTTL time.Duration
)

// SetDefaultCache makes every later NewClient serve repeat reads from dir for
// ttl. Call it once at startup: threading the setting through each of the
// TUI's own client constructions would touch every view for one value.
func SetDefaultCache(dir string, ttl time.Duration) {
	defaultCacheMu.Lock()
	defer defaultCacheMu.Unlock()
	defaultCacheDir, defaultCacheTTL = dir, ttl
}

func installedCache() (string, time.Duration) {
	defaultCacheMu.Lock()
	defer defaultCacheMu.Unlock()

	return defaultCacheDir, defaultCacheTTL
}
