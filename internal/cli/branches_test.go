package cli

import (
	"testing"
	"time"

	"github.com/KyleKing/gh-sweep/internal/github"
)

func TestFormatCommitAge(t *testing.T) {
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		commit time.Time
		want   string
	}{
		{name: "zero time", commit: time.Time{}, want: "-"},
		{name: "same day", commit: now.Add(-2 * time.Hour), want: "<1d"},
		{name: "days", commit: now.AddDate(0, 0, -3), want: "3d"},
		{name: "just under a year", commit: now.AddDate(0, 0, -364), want: "364d"},
		{name: "years", commit: now.AddDate(-2, 0, -1), want: "2y"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatCommitAge(tt.commit, now); got != tt.want {
				t.Errorf("formatCommitAge() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatBranchPR(t *testing.T) {
	tests := []struct {
		name string
		pr   *github.PullRequest
		want string
	}{
		{name: "no PR", pr: nil, want: "-"},
		{name: "open PR", pr: &github.PullRequest{Number: 12, State: "open"}, want: "#12 (open)"},
		{
			name: "closed PR",
			pr:   &github.PullRequest{Number: 4, State: "closed"},
			want: "#4 (closed)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatBranchPR(tt.pr); got != tt.want {
				t.Errorf("formatBranchPR() = %q, want %q", got, tt.want)
			}
		})
	}
}
