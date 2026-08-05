package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/KyleKing/gh-sweep/internal/config"
	"github.com/KyleKing/gh-sweep/internal/orphans"
	"github.com/KyleKing/gh-sweep/internal/tui/components/analytics"
	"github.com/KyleKing/gh-sweep/internal/tui/components/branches"
	"github.com/KyleKing/gh-sweep/internal/tui/components/collaborators"
	"github.com/KyleKing/gh-sweep/internal/tui/components/comments"
	"github.com/KyleKing/gh-sweep/internal/tui/components/ghaperf"
	orphanstui "github.com/KyleKing/gh-sweep/internal/tui/components/orphans"
	policytui "github.com/KyleKing/gh-sweep/internal/tui/components/policy"
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
	ViewPolicy
)

// menuItem describes one selectable entry in the home menu list.
type menuItem struct {
	key     string
	icon    string
	label   string
	desc    string
	section string
	view    ViewMode
	enabled bool
}

// MainModel represents the main TUI application state with navigation.
type MainModel struct {
	width  int
	height int
	ready  bool
	mode   ViewMode

	// Home menu list state
	menuCursor    int
	menuFilter    string
	menuFiltering bool

	// Sub-models for each view
	analyticsModel     analytics.Model
	branchesModel      branches.Model
	collaboratorsModel collaborators.Model
	commentsModel      comments.Model
	ghaPerfModel       ghaperf.Model
	orphansModel       orphanstui.Model
	policyModel        policytui.Model
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
		if m.mode == ViewHome {
			return m.updateHome(msg)
		}

		// Handle back navigation
		if msg.String() == "esc" {
			m.mode = ViewHome
			return m, nil
		}

		return m.updateActive(msg)
	}

	// Forward async messages (e.g. component load results) to the active sub-model.
	if m.mode != ViewHome {
		return m.updateActive(msg)
	}

	return m, nil
}

// buildMenuItems returns the full home menu, marking items whose repo/org
// preconditions aren't met as disabled.
func (m MainModel) buildMenuItems() []menuItem {
	hasRepo := m.repo != ""
	hasRepos := len(m.repos) > 0
	hasOrgAndRepos := m.org != "" && hasRepos

	return []menuItem{
		{
			key:     "0",
			icon:    "👁️ ",
			label:   "Watch Status",
			desc:    "Audit and manage repo watching",
			section: "Namespace Audit",
			view:    ViewWatching,
			enabled: true,
		},
		{
			key:     "o",
			icon:    "🌿",
			label:   "Orphan Branches",
			desc:    "Detect and clean up orphaned branches",
			section: "Namespace Audit",
			view:    ViewOrphans,
			enabled: true,
		},
		{
			key:     "1",
			icon:    "🌳",
			label:   "Branch Management",
			desc:    "Interactive branch operations",
			section: "Phase 1: Core Management",
			view:    ViewBranches,
			enabled: hasRepo,
		},
		{
			key:     "2",
			icon:    "🛡️ ",
			label:   "Branch Protection",
			desc:    "Compare and sync protection rules",
			section: "Phase 1: Core Management",
			view:    ViewProtection,
			enabled: hasRepos,
		},
		{
			key:     "3",
			icon:    "💬",
			label:   "PR Comments",
			desc:    "Review unresolved comments",
			section: "Phase 1: Core Management",
			view:    ViewComments,
			enabled: hasRepo,
		},
		{
			key:     "4",
			icon:    "📊",
			label:   "Analytics",
			desc:    "CI/CD and repository statistics",
			section: "Phase 1: Core Management",
			view:    ViewAnalytics,
			enabled: hasRepo,
		},
		{
			key:     "p",
			icon:    "⏱️ ",
			label:   "GHA Performance",
			desc:    "Workflow timing analysis",
			section: "Phase 1: Core Management",
			view:    ViewGHAPerf,
			enabled: hasRepo,
		},
		{
			key:     "5",
			icon:    "⚙️ ",
			label:   "Settings Comparison",
			desc:    "Cross-repo settings diff",
			section: "Phase 2: Analytics & Settings",
			view:    ViewSettings,
			enabled: hasRepos,
		},
		{
			key:     "6",
			icon:    "🔔",
			label:   "Webhooks",
			desc:    "Webhook health monitoring",
			section: "Phase 2: Analytics & Settings",
			view:    ViewWebhooks,
			enabled: hasRepos,
		},
		{
			key:     "7",
			icon:    "👥",
			label:   "Collaborators",
			desc:    "Manage repository access",
			section: "Phase 3: Access & Releases",
			view:    ViewCollaborators,
			enabled: hasRepos,
		},
		{
			key:     "8",
			icon:    "🔐",
			label:   "Secrets Audit",
			desc:    "Review secrets usage (read-only)",
			section: "Phase 3: Access & Releases",
			view:    ViewSecrets,
			enabled: hasOrgAndRepos,
		},
		{
			key:     "9",
			icon:    "📦",
			label:   "Releases",
			desc:    "Release version overview",
			section: "Phase 3: Access & Releases",
			view:    ViewReleases,
			enabled: hasRepos,
		},
		{
			key:     "y",
			icon:    "📐",
			label:   "Policy",
			desc:    "Diff and sync settings against a policy file",
			section: "Phase 3: Access & Releases",
			view:    ViewPolicy,
			enabled: true,
		},
	}
}

// fuzzyMatch reports whether query's characters appear in target, in order,
// case-insensitively (a subsequence match, as in fzf-style pickers).
func fuzzyMatch(query, target string) bool {
	if query == "" {
		return true
	}

	query = strings.ToLower(query)
	target = strings.ToLower(target)

	qi := 0
	for i := 0; i < len(target) && qi < len(query); i++ {
		if target[i] == query[qi] {
			qi++
		}
	}

	return qi == len(query)
}

func (m MainModel) filteredMenuItems() []menuItem {
	items := m.buildMenuItems()
	if m.menuFilter == "" {
		return items
	}

	filtered := make([]menuItem, 0, len(items))
	for _, item := range items {
		if fuzzyMatch(m.menuFilter, item.label) || fuzzyMatch(m.menuFilter, item.key) {
			filtered = append(filtered, item)
		}
	}

	return filtered
}

func (m *MainModel) moveMenuCursor(delta int) {
	items := m.filteredMenuItems()
	if len(items) == 0 {
		m.menuCursor = 0
		return
	}

	m.menuCursor += delta

	if m.menuCursor < 0 {
		m.menuCursor = 0
	}
	if m.menuCursor > len(items)-1 {
		m.menuCursor = len(items) - 1
	}
}

func (m MainModel) menuItemIndexForKey(key string) int {
	items := m.filteredMenuItems()
	for i, item := range items {
		if item.key == key {
			return i
		}
	}

	return -1
}

// updateHome handles key presses while the home menu list is shown, both in
// normal navigation mode (j/k, arrows, direct shortcuts) and while typing a
// fuzzy filter (entered with "/").
func (m MainModel) updateHome(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if m.menuFiltering {
		switch key {
		case "ctrl+c":
			return m, tea.Quit

		case "esc":
			m.menuFiltering = false
			m.menuFilter = ""
			m.menuCursor = 0

		case "enter":
			m.menuFiltering = false
			return m.activateSelected()

		case "backspace":
			if len(m.menuFilter) > 0 {
				runes := []rune(m.menuFilter)
				m.menuFilter = string(runes[:len(runes)-1])
				m.menuCursor = 0
			}

		case "up", "ctrl+p":
			m.moveMenuCursor(-1)

		case "down", "ctrl+n":
			m.moveMenuCursor(1)

		default:
			if runes := []rune(key); len(runes) == 1 {
				m.menuFilter += key
				m.menuCursor = 0
			}
		}

		return m, nil
	}

	switch key {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "/":
		m.menuFiltering = true
		m.menuFilter = ""
		m.menuCursor = 0

	case "up", "k":
		m.moveMenuCursor(-1)

	case "down", "j":
		m.moveMenuCursor(1)

	case "enter":
		return m.activateSelected()

	default:
		if idx := m.menuItemIndexForKey(key); idx >= 0 {
			m.menuCursor = idx
			return m.activateSelected()
		}
	}

	return m, nil
}

func (m MainModel) activateSelected() (tea.Model, tea.Cmd) {
	items := m.filteredMenuItems()
	if m.menuCursor < 0 || m.menuCursor >= len(items) {
		return m, nil
	}

	item := items[m.menuCursor]

	m.menuFilter = ""
	m.menuFiltering = false
	m.menuCursor = 0

	return m.activateItem(item)
}

// activateItem switches to the view backing item and initializes its
// sub-model, mirroring each view's repo/repos/org precondition.
func (m MainModel) activateItem(item menuItem) (tea.Model, tea.Cmd) {
	m.mode = item.view

	switch item.view {
	case ViewWatching:
		m.watchingModel = watching.NewModel()
		return m, m.watchingModel.Init()

	case ViewBranches:
		if m.repo != "" {
			m.branchesModel = branches.NewModel(m.repo, "")
			return m, m.branchesModel.Init()
		}

	case ViewProtection:
		if len(m.repos) > 0 {
			m.protectionModel = protection.NewModel(m.repos, m.baseline)
			return m, m.protectionModel.Init()
		}

	case ViewComments:
		if m.repo != "" {
			m.commentsModel = comments.NewModel(m.repo)
			return m, m.commentsModel.Init()
		}

	case ViewAnalytics:
		if m.repo != "" {
			m.analyticsModel = analytics.NewModel(m.repo)
			return m, m.analyticsModel.Init()
		}

	case ViewGHAPerf:
		if m.repo != "" {
			m.ghaPerfModel = ghaperf.NewModel(m.repo)
			return m, m.ghaPerfModel.Init()
		}

	case ViewSettings:
		if len(m.repos) > 0 {
			m.settingsModel = settings.NewModel(m.repos, m.baseline)
			return m, m.settingsModel.Init()
		}

	case ViewWebhooks:
		if len(m.repos) > 0 {
			m.webhooksModel = webhooks.NewModel(m.repos)
			return m, m.webhooksModel.Init()
		}

	case ViewCollaborators:
		if len(m.repos) > 0 {
			m.collaboratorsModel = collaborators.NewModel(m.repos)
			return m, m.collaboratorsModel.Init()
		}

	case ViewSecrets:
		if m.org != "" && len(m.repos) > 0 {
			m.secretsModel = secrets.NewModel(m.org, m.repos)
			return m, m.secretsModel.Init()
		}

	case ViewReleases:
		if len(m.repos) > 0 {
			m.releasesModel = releases.NewModel(m.repos)
			return m, m.releasesModel.Init()
		}

	case ViewOrphans:
		m.orphansModel = orphanstui.NewModel(m.org, orphans.DefaultScanOptions())
		return m, m.orphansModel.Init()

	case ViewPolicy:
		policyCfg, err := config.LoadPolicy("")
		if err != nil {
			m.policyModel = policytui.NewModelWithConfigError(err)
		} else {
			m.policyModel = policytui.NewModel(policyCfg)
		}

		return m, m.policyModel.Init()
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

	case ViewPolicy:
		m.policyModel, cmd = m.policyModel.Update(msg)
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
	case ViewPolicy:
		return m.policyModel.View()
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
		Foreground(theme.Current().Warning)

	menuItemStyle := lipgloss.NewStyle().
		Foreground(theme.Current().Text)

	selectedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Warning)

	disabledStyle := lipgloss.NewStyle().
		Foreground(theme.Current().Muted)

	descStyle := lipgloss.NewStyle().
		Foreground(theme.Current().Muted)

	filterStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Accent)

	helpStyle := lipgloss.NewStyle().
		Foreground(theme.Current().Muted)

	content := titleStyle.Render("🧹 gh-sweep") + "\n"
	content += titleStyle.Render("GitHub Repository Management TUI") + "\n\n"

	if m.menuFiltering || m.menuFilter != "" {
		content += filterStyle.Render("/ "+m.menuFilter) + "\n\n"
	}

	items := m.filteredMenuItems()
	if len(items) == 0 {
		content += helpStyle.Render("No views match filter") + "\n"
	} else {
		currentSection := ""
		var contentSb541 strings.Builder
		for i, item := range items {
			if item.section != currentSection {
				currentSection = item.section
				contentSb541.WriteString("\n" + sectionStyle.Render(currentSection) + "\n")
			}

			cursor := "  "
			if i == m.menuCursor {
				cursor = "> "
			}

			label := fmt.Sprintf("[%s] %s %s", item.key, item.icon, item.label)

			lineStyle := menuItemStyle
			switch {
			case i == m.menuCursor:
				lineStyle = selectedStyle
			case !item.enabled:
				lineStyle = disabledStyle
			}

			contentSb541.WriteString(lineStyle.Render(cursor + label))
			contentSb541.WriteString(descStyle.Render(" - " + item.desc))
			if !item.enabled {
				contentSb541.WriteString(descStyle.Render(" (unavailable)"))
			}
			contentSb541.WriteString("\n")
		}
		content += contentSb541.String()
		content += "\n"
	}

	if m.repo == "" && len(m.repos) == 0 {
		content += helpStyle.Render("💡 Configure with --repo flag or .gh-sweep.yaml") + "\n\n"
	}

	if m.menuFiltering {
		content += helpStyle.Render("↑/↓ or ctrl+p/n: move | enter: select | esc: clear filter")
	} else {
		content += helpStyle.Render(
			"j/k or ↑/↓: move | enter: select | /: filter | 0-9/o/p: quick jump | q: quit",
		)
	}

	return content
}
