package protection

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/tui/theme"
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
		parts := strings.Split(repoStr, "/")
		if len(parts) != 2 {
			continue
		}
		owner, repo := parts[0], parts[1]

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

// Update handles messages.
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

	var b strings.Builder

	// Header
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Primary)

	b.WriteString(titleStyle.Render("🛡️  Branch Protection Rules"))
	b.WriteString("\n\n")

	if m.baseline != "" {
		fmt.Fprintf(&b, "Baseline: %s\n\n", m.baseline)
	}

	if m.showHelp {
		return m.renderHelp(&b)
	}

	// Repository list with rules
	for i, repo := range m.repos {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		rule := m.rules[repo]
		if rule == nil {
			fmt.Fprintf(&b, "%s %s: No protection\n", cursor, repo)
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

		b.WriteString(statusStyle.Render(line))
		b.WriteString("\n")
	}

	// Differences
	if len(m.diffs) > 0 {
		b.WriteString("\n⚠️  Differences from baseline:\n\n")
		for field, differences := range m.diffs {
			b.WriteString(field + ":\n")
			for _, diff := range differences {
				fmt.Fprintf(&b, "  - %s\n", diff)
			}
		}
	}

	// Help
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	b.WriteString(helpStyle.Render("↑/↓: navigate | ?: help | q: quit"))

	return b.String()
}

func (m Model) renderHelp(b *strings.Builder) string {
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
		keyStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Warning).Width(16)
		fmt.Fprintf(b, "%s %s\n", keyStyle.Render(binding[0]), binding[1])
	}

	b.WriteString("\nPress '?' or 'esc' to close\n")

	return b.String()
}
