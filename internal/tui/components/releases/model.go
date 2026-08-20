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
	"github.com/KyleKing/gh-sweep/internal/tui/scroll"
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

	var header strings.Builder

	// Header
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Primary)

	header.WriteString(titleStyle.Render("📦 Release Overview"))
	header.WriteString("\n\n")

	if m.showHelp {
		return renderHelp(&header)
	}

	// View mode tabs
	activeTab := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Warning)

	inactiveTab := lipgloss.NewStyle().
		Foreground(theme.Current().Muted)

	if m.viewMode == viewModeLatest {
		header.WriteString(activeTab.Render("[1] Latest"))
	} else {
		header.WriteString(inactiveTab.Render("[1] Latest"))
	}
	header.WriteString("  ")
	if m.viewMode == viewModeAll {
		header.WriteString(activeTab.Render("[2] All Releases"))
	} else {
		header.WriteString(inactiveTab.Render("[2] All Releases"))
	}
	header.WriteString("  ")
	if m.viewMode == viewModeOutdated {
		header.WriteString(activeTab.Render("[3] Outdated"))
	} else {
		header.WriteString(inactiveTab.Render("[3] Outdated"))
	}
	header.WriteString("\n\n")

	switch m.viewMode {
	case viewModeAll:
		header.WriteString("📋 All Releases\n\n")
	case viewModeOutdated:
		header.WriteString("⚠️  Outdated Releases (>90 days)\n\n")
	default:
		header.WriteString("📌 Latest Releases\n\n")
	}

	var footer strings.Builder
	footer.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	footer.WriteString(helpStyle.Render("↑/↓: navigate | 1/2/3: switch view | ?: help | q: quit"))

	headerLines := strings.Count(header.String(), "\n")
	footerLines := strings.Count(footer.String(), "\n")
	available := m.height - headerLines - footerLines

	var body strings.Builder

	switch m.viewMode {
	case viewModeAll:
		lines, cursorLine := m.buildAllLines()
		renderScrolledList(&body, lines, cursorLine, available)

	case viewModeOutdated:
		lines, cursorLine, outdatedCount := m.buildOutdatedLines()
		if outdatedCount == 0 {
			body.WriteString("✅ All repositories have recent releases!\n")
		} else {
			renderScrolledList(&body, lines, cursorLine, available)
			fmt.Fprintf(&body, "\nFound %d repositories with outdated releases.\n", outdatedCount)
		}

	default:
		lines, cursorLine := m.buildLatestLines()
		renderScrolledList(&body, lines, cursorLine, available)
	}

	return header.String() + body.String() + footer.String()
}

// renderScrolledList windows lines to the available height, keeping
// cursorLine visible, and prepends/appends a muted scroll hint when the
// list overflows the viewport.
func renderScrolledList(b *strings.Builder, lines []string, cursorLine, available int) {
	if len(lines) == 0 {
		return
	}

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

// buildLatestLines renders each repo's latest release as one or more lines,
// returning the lines and the cursor row's index among them so the caller
// can window by line rather than by item.
func (m Model) buildLatestLines() ([]string, int) {
	if len(m.latest) == 0 {
		return []string{"No releases found."}, 0
	}

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

		release := m.latest[repo]

		releaseStyle := lipgloss.NewStyle()
		if m.cursor == i {
			releaseStyle = releaseStyle.Bold(true).Foreground(theme.Current().Warning)
		}

		if release == nil {
			line := fmt.Sprintf("%s %s: No releases\n", cursor, repo)
			body.WriteString(releaseStyle.Render(line))

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

		body.WriteString(releaseStyle.Render(line))
		body.WriteString("\n")
	}

	trimmed := strings.TrimSuffix(body.String(), "\n")

	return strings.Split(trimmed, "\n"), cursorLine
}

// buildAllLines renders each repo's release history as one or more lines,
// returning the lines and the cursor row's index among them so the caller
// can window by line rather than by item.
func (m Model) buildAllLines() ([]string, int) {
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

		body.WriteString(repoStyle.Render(line))
		body.WriteString("\n")
	}

	if body.Len() == 0 {
		return nil, 0
	}

	trimmed := strings.TrimSuffix(body.String(), "\n")

	return strings.Split(trimmed, "\n"), cursorLine
}

// buildOutdatedLines renders each outdated repo as one or more lines,
// returning the lines, the cursor row's index among them, and the count of
// outdated repos found.
func (m Model) buildOutdatedLines() ([]string, int, int) {
	var body strings.Builder

	cursorLine := 0
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

		if i == m.cursor {
			cursorLine = strings.Count(body.String(), "\n")
		}

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

		body.WriteString(releaseStyle.Render(line))
		body.WriteString("\n")
	}

	if body.Len() == 0 {
		return nil, 0, outdatedCount
	}

	trimmed := strings.TrimSuffix(body.String(), "\n")

	return strings.Split(trimmed, "\n"), cursorLine, outdatedCount
}
