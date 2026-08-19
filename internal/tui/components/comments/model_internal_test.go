package comments

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
)

var errRateLimited = errors.New("rate limited")

func threadsFixture() []github.ReviewThread {
	return []github.ReviewThread{
		{
			Repository: "acme/widgets",
			PRNumber:   42,
			PRTitle:    "Add login flow",
			Path:       "internal/auth/session.go",
			Comments: []github.ReviewComment{
				{
					Author:    "alice",
					Body:      "Extract a helper here.",
					CreatedAt: time.Now().Add(-3 * time.Hour),
				},
			},
		},
		{
			Repository: "acme/widgets",
			PRNumber:   42,
			PRTitle:    "Add login flow",
			Path:       "internal/auth/token.go",
			IsResolved: true,
			Comments: []github.ReviewComment{
				{Author: "bob", Body: "Nit: rename.", CreatedAt: time.Now().Add(-49 * time.Hour)},
			},
		},
	}
}

func TestLoadedThreadsView(t *testing.T) {
	t.Parallel()

	m := NewModel("acme/widgets")
	m, _ = m.Update(threadsLoadedMsg{threads: threadsFixture()})

	view := m.View()
	for _, want := range []string{"PR Review Threads: acme/widgets", "Total: 2 | Unresolved: 1", "@alice"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q", want)
		}
	}

	if strings.Contains(view, "token.go") {
		t.Error("resolved thread shown in unresolved-only view")
	}
}

func TestToggleResolvedThreads(t *testing.T) {
	t.Parallel()

	m := NewModel("acme/widgets")
	m, _ = m.Update(threadsLoadedMsg{threads: threadsFixture()})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})

	if !m.showResolved {
		t.Fatal("expected showResolved after r")
	}

	if !strings.Contains(m.View(), "token.go") {
		t.Errorf("resolved thread missing after toggle: %q", m.View())
	}
}

func TestJumpTopBottom(t *testing.T) {
	t.Parallel()

	m := NewModel("acme/widgets")
	m, _ = m.Update(threadsLoadedMsg{threads: threadsFixture()})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})

	m, _ = m.Update(tea.KeyPressMsg{Code: 'G', Text: "G"})
	if want := len(m.visibleThreads()) - 1; m.cursor != want {
		t.Errorf("cursor after G = %d, want %d", m.cursor, want)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	if m.cursor != 0 {
		t.Errorf("cursor after g = %d, want 0", m.cursor)
	}
}

func TestHelpToggle(t *testing.T) {
	t.Parallel()

	m := NewModel("acme/widgets")
	m, _ = m.Update(threadsLoadedMsg{threads: threadsFixture()})

	m, _ = m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	if !m.showHelp {
		t.Fatal("expected showHelp true after ?")
	}
	if !strings.Contains(m.View(), "Keybindings") {
		t.Error("help view missing Keybindings title")
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.showHelp {
		t.Error("expected showHelp false after esc")
	}
}

func TestThreadsLoadError(t *testing.T) {
	t.Parallel()

	m := NewModel("acme/widgets")
	m, _ = m.Update(threadsLoadedMsg{err: errRateLimited})

	if !strings.Contains(m.View(), "rate limited") {
		t.Errorf("view = %q", m.View())
	}
}

func TestFormatAge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		offset time.Duration
		want   string
	}{
		{30 * time.Minute, "30m"},
		{3 * time.Hour, "3h"},
		{49 * time.Hour, "2d"},
		{45 * 24 * time.Hour, "1mo"},
	}

	for _, tt := range tests {
		if got := formatAge(time.Now().Add(-tt.offset)); got != tt.want {
			t.Errorf("formatAge(-%v) = %q, want %q", tt.offset, got, tt.want)
		}
	}
}

func TestExcerpt(t *testing.T) {
	t.Parallel()

	if got := excerpt("one  two\nthree", 100); got != "one two three" {
		t.Errorf("excerpt() = %q", got)
	}

	if got := excerpt("abcdefghij", 4); got != "abcd..." {
		t.Errorf("excerpt() truncated = %q", got)
	}
}
