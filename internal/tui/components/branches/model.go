package branches

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/KyleKing/gh-sweep/internal/git"
	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/tui/theme"
)

// Model represents the branch management TUI state.
type Model struct {
	repo          string
	baseBranch    string
	branches      []github.BranchStatus
	selected      map[string]bool
	cursor        int
	width         int
	height        int
	loading       bool
	err           error
	statusMsg     string
	confirmDelete bool
	deleteTargets []github.BranchStatus
	searching     bool
	searchQuery   string
	showHelp      bool
}

// NewModel creates a new branch management model.
func NewModel(repo, baseBranch string) Model {
	return Model{
		repo:       repo,
		baseBranch: baseBranch,
		selected:   make(map[string]bool),
		loading:    true,
	}
}

type branchesLoadedMsg struct {
	branches []github.BranchStatus
	err      error
}

type deleteResultMsg struct {
	branch string
	err    error
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return m.loadBranches
}

func (m Model) loadBranches() tea.Msg {
	if m.repo == "" {
		return branchesLoadedMsg{err: errors.New("no repository specified")}
	}

	owner, name, ok := splitRepo(m.repo)
	if !ok {
		return branchesLoadedMsg{err: errors.New("invalid repo format, expected owner/repo")}
	}

	ctx := context.Background()
	client, err := github.NewClient(ctx)
	if err != nil {
		return branchesLoadedMsg{err: fmt.Errorf("failed to create GitHub client: %w", err)}
	}

	branches, err := client.ListBranchStatuses(owner, name, m.baseBranch)
	if err != nil {
		return branchesLoadedMsg{err: fmt.Errorf("failed to load branches: %w", err)}
	}

	return branchesLoadedMsg{branches: branches}
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		return m, nil

	case branchesLoadedMsg:
		m.loading = false
		m.branches = msg.branches
		m.err = msg.err

		return m, nil

	case deleteResultMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Failed to delete %s: %v", msg.branch, msg.err)
		} else {
			m.statusMsg = "Deleted: " + msg.branch
			delete(m.selected, msg.branch)
			m.removeBranch(msg.branch)
		}
		m.confirmDelete = false
		m.deleteTargets = nil

		return m, nil

	case tea.KeyPressMsg:
		if m.confirmDelete {
			return m.handleConfirmKeys(msg)
		}

		if m.searching {
			return m.handleSearchKeys(msg)
		}

		if m.showHelp {
			if msg.String() == "?" || msg.String() == "esc" {
				m.showHelp = false
			}

			return m, nil
		}

		return m.handleListKeys(msg)
	}

	return m, nil
}

func (m Model) getVisibleBranches() []github.BranchStatus {
	if m.searchQuery == "" {
		return m.branches
	}

	query := strings.ToLower(m.searchQuery)

	var visible []github.BranchStatus
	for _, branch := range m.branches {
		if strings.Contains(strings.ToLower(branch.Name), query) {
			visible = append(visible, branch)
		}
	}

	return visible
}

func (m Model) handleSearchKeys(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc":
		m.searching = false
		m.searchQuery = ""
		m.cursor = 0

	case "enter":
		m.searching = false

	case "backspace":
		if len(m.searchQuery) > 0 {
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

func (m Model) handleListKeys(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "?":
		m.showHelp = true

	case "/":
		m.searching = true

	case "g":
		m.cursor = 0

	case "G":
		if visible := m.getVisibleBranches(); len(visible) > 0 {
			m.cursor = len(visible) - 1
		}

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.getVisibleBranches())-1 {
			m.cursor++
		}

	case "space":
		visible := m.getVisibleBranches()
		if m.cursor < len(visible) {
			m.selected[visible[m.cursor].Name] = !m.selected[visible[m.cursor].Name]
		}

	case "a":
		for _, branch := range m.getVisibleBranches() {
			m.selected[branch.Name] = true
		}

	case "n":
		m.selected = make(map[string]bool)

	case "I":
		for _, branch := range m.getVisibleBranches() {
			m.selected[branch.Name] = !m.selected[branch.Name]
		}

	case "d":
		return m.handleDelete()

	case "r":
		m.loading = true
		m.branches = nil
		m.err = nil
		m.cursor = 0
		m.statusMsg = ""
		m.selected = make(map[string]bool)

		return m, m.loadBranches
	}

	return m, nil
}

func (m Model) handleConfirmKeys(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		return m.executeDelete()
	case "n", "N", "esc":
		m.confirmDelete = false
		m.deleteTargets = nil
		m.statusMsg = "Delete canceled"

		return m, nil
	}

	return m, nil
}

func (m Model) handleDelete() (Model, tea.Cmd) {
	eligible, blocked := collectDeleteTargets(m.getVisibleBranches(), m.selected, m.cursor)

	if len(blocked) > 0 {
		m.statusMsg = describeBlocked(blocked)
	}

	if len(eligible) == 0 {
		if len(blocked) == 0 {
			m.statusMsg = "No branches selected"
		}

		return m, nil
	}

	m.confirmDelete = true
	m.deleteTargets = eligible

	return m, nil
}

func (m Model) executeDelete() (Model, tea.Cmd) {
	owner, name, ok := splitRepo(m.repo)
	if !ok {
		m.confirmDelete = false
		m.deleteTargets = nil
		m.statusMsg = "Invalid repository: " + m.repo

		return m, nil
	}

	cmds := make([]tea.Cmd, 0, len(m.deleteTargets))
	for _, target := range m.deleteTargets {
		branch := target.Name
		cmds = append(cmds, func() tea.Msg {
			ctx := context.Background()
			client, err := github.NewClient(ctx)
			if err != nil {
				return deleteResultMsg{branch: branch, err: err}
			}

			return deleteResultMsg{branch: branch, err: client.DeleteBranch(owner, name, branch)}
		})
	}

	m.confirmDelete = false

	return m, tea.Batch(cmds...)
}

func (m *Model) removeBranch(name string) {
	for i := range m.branches {
		if m.branches[i].Name == name {
			m.branches = append(m.branches[:i], m.branches[i+1:]...)

			break
		}
	}

	if m.cursor >= len(m.branches) && m.cursor > 0 {
		m.cursor = len(m.branches) - 1
	}
}

func collectDeleteTargets(
	branches []github.BranchStatus,
	selected map[string]bool,
	cursor int,
) ([]github.BranchStatus, []github.BranchStatus) {
	var targets []github.BranchStatus

	for _, branch := range branches {
		if selected[branch.Name] {
			targets = append(targets, branch)
		}
	}

	if len(targets) == 0 && cursor >= 0 && cursor < len(branches) {
		targets = append(targets, branches[cursor])
	}

	var eligible, blocked []github.BranchStatus

	for _, branch := range targets {
		if branch.DeleteBlocked() != nil {
			blocked = append(blocked, branch)
		} else {
			eligible = append(eligible, branch)
		}
	}

	return eligible, blocked
}

func describeBlocked(blocked []github.BranchStatus) string {
	reasons := make([]string, 0, len(blocked))
	for _, branch := range blocked {
		reasons = append(reasons, fmt.Sprintf("%s (%v)", branch.Name, branch.DeleteBlocked()))
	}

	return "Blocked: " + strings.Join(reasons, ", ")
}

// View renders the model.
func (m Model) View() string {
	if m.loading {
		return "Loading branches...\n"
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress 'r' to retry or 'q' to quit\n", m.err)
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Primary)

	b.WriteString(titleStyle.Render("Branches for " + m.repo))
	b.WriteString("\n\n")

	if m.confirmDelete {
		return m.renderConfirmDialog(&b)
	}

	if m.showHelp {
		return m.renderHelp(&b)
	}

	if m.searching || m.searchQuery != "" {
		searchStyle := lipgloss.NewStyle().Foreground(theme.Current().Warning)
		fmt.Fprintf(&b, "%s %s\n\n", searchStyle.Render("Search:"), m.searchQuery)
	}

	visible := m.getVisibleBranches()

	if len(visible) == 0 {
		b.WriteString("No branches found.\n")
	} else {
		for i, branch := range visible {
			m.renderBranchLine(&b, i, branch)
		}
	}

	if m.statusMsg != "" {
		b.WriteString("\n")
		statusStyle := lipgloss.NewStyle().Foreground(theme.Current().Primary)
		b.WriteString(statusStyle.Render(m.statusMsg))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	b.WriteString(
		helpStyle.Render(
			"j/k: navigate | space: select | a/n/I: all/none/invert | d: delete | r: refresh | ?: help | q: quit",
		),
	)

	return b.String()
}

func (m Model) renderHelp(b *strings.Builder) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Primary)
	b.WriteString(titleStyle.Render("Keybindings"))
	b.WriteString("\n\n")

	bindings := [][2]string{
		{"j/k, up/down", "move the cursor"},
		{"g / G", "jump to top / bottom"},
		{"space", "toggle selection on the cursor row"},
		{"a / n / I", "select all / none / invert selection"},
		{"/", "search branch name"},
		{"d", "delete the selection (or the cursor row)"},
		{"r", "refresh"},
		{"?", "toggle this help"},
		{"esc / q", "back / quit"},
	}

	for _, binding := range bindings {
		keyStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Warning).Width(16)
		fmt.Fprintf(b, "%s %s\n", keyStyle.Render(binding[0]), binding[1])
	}

	b.WriteString("\nPress '?' or 'esc' to close\n")

	return b.String()
}

func (m Model) renderBranchLine(b *strings.Builder, i int, branch github.BranchStatus) {
	cursor := " "
	if m.cursor == i {
		cursor = ">"
	}

	checkbox := "[ ]"
	if m.selected[branch.Name] {
		checkbox = "[x]"
	}

	line := fmt.Sprintf(
		"%s %s %s ↑%d ↓%d",
		cursor,
		checkbox,
		branch.Name,
		branch.Ahead,
		branch.Behind,
	)

	if m.cursor == i {
		selectedStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Current().Warning)
		b.WriteString(selectedStyle.Render(line))
	} else {
		b.WriteString(line)
	}

	mutedStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	b.WriteString(mutedStyle.Render(branchAnnotations(branch)))
	b.WriteString("\n")
}

func branchAnnotations(branch github.BranchStatus) string {
	var parts []string

	if branch.IsDefault {
		parts = append(parts, "[default]")
	}
	if branch.Protected {
		parts = append(parts, "[protected]")
	}
	if branch.PR != nil {
		parts = append(parts, fmt.Sprintf("PR #%d (%s)", branch.PR.Number, branch.PR.State))
	}
	if !branch.LastCommitDate.IsZero() {
		days := int(time.Since(branch.LastCommitDate).Hours() / 24)
		parts = append(parts, fmt.Sprintf("%dd", days))
	}

	if len(parts) == 0 {
		return ""
	}

	return " " + strings.Join(parts, " ")
}

func (m Model) renderConfirmDialog(b *strings.Builder) string {
	warnStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Error)
	b.WriteString(warnStyle.Render("Confirm Delete"))
	b.WriteString("\n\n")

	fmt.Fprintf(b, "Delete %d branch(es) from %s?\n\n", len(m.deleteTargets), m.repo)

	for _, branch := range m.deleteTargets {
		fmt.Fprintf(b, "  - %s\n", branch.Name)
	}

	b.WriteString("\n")
	b.WriteString("Press 'y' to confirm, 'n' or 'esc' to cancel\n")

	return b.String()
}

func splitRepo(repo string) (string, string, bool) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}

	return parts[0], parts[1], true
}

// GetLocalBranches loads branches from local Git repository.
func GetLocalBranches(repoPath string) ([]git.BranchInfo, error) {
	repo := git.NewLocalRepo(repoPath)

	branches, err := repo.ListBranches()
	if err != nil {
		return nil, fmt.Errorf("failed to list local branches: %w", err)
	}

	return branches, nil
}
