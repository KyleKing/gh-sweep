package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/KyleKing/gh-sweep/internal/orphans"
	"github.com/KyleKing/gh-sweep/internal/tui/components/analytics"
	"github.com/KyleKing/gh-sweep/internal/tui/components/branches"
	"github.com/KyleKing/gh-sweep/internal/tui/components/collaborators"
	"github.com/KyleKing/gh-sweep/internal/tui/components/comments"
	"github.com/KyleKing/gh-sweep/internal/tui/components/ghaperf"
	orphanstui "github.com/KyleKing/gh-sweep/internal/tui/components/orphans"
	"github.com/KyleKing/gh-sweep/internal/tui/components/protection"
	"github.com/KyleKing/gh-sweep/internal/tui/components/releases"
	"github.com/KyleKing/gh-sweep/internal/tui/components/secrets"
	"github.com/KyleKing/gh-sweep/internal/tui/components/settings"
	"github.com/KyleKing/gh-sweep/internal/tui/components/watching"
	"github.com/KyleKing/gh-sweep/internal/tui/components/webhooks"
	"github.com/KyleKing/gh-sweep/internal/tui/theme"
)

// ViewMode represents different TUI views.
type ViewMode int

const (
	ViewHome ViewMode = iota
	ViewBranches
	ViewProtection
	ViewComments
	ViewAnalytics
	ViewGHAPerf
	ViewSettings
	ViewWatching
	ViewWebhooks
	ViewCollaborators
	ViewSecrets
	ViewReleases
	ViewOrphans
)

// MainModel represents the main TUI application state with navigation.
type MainModel struct {
	width  int
	height int
	ready  bool
	mode   ViewMode

	// Sub-models for each view
	analyticsModel     analytics.Model
	branchesModel      branches.Model
	collaboratorsModel collaborators.Model
	commentsModel      comments.Model
	ghaPerfModel       ghaperf.Model
	orphansModel       orphanstui.Model
	protectionModel    protection.Model
	releasesModel      releases.Model
	secretsModel       secrets.Model
	settingsModel      settings.Model
	watchingModel      watching.Model
	webhooksModel      webhooks.Model

	// Configuration
	repo     string
	repos    []string
	baseline string
	org      string
}

// MainModelOptions configures the initial state of the main TUI model.
type MainModelOptions struct {
	Baseline string
	Org      string
	Repo     string
	Repos    []string
}

// NewMainModel creates a new main TUI model.
func NewMainModel(opts MainModelOptions) MainModel {
	return MainModel{
		ready:    false,
		mode:     ViewHome,
		repo:     opts.Repo,
		repos:    opts.Repos,
		baseline: opts.Baseline,
		org:      opts.Org,
	}
}

// Init initializes the model.
func (m MainModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model.
func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Forward to sub-models
		m.branchesModel, _ = m.branchesModel.Update(msg)
		m.protectionModel, _ = m.protectionModel.Update(msg)
		m.commentsModel, _ = m.commentsModel.Update(msg)
		m.analyticsModel, _ = m.analyticsModel.Update(msg)
		m.ghaPerfModel, _ = m.ghaPerfModel.Update(msg)
		m.settingsModel, _ = m.settingsModel.Update(msg)
		m.webhooksModel, _ = m.webhooksModel.Update(msg)
		m.collaboratorsModel, _ = m.collaboratorsModel.Update(msg)
		m.secretsModel, _ = m.secretsModel.Update(msg)
		m.releasesModel, _ = m.releasesModel.Update(msg)
		m.watchingModel, _ = m.watchingModel.Update(msg)
		m.orphansModel, _ = m.orphansModel.Update(msg)

		return m, nil

	case tea.KeyPressMsg:
		// Handle navigation in home view
		if m.mode == ViewHome {
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit

			case "0":
				m.mode = ViewWatching
				m.watchingModel = watching.NewModel()

				return m, m.watchingModel.Init()

			case "1":
				m.mode = ViewBranches
				if m.repo != "" {
					m.branchesModel = branches.NewModel(m.repo, "")
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

			case "p":
				m.mode = ViewGHAPerf
				if m.repo != "" {
					m.ghaPerfModel = ghaperf.NewModel(m.repo)
					return m, m.ghaPerfModel.Init()
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

			case "o":
				m.mode = ViewOrphans
				namespace := m.org
				if namespace == "" {
					namespace = ""
				}
				m.orphansModel = orphanstui.NewModel(namespace, orphans.DefaultScanOptions())

				return m, m.orphansModel.Init()
			}
		} else {
			// Handle back navigation
			if msg.String() == "esc" {
				m.mode = ViewHome
				return m, nil
			}

			return m.updateActive(msg)
		}

		return m, nil
	}

	// Forward async messages (e.g. component load results) to the active sub-model.
	if m.mode != ViewHome {
		return m.updateActive(msg)
	}

	return m, nil
}

func (m MainModel) updateActive(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.mode {
	case ViewBranches:
		m.branchesModel, cmd = m.branchesModel.Update(msg)

	case ViewProtection:
		m.protectionModel, cmd = m.protectionModel.Update(msg)

	case ViewComments:
		m.commentsModel, cmd = m.commentsModel.Update(msg)

	case ViewAnalytics:
		m.analyticsModel, cmd = m.analyticsModel.Update(msg)

	case ViewGHAPerf:
		m.ghaPerfModel, cmd = m.ghaPerfModel.Update(msg)

	case ViewSettings:
		m.settingsModel, cmd = m.settingsModel.Update(msg)

	case ViewWebhooks:
		m.webhooksModel, cmd = m.webhooksModel.Update(msg)

	case ViewCollaborators:
		m.collaboratorsModel, cmd = m.collaboratorsModel.Update(msg)

	case ViewSecrets:
		m.secretsModel, cmd = m.secretsModel.Update(msg)

	case ViewReleases:
		m.releasesModel, cmd = m.releasesModel.Update(msg)

	case ViewWatching:
		m.watchingModel, cmd = m.watchingModel.Update(msg)

	case ViewOrphans:
		m.orphansModel, cmd = m.orphansModel.Update(msg)
	}

	return m, cmd
}

// View renders the model.
func (m MainModel) View() tea.View {
	v := tea.NewView(m.renderContent())
	v.AltScreen = true

	return v
}

func (m MainModel) renderContent() string {
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
	case ViewGHAPerf:
		return m.ghaPerfModel.View()
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
	case ViewWatching:
		return m.watchingModel.View()
	case ViewOrphans:
		return m.orphansModel.View()
	default:
		return m.renderHome()
	}
}

func (m MainModel) renderHome() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Primary).
		Padding(1, 0)

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Warning).
		Padding(0, 0)

	menuItemStyle := lipgloss.NewStyle().
		Foreground(theme.Current().Text).
		Padding(0, 2)

	helpStyle := lipgloss.NewStyle().
		Foreground(theme.Current().Muted)

	content := titleStyle.Render("🧹 gh-sweep") + "\n"
	content += titleStyle.Render("GitHub Repository Management TUI") + "\n\n"

	// Namespace Audit
	content += sectionStyle.Render("Namespace Audit") + "\n"
	content += menuItemStyle.Render("[0] 👁️  Watch Status")
	content += " - Audit and manage repo watching\n"
	content += menuItemStyle.Render("[o] 🌿 Orphan Branches")
	content += " - Detect and clean up orphaned branches\n\n"

	// Phase 1: Core Management
	content += sectionStyle.Render("Phase 1: Core Management") + "\n"
	content += menuItemStyle.Render("[1] 🌳 Branch Management")
	content += " - Interactive branch operations\n"
	content += menuItemStyle.Render("[2] 🛡️  Branch Protection")
	content += " - Compare and sync protection rules\n"
	content += menuItemStyle.Render("[3] 💬 PR Comments")
	content += " - Review unresolved comments\n"
	content += menuItemStyle.Render("[4] 📊 Analytics")
	content += " - CI/CD and repository statistics\n"
	content += menuItemStyle.Render("[p] ⏱️  GHA Performance")
	content += " - Workflow timing analysis\n\n"

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

	content += helpStyle.Render("Press 0-9/o/p to select a view | q to quit")

	return content
}
