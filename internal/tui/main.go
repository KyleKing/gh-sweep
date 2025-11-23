package tui

import (
	"github.com/KyleKing/gh-sweep/internal/tui/components/analytics"
	"github.com/KyleKing/gh-sweep/internal/tui/components/branches"
	"github.com/KyleKing/gh-sweep/internal/tui/components/collaborators"
	"github.com/KyleKing/gh-sweep/internal/tui/components/comments"
	"github.com/KyleKing/gh-sweep/internal/tui/components/protection"
	"github.com/KyleKing/gh-sweep/internal/tui/components/releases"
	"github.com/KyleKing/gh-sweep/internal/tui/components/secrets"
	"github.com/KyleKing/gh-sweep/internal/tui/components/settings"
	"github.com/KyleKing/gh-sweep/internal/tui/components/webhooks"
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
	ViewSettings
	ViewWebhooks
	ViewCollaborators
	ViewSecrets
	ViewReleases
)

// MainModel represents the main TUI application state with navigation
type MainModel struct {
	width  int
	height int
	ready  bool
	mode   ViewMode

	// Sub-models for each view
	branchesModel      branches.Model
	protectionModel    protection.Model
	commentsModel      comments.Model
	analyticsModel     analytics.Model
	settingsModel      settings.Model
	webhooksModel      webhooks.Model
	collaboratorsModel collaborators.Model
	secretsModel       secrets.Model
	releasesModel      releases.Model

	// Configuration
	repo     string
	repos    []string
	baseline string
	org      string
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
		newModel, _ = m.settingsModel.Update(msg)
		m.settingsModel = newModel.(settings.Model)
		newModel, _ = m.webhooksModel.Update(msg)
		m.webhooksModel = newModel.(webhooks.Model)
		newModel, _ = m.collaboratorsModel.Update(msg)
		m.collaboratorsModel = newModel.(collaborators.Model)
		newModel, _ = m.secretsModel.Update(msg)
		m.secretsModel = newModel.(secrets.Model)
		newModel, _ = m.releasesModel.Update(msg)
		m.releasesModel = newModel.(releases.Model)

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

			case "5":
				m.mode = ViewSettings
				if len(m.repos) > 0 {
					m.settingsModel = settings.NewModel(m.repos, m.baseline)
					return m, m.settingsModel.Init()
				}

			case "6":
				m.mode = ViewWebhooks
				if len(m.repos) > 0 {
					m.webhooksModel = webhooks.NewModel(m.repos)
					return m, m.webhooksModel.Init()
				}

			case "7":
				m.mode = ViewCollaborators
				if len(m.repos) > 0 {
					m.collaboratorsModel = collaborators.NewModel(m.repos)
					return m, m.collaboratorsModel.Init()
				}

			case "8":
				m.mode = ViewSecrets
				if m.org != "" && len(m.repos) > 0 {
					m.secretsModel = secrets.NewModel(m.org, m.repos)
					return m, m.secretsModel.Init()
				}

			case "9":
				m.mode = ViewReleases
				if len(m.repos) > 0 {
					m.releasesModel = releases.NewModel(m.repos)
					return m, m.releasesModel.Init()
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

			case ViewSettings:
				var newModel tea.Model
				newModel, cmd = m.settingsModel.Update(msg)
				m.settingsModel = newModel.(settings.Model)

			case ViewWebhooks:
				var newModel tea.Model
				newModel, cmd = m.webhooksModel.Update(msg)
				m.webhooksModel = newModel.(webhooks.Model)

			case ViewCollaborators:
				var newModel tea.Model
				newModel, cmd = m.collaboratorsModel.Update(msg)
				m.collaboratorsModel = newModel.(collaborators.Model)

			case ViewSecrets:
				var newModel tea.Model
				newModel, cmd = m.secretsModel.Update(msg)
				m.secretsModel = newModel.(secrets.Model)

			case ViewReleases:
				var newModel tea.Model
				newModel, cmd = m.releasesModel.Update(msg)
				m.releasesModel = newModel.(releases.Model)
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
	case ViewSettings:
		return m.settingsModel.View()
	case ViewWebhooks:
		return m.webhooksModel.View()
	case ViewCollaborators:
		return m.collaboratorsModel.View()
	case ViewSecrets:
		return m.secretsModel.View()
	case ViewReleases:
		return m.releasesModel.View()
	default:
		return m.renderHome()
	}
}

func (m MainModel) renderHome() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FFFF")).
		Padding(1, 0)

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFF00")).
		Padding(0, 0)

	menuItemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 2)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#777777"))

	content := titleStyle.Render("🧹 gh-sweep") + "\n"
	content += titleStyle.Render("GitHub Repository Management TUI") + "\n\n"

	// Phase 1: Core Management
	content += sectionStyle.Render("Phase 1: Core Management") + "\n"
	content += menuItemStyle.Render("[1] 🌳 Branch Management")
	content += " - Interactive branch operations\n"
	content += menuItemStyle.Render("[2] 🛡️  Branch Protection")
	content += " - Compare and sync protection rules\n"
	content += menuItemStyle.Render("[3] 💬 PR Comments")
	content += " - Review unresolved comments\n"
	content += menuItemStyle.Render("[4] 📊 Analytics")
	content += " - CI/CD and repository statistics\n\n"

	// Phase 2: Analytics & Settings
	content += sectionStyle.Render("Phase 2: Analytics & Settings") + "\n"
	content += menuItemStyle.Render("[5] ⚙️  Settings Comparison")
	content += " - Cross-repo settings diff\n"
	content += menuItemStyle.Render("[6] 🔔 Webhooks")
	content += " - Webhook health monitoring\n\n"

	// Phase 3: Access & Releases
	content += sectionStyle.Render("Phase 3: Access & Releases") + "\n"
	content += menuItemStyle.Render("[7] 👥 Collaborators")
	content += " - Manage repository access\n"
	content += menuItemStyle.Render("[8] 🔐 Secrets Audit")
	content += " - Review secrets usage (read-only)\n"
	content += menuItemStyle.Render("[9] 📦 Releases")
	content += " - Release version overview\n\n"

	if m.repo == "" && len(m.repos) == 0 {
		content += helpStyle.Render("💡 Configure with --repo flag or .gh-sweep.yaml\n\n")
	}

	content += helpStyle.Render("Press 1-9 to select a view | q to quit")

	return content
}
