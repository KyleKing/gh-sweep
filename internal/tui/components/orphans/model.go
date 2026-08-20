// Package orphans is the TUI view for auditing and deleting orphaned
// branches across a GitHub namespace.
package orphans

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/orphans"
	"github.com/KyleKing/gh-sweep/internal/tui/scroll"
	"github.com/KyleKing/gh-sweep/internal/tui/theme"
)

const (
	repoPartsCount = 2
	helpKeyWidth   = 16
	keyEsc         = "esc"
	keySpace       = "space"
)

var errInvalidRepository = errors.New("invalid repository")

// ViewMode selects how orphaned branches are grouped in the list view.
type ViewMode string

// Grouping modes for the orphaned-branch list.
const (
	ViewModeByRepo ViewMode = "by_repo"
	ViewModeByType ViewMode = "by_type"
	ViewModeFlat   ViewMode = "flat"
)

// Model represents the orphaned branches TUI state.
type Model struct {
	namespace     string
	options       orphans.ScanOptions
	result        *orphans.NamespaceScanResult
	viewMode      ViewMode
	cursor        int
	selected      map[string]bool
	filterType    *orphans.OrphanType
	loading       bool
	scanning      string
	progress      int
	total         int
	orphansFound  int
	statusMsg     string
	err           error
	width         int
	height        int
	confirmDelete bool
	deleteTargets []orphans.OrphanedBranch
	searching     bool
	searchQuery   string
	sortReverse   bool
	showHelp      bool
}

// NewModel creates a new orphaned branches model.
func NewModel(namespace string, options orphans.ScanOptions) Model {
	return Model{
		namespace: namespace,
		options:   options,
		viewMode:  ViewModeByRepo,
		selected:  make(map[string]bool),
		loading:   true,
	}
}

type scanCompleteMsg struct {
	namespace string
	result    *orphans.NamespaceScanResult
	err       error
}

type scanProgressMsg struct {
	current     int
	total       int
	currentRepo string
	orphans     int
}

type deleteResultMsg struct {
	branch string
	err    error
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return m.startScan
}

func (m Model) startScan() tea.Msg {
	ctx := context.Background()
	client, err := github.NewClient(ctx)
	if err != nil {
		return scanCompleteMsg{err: fmt.Errorf("failed to create GitHub client: %w", err)}
	}

	namespace := m.namespace
	if namespace == "" {
		namespace, err = client.GetAuthenticatedUser()
		if err != nil {
			return scanCompleteMsg{err: fmt.Errorf("failed to get authenticated user: %w", err)}
		}
	}

	scanner := orphans.NewNamespaceScanner(client, m.options)
	result, err := scanner.ScanNamespace(ctx, namespace)

	return scanCompleteMsg{namespace: namespace, result: result, err: err}
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		return m, nil

	case scanCompleteMsg:
		return m.handleScanComplete(msg), nil

	case scanProgressMsg:
		m.progress = msg.current
		m.total = msg.total
		m.scanning = msg.currentRepo
		m.orphansFound = msg.orphans

		return m, nil

	case deleteResultMsg:
		return m.handleDeleteResult(msg), nil

	case tea.KeyPressMsg:
		if m.confirmDelete {
			return m.handleConfirmKeys(msg)
		}

		if m.searching {
			return m.handleSearchKeys(msg)
		}

		return m.updateKeyMsg(msg)
	}

	return m, nil
}

func (m Model) handleScanComplete(msg scanCompleteMsg) Model {
	m.loading = false
	if msg.namespace != "" {
		m.namespace = msg.namespace
	}
	m.result = msg.result
	m.err = msg.err

	return m
}

func (m Model) handleDeleteResult(msg deleteResultMsg) Model {
	if msg.err != nil {
		m.statusMsg = fmt.Sprintf("Failed to delete %s: %v", msg.branch, msg.err)
	} else {
		m.statusMsg = "Deleted: " + msg.branch
		delete(m.selected, msg.branch)
		m.removeOrphanFromResult(msg.branch)
	}
	m.confirmDelete = false
	m.deleteTargets = nil

	return m
}

func (m Model) updateKeyMsg(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	if m.showHelp {
		if msg.String() == "?" || msg.String() == keyEsc {
			m.showHelp = false
		}

		return m, nil
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "?":
		m.showHelp = true

	case "/":
		m.searching = true

	case "g", "G", "up", "k", "down", "j":
		m.updateCursor(msg.String())

	case "R":
		m.sortReverse = !m.sortReverse

	case keySpace, "a", "n", "I":
		m.updateSelection(msg.String())

	case "d":
		return m.handleDelete()

	case "1", "2", "3", "4":
		m.updateFilterType(msg.String())

	case "v":
		m.cycleViewMode()

	case "r":
		return m.startRescan()
	}

	return m, nil
}

func (m *Model) updateCursor(key string) {
	filtered := m.getFilteredOrphans()

	switch key {
	case "g":
		m.cursor = 0

	case "G":
		if len(filtered) > 0 {
			m.cursor = len(filtered) - 1
		}

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(filtered)-1 {
			m.cursor++
		}
	}
}

func (m *Model) updateSelection(key string) {
	switch key {
	case keySpace:
		m.toggleCursorSelection()
	case "a":
		m.selectAll()
	case "n":
		m.selected = make(map[string]bool)
	case "I":
		m.invertSelection()
	}
}

func (m *Model) toggleCursorSelection() {
	filtered := m.getFilteredOrphans()
	if m.cursor < len(filtered) {
		key := filtered[m.cursor].Key()
		m.selected[key] = !m.selected[key]
	}
}

func (m *Model) selectAll() {
	for _, orphan := range m.getFilteredOrphans() {
		m.selected[orphan.Key()] = true
	}
}

func (m *Model) invertSelection() {
	for _, orphan := range m.getFilteredOrphans() {
		key := orphan.Key()
		m.selected[key] = !m.selected[key]
	}
}

func (m *Model) updateFilterType(key string) {
	switch key {
	case "1":
		m.filterType = nil
	case "2":
		t := orphans.OrphanTypeMergedPR
		m.filterType = &t
	case "3":
		t := orphans.OrphanTypeClosedPR
		m.filterType = &t
	case "4":
		t := orphans.OrphanTypeStale
		m.filterType = &t
	}

	m.cursor = 0
}

func (m *Model) cycleViewMode() {
	switch m.viewMode {
	case ViewModeByRepo:
		m.viewMode = ViewModeByType
	case ViewModeByType:
		m.viewMode = ViewModeFlat
	case ViewModeFlat:
		m.viewMode = ViewModeByRepo
	}

	m.cursor = 0
}

func (m Model) startRescan() (Model, tea.Cmd) {
	m.loading = true
	m.result = nil
	m.err = nil
	m.cursor = 0
	m.selected = make(map[string]bool)

	return m, m.startScan
}

func (m Model) handleSearchKeys(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case keyEsc:
		m.searching = false
		m.searchQuery = ""
		m.cursor = 0

	case "enter":
		m.searching = false

	case "backspace":
		if m.searchQuery != "" {
			runes := []rune(m.searchQuery)
			m.searchQuery = string(runes[:len(runes)-1])
			m.cursor = 0
		}

	default:
		if key := msg.String(); len([]rune(key)) == 1 {
			m.searchQuery += key
			m.cursor = 0
		}
	}

	return m, nil
}

func (m Model) handleConfirmKeys(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		return m.executeDelete()
	case "n", "N", keyEsc:
		m.confirmDelete = false
		m.deleteTargets = nil
		m.statusMsg = "Delete canceled"

		return m, nil
	}

	return m, nil
}

func (m Model) handleDelete() (Model, tea.Cmd) {
	filtered := m.getFilteredOrphans()
	var targets []orphans.OrphanedBranch

	hasSelection := false
	for _, orphan := range filtered {
		if m.selected[orphan.Key()] {
			hasSelection = true
			targets = append(targets, orphan)
		}
	}

	if !hasSelection && m.cursor < len(filtered) {
		targets = append(targets, filtered[m.cursor])
	}

	if len(targets) == 0 {
		m.statusMsg = "No branches selected"
		return m, nil
	}

	m.confirmDelete = true
	m.deleteTargets = targets

	return m, nil
}

func (m Model) executeDelete() (Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0, len(m.deleteTargets))

	for _, orphan := range m.deleteTargets {
		cmds = append(cmds, func() tea.Msg {
			ctx := context.Background()
			client, err := github.NewClient(ctx)
			if err != nil {
				return deleteResultMsg{branch: orphan.Key(), err: err}
			}

			parts := strings.SplitN(orphan.Repository, "/", repoPartsCount)
			if len(parts) != repoPartsCount {
				return deleteResultMsg{
					branch: orphan.Key(),
					err:    fmt.Errorf("%w: %s", errInvalidRepository, orphan.Repository),
				}
			}

			err = client.DeleteBranch(parts[0], parts[1], orphan.BranchName)

			return deleteResultMsg{branch: orphan.Key(), err: err}
		})
	}

	m.confirmDelete = false

	return m, tea.Batch(cmds...)
}

func (m *Model) removeOrphanFromResult(key string) {
	if m.result == nil {
		return
	}

	for i := range m.result.Results {
		result := &m.result.Results[i]
		for j := len(result.Orphans) - 1; j >= 0; j-- {
			if result.Orphans[j].Key() == key {
				result.Orphans = append(result.Orphans[:j], result.Orphans[j+1:]...)
				m.result.TotalOrphans--

				break
			}
		}
	}
}

func (m Model) getFilteredOrphans() []orphans.OrphanedBranch {
	if m.result == nil {
		return nil
	}

	var filtered []orphans.OrphanedBranch

	query := strings.ToLower(m.searchQuery)

	for _, orphan := range m.result.AllOrphans() {
		if m.filterType != nil && orphan.Type != *m.filterType {
			continue
		}
		if query != "" &&
			!strings.Contains(strings.ToLower(orphan.BranchName), query) &&
			!strings.Contains(strings.ToLower(orphan.Repository), query) {
			continue
		}
		filtered = append(filtered, orphan)
	}

	switch m.viewMode {
	case ViewModeByType:
		sort.Slice(filtered, func(i, j int) bool {
			if filtered[i].Type != filtered[j].Type {
				return filtered[i].Type < filtered[j].Type
			}

			return filtered[i].Key() < filtered[j].Key()
		})
	case ViewModeFlat:
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].LastCommitDate.Before(filtered[j].LastCommitDate)
		})
	default:
		sort.Slice(filtered, func(i, j int) bool {
			if filtered[i].Repository != filtered[j].Repository {
				return filtered[i].Repository < filtered[j].Repository
			}

			return filtered[i].BranchName < filtered[j].BranchName
		})
	}

	if m.sortReverse {
		for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
			filtered[i], filtered[j] = filtered[j], filtered[i]
		}
	}

	return filtered
}

// View renders the model.
func (m Model) View() string {
	if m.loading {
		return m.renderLoading()
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress 'r' to retry or 'q' to quit\n", m.err)
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Primary)

	b.WriteString(titleStyle.Render("Orphaned Branches"))
	b.WriteString("\n")
	fmt.Fprintf(&b, "Namespace: %s\n\n", m.namespace)

	if m.confirmDelete {
		return m.renderConfirmDialog(&b)
	}

	if m.showHelp {
		return renderHelp(&b)
	}

	m.renderTypeTabs(&b)

	summaryStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	sortLabel := ""
	if m.sortReverse {
		sortLabel = " | sort: reversed"
	}
	b.WriteString(summaryStyle.Render(fmt.Sprintf("Repos: %d | Orphans: %d | View: %s%s\n\n",
		m.result.TotalRepos, m.result.TotalOrphans, m.viewMode, sortLabel)))

	if m.searching || m.searchQuery != "" {
		searchStyle := lipgloss.NewStyle().Foreground(theme.Current().Warning)
		fmt.Fprintf(&b, "%s %s\n\n", searchStyle.Render("Search:"), m.searchQuery)
	}

	var footer strings.Builder
	if m.statusMsg != "" {
		footer.WriteString("\n")
		statusStyle := lipgloss.NewStyle().Foreground(theme.Current().Primary)
		footer.WriteString(statusStyle.Render(m.statusMsg))
		footer.WriteString("\n")
	}
	footer.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	footer.WriteString(
		helpStyle.Render(
			"j/k: navigate | space: select | a/n/I: all/none/invert | d: delete | v: view mode | " +
				"r: refresh | ?: help | esc: back",
		),
	)

	headerLines := strings.Count(b.String(), "\n")
	footerLines := strings.Count(footer.String(), "\n")
	available := m.height - headerLines - footerLines

	filtered := m.getFilteredOrphans()
	m.renderOrphanList(&b, filtered, available)
	b.WriteString(footer.String())

	return b.String()
}

func (m Model) renderLoading() string {
	if m.total > 0 {
		return fmt.Sprintf(
			"Scanning repositories...\nProgress: %d/%d repos\nCurrently: %s\nOrphans found: %d\n",
			m.progress,
			m.total,
			m.scanning,
			m.orphansFound,
		)
	}

	return "Loading repositories...\n"
}

func (m Model) renderTypeTabs(b *strings.Builder) {
	activeTab := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Warning)

	inactiveTab := lipgloss.NewStyle().
		Foreground(theme.Current().Muted)

	tab := func(label string, t *orphans.OrphanType) string {
		match := (t == nil && m.filterType == nil) || (t != nil && m.filterType != nil && *m.filterType == *t)
		if match {
			return activeTab.Render(label)
		}

		return inactiveTab.Render(label)
	}

	merged := orphans.OrphanTypeMergedPR
	closed := orphans.OrphanTypeClosedPR
	stale := orphans.OrphanTypeStale

	b.WriteString(tab("[1] All", nil))
	b.WriteString("  ")
	b.WriteString(tab("[2] Merged", &merged))
	b.WriteString("  ")
	b.WriteString(tab("[3] Closed", &closed))
	b.WriteString("  ")
	b.WriteString(tab("[4] Stale", &stale))
	b.WriteString("\n\n")
}

func (m Model) renderOrphanList(b *strings.Builder, filtered []orphans.OrphanedBranch, available int) {
	if len(filtered) == 0 {
		b.WriteString("No orphaned branches in this view.\n")
		return
	}

	lines, cursorLine := m.buildOrphanLines(filtered)

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

// buildOrphanLines renders each orphan, and its group header in by-repo
// mode, as one line each so the caller can window by line rather than by
// item, returning the lines and the cursor row's index among them.
func (m Model) buildOrphanLines(filtered []orphans.OrphanedBranch) ([]string, int) {
	var lines []string

	cursorLine := 0
	currentRepo := ""

	for i, orphan := range filtered {
		if m.viewMode == ViewModeByRepo && orphan.Repository != currentRepo {
			currentRepo = orphan.Repository
			repoStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Primary)
			lines = append(lines, "", repoStyle.Render(currentRepo))
		}

		if i == m.cursor {
			cursorLine = len(lines)
		}

		lines = append(lines, m.renderOrphanLine(orphan, i))
	}

	return lines, cursorLine
}

func (m Model) renderOrphanLine(orphan orphans.OrphanedBranch, i int) string {
	cursor := " "
	if m.cursor == i {
		cursor = ">"
	}

	selectMark := " "
	if m.selected[orphan.Key()] {
		selectMark = "*"
	}

	typeStyle := getTypeStyle(orphan.Type)

	lineStyle := lipgloss.NewStyle()
	if m.cursor == i {
		lineStyle = lineStyle.Bold(true).Foreground(theme.Current().Warning)
	}

	prInfo := ""
	if orphan.PRNumber != nil {
		prInfo = fmt.Sprintf(" #%d", *orphan.PRNumber)
	}

	line := fmt.Sprintf("%s%s %s ", cursor, selectMark, orphan.BranchName)

	var b strings.Builder
	b.WriteString(lineStyle.Render(line))
	b.WriteString(typeStyle.Render(fmt.Sprintf("[%s]", orphan.Type.Label())))
	fmt.Fprintf(&b, " %dd%s", orphan.DaysSinceActivity, prInfo)

	return b.String()
}

func (m Model) renderConfirmDialog(b *strings.Builder) string {
	warnStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Error)
	b.WriteString(warnStyle.Render("Confirm Delete"))
	b.WriteString("\n\n")

	fmt.Fprintf(b, "Delete %d branch(es)?\n\n", len(m.deleteTargets))

	for _, orphan := range m.deleteTargets {
		fmt.Fprintf(b, "  - %s/%s\n", orphan.Repository, orphan.BranchName)
	}

	b.WriteString("\n")
	b.WriteString("Press 'y' to confirm, 'n' or 'esc' to cancel\n")

	return b.String()
}

func renderHelp(b *strings.Builder) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Primary)
	b.WriteString(titleStyle.Render("Keybindings"))
	b.WriteString("\n\n")

	bindings := [][2]string{
		{"j/k, up/down", "move the cursor"},
		{"g / G", "jump to top / bottom"},
		{"space", "toggle selection on the cursor row"},
		{"a / n / I", "select all / none / invert selection"},
		{"/", "search branch or repository name"},
		{"1-4", "filter by all, merged, closed, or stale"},
		{"v", "cycle grouping: by repo, by type, flat"},
		{"R", "reverse the current sort order"},
		{"d", "delete the selection (or the cursor row)"},
		{"r", "refresh"},
		{"?", "toggle this help"},
		{"esc / q", "back / quit"},
	}

	for _, binding := range bindings {
		keyStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Warning).Width(helpKeyWidth)
		fmt.Fprintf(b, "%s %s\n", keyStyle.Render(binding[0]), binding[1])
	}

	b.WriteString("\nPress '?' or 'esc' to close\n")

	return b.String()
}

func getTypeStyle(t orphans.OrphanType) lipgloss.Style {
	switch t {
	case orphans.OrphanTypeMergedPR:
		return lipgloss.NewStyle().Foreground(theme.Current().Success)
	case orphans.OrphanTypeClosedPR:
		return lipgloss.NewStyle().Foreground(theme.Current().Error)
	case orphans.OrphanTypeStale:
		return lipgloss.NewStyle().Foreground(theme.Current().Warning)
	case orphans.OrphanTypeRecentNoPR:
		return lipgloss.NewStyle().Foreground(theme.Current().Muted)
	default:
		return lipgloss.NewStyle()
	}
}
