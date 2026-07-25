package comments

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/tui/theme"
)

// Model represents the comments review TUI state.
type Model struct {
	repo         string
	prNumber     int
	filter       github.ThreadFilter
	threads      []github.ReviewThread
	unresolved   []github.ReviewThread
	cursor       int
	width        int
	height       int
	loading      bool
	err          error
	showResolved bool
}

// NewModel creates a comments model scanning all open PRs of the repo.
func NewModel(repo string) Model {
	return NewModelWithOptions(repo, 0, github.ThreadFilter{})
}

// NewModelWithOptions creates a comments model for a single PR (prNumber > 0) with filters applied.
func NewModelWithOptions(repo string, prNumber int, filter github.ThreadFilter) Model {
	return Model{
		repo:     repo,
		prNumber: prNumber,
		filter:   filter,
		loading:  true,
	}
}

type threadsLoadedMsg struct {
	threads []github.ReviewThread
	err     error
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return m.loadThreads
}

func (m Model) loadThreads() tea.Msg {
	if m.repo == "" {
		return threadsLoadedMsg{err: errors.New("no repository specified")}
	}

	parts := strings.Split(m.repo, "/")
	if len(parts) != 2 {
		return threadsLoadedMsg{err: errors.New("invalid repo format, expected owner/repo")}
	}
	owner, repo := parts[0], parts[1]

	client, err := github.NewClient(context.Background())
	if err != nil {
		return threadsLoadedMsg{err: fmt.Errorf("failed to create GitHub client: %w", err)}
	}

	gql, err := github.NewGQLClient()
	if err != nil {
		return threadsLoadedMsg{err: err}
	}

	threads, err := gql.ListRepoReviewThreads(client, owner, repo, m.prNumber, github.DefaultOpenPRCap)
	if err != nil {
		return threadsLoadedMsg{err: err}
	}

	return threadsLoadedMsg{threads: threads}
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		return m, nil

	case threadsLoadedMsg:
		m.loading = false
		m.threads = msg.threads
		m.unresolved = github.FilterUnresolvedThreads(msg.threads)
		m.err = msg.err

		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.visibleThreads())-1 {
				m.cursor++
			}

		case "r":
			m.showResolved = !m.showResolved
			m.cursor = 0
		}
	}

	return m, nil
}

func (m Model) visibleThreads() []github.ReviewThread {
	if m.showResolved {
		return m.filter.Apply(m.threads)
	}

	return m.filter.Apply(m.unresolved)
}

// View renders the model.
func (m Model) View() string {
	if m.loading {
		return "Loading review threads...\n"
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Primary)

	b.WriteString(titleStyle.Render("💬 PR Review Threads: " + m.repo))
	b.WriteString("\n\n")

	if m.showResolved {
		b.WriteString("Showing: All threads\n")
	} else {
		b.WriteString("Showing: Unresolved only\n")
	}
	b.WriteString(fmt.Sprintf("Total: %d | Unresolved: %d\n\n", len(m.threads), len(m.unresolved)))

	visible := m.visibleThreads()
	if len(visible) == 0 {
		b.WriteString("No review threads found.\n")
	} else {
		m.renderThreads(&b, visible)
	}

	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	b.WriteString(helpStyle.Render("↑/↓: navigate | r: toggle resolved | q: quit"))

	return b.String()
}

func (m Model) renderThreads(b *strings.Builder, visible []github.ReviewThread) {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Secondary)

	lastPR := 0
	rendered := 0
	for i, thread := range visible {
		if rendered >= m.height-10 && m.height > 0 {
			break
		}

		if thread.PRNumber != lastPR {
			b.WriteString(headerStyle.Render(fmt.Sprintf("PR #%d: %s", thread.PRNumber, thread.PRTitle)))
			b.WriteString("\n")
			lastPR = thread.PRNumber
		}

		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		threadStyle := lipgloss.NewStyle()
		if m.cursor == i {
			threadStyle = threadStyle.Bold(true).Foreground(theme.Current().Warning)
		}

		b.WriteString(threadStyle.Render(renderThread(cursor, thread)))
		b.WriteString("\n")
		rendered++
	}
}

func renderThread(cursor string, thread github.ReviewThread) string {
	outdated := ""
	if thread.IsOutdated {
		outdated = " [outdated]"
	}

	author, age, body := "unknown", "", ""
	if first, ok := thread.FirstComment(); ok {
		author = first.Author
		age = formatAge(first.CreatedAt)
		body = excerpt(first.Body, 60)
	}

	line := fmt.Sprintf("%s %s%s\n", cursor, thread.Path, outdated)
	line += fmt.Sprintf("  @%s · %s\n", author, age)
	line += fmt.Sprintf("  %s\n", body)

	return line
}

func excerpt(body string, limit int) string {
	flat := strings.Join(strings.Fields(body), " ")
	if len(flat) > limit {
		return flat[:limit] + "..."
	}

	return flat
}

func formatAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%dmo", int(d.Hours()/(24*30)))
	}
}
