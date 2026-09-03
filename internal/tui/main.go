// Package tui implements the top-level Bubble Tea application: the home
// menu and navigation between each view's sub-model.
package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/KyleKing/gh-sweep/internal/config"
	"github.com/KyleKing/gh-sweep/internal/orphans"
	"github.com/KyleKing/gh-sweep/internal/policy"
	"github.com/KyleKing/gh-sweep/internal/tui/components/analytics"
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
	"github.com/KyleKing/gh-sweep/internal/tui/scroll"
	"github.com/KyleKing/gh-sweep/internal/tui/theme"
)

// ViewMode represents different TUI views.
type ViewMode int

// View modes for MainModel, one per home-menu entry plus the home menu itself.
const (
	ViewHome ViewMode = iota
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

const (
	keyEsc   = "esc"
	keyCtrlC = "ctrl+c"

	sectionNamespace  = "Namespace Audit (scans your whole account or org, ignores --repo)"
	sectionSingleRepo = "Single Repo (needs --repo)"
	sectionCrossRepo  = "Cross-Repo (needs --repos or --org)"
	sectionPolicy     = "Policy (edits, diffs, and applies a declared policy file)"
)

// menuItem describes one selectable entry in the home menu list, where
// label carries its own keyboard shortcut inline as a bracketed letter
// (e.g. "[w]atch status") and key is that same letter for direct activation.
type menuItem struct {
	key     string
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
	repo                string
	repos               []string
	baseline            string
	org                 string
	policyPath          string
	applyOpts           policy.ApplyOptions
	scanOptions         orphans.ScanOptions
	regressionThreshold float64
}

// MainModelOptions configures the initial state of the main TUI model.
type MainModelOptions struct {
	Baseline string
	Org      string
	Repo     string
	// PolicyPath is the policy file the policy view diffs against. Empty
	// searches the default locations.
	PolicyPath          string
	ApplyOpts           policy.ApplyOptions
	Repos               []string
	ScanOptions         orphans.ScanOptions
	RegressionThreshold float64
}

// NewMainModel creates a new main TUI model.
func NewMainModel(opts MainModelOptions) MainModel {
	return MainModel{
		ready:               false,
		mode:                ViewHome,
		repo:                opts.Repo,
		repos:               opts.Repos,
		baseline:            opts.Baseline,
		org:                 opts.Org,
		policyPath:          opts.PolicyPath,
		applyOpts:           opts.ApplyOpts,
		scanOptions:         opts.ScanOptions,
		regressionThreshold: opts.RegressionThreshold,
	}
}

// Init initializes the model.
func (MainModel) Init() tea.Cmd {
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
		if msg.String() == keyEsc {
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

// buildMenuItems returns the full home menu, grouped by scope (namespace,
// single-repo, cross-repo, or policy) rather than by the phase each view
// happened to ship in, marking items whose repo/org preconditions aren't
// met as disabled.
func (m MainModel) buildMenuItems() []menuItem {
	items := m.buildNamespaceAuditItems()
	items = append(items, m.buildSingleRepoItems()...)
	items = append(items, m.buildCrossRepoItems()...)
	items = append(items, m.buildPolicyItems()...)

	return items
}

func (MainModel) buildNamespaceAuditItems() []menuItem {
	return []menuItem{
		{
			key:     "w",
			label:   "[w]atch status",
			desc:    "Audit and manage repo watching",
			section: sectionNamespace,
			view:    ViewWatching,
			enabled: true,
		},
		{
			key:     "o",
			label:   "[o]rphan branches",
			desc:    "Detect and clean up orphaned branches",
			section: sectionNamespace,
			view:    ViewOrphans,
			enabled: true,
		},
	}
}

// activeScope names what the resolved flags and config point this session at,
// so a view opened from the menu is never a surprise about which repos it
// touches or which policy file it would apply.
func (m MainModel) activeScope() string {
	parts := []string{}

	if m.org != "" {
		parts = append(parts, "org "+m.org)
	}

	switch {
	case len(m.repos) == 1:
		parts = append(parts, "repo "+m.repos[0])
	case len(m.repos) > 1:
		parts = append(parts, fmt.Sprintf("%d repos", len(m.repos)))
	}

	if m.policyPath != "" {
		parts = append(parts, "policy "+filepath.Base(m.policyPath))
	}

	if m.applyOpts.PruneBranches {
		parts = append(parts, "prune on")
	}

	if len(parts) == 0 {
		return "no org or repos configured; pass --org, --repos, or --config"
	}

	return strings.Join(parts, " | ")
}

func (m MainModel) buildSingleRepoItems() []menuItem {
	hasRepo := m.repo != ""

	return []menuItem{
		{
			key:     "c",
			label:   "pr [c]omments",
			desc:    "Review unresolved comments",
			section: sectionSingleRepo,
			view:    ViewComments,
			enabled: hasRepo,
		},
		{
			key:     "a",
			label:   "[a]nalytics",
			desc:    "CI/CD and repository statistics",
			section: sectionSingleRepo,
			view:    ViewAnalytics,
			enabled: hasRepo,
		},
		{
			key:     "g",
			label:   "[g]ha performance",
			desc:    "Workflow timing analysis",
			section: sectionSingleRepo,
			view:    ViewGHAPerf,
			enabled: hasRepo,
		},
	}
}

func (m MainModel) buildCrossRepoItems() []menuItem {
	hasRepos := len(m.repos) > 0
	hasOrgAndRepos := m.org != "" && hasRepos

	return []menuItem{
		{
			key:     "t",
			label:   "branch pro[t]ection",
			desc:    "Compare and sync protection rules",
			section: sectionCrossRepo,
			view:    ViewProtection,
			enabled: hasRepos,
		},
		{
			key:     "s",
			label:   "[s]ettings comparison",
			desc:    "Cross-repo settings diff",
			section: sectionCrossRepo,
			view:    ViewSettings,
			enabled: hasRepos,
		},
		{
			key:     "h",
			label:   "web[h]ooks",
			desc:    "Webhook health monitoring",
			section: sectionCrossRepo,
			view:    ViewWebhooks,
			enabled: hasRepos,
		},
		{
			key:     "l",
			label:   "co[l]laborators",
			desc:    "Manage repository access",
			section: sectionCrossRepo,
			view:    ViewCollaborators,
			enabled: hasRepos,
		},
		{
			key:     "e",
			label:   "s[e]crets audit",
			desc:    "Review secrets usage (read-only)",
			section: sectionCrossRepo,
			view:    ViewSecrets,
			enabled: hasOrgAndRepos,
		},
		{
			key:     "r",
			label:   "[r]eleases",
			desc:    "Release version overview",
			section: sectionCrossRepo,
			view:    ViewReleases,
			enabled: hasRepos,
		},
	}
}

func (MainModel) buildPolicyItems() []menuItem {
	return []menuItem{
		{
			key:     "y",
			label:   "polic[y]",
			desc:    "Diff and sync settings against a policy file",
			section: sectionPolicy,
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
	if m.menuFiltering {
		return m.updateHomeFiltering(msg.String())
	}

	return m.updateHomeNormal(msg.String())
}

func (m MainModel) updateHomeFiltering(key string) (tea.Model, tea.Cmd) {
	switch key {
	case keyCtrlC:
		return m, tea.Quit

	case keyEsc:
		m.menuFiltering = false
		m.menuFilter = ""
		m.menuCursor = 0

	case "enter":
		m.menuFiltering = false
		return m.activateSelected()

	case "backspace":
		if m.menuFilter != "" {
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

func (m MainModel) updateHomeNormal(key string) (tea.Model, tea.Cmd) {
	switch key {
	case keyCtrlC, "q":
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
		return m.activateWatching()
	case ViewProtection:
		return m.activateProtection()
	case ViewComments:
		return m.activateComments()
	case ViewAnalytics:
		return m.activateAnalytics()
	case ViewGHAPerf:
		return m.activateGHAPerf()
	case ViewSettings:
		return m.activateSettings()
	case ViewWebhooks:
		return m.activateWebhooks()
	case ViewCollaborators:
		return m.activateCollaborators()
	case ViewSecrets:
		return m.activateSecrets()
	case ViewReleases:
		return m.activateReleases()
	case ViewOrphans:
		return m.activateOrphans()
	case ViewPolicy:
		return m.activatePolicy()
	default:
		return m, nil
	}
}

func (m MainModel) activateWatching() (tea.Model, tea.Cmd) {
	m.watchingModel = watching.NewModel(m.org)
	return m, m.watchingModel.Init()
}

func (m MainModel) activateProtection() (tea.Model, tea.Cmd) {
	if len(m.repos) == 0 {
		return m, nil
	}

	m.protectionModel = protection.NewModel(m.repos, m.baseline)

	return m, m.protectionModel.Init()
}

func (m MainModel) activateComments() (tea.Model, tea.Cmd) {
	if m.repo == "" {
		return m, nil
	}

	m.commentsModel = comments.NewModel(m.repo)

	return m, m.commentsModel.Init()
}

func (m MainModel) activateAnalytics() (tea.Model, tea.Cmd) {
	if m.repo == "" {
		return m, nil
	}

	m.analyticsModel = analytics.NewModel(m.repo)

	return m, m.analyticsModel.Init()
}

func (m MainModel) activateGHAPerf() (tea.Model, tea.Cmd) {
	if m.repo == "" {
		return m, nil
	}

	opts := []ghaperf.Option{}
	if m.regressionThreshold > 0 {
		opts = append(opts, ghaperf.WithRegressionThreshold(m.regressionThreshold))
	}

	m.ghaPerfModel = ghaperf.NewModel(m.repo, opts...)

	return m, m.ghaPerfModel.Init()
}

func (m MainModel) activateSettings() (tea.Model, tea.Cmd) {
	if len(m.repos) == 0 {
		return m, nil
	}

	m.settingsModel = settings.NewModel(m.repos, m.baseline)

	return m, m.settingsModel.Init()
}

func (m MainModel) activateWebhooks() (tea.Model, tea.Cmd) {
	if len(m.repos) == 0 {
		return m, nil
	}

	m.webhooksModel = webhooks.NewModel(m.repos)

	return m, m.webhooksModel.Init()
}

func (m MainModel) activateCollaborators() (tea.Model, tea.Cmd) {
	if len(m.repos) == 0 {
		return m, nil
	}

	m.collaboratorsModel = collaborators.NewModel(m.repos)

	return m, m.collaboratorsModel.Init()
}

func (m MainModel) activateSecrets() (tea.Model, tea.Cmd) {
	if m.org == "" || len(m.repos) == 0 {
		return m, nil
	}

	m.secretsModel = secrets.NewModel(m.org, m.repos)

	return m, m.secretsModel.Init()
}

func (m MainModel) activateReleases() (tea.Model, tea.Cmd) {
	if len(m.repos) == 0 {
		return m, nil
	}

	m.releasesModel = releases.NewModel(m.repos)

	return m, m.releasesModel.Init()
}

func (m MainModel) activateOrphans() (tea.Model, tea.Cmd) {
	m.orphansModel = orphanstui.NewModel(m.org, m.scanOptions)
	return m, m.orphansModel.Init()
}

func (m MainModel) activatePolicy() (tea.Model, tea.Cmd) {
	policyCfg, err := config.LoadPolicy(m.policyPath)
	if err != nil {
		m.policyModel = policytui.NewModelWithConfigError(err)
	} else {
		m.policyModel = policytui.NewModel(policyCfg, config.PolicyPathOrDefault(m.policyPath), m.applyOpts)
	}

	return m, m.policyModel.Init()
}

func (m MainModel) updateActive(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.mode {
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

	default:
	}

	return m, cmd
}

// View renders the model.
func (m MainModel) View() tea.View {
	v := tea.NewView(m.renderContent())
	v.AltScreen = true

	return v
}

// viewRenderers maps each non-home view to the function that renders its
// sub-model, so renderContent can dispatch with a lookup instead of a
// 13-case switch.
var viewRenderers = map[ViewMode]func(MainModel) string{
	ViewProtection:    func(m MainModel) string { return m.protectionModel.View() },
	ViewComments:      func(m MainModel) string { return m.commentsModel.View() },
	ViewAnalytics:     func(m MainModel) string { return m.analyticsModel.View() },
	ViewGHAPerf:       func(m MainModel) string { return m.ghaPerfModel.View() },
	ViewSettings:      func(m MainModel) string { return m.settingsModel.View() },
	ViewWebhooks:      func(m MainModel) string { return m.webhooksModel.View() },
	ViewCollaborators: func(m MainModel) string { return m.collaboratorsModel.View() },
	ViewSecrets:       func(m MainModel) string { return m.secretsModel.View() },
	ViewReleases:      func(m MainModel) string { return m.releasesModel.View() },
	ViewWatching:      func(m MainModel) string { return m.watchingModel.View() },
	ViewOrphans:       func(m MainModel) string { return m.orphansModel.View() },
	ViewPolicy:        func(m MainModel) string { return m.policyModel.View() },
}

func (m MainModel) renderContent() string {
	if !m.ready {
		return "Initializing..."
	}

	if render, ok := viewRenderers[m.mode]; ok {
		return render(m)
	}

	return m.renderHome()
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

	var header strings.Builder
	header.WriteString(titleStyle.Render("gh-sweep") + "\n")
	header.WriteString(titleStyle.Render("GitHub Repository Management TUI") + "\n")
	header.WriteString(helpStyle.MaxWidth(m.width).Render(m.activeScope()) + "\n\n")

	if m.menuFiltering || m.menuFilter != "" {
		header.WriteString(filterStyle.Render("/ "+m.menuFilter) + "\n\n")
	}

	var footer strings.Builder
	if m.repo == "" && len(m.repos) == 0 {
		footer.WriteString(helpStyle.Render("Configure with --repo flag or .gh-sweep.yaml") + "\n\n")
	}
	if m.menuFiltering {
		footer.WriteString(helpStyle.Render("↑/↓ or ctrl+p/n: move | enter: select | esc: clear filter"))
	} else {
		footer.WriteString(helpStyle.Render(
			"j/k or ↑/↓: move | enter: select | /: filter | letter in [brackets]: jump | ?: help | q: quit",
		))
	}

	items := m.filteredMenuItems()
	if len(items) == 0 {
		return header.String() + helpStyle.Render("No views match filter") + "\n" + footer.String()
	}

	headerLines := strings.Count(header.String(), "\n")
	footerLines := strings.Count(footer.String(), "\n")
	available := m.height - headerLines - footerLines

	lines, cursorLine := m.buildMenuLines(items, sectionStyle, menuItemStyle, selectedStyle, disabledStyle, descStyle)
	start, end := scroll.Window(len(lines), cursorLine, available)

	var body strings.Builder

	scrollHintStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	if start > 0 {
		fmt.Fprintf(&body, "%s\n", scrollHintStyle.Render(fmt.Sprintf("↑ %d more above", start)))
	}

	body.WriteString(strings.Join(lines[start:end], "\n"))
	body.WriteString("\n")

	if end < len(lines) {
		fmt.Fprintf(&body, "%s\n", scrollHintStyle.Render(fmt.Sprintf("↓ %d more below", len(lines)-end)))
	}

	return header.String() + body.String() + "\n" + footer.String()
}

// buildMenuLines renders each menu item, and its section header, as one
// line each, aligning every item's " - description" on the same column so
// the caller can window by line, returning the lines and the selected
// row's index among them.
func (m MainModel) buildMenuLines(
	items []menuItem,
	sectionStyle, menuItemStyle, selectedStyle, disabledStyle, descStyle lipgloss.Style,
) ([]string, int) {
	maxLabelWidth := 0
	for _, item := range items {
		if w := lipgloss.Width(item.label); w > maxLabelWidth {
			maxLabelWidth = w
		}
	}

	var lines []string

	cursorLine := 0
	currentSection := ""

	for i, item := range items {
		if item.section != currentSection {
			currentSection = item.section
			lines = append(lines, "", sectionStyle.Render(currentSection))
		}

		if i == m.menuCursor {
			cursorLine = len(lines)
		}

		cursor := "  "
		if i == m.menuCursor {
			cursor = "> "
		}

		lineStyle := menuItemStyle
		switch {
		case i == m.menuCursor:
			lineStyle = selectedStyle
		case !item.enabled:
			lineStyle = disabledStyle
		}

		padding := strings.Repeat(" ", maxLabelWidth-lipgloss.Width(item.label))

		var line strings.Builder
		line.WriteString(lineStyle.Render(cursor + item.label + padding))
		line.WriteString(descStyle.Render(" - " + item.desc))
		if !item.enabled {
			line.WriteString(descStyle.Render(" (unavailable)"))
		}

		lines = append(lines, line.String())
	}

	return lines, cursorLine
}
