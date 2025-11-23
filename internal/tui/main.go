package tui

import (
	"github.com/KyleKing/gh-sweep/internal/tui/components/analytics"
	"github.com/KyleKing/gh-sweep/internal/tui/components/branches"
	"github.com/KyleKing/gh-sweep/internal/tui/components/comments"
	"github.com/KyleKing/gh-sweep/internal/tui/components/protection"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ViewMode represents different TUI views
type ViewMode int

const (
	ViewHome ViewMode = iota
	ViewBranches
	ViewProtection
	ViewComments
	ViewAnalytics
)

// MainModel represents the main TUI application state with navigation
type MainModel struct {
	width  int
	height int
	ready  bool
	mode   ViewMode

	// Sub-models for each view
	branchesModel   branches.Model
	protectionModel protection.Model
	commentsModel   comments.Model
	analyticsModel  analytics.Model

	// Configuration
	repo     string
	repos    []string
	baseline string
}

// NewMainModel creates a new main TUI model
func NewMainModel(repo string) MainModel {
	return MainModel{
		ready: false,
		mode:  ViewHome,
		repo:  repo,
	}
}

// Init initializes the model
func (m MainModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Forward to sub-models
		var newModel tea.Model
		newModel, _ = m.branchesModel.Update(msg)
		m.branchesModel = newModel.(branches.Model)
		newModel, _ = m.protectionModel.Update(msg)
		m.protectionModel = newModel.(protection.Model)
		newModel, _ = m.commentsModel.Update(msg)
		m.commentsModel = newModel.(comments.Model)
		newModel, _ = m.analyticsModel.Update(msg)
		m.analyticsModel = newModel.(analytics.Model)

		return m, nil

	case tea.KeyMsg:
		// Handle navigation in home view
		if m.mode == ViewHome {
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit

			case "1":
				m.mode = ViewBranches
				if m.repo != "" {
					m.branchesModel = branches.NewModel(m.repo, "main")
					return m, m.branchesModel.Init()
				}

			case "2":
				m.mode = ViewProtection
				if len(m.repos) > 0 {
					m.protectionModel = protection.NewModel(m.repos, m.baseline)
					return m, m.protectionModel.Init()
				}

			case "3":
				m.mode = ViewComments
				if m.repo != "" {
					m.commentsModel = comments.NewModel(m.repo)
					return m, m.commentsModel.Init()
				}

			case "4":
				m.mode = ViewAnalytics
				if m.repo != "" {
					m.analyticsModel = analytics.NewModel(m.repo)
					return m, m.analyticsModel.Init()
				}
			}
		} else {
			// Handle back navigation
			if msg.String() == "esc" {
				m.mode = ViewHome
				return m, nil
			}

			// Forward to active sub-model
			var cmd tea.Cmd
			switch m.mode {
			case ViewBranches:
				var newModel tea.Model
				newModel, cmd = m.branchesModel.Update(msg)
				m.branchesModel = newModel.(branches.Model)

			case ViewProtection:
				var newModel tea.Model
				newModel, cmd = m.protectionModel.Update(msg)
				m.protectionModel = newModel.(protection.Model)

			case ViewComments:
				var newModel tea.Model
				newModel, cmd = m.commentsModel.Update(msg)
				m.commentsModel = newModel.(comments.Model)

			case ViewAnalytics:
				var newModel tea.Model
				newModel, cmd = m.analyticsModel.Update(msg)
				m.analyticsModel = newModel.(analytics.Model)
			}

			return m, cmd
		}
	}

	return m, nil
}

// View renders the model
func (m MainModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Render active view
	switch m.mode {
	case ViewBranches:
		return m.branchesModel.View()
	case ViewProtection:
		return m.protectionModel.View()
	case ViewComments:
		return m.commentsModel.View()
	case ViewAnalytics:
		return m.analyticsModel.View()
	default:
		return m.renderHome()
	}
}

func (m MainModel) renderHome() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FFFF")).
		Padding(1, 0)

	menuItemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 2)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#777777"))

	content := titleStyle.Render("🧹 gh-sweep") + "\n"
	content += titleStyle.Render("GitHub Repository Management TUI") + "\n\n"

	content += menuItemStyle.Render("[1] 🌳 Branch Management")
	content += " - Interactive branch operations\n"

	content += menuItemStyle.Render("[2] 🛡️  Branch Protection")
	content += " - Compare and sync protection rules\n"

	content += menuItemStyle.Render("[3] 💬 PR Comments")
	content += " - Review unresolved comments\n"

	content += menuItemStyle.Render("[4] 📊 Analytics")
	content += " - CI/CD and repository statistics\n\n"

	if m.repo == "" {
		content += helpStyle.Render("💡 Configure a repository with --repo flag\n")
		content += helpStyle.Render("   or set repositories in .gh-sweep.yaml\n\n")
	}

	content += helpStyle.Render("Press 1-4 to select a view | q to quit")

	return content
}
