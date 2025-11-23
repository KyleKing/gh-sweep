package settings

import (
	"context"
	"fmt"
	"strings"

	"github.com/KyleKing/gh-sweep/internal/github"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the settings comparison TUI state
type Model struct {
	repos      []string
	settings   map[string]*github.RepoSettings
	baseline   string
	diffs      map[string][]github.SettingsDiff
	cursor     int
	width      int
	height     int
	loading    bool
	err        error
	viewMode   string // "overview", "diff"
}

// NewModel creates a new settings comparison model
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

// Init initializes the model
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

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

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

// View renders the model
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
		Foreground(lipgloss.Color("#00FFFF"))

	b.WriteString(titleStyle.Render("⚙️  Repository Settings Comparison"))
	b.WriteString("\n\n")

	if m.baseline != "" {
		b.WriteString(fmt.Sprintf("Baseline: %s\n\n", m.baseline))
	}

	// View mode tabs
	activeTab := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFF00"))

	inactiveTab := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#777777"))

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
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))
	b.WriteString(helpStyle.Render("↑/↓: navigate | 1/2: switch view | q: quit"))

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
			b.WriteString(fmt.Sprintf("%s %s: No settings loaded\n", cursor, repo))
			continue
		}

		statusStyle := lipgloss.NewStyle()
		if m.cursor == i {
			statusStyle = statusStyle.Bold(true).Foreground(lipgloss.Color("#FFFF00"))
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
		b.WriteString(fmt.Sprintf("📦 %s:\n", repo))
		for _, diff := range diffs {
			severityColor := "#FFFF00" // warning
			if diff.Severity == "critical" {
				severityColor = "#FF0000"
			} else if diff.Severity == "info" {
				severityColor = "#00FF00"
			}

			diffStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(severityColor))
			b.WriteString(diffStyle.Render(fmt.Sprintf("   [%s] %s: %v → %v\n",
				diff.Severity, diff.Field, diff.Baseline, diff.Current)))
		}
		b.WriteString("\n")
	}

	return b.String()
}
