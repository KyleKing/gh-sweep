// Package releases is the TUI view that overviews repository releases: the
// latest per repo, the full history, and which repos have gone stale.
package releases

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/tui/theme"
)

const (
	repoPartsCount = 2
	helpKeyWidth   = 16
	hoursPerDay    = 24

	outdatedThresholdDays = 90
	agingThresholdDays    = 30
	maxReleasesShown      = 5

	viewModeLatest   = "latest"
	viewModeAll      = "all"
	viewModeOutdated = "outdated"
)

// Model represents the releases overview TUI state.
type Model struct {
	repos    []string
	releases map[string][]github.Release
	latest   map[string]*github.Release
	cursor   int
	width    int
	height   int
	loading  bool
	err      error
	viewMode string // "latest", "all", "outdated"
	showHelp bool
}

// NewModel creates a new releases overview model.
func NewModel(repos []string) Model {
	return Model{
		repos:    repos,
		releases: make(map[string][]github.Release),
		latest:   make(map[string]*github.Release),
		loading:  true,
		viewMode: viewModeLatest,
	}
}

type releasesLoadedMsg struct {
	releases map[string][]github.Release
	latest   map[string]*github.Release
	err      error
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return m.loadReleases
}

func (m Model) loadReleases() tea.Msg {
	// Create GitHub client
	ctx := context.Background()
	client, err := github.NewClient(ctx)
	if err != nil {
		return releasesLoadedMsg{
			releases: make(map[string][]github.Release),
			latest:   make(map[string]*github.Release),
			err:      fmt.Errorf("failed to create GitHub client: %w", err),
		}
	}

	// Load releases for each repo
	releases := make(map[string][]github.Release)
	latest := make(map[string]*github.Release)

	for _, repoStr := range m.repos {
		owner, repo, ok := splitRepo(repoStr)
		if !ok {
			continue
		}

		// Get all releases
		repoReleases, err := client.ListReleases(owner, repo)
		if err != nil {
			// Skip repos on error
			continue
		}
		releases[repoStr] = repoReleases

		// Get latest release
		latestRelease, err := client.GetLatestRelease(owner, repo)
		if err != nil {
			// Skip if no release
			continue
		}
		latest[repoStr] = latestRelease
	}

	return releasesLoadedMsg{
		releases: releases,
		latest:   latest,
		err:      nil,
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

	case releasesLoadedMsg:
		m.loading = false
		m.releases = msg.releases
		m.latest = msg.latest
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

	case "1", "2", "3":
		m.updateViewMode(msg.String())

	default:
		m.updateCursor(msg.String())
	}

	return m, nil
}

func (m *Model) updateViewMode(key string) {
	switch key {
	case "1":
		m.viewMode = viewModeLatest
	case "2":
		m.viewMode = viewModeAll
	case "3":
		m.viewMode = viewModeOutdated
	}

	m.cursor = 0
}

func (m *Model) updateCursor(key string) {
	switch key {
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
		maxCursor := len(m.repos) - 1
		if m.cursor < maxCursor {
			m.cursor++
		}
	}
}

// View renders the model.
func (m Model) View() string {
	if m.loading {
		return "Loading releases...\n"
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var b strings.Builder

	// Header
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Primary)

	b.WriteString(titleStyle.Render("📦 Release Overview"))
	b.WriteString("\n\n")

	if m.showHelp {
		return renderHelp(&b)
	}

	// View mode tabs
	activeTab := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Warning)

	inactiveTab := lipgloss.NewStyle().
		Foreground(theme.Current().Muted)

	if m.viewMode == viewModeLatest {
		b.WriteString(activeTab.Render("[1] Latest"))
	} else {
		b.WriteString(inactiveTab.Render("[1] Latest"))
	}
	b.WriteString("  ")
	if m.viewMode == viewModeAll {
		b.WriteString(activeTab.Render("[2] All Releases"))
	} else {
		b.WriteString(inactiveTab.Render("[2] All Releases"))
	}
	b.WriteString("  ")
	if m.viewMode == viewModeOutdated {
		b.WriteString(activeTab.Render("[3] Outdated"))
	} else {
		b.WriteString(inactiveTab.Render("[3] Outdated"))
	}
	b.WriteString("\n\n")

	// Content based on view mode
	switch m.viewMode {
	case viewModeLatest:
		b.WriteString(m.renderLatest())
	case viewModeAll:
		b.WriteString(m.renderAll())
	case viewModeOutdated:
		b.WriteString(m.renderOutdated())
	}

	// Help
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	b.WriteString(helpStyle.Render("↑/↓: navigate | 1/2/3: switch view | ?: help | q: quit"))

	return b.String()
}

func renderHelp(b *strings.Builder) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Primary)
	b.WriteString(titleStyle.Render("Keybindings"))
	b.WriteString("\n\n")

	bindings := [][2]string{
		{"j/k, up/down", "move the cursor"},
		{"g / G", "jump to top / bottom"},
		{"1/2/3", "switch view: latest / all / outdated"},
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

func (m Model) renderLatest() string {
	var b strings.Builder

	b.WriteString("📌 Latest Releases\n\n")

	if len(m.latest) == 0 {
		b.WriteString("No releases found.\n")
		return b.String()
	}

	for i, repo := range m.repos {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		release := m.latest[repo]

		releaseStyle := lipgloss.NewStyle()
		if m.cursor == i {
			releaseStyle = releaseStyle.Bold(true).Foreground(theme.Current().Warning)
		}

		if release == nil {
			line := fmt.Sprintf("%s %s: No releases\n", cursor, repo)
			b.WriteString(releaseStyle.Render(line))

			continue
		}

		// Calculate days since release
		daysSince := int(time.Since(release.PublishedAt).Hours() / hoursPerDay)
		ageColor := theme.Current().Success
		if daysSince > outdatedThresholdDays {
			ageColor = theme.Current().Error
		} else if daysSince > agingThresholdDays {
			ageColor = theme.Current().Warning
		}
		ageStyle := lipgloss.NewStyle().Foreground(ageColor)

		line := fmt.Sprintf("%s %s:\n", cursor, repo)
		line += fmt.Sprintf("   Version: %s\n", release.TagName)
		line += fmt.Sprintf("   Published: %s ", release.PublishedAt.Format("2006-01-02"))
		line += ageStyle.Render(fmt.Sprintf("(%d days ago)", daysSince))
		line += "\n"
		line += fmt.Sprintf("   Author: %s\n", release.Author)

		b.WriteString(releaseStyle.Render(line))
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderAll() string {
	var b strings.Builder

	b.WriteString("📋 All Releases\n\n")

	for i, repo := range m.repos {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		releases := m.releases[repo]

		repoStyle := lipgloss.NewStyle()
		if m.cursor == i {
			repoStyle = repoStyle.Bold(true).Foreground(theme.Current().Warning)
		}

		line := fmt.Sprintf("%s %s (%d releases):\n", cursor, repo, len(releases))

		// Show first few releases
		var lineSb281 strings.Builder
		for j := range releases {
			if j >= maxReleasesShown {
				fmt.Fprintf(&lineSb281, "   ... and %d more\n", len(releases)-maxReleasesShown)
				break
			}
			release := releases[j]
			fmt.Fprintf(&lineSb281, "   - %s (%s)\n",
				release.TagName,
				release.PublishedAt.Format("2006-01-02"))
		}
		line += lineSb281.String()

		b.WriteString(repoStyle.Render(line))
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderOutdated() string {
	var b strings.Builder

	b.WriteString("⚠️  Outdated Releases (>90 days)\n\n")

	outdatedCount := 0
	for i, repo := range m.repos {
		release := m.latest[repo]
		if release == nil {
			continue
		}

		daysSince := int(time.Since(release.PublishedAt).Hours() / hoursPerDay)
		if daysSince <= outdatedThresholdDays {
			continue
		}

		outdatedCount++
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		releaseStyle := lipgloss.NewStyle()
		if m.cursor == i {
			releaseStyle = releaseStyle.Bold(true).Foreground(theme.Current().Warning)
		}

		warningStyle := lipgloss.NewStyle().Foreground(theme.Current().Error)

		line := fmt.Sprintf("%s %s:\n", cursor, repo)
		line += fmt.Sprintf("   Last Release: %s\n", release.TagName)
		line += "   "
		line += warningStyle.Render(fmt.Sprintf("⚠️  %d days old", daysSince))
		line += "\n"

		b.WriteString(releaseStyle.Render(line))
		b.WriteString("\n")
	}

	if outdatedCount == 0 {
		b.WriteString("✅ All repositories have recent releases!\n")
	} else {
		fmt.Fprintf(&b, "\nFound %d repositories with outdated releases.\n", outdatedCount)
	}

	return b.String()
}
