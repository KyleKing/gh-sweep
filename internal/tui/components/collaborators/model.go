// Package collaborators is the TUI view that audits repository collaborators
// and their permission levels, grouped either by repository or by user.
package collaborators

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/tui/theme"
)

const (
	repoPartsCount  = 2
	helpKeyWidth    = 16
	maxPreviewItems = 3

	viewModeByRepo = "byrepo"
	viewModeByUser = "byuser"

	permissionAdmin = "admin"
	permissionWrite = "write"
)

// Model represents the collaborator management TUI state.
type Model struct {
	repos         []string
	collaborators map[string][]github.Collaborator
	cursor        int
	width         int
	height        int
	loading       bool
	err           error
	viewMode      string // viewModeByRepo, viewModeByUser
	showHelp      bool
}

// NewModel creates a new collaborator management model.
func NewModel(repos []string) Model {
	return Model{
		repos:         repos,
		collaborators: make(map[string][]github.Collaborator),
		loading:       true,
		viewMode:      viewModeByRepo,
	}
}

type collaboratorsLoadedMsg struct {
	collaborators map[string][]github.Collaborator
	err           error
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return m.loadCollaborators
}

func (m Model) loadCollaborators() tea.Msg {
	// Create GitHub client
	ctx := context.Background()
	client, err := github.NewClient(ctx)
	if err != nil {
		return collaboratorsLoadedMsg{
			collaborators: make(map[string][]github.Collaborator),
			err:           fmt.Errorf("failed to create GitHub client: %w", err),
		}
	}

	// Load collaborators for each repo
	collaborators := make(map[string][]github.Collaborator)
	for _, repoStr := range m.repos {
		parts := strings.Split(repoStr, "/")
		if len(parts) != repoPartsCount {
			continue
		}
		owner, repo := parts[0], parts[1]

		repoCollaborators, err := client.ListCollaborators(owner, repo)
		if err != nil {
			// Skip repos on error
			continue
		}

		collaborators[repoStr] = repoCollaborators
	}

	return collaboratorsLoadedMsg{
		collaborators: collaborators,
		err:           nil,
	}
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

	case collaboratorsLoadedMsg:
		m.loading = false
		m.collaborators = msg.collaborators
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
		if maxCursor := m.maxCursor(); maxCursor >= 0 {
			m.cursor = maxCursor
		}

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < m.maxCursor() {
			m.cursor++
		}

	case "1":
		m.viewMode = viewModeByRepo
		m.cursor = 0
	case "2":
		m.viewMode = viewModeByUser
		m.cursor = 0
	}

	return m, nil
}

func (m Model) maxCursor() int {
	if m.viewMode == viewModeByUser {
		return m.getTotalCollaborators() - 1
	}

	return len(m.repos) - 1
}

func (m Model) getTotalCollaborators() int {
	// Get unique collaborators across all repos
	uniqueUsers := make(map[string]bool)
	for _, collabs := range m.collaborators {
		for _, collab := range collabs {
			uniqueUsers[collab.Login] = true
		}
	}

	return len(uniqueUsers)
}

// View renders the model.
func (m Model) View() string {
	if m.loading {
		return "Loading collaborators...\n"
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var b strings.Builder

	// Header
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Primary)

	b.WriteString(titleStyle.Render("👥 Collaborator Management"))
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

	if m.viewMode == viewModeByRepo {
		b.WriteString(activeTab.Render("[1] By Repository"))
	} else {
		b.WriteString(inactiveTab.Render("[1] By Repository"))
	}
	b.WriteString("  ")
	if m.viewMode == viewModeByUser {
		b.WriteString(activeTab.Render("[2] By User"))
	} else {
		b.WriteString(inactiveTab.Render("[2] By User"))
	}
	b.WriteString("\n\n")

	// Content based on view mode
	switch m.viewMode {
	case viewModeByRepo:
		b.WriteString(m.renderByRepo())
	case viewModeByUser:
		b.WriteString(m.renderByUser())
	}

	// Help
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	b.WriteString(helpStyle.Render("↑/↓: navigate | 1/2: switch view | ?: help | q: quit"))

	return b.String()
}

func renderHelp(b *strings.Builder) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Primary)
	b.WriteString(titleStyle.Render("Keybindings"))
	b.WriteString("\n\n")

	bindings := [][2]string{
		{"j/k, up/down", "move the cursor"},
		{"g / G", "jump to top / bottom"},
		{"1 / 2", "switch view: by repository / by user"},
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

func (m Model) renderByRepo() string {
	var b strings.Builder

	b.WriteString("📦 Collaborators by Repository\n\n")

	for i, repo := range m.repos {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		collabs := m.collaborators[repo]

		statusStyle := lipgloss.NewStyle()
		if m.cursor == i {
			statusStyle = statusStyle.Bold(true).Foreground(theme.Current().Warning)
		}

		line := fmt.Sprintf("%s %s (%d collaborators):\n", cursor, repo, len(collabs))

		// Show first few collaborators
		var lineSb214 strings.Builder
		for j, collab := range collabs {
			if j >= maxPreviewItems {
				fmt.Fprintf(&lineSb214, "   ... and %d more\n", len(collabs)-maxPreviewItems)
				break
			}
			permColor := theme.Current().Success
			switch collab.Permission {
			case permissionAdmin:
				permColor = theme.Current().Error
			case permissionWrite:
				permColor = theme.Current().Warning
			}
			permStyle := lipgloss.NewStyle().Foreground(permColor)
			fmt.Fprintf(&lineSb214, "   - %s ", collab.Login)
			lineSb214.WriteString(permStyle.Render(fmt.Sprintf("[%s]", collab.Permission)))
			lineSb214.WriteString("\n")
		}
		line += lineSb214.String()

		b.WriteString(statusStyle.Render(line))
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderByUser() string {
	var b strings.Builder

	b.WriteString("👤 Cross-Repo Access by User\n\n")

	// Build user -> repos mapping
	userRepos := make(map[string][]string)
	userPerms := make(map[string]map[string]string) // user -> repo -> permission

	for repo, collabs := range m.collaborators {
		for _, collab := range collabs {
			userRepos[collab.Login] = append(userRepos[collab.Login], repo)
			if userPerms[collab.Login] == nil {
				userPerms[collab.Login] = make(map[string]string)
			}
			userPerms[collab.Login][repo] = collab.Permission
		}
	}

	// Display users
	currentIdx := 0
	for user, repos := range userRepos {
		cursor := " "
		if m.cursor == currentIdx {
			cursor = ">"
		}

		userStyle := lipgloss.NewStyle()
		if m.cursor == currentIdx {
			userStyle = userStyle.Bold(true).Foreground(theme.Current().Warning)
		}

		line := fmt.Sprintf("%s %s (access to %d repos):\n", cursor, user, len(repos))

		// Show repos with permissions
		var lineSb273 strings.Builder
		for j, repo := range repos {
			if j >= maxPreviewItems {
				fmt.Fprintf(&lineSb273, "   ... and %d more\n", len(repos)-maxPreviewItems)
				break
			}
			perm := userPerms[user][repo]
			permColor := theme.Current().Success
			switch perm {
			case permissionAdmin:
				permColor = theme.Current().Error
			case permissionWrite:
				permColor = theme.Current().Warning
			}
			permStyle := lipgloss.NewStyle().Foreground(permColor)
			fmt.Fprintf(&lineSb273, "   - %s ", repo)
			lineSb273.WriteString(permStyle.Render(fmt.Sprintf("[%s]", perm)))
			lineSb273.WriteString("\n")
		}
		line += lineSb273.String()

		b.WriteString(userStyle.Render(line))
		b.WriteString("\n")

		currentIdx++
	}

	return b.String()
}
