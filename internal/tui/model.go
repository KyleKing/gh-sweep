package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the main TUI application state
type Model struct {
	width  int
	height int
	ready  bool
}

// NewModel creates a new TUI model
func NewModel() Model {
	return Model{
		ready: false,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	return m, nil
}

// View renders the model
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FFFF")).
		Padding(1, 0)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#777777"))

	content := titleStyle.Render("🧹 gh-sweep") + "\n\n"
	content += "Welcome to gh-sweep - GitHub Repository Management TUI\n\n"
	content += "Available features:\n"
	content += "  • Branch management (coming soon)\n"
	content += "  • Branch protection rules (coming soon)\n"
	content += "  • Unresolved PR comments (coming soon)\n\n"
	content += helpStyle.Render("Press 'q' or Ctrl+C to quit")

	return content
}
