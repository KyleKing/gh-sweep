package settings

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/tui/theme"
)

// Model represents the settings comparison TUI state.
type Model struct {
	repos    []string
	settings map[string]*github.RepoSettings
	baseline string
	diffs    map[string][]github.SettingsDiff
	cursor   int
	width    int
	height   int
	loading  bool
	err      error
	viewMode string // "overview", "diff"
	showHelp bool
}

// NewModel creates a new settings comparison model.
func NewModel(repos []string, baseline string) Model {
	return Model{
		repos:    repos,
		baseline: baseline,
		settings: make(map[string]*github.RepoSettings),
		diffs:    make(map[string][]github.SettingsDiff),
		loading:  true,
		viewMode: "overview",
	}
}

type settingsLoadedMsg struct {
	settings map[string]*github.RepoSettings
	diffs    map[string][]github.SettingsDiff
	err      error
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return m.loadSettings
}

func (m Model) loadSettings() tea.Msg {
	// Create GitHub client
	ctx := context.Background()
	client, err := github.NewClient(ctx)
	if err != nil {
		return settingsLoadedMsg{
			settings: make(map[string]*github.RepoSettings),
			diffs:    make(map[string][]github.SettingsDiff),
			err:      fmt.Errorf("failed to create GitHub client: %w", err),
		}
	}

	// Load settings for each repo
	settings := make(map[string]*github.RepoSettings)
	for _, repoStr := range m.repos {
		parts := strings.Split(repoStr, "/")
		if len(parts) != 2 {
			continue
		}
		owner, repo := parts[0], parts[1]

		repoSettings, err := client.GetRepoSettings(owner, repo)
		if err != nil {
			// Skip repos on error
			continue
		}

		settings[repoStr] = repoSettings
	}

	// Compare settings if baseline is specified
	diffs := make(map[string][]github.SettingsDiff)
	if m.baseline != "" {
		baselineSettings := settings[m.baseline]
		if baselineSettings != nil {
			for repoStr, repoSettings := range settings {
				if repoStr != m.baseline {
					repoDiffs := github.CompareSettings(baselineSettings, repoSettings)
					if len(repoDiffs) > 0 {
						diffs[repoStr] = repoDiffs
					}
				}
			}
		}
	}

	return settingsLoadedMsg{
		settings: settings,
		diffs:    diffs,
		err:      nil,
	}
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		return m, nil

	case settingsLoadedMsg:
		m.loading = false
		m.settings = msg.settings
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

		case "1":
			m.viewMode = "overview"
		case "2":
			m.viewMode = "diff"
		}
	}

	return m, nil
}

// View renders the model.
func (m Model) View() string {
	if m.loading {
		return "Loading repository settings...\n"
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var b strings.Builder

	// Header
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Primary)

	b.WriteString(titleStyle.Render("⚙️  Repository Settings Comparison"))
	b.WriteString("\n\n")

	if m.baseline != "" {
		fmt.Fprintf(&b, "Baseline: %s\n\n", m.baseline)
	}

	if m.showHelp {
		return m.renderHelp(&b)
	}

	// View mode tabs
	activeTab := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Warning)

	inactiveTab := lipgloss.NewStyle().
		Foreground(theme.Current().Muted)

	if m.viewMode == "overview" {
		b.WriteString(activeTab.Render("[1] Overview"))
	} else {
		b.WriteString(inactiveTab.Render("[1] Overview"))
	}
	b.WriteString("  ")
	if m.viewMode == "diff" {
		b.WriteString(activeTab.Render("[2] Differences"))
	} else {
		b.WriteString(inactiveTab.Render("[2] Differences"))
	}
	b.WriteString("\n\n")

	// Content based on view mode
	switch m.viewMode {
	case "overview":
		b.WriteString(m.renderOverview())
	case "diff":
		b.WriteString(m.renderDiff())
	}

	// Help
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	b.WriteString(helpStyle.Render("↑/↓: navigate | 1/2: switch view | ?: help | q: quit"))

	return b.String()
}

func (m Model) renderHelp(b *strings.Builder) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Primary)
	b.WriteString(titleStyle.Render("Keybindings"))
	b.WriteString("\n\n")

	bindings := [][2]string{
		{"j/k, up/down", "move the cursor"},
		{"g / G", "jump to top / bottom"},
		{"1 / 2", "switch view: overview / differences"},
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

func (m Model) renderOverview() string {
	var b strings.Builder

	b.WriteString("📋 Repository Settings\n\n")

	for i, repo := range m.repos {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		settings := m.settings[repo]
		if settings == nil {
			fmt.Fprintf(&b, "%s %s: No settings loaded\n", cursor, repo)
			continue
		}

		statusStyle := lipgloss.NewStyle()
		if m.cursor == i {
			statusStyle = statusStyle.Bold(true).Foreground(theme.Current().Warning)
		}

		line := fmt.Sprintf("%s %s:\n", cursor, repo)
		line += fmt.Sprintf("   Default Branch: %s\n", settings.DefaultBranch)
		line += fmt.Sprintf("   Merge: %v | Squash: %v | Rebase: %v\n",
			settings.AllowMergeCommit, settings.AllowSquashMerge, settings.AllowRebaseMerge)
		line += fmt.Sprintf("   Delete on Merge: %v | Issues: %v | Wiki: %v\n",
			settings.DeleteBranchOnMerge, settings.HasIssues, settings.HasWiki)

		b.WriteString(statusStyle.Render(line))
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderDiff() string {
	var b strings.Builder

	if len(m.diffs) == 0 {
		b.WriteString("✅ No differences found - all repositories match baseline\n")
		return b.String()
	}

	b.WriteString("⚠️  Differences from Baseline\n\n")

	for repo, diffs := range m.diffs {
		fmt.Fprintf(&b, "📦 %s:\n", repo)
		for _, diff := range diffs {
			severityColor := theme.Current().Warning
			switch diff.Severity {
			case "critical":
				severityColor = theme.Current().Error
			case "info":
				severityColor = theme.Current().Success
			}

			diffStyle := lipgloss.NewStyle().Foreground(severityColor)
			b.WriteString(diffStyle.Render(fmt.Sprintf("   [%s] %s: %v → %v\n",
				diff.Severity, diff.Field, diff.Baseline, diff.Current)))
		}
		b.WriteString("\n")
	}

	return b.String()
}
