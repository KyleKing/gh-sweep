package dns_test

import (
	"context"
	"errors"
	"testing"

	"github.com/KyleKing/gh-sweep/internal/dns"
)

var errNoSuchHost = errors.New("no such host")

type fakeResolver struct {
	cname    string
	cnameErr error
	addrs    []string
	hostErr  error
}

func (f fakeResolver) LookupCNAME(_ context.Context, _ string) (string, error) {
	return f.cname, f.cnameErr
}

func (f fakeResolver) LookupHost(_ context.Context, _ string) ([]string, error) {
	return f.addrs, f.hostErr
}

func TestResolve(t *testing.T) {
	t.Parallel()

	notFound := errNoSuchHost

	tests := []struct {
		name          string
		resolver      fakeResolver
		wantResolves  bool
		wantPagesLink bool
	}{
		{
			name:          "CNAME to github.io",
			resolver:      fakeResolver{cname: "user.github.io.", addrs: []string{"185.199.108.153"}},
			wantResolves:  true,
			wantPagesLink: true,
		},
		{
			name:          "apex A record at a documented Pages IP",
			resolver:      fakeResolver{cname: "apex.example.com.", addrs: []string{"185.199.109.153"}},
			wantResolves:  true,
			wantPagesLink: true,
		},
		{
			name:          "resolves elsewhere",
			resolver:      fakeResolver{cname: "apex.example.com.", addrs: []string{"93.184.216.34"}},
			wantResolves:  true,
			wantPagesLink: false,
		},
		{
			name:          "does not resolve at all",
			resolver:      fakeResolver{cnameErr: notFound, hostErr: notFound},
			wantResolves:  false,
			wantPagesLink: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := dns.Resolve(context.Background(), tt.resolver, "example.com")
			if got.Resolves != tt.wantResolves {
				t.Errorf("Resolves = %v, want %v", got.Resolves, tt.wantResolves)
			}
			if got.PointsAtPages != tt.wantPagesLink {
				t.Errorf("PointsAtPages = %v, want %v", got.PointsAtPages, tt.wantPagesLink)
			}
		})
	}
}
