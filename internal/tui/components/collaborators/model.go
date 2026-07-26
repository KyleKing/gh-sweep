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

// Model represents the collaborator management TUI state.
type Model struct {
	repos         []string
	collaborators map[string][]github.Collaborator
	cursor        int
	width         int
	height        int
	loading       bool
	err           error
	viewMode      string // "byrepo", "byuser"
}

// NewModel creates a new collaborator management model.
func NewModel(repos []string) Model {
	return Model{
		repos:         repos,
		collaborators: make(map[string][]github.Collaborator),
		loading:       true,
		viewMode:      "byrepo",
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
		if len(parts) != 2 {
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
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			maxCursor := len(m.repos) - 1
			if m.viewMode == "byuser" {
				maxCursor = m.getTotalCollaborators() - 1
			}
			if m.cursor < maxCursor {
				m.cursor++
			}

		case "1":
			m.viewMode = "byrepo"
			m.cursor = 0
		case "2":
			m.viewMode = "byuser"
			m.cursor = 0
		}
	}

	return m, nil
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

	// View mode tabs
	activeTab := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Warning)

	inactiveTab := lipgloss.NewStyle().
		Foreground(theme.Current().Muted)

	if m.viewMode == "byrepo" {
		b.WriteString(activeTab.Render("[1] By Repository"))
	} else {
		b.WriteString(inactiveTab.Render("[1] By Repository"))
	}
	b.WriteString("  ")
	if m.viewMode == "byuser" {
		b.WriteString(activeTab.Render("[2] By User"))
	} else {
		b.WriteString(inactiveTab.Render("[2] By User"))
	}
	b.WriteString("\n\n")

	// Content based on view mode
	switch m.viewMode {
	case "byrepo":
		b.WriteString(m.renderByRepo())
	case "byuser":
		b.WriteString(m.renderByUser())
	}

	// Help
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	b.WriteString(helpStyle.Render("↑/↓: navigate | 1/2: switch view | q: quit"))

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
			if j >= 3 {
				fmt.Fprintf(&lineSb214, "   ... and %d more\n", len(collabs)-3)
				break
			}
			permColor := theme.Current().Success
			switch collab.Permission {
			case "admin":
				permColor = theme.Current().Error
			case "write":
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
			if j >= 3 {
				fmt.Fprintf(&lineSb273, "   ... and %d more\n", len(repos)-3)
				break
			}
			perm := userPerms[user][repo]
			permColor := theme.Current().Success
			switch perm {
			case "admin":
				permColor = theme.Current().Error
			case "write":
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
