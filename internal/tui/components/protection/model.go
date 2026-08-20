// Package protection is the TUI view that compares branch protection rules
// across repositories against a baseline repo.
package protection

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/tui/scroll"
	"github.com/KyleKing/gh-sweep/internal/tui/theme"
)

const (
	repoPartsCount = 2
	helpKeyWidth   = 16
)

// Model represents the protection rules TUI state.
type Model struct {
	repos    []string
	rules    map[string]*github.ProtectionRule
	baseline string
	diffs    map[string][]string
	cursor   int
	width    int
	height   int
	loading  bool
	err      error
	showHelp bool
}

// NewModel creates a new protection rules model.
func NewModel(repos []string, baseline string) Model {
	return Model{
		repos:    repos,
		baseline: baseline,
		rules:    make(map[string]*github.ProtectionRule),
		diffs:    make(map[string][]string),
		loading:  true,
	}
}

type rulesLoadedMsg struct {
	rules map[string]*github.ProtectionRule
	diffs map[string][]string
	err   error
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return m.loadRules
}

func (m Model) loadRules() tea.Msg {
	// Create GitHub client
	ctx := context.Background()
	client, err := github.NewClient(ctx)
	if err != nil {
		return rulesLoadedMsg{
			rules: make(map[string]*github.ProtectionRule),
			diffs: make(map[string][]string),
			err:   fmt.Errorf("failed to create GitHub client: %w", err),
		}
	}

	// Load protection rules for each repo
	rules := make(map[string]*github.ProtectionRule)
	for _, repoStr := range m.repos {
		owner, repo, ok := splitRepo(repoStr)
		if !ok {
			continue
		}

		rule, err := client.GetDefaultBranchProtection(owner, repo)
		if err != nil {
			continue
		}

		rules[repoStr] = rule
	}

	diffs := make(map[string][]string)
	if m.baseline != "" && rules[m.baseline] != nil {
		rulesSlice := make([]*github.ProtectionRule, 0, len(rules))
		rulesSlice = append(rulesSlice, rules[m.baseline])
		for repo, rule := range rules {
			if repo != m.baseline {
				rulesSlice = append(rulesSlice, rule)
			}
		}
		diffs = github.CompareProtectionRules(rulesSlice)
	}

	return rulesLoadedMsg{
		rules: rules,
		diffs: diffs,
		err:   nil,
	}
}

func splitRepo(repo string) (string, string, bool) {
	parts := strings.SplitN(repo, "/", repoPartsCount)
	if len(parts) != repoPartsCount {
		return "", "", false
	}

	return parts[0], parts[1], true
}

// Update handles messages.
//
//nolint:unparam // matches every TUI component's Update(Model, tea.Cmd) shape
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		return m, nil

	case rulesLoadedMsg:
		m.loading = false
		m.rules = msg.rules
		m.diffs = msg.diffs
		m.err = msg.err

		return m, nil

	case tea.KeyPressMsg:
		return m.updateKeyMsg(msg)
	}

	return m, nil
}

func (m Model) updateKeyMsg(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	if m.showHelp {
		if msg.String() == "?" || msg.String() == "esc" {
			m.showHelp = false
		}

		return m, nil
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "?":
		m.showHelp = true

	case "g":
		m.cursor = 0

	case "G":
		if len(m.repos) > 0 {
			m.cursor = len(m.repos) - 1
		}

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.repos)-1 {
			m.cursor++
		}
	}

	return m, nil
}

// View renders the model.
func (m Model) View() string {
	if m.loading {
		return "Loading protection rules...\n"
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var header strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Primary)

	header.WriteString(titleStyle.Render("Branch Protection Rules"))
	header.WriteString("\n\n")

	if m.baseline != "" {
		fmt.Fprintf(&header, "Baseline: %s\n\n", m.baseline)
	}

	if m.showHelp {
		return renderHelp(&header)
	}

	var footer strings.Builder

	if len(m.diffs) > 0 {
		footer.WriteString("\nDifferences from baseline:\n\n")
		for field, differences := range m.diffs {
			footer.WriteString(field + ":\n")
			for _, diff := range differences {
				fmt.Fprintf(&footer, "  - %s\n", diff)
			}
		}
	}

	footer.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	footer.WriteString(helpStyle.Render("↑/↓: navigate | ?: help | q: quit"))

	headerLines := strings.Count(header.String(), "\n")
	footerLines := strings.Count(footer.String(), "\n")
	available := m.height - headerLines - footerLines

	var b strings.Builder
	b.WriteString(header.String())
	m.renderRepoList(&b, available)
	b.WriteString(footer.String())

	return b.String()
}

func (m Model) renderRepoList(b *strings.Builder, available int) {
	if len(m.repos) == 0 {
		return
	}

	lines, cursorLine := m.buildRepoLines()

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

// buildRepoLines renders each repo's protection summary as one multi-line
// block, tracking cursorLine as the index of the cursor's block within the
// flattened lines.
func (m Model) buildRepoLines() ([]string, int) {
	var body strings.Builder

	cursorLine := 0

	for i, repo := range m.repos {
		if i == m.cursor {
			cursorLine = strings.Count(body.String(), "\n")
		}

		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		rule := m.rules[repo]
		if rule == nil {
			fmt.Fprintf(&body, "%s %s: No protection\n", cursor, repo)
			continue
		}

		statusStyle := lipgloss.NewStyle()
		if m.cursor == i {
			statusStyle = statusStyle.Bold(true).Foreground(theme.Current().Warning)
		}

		line := fmt.Sprintf("%s %s:\n", cursor, repo)
		line += fmt.Sprintf("   Reviews: %d | Code Owners: %v | Admins: %v\n",
			rule.RequiredReviews,
			rule.RequireCodeOwnerReviews,
			rule.EnforceAdmins,
		)
		line += fmt.Sprintf("   Status Checks: %s\n",
			strings.Join(rule.RequireStatusChecks, ", "))

		body.WriteString(statusStyle.Render(line))
		body.WriteString("\n")
	}

	lines := strings.Split(strings.TrimSuffix(body.String(), "\n"), "\n")

	return lines, cursorLine
}

func renderHelp(b *strings.Builder) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Primary)
	b.WriteString(titleStyle.Render("Keybindings"))
	b.WriteString("\n\n")

	bindings := [][2]string{
		{"j/k, up/down", "move the cursor"},
		{"g / G", "jump to top / bottom"},
		{"?", "toggle this help"},
		{"q", "quit"},
	}

	for _, binding := range bindings {
		keyStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Warning).Width(helpKeyWidth)
		fmt.Fprintf(b, "%s %s\n", keyStyle.Render(binding[0]), binding[1])
	}

	b.WriteString("\nPress '?' or 'esc' to close\n")

	return b.String()
}
