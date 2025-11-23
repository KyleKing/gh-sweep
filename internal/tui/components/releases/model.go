package releases

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/KyleKing/gh-sweep/internal/github"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the releases overview TUI state
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
}

// NewModel creates a new releases overview model
func NewModel(repos []string) Model {
	return Model{
		repos:    repos,
		releases: make(map[string][]github.Release),
		latest:   make(map[string]*github.Release),
		loading:  true,
		viewMode: "latest",
	}
}

type releasesLoadedMsg struct {
	releases map[string][]github.Release
	latest   map[string]*github.Release
	err      error
}

// Init initializes the model
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
		parts := strings.Split(repoStr, "/")
		if len(parts) != 2 {
			continue
		}
		owner, repo := parts[0], parts[1]

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

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			maxCursor := len(m.repos) - 1
			if m.cursor < maxCursor {
				m.cursor++
			}

		case "1":
			m.viewMode = "latest"
			m.cursor = 0
		case "2":
			m.viewMode = "all"
			m.cursor = 0
		case "3":
			m.viewMode = "outdated"
			m.cursor = 0
		}
	}

	return m, nil
}

// View renders the model
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
		Foreground(lipgloss.Color("#00FFFF"))

	b.WriteString(titleStyle.Render("📦 Release Overview"))
	b.WriteString("\n\n")

	// View mode tabs
	activeTab := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFF00"))

	inactiveTab := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#777777"))

	if m.viewMode == "latest" {
		b.WriteString(activeTab.Render("[1] Latest"))
	} else {
		b.WriteString(inactiveTab.Render("[1] Latest"))
	}
	b.WriteString("  ")
	if m.viewMode == "all" {
		b.WriteString(activeTab.Render("[2] All Releases"))
	} else {
		b.WriteString(inactiveTab.Render("[2] All Releases"))
	}
	b.WriteString("  ")
	if m.viewMode == "outdated" {
		b.WriteString(activeTab.Render("[3] Outdated"))
	} else {
		b.WriteString(inactiveTab.Render("[3] Outdated"))
	}
	b.WriteString("\n\n")

	// Content based on view mode
	switch m.viewMode {
	case "latest":
		b.WriteString(m.renderLatest())
	case "all":
		b.WriteString(m.renderAll())
	case "outdated":
		b.WriteString(m.renderOutdated())
	}

	// Help
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))
	b.WriteString(helpStyle.Render("↑/↓: navigate | 1/2/3: switch view | q: quit"))

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
			releaseStyle = releaseStyle.Bold(true).Foreground(lipgloss.Color("#FFFF00"))
		}

		if release == nil {
			line := fmt.Sprintf("%s %s: No releases\n", cursor, repo)
			b.WriteString(releaseStyle.Render(line))
			continue
		}

		// Calculate days since release
		daysSince := int(time.Since(release.PublishedAt).Hours() / 24)
		ageColor := "#00FF00" // green (recent)
		if daysSince > 90 {
			ageColor = "#FF0000" // red (old)
		} else if daysSince > 30 {
			ageColor = "#FFFF00" // yellow (moderate)
		}
		ageStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ageColor))

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
			repoStyle = repoStyle.Bold(true).Foreground(lipgloss.Color("#FFFF00"))
		}

		line := fmt.Sprintf("%s %s (%d releases):\n", cursor, repo, len(releases))

		// Show first few releases
		for j, release := range releases {
			if j >= 5 {
				line += fmt.Sprintf("   ... and %d more\n", len(releases)-5)
				break
			}
			line += fmt.Sprintf("   - %s (%s)\n",
				release.TagName,
				release.PublishedAt.Format("2006-01-02"))
		}

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

		daysSince := int(time.Since(release.PublishedAt).Hours() / 24)
		if daysSince <= 90 {
			continue
		}

		outdatedCount++
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		releaseStyle := lipgloss.NewStyle()
		if m.cursor == i {
			releaseStyle = releaseStyle.Bold(true).Foreground(lipgloss.Color("#FFFF00"))
		}

		warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))

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
		b.WriteString(fmt.Sprintf("\nFound %d repositories with outdated releases.\n", outdatedCount))
	}

	return b.String()
}
