package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/KyleKing/gh-sweep/internal/tui/theme"
)

// Model represents the main TUI application state.
type Model struct {
	width  int
	height int
	ready  bool
}

// NewModel creates a new TUI model.
func NewModel() Model {
	return Model{
		ready: false,
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	return m, nil
}

// View renders the model.
func (m Model) View() tea.View {
	return tea.NewView(m.renderContent())
}

func (m Model) renderContent() string {
	if !m.ready {
		return "Initializing..."
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Primary).
		Padding(1, 0)

	helpStyle := lipgloss.NewStyle().
		Foreground(theme.Current().Muted)

	content := titleStyle.Render("🧹 gh-sweep") + "\n\n"
	content += "Welcome to gh-sweep - GitHub Repository Management TUI\n\n"
	content += "Available features:\n"
	content += "  • Branch management (coming soon)\n"
	content += "  • Branch protection rules (coming soon)\n"
	content += "  • Unresolved PR comments (coming soon)\n\n"
	content += helpStyle.Render("Press 'q' or Ctrl+C to quit")

	return content
}
