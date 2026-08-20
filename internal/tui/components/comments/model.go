// Package comments is the TUI view for reviewing PR review threads across a
// repository, filterable to unresolved threads only.
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
	"github.com/KyleKing/gh-sweep/internal/tui/scroll"
	"github.com/KyleKing/gh-sweep/internal/tui/theme"
)

const (
	repoPartsCount = 2
	helpKeyWidth   = 16
	excerptLimit   = 60
	hoursPerDay    = 24
	daysPerMonth   = 30
)

// ErrNoRepository means the model was constructed without a repository to load threads from.
var ErrNoRepository = errors.New("no repository specified")

// ErrInvalidRepoFormat means the repository string was not in owner/repo form.
var ErrInvalidRepoFormat = errors.New("invalid repo format, expected owner/repo")

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
	showHelp     bool
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
		return threadsLoadedMsg{err: ErrNoRepository}
	}

	owner, repo, ok := splitRepo(m.repo)
	if !ok {
		return threadsLoadedMsg{err: ErrInvalidRepoFormat}
	}

	client, err := github.NewClient(context.Background())
	if err != nil {
		return threadsLoadedMsg{err: fmt.Errorf("failed to create GitHub client: %w", err)}
	}

	gql, err := github.NewGQLClient()
	if err != nil {
		return threadsLoadedMsg{err: err}
	}

	threads, err := gql.ListRepoReviewThreads(
		client,
		owner,
		repo,
		m.prNumber,
		github.DefaultOpenPRCap,
	)
	if err != nil {
		return threadsLoadedMsg{err: err}
	}

	return threadsLoadedMsg{threads: threads}
}

func splitRepo(repo string) (string, string, bool) {
	parts := strings.SplitN(repo, "/", repoPartsCount)
	if len(parts) != repoPartsCount {
		return "", "", false
	}

	return parts[0], parts[1], true
}

// Update handles messages.
//
//nolint:unparam // matches every TUI component's Update(Model, tea.Cmd) shape
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
		return m.updateKeyMsg(msg)
	}

	return m, nil
}

func (m Model) updateKeyMsg(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	if m.showHelp {
		if msg.String() == "?" || msg.String() == "esc" {
			m.showHelp = false
		}

		return m, nil
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "?":
		m.showHelp = true

	case "g":
		m.cursor = 0

	case "G":
		if visible := m.visibleThreads(); len(visible) > 0 {
			m.cursor = len(visible) - 1
		}

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

	var header strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Primary)

	header.WriteString(titleStyle.Render("💬 PR Review Threads: " + m.repo))
	header.WriteString("\n\n")

	if m.showResolved {
		header.WriteString("Showing: All threads\n")
	} else {
		header.WriteString("Showing: Unresolved only\n")
	}
	fmt.Fprintf(&header, "Total: %d | Unresolved: %d\n\n", len(m.threads), len(m.unresolved))

	if m.showHelp {
		return renderHelp(&header)
	}

	var footer strings.Builder
	footer.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	footer.WriteString(helpStyle.Render("↑/↓: navigate | r: toggle resolved | ?: help | q: quit"))

	var b strings.Builder
	b.WriteString(header.String())

	visible := m.visibleThreads()
	if len(visible) == 0 {
		b.WriteString("No review threads found.\n")
	} else {
		headerLines := strings.Count(header.String(), "\n")
		footerLines := strings.Count(footer.String(), "\n")
		available := m.height - headerLines - footerLines
		m.renderThreads(&b, visible, available)
	}

	b.WriteString(footer.String())

	return b.String()
}

func renderHelp(b *strings.Builder) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Primary)
	b.WriteString(titleStyle.Render("Keybindings"))
	b.WriteString("\n\n")

	bindings := [][2]string{
		{"j/k, up/down", "move the cursor"},
		{"g / G", "jump to top / bottom"},
		{"r", "toggle resolved threads"},
		{"?", "toggle this help"},
		{"q", "quit"},
	}

	for _, binding := range bindings {
		keyStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Warning).Width(helpKeyWidth)
		fmt.Fprintf(b, "%s %s\n", keyStyle.Render(binding[0]), binding[1])
	}

	b.WriteString("\nPress '?' or 'esc' to close\n")

	return b.String()
}

func (m Model) renderThreads(b *strings.Builder, visible []github.ReviewThread, available int) {
	lines, cursorLine := m.buildThreadLines(visible)

	start, end := scroll.Window(len(lines), cursorLine, available)

	scrollHintStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	if start > 0 {
		fmt.Fprintf(b, "%s\n", scrollHintStyle.Render(fmt.Sprintf("↑ %d more above", start)))
	}

	b.WriteString(strings.Join(lines[start:end], "\n"))
	b.WriteString("\n")

	if end < len(lines) {
		fmt.Fprintf(b, "%s\n", scrollHintStyle.Render(fmt.Sprintf("↓ %d more below", len(lines)-end)))
	}
}

// buildThreadLines renders each thread, and its PR group header when the PR
// changes, as one line each so the caller can window by line rather than by
// item, returning the lines and the cursor row's index among them.
func (m Model) buildThreadLines(visible []github.ReviewThread) ([]string, int) {
	var lines []string

	cursorLine := 0
	lastPR := 0
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Secondary)

	for i, thread := range visible {
		if thread.PRNumber != lastPR {
			lines = append(lines, headerStyle.Render(fmt.Sprintf("PR #%d: %s", thread.PRNumber, thread.PRTitle)))
			lastPR = thread.PRNumber
		}

		if i == m.cursor {
			cursorLine = len(lines)
		}

		lines = append(lines, m.renderThreadLines(thread, i)...)
	}

	return lines, cursorLine
}

func (m Model) renderThreadLines(thread github.ReviewThread, i int) []string {
	cursor := " "
	if m.cursor == i {
		cursor = ">"
	}

	threadStyle := lipgloss.NewStyle()
	if m.cursor == i {
		threadStyle = threadStyle.Bold(true).Foreground(theme.Current().Warning)
	}

	rendered := threadStyle.Render(renderThread(cursor, thread))

	return strings.Split(rendered, "\n")
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
		body = excerpt(first.Body, excerptLimit)
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
	case d < hoursPerDay*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < daysPerMonth*hoursPerDay*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/hoursPerDay))
	default:
		return fmt.Sprintf("%dmo", int(d.Hours()/(hoursPerDay*daysPerMonth)))
	}
}
