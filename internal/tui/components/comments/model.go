package comments

import (
	"context"
	"fmt"
	"strings"

	"github.com/KyleKing/gh-sweep/internal/github"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the comments review TUI state
type Model struct {
	repo         string
	comments     []github.Comment
	unresolved   []github.Comment
	cursor       int
	width        int
	height       int
	loading      bool
	err          error
	filterAuthor string
	showResolved bool
}

// NewModel creates a new comments model
func NewModel(repo string) Model {
	return Model{
		repo:         repo,
		loading:      true,
		showResolved: false,
	}
}

type commentsLoadedMsg struct {
	comments   []github.Comment
	unresolved []github.Comment
	err        error
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return m.loadComments
}

func (m Model) loadComments() tea.Msg {
	// If no repo specified, return empty
	if m.repo == "" {
		return commentsLoadedMsg{
			comments:   []github.Comment{},
			unresolved: []github.Comment{},
			err:        fmt.Errorf("no repository specified"),
		}
	}

	// Parse repo (owner/name format)
	parts := strings.Split(m.repo, "/")
	if len(parts) != 2 {
		return commentsLoadedMsg{
			comments:   []github.Comment{},
			unresolved: []github.Comment{},
			err:        fmt.Errorf("invalid repo format, expected owner/repo"),
		}
	}
	owner, repo := parts[0], parts[1]

	// Create GitHub client
	ctx := context.Background()
	client, err := github.NewClient(ctx)
	if err != nil {
		return commentsLoadedMsg{
			comments:   []github.Comment{},
			unresolved: []github.Comment{},
			err:        fmt.Errorf("failed to create GitHub client: %w", err),
		}
	}

	// Load comments from GitHub
	// Note: ListPRComments loads comments for a specific PR
	// For now, we'll load from a recent PR (this is a simplification)
	// In a real implementation, you'd want to iterate through recent PRs
	comments, err := client.ListPRComments(owner, repo, 1) // PR #1 as example
	if err != nil {
		// Return empty on error (repo might not have PR #1)
		return commentsLoadedMsg{
			comments:   []github.Comment{},
			unresolved: []github.Comment{},
			err:        nil, // Don't error out, just show empty
		}
	}

	// Filter unresolved
	unresolved := github.FilterUnresolvedComments(comments)

	return commentsLoadedMsg{
		comments:   comments,
		unresolved: unresolved,
		err:        nil,
	}
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case commentsLoadedMsg:
		m.loading = false
		m.comments = msg.comments
		m.unresolved = msg.unresolved
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			activeList := m.getActiveList()
			if m.cursor < len(activeList)-1 {
				m.cursor++
			}

		case "r":
			m.showResolved = !m.showResolved
			m.cursor = 0
		}
	}

	return m, nil
}

func (m Model) getActiveList() []github.Comment {
	if m.showResolved {
		return m.comments
	}
	return m.unresolved
}

// View renders the model
func (m Model) View() string {
	if m.loading {
		return "Loading comments...\n"
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var b strings.Builder

	// Header
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FFFF"))

	b.WriteString(titleStyle.Render(fmt.Sprintf("💬 PR Comments: %s", m.repo)))
	b.WriteString("\n\n")

	// Filter status
	if m.showResolved {
		b.WriteString("Showing: All comments\n")
	} else {
		b.WriteString("Showing: Unresolved only\n")
	}
	b.WriteString(fmt.Sprintf("Total: %d | Unresolved: %d\n\n", len(m.comments), len(m.unresolved)))

	// Comment list
	activeList := m.getActiveList()
	if len(activeList) == 0 {
		b.WriteString("No comments found.\n")
	} else {
		for i, comment := range activeList {
			if i >= m.height-10 { // Limit visible items
				break
			}

			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}

			commentStyle := lipgloss.NewStyle()
			if m.cursor == i {
				commentStyle = commentStyle.Bold(true).Foreground(lipgloss.Color("#FFFF00"))
			}

			// Truncate body if too long
			body := comment.Body
			if len(body) > 60 {
				body = body[:60] + "..."
			}

			line := fmt.Sprintf("%s PR#%d @%s\n", cursor, comment.PRNumber, comment.Author)
			line += fmt.Sprintf("  %s:%d\n", comment.Path, comment.Line)
			line += fmt.Sprintf("  %s\n", body)

			b.WriteString(commentStyle.Render(line))
			b.WriteString("\n")
		}
	}

	// Help
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))
	b.WriteString(helpStyle.Render("↑/↓: navigate | r: toggle resolved | q: quit"))

	return b.String()
}
