package orphans

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/orphans"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ViewMode string

const (
	ViewModeByRepo ViewMode = "by_repo"
	ViewModeByType ViewMode = "by_type"
	ViewModeFlat   ViewMode = "flat"
)

type Model struct {
	namespace      string
	options        orphans.ScanOptions
	result         *orphans.NamespaceScanResult
	viewMode       ViewMode
	cursor         int
	selected       map[string]bool
	filterType     *orphans.OrphanType
	loading        bool
	scanning       string
	progress       int
	total          int
	orphansFound   int
	statusMsg      string
	err            error
	width          int
	height         int
	confirmDelete  bool
	deleteTargets  []orphans.OrphanedBranch
}

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
	result *orphans.NamespaceScanResult
	err    error
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

func (m Model) Init() tea.Cmd {
	return m.startScan
}

func (m Model) startScan() tea.Msg {
	ctx := context.Background()
	client, err := github.NewClient(ctx)
	if err != nil {
		return scanCompleteMsg{err: fmt.Errorf("failed to create GitHub client: %w", err)}
	}

	scanner := orphans.NewNamespaceScanner(client, m.options)
	result, err := scanner.ScanNamespace(ctx, m.namespace)

	return scanCompleteMsg{result: result, err: err}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case scanCompleteMsg:
		m.loading = false
		m.result = msg.result
		m.err = msg.err
		return m, nil

	case scanProgressMsg:
		m.progress = msg.current
		m.total = msg.total
		m.scanning = msg.currentRepo
		m.orphansFound = msg.orphans
		return m, nil

	case deleteResultMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Failed to delete %s: %v", msg.branch, msg.err)
		} else {
			m.statusMsg = fmt.Sprintf("Deleted: %s", msg.branch)
			delete(m.selected, msg.branch)
			m.removeOrphanFromResult(msg.branch)
		}
		m.confirmDelete = false
		m.deleteTargets = nil
		return m, nil

	case tea.KeyMsg:
		if m.confirmDelete {
			return m.handleConfirmKeys(msg)
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			filtered := m.getFilteredOrphans()
			if m.cursor < len(filtered)-1 {
				m.cursor++
			}

		case " ":
			filtered := m.getFilteredOrphans()
			if m.cursor < len(filtered) {
				key := filtered[m.cursor].Key()
				m.selected[key] = !m.selected[key]
			}

		case "a":
			filtered := m.getFilteredOrphans()
			for _, orphan := range filtered {
				m.selected[orphan.Key()] = true
			}

		case "n":
			m.selected = make(map[string]bool)

		case "d":
			return m.handleDelete()

		case "1":
			m.filterType = nil
			m.cursor = 0

		case "2":
			t := orphans.OrphanTypeMergedPR
			m.filterType = &t
			m.cursor = 0

		case "3":
			t := orphans.OrphanTypeClosedPR
			m.filterType = &t
			m.cursor = 0

		case "4":
			t := orphans.OrphanTypeStale
			m.filterType = &t
			m.cursor = 0

		case "v":
			switch m.viewMode {
			case ViewModeByRepo:
				m.viewMode = ViewModeByType
			case ViewModeByType:
				m.viewMode = ViewModeFlat
			case ViewModeFlat:
				m.viewMode = ViewModeByRepo
			}
			m.cursor = 0

		case "r":
			m.loading = true
			m.result = nil
			m.err = nil
			m.cursor = 0
			m.selected = make(map[string]bool)
			return m, m.startScan
		}
	}

	return m, nil
}

func (m Model) handleConfirmKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		return m.executeDelete()
	case "n", "N", "esc":
		m.confirmDelete = false
		m.deleteTargets = nil
		m.statusMsg = "Delete cancelled"
		return m, nil
	}
	return m, nil
}

func (m Model) handleDelete() (tea.Model, tea.Cmd) {
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

func (m Model) executeDelete() (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	for _, orphan := range m.deleteTargets {
		orphan := orphan
		cmds = append(cmds, func() tea.Msg {
			ctx := context.Background()
			client, err := github.NewClient(ctx)
			if err != nil {
				return deleteResultMsg{branch: orphan.Key(), err: err}
			}

			parts := strings.SplitN(orphan.Repository, "/", 2)
			if len(parts) != 2 {
				return deleteResultMsg{branch: orphan.Key(), err: fmt.Errorf("invalid repository: %s", orphan.Repository)}
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

	for _, orphan := range m.result.AllOrphans() {
		if m.filterType != nil && orphan.Type != *m.filterType {
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

	return filtered
}

func (m Model) View() string {
	if m.loading {
		if m.total > 0 {
			return fmt.Sprintf("Scanning repositories...\nProgress: %d/%d repos\nCurrently: %s\nOrphans found: %d\n",
				m.progress, m.total, m.scanning, m.orphansFound)
		}
		return "Loading repositories...\n"
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress 'r' to retry or 'q' to quit\n", m.err)
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FFFF"))

	b.WriteString(titleStyle.Render("Orphaned Branches"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Namespace: %s\n\n", m.namespace))

	if m.confirmDelete {
		return m.renderConfirmDialog(&b)
	}

	activeTab := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFF00"))

	inactiveTab := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#777777"))

	if m.filterType == nil {
		b.WriteString(activeTab.Render("[1] All"))
	} else {
		b.WriteString(inactiveTab.Render("[1] All"))
	}
	b.WriteString("  ")

	if m.filterType != nil && *m.filterType == orphans.OrphanTypeMergedPR {
		b.WriteString(activeTab.Render("[2] Merged"))
	} else {
		b.WriteString(inactiveTab.Render("[2] Merged"))
	}
	b.WriteString("  ")

	if m.filterType != nil && *m.filterType == orphans.OrphanTypeClosedPR {
		b.WriteString(activeTab.Render("[3] Closed"))
	} else {
		b.WriteString(inactiveTab.Render("[3] Closed"))
	}
	b.WriteString("  ")

	if m.filterType != nil && *m.filterType == orphans.OrphanTypeStale {
		b.WriteString(activeTab.Render("[4] Stale"))
	} else {
		b.WriteString(inactiveTab.Render("[4] Stale"))
	}
	b.WriteString("\n\n")

	summaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))
	b.WriteString(summaryStyle.Render(fmt.Sprintf("Repos: %d | Orphans: %d | View: %s\n\n",
		m.result.TotalRepos, m.result.TotalOrphans, m.viewMode)))

	filtered := m.getFilteredOrphans()

	if len(filtered) == 0 {
		b.WriteString("No orphaned branches in this view.\n")
	} else {
		currentRepo := ""
		for i, orphan := range filtered {
			if m.viewMode == ViewModeByRepo && orphan.Repository != currentRepo {
				currentRepo = orphan.Repository
				repoStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FFFF"))
				b.WriteString("\n")
				b.WriteString(repoStyle.Render(currentRepo))
				b.WriteString("\n")
			}

			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}

			selectMark := " "
			if m.selected[orphan.Key()] {
				selectMark = "*"
			}

			typeStyle := m.getTypeStyle(orphan.Type)

			lineStyle := lipgloss.NewStyle()
			if m.cursor == i {
				lineStyle = lineStyle.Bold(true).Foreground(lipgloss.Color("#FFFF00"))
			}

			prInfo := ""
			if orphan.PRNumber != nil {
				prInfo = fmt.Sprintf(" #%d", *orphan.PRNumber)
			}

			line := fmt.Sprintf("%s%s %s ", cursor, selectMark, orphan.BranchName)
			b.WriteString(lineStyle.Render(line))
			b.WriteString(typeStyle.Render(fmt.Sprintf("[%s]", orphan.Type.Label())))
			b.WriteString(fmt.Sprintf(" %dd%s\n", orphan.DaysSinceActivity, prInfo))
		}
	}

	if m.statusMsg != "" {
		b.WriteString("\n")
		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF"))
		b.WriteString(statusStyle.Render(m.statusMsg))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))
	b.WriteString(helpStyle.Render("j/k: navigate | space: select | a/n: all/none | d: delete | v: view mode | r: refresh | esc: back"))

	return b.String()
}

func (m Model) renderConfirmDialog(b *strings.Builder) string {
	warnStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF0000"))
	b.WriteString(warnStyle.Render("Confirm Delete"))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("Delete %d branch(es)?\n\n", len(m.deleteTargets)))

	for _, orphan := range m.deleteTargets {
		b.WriteString(fmt.Sprintf("  - %s/%s\n", orphan.Repository, orphan.BranchName))
	}

	b.WriteString("\n")
	b.WriteString("Press 'y' to confirm, 'n' or 'esc' to cancel\n")

	return b.String()
}

func (m Model) getTypeStyle(t orphans.OrphanType) lipgloss.Style {
	switch t {
	case orphans.OrphanTypeMergedPR:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	case orphans.OrphanTypeClosedPR:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	case orphans.OrphanTypeStale:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00"))
	case orphans.OrphanTypeRecentNoPR:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))
	default:
		return lipgloss.NewStyle()
	}
}
