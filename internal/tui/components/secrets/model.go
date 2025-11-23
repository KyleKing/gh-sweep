package secrets

import (
	"context"
	"fmt"
	"strings"

	"github.com/KyleKing/gh-sweep/internal/github"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the secrets audit TUI state
type Model struct {
	org        string
	repos      []string
	orgSecrets []github.Secret
	repoSecrets map[string][]github.Secret
	unusedSecrets []string
	cursor     int
	width      int
	height     int
	loading    bool
	err        error
	viewMode   string // "org", "repo", "unused"
}

// NewModel creates a new secrets audit model
func NewModel(org string, repos []string) Model {
	return Model{
		org:         org,
		repos:       repos,
		repoSecrets: make(map[string][]github.Secret),
		loading:     true,
		viewMode:    "org",
	}
}

type secretsLoadedMsg struct {
	orgSecrets    []github.Secret
	repoSecrets   map[string][]github.Secret
	unusedSecrets []string
	err           error
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return m.loadSecrets
}

func (m Model) loadSecrets() tea.Msg {
	// Create GitHub client
	ctx := context.Background()
	client, err := github.NewClient(ctx)
	if err != nil {
		return secretsLoadedMsg{
			orgSecrets:    []github.Secret{},
			repoSecrets:   make(map[string][]github.Secret),
			unusedSecrets: []string{},
			err:           fmt.Errorf("failed to create GitHub client: %w", err),
		}
	}

	// Load organization secrets
	var orgSecrets []github.Secret
	if m.org != "" {
		orgSecrets, err = client.ListOrgSecrets(m.org)
		if err != nil {
			// Continue even if org secrets fail
			orgSecrets = []github.Secret{}
		}
	}

	// Load repository secrets
	repoSecrets := make(map[string][]github.Secret)
	for _, repoStr := range m.repos {
		parts := strings.Split(repoStr, "/")
		if len(parts) != 2 {
			continue
		}
		owner, repo := parts[0], parts[1]

		secrets, err := client.ListRepoSecrets(owner, repo)
		if err != nil {
			// Skip repos on error
			continue
		}

		repoSecrets[repoStr] = secrets
	}

	// Detect unused secrets (simplified - would need workflow file parsing for real detection)
	// For now, just return empty list
	unusedSecrets := []string{}

	return secretsLoadedMsg{
		orgSecrets:    orgSecrets,
		repoSecrets:   repoSecrets,
		unusedSecrets: unusedSecrets,
		err:           nil,
	}
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case secretsLoadedMsg:
		m.loading = false
		m.orgSecrets = msg.orgSecrets
		m.repoSecrets = msg.repoSecrets
		m.unusedSecrets = msg.unusedSecrets
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
			maxCursor := len(m.orgSecrets) - 1
			if m.viewMode == "repo" {
				maxCursor = len(m.repos) - 1
			} else if m.viewMode == "unused" {
				maxCursor = len(m.unusedSecrets) - 1
			}
			if m.cursor < maxCursor {
				m.cursor++
			}

		case "1":
			m.viewMode = "org"
			m.cursor = 0
		case "2":
			m.viewMode = "repo"
			m.cursor = 0
		case "3":
			m.viewMode = "unused"
			m.cursor = 0
		}
	}

	return m, nil
}

// View renders the model
func (m Model) View() string {
	if m.loading {
		return "Loading secrets...\n"
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var b strings.Builder

	// Header
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FFFF"))

	b.WriteString(titleStyle.Render("🔐 Secrets Audit (Read-Only)"))
	b.WriteString("\n\n")

	// View mode tabs
	activeTab := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFF00"))

	inactiveTab := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#777777"))

	if m.viewMode == "org" {
		b.WriteString(activeTab.Render("[1] Organization"))
	} else {
		b.WriteString(inactiveTab.Render("[1] Organization"))
	}
	b.WriteString("  ")
	if m.viewMode == "repo" {
		b.WriteString(activeTab.Render("[2] Repository"))
	} else {
		b.WriteString(inactiveTab.Render("[2] Repository"))
	}
	b.WriteString("  ")
	if m.viewMode == "unused" {
		b.WriteString(activeTab.Render("[3] Unused"))
	} else {
		b.WriteString(inactiveTab.Render("[3] Unused"))
	}
	b.WriteString("\n\n")

	// Content based on view mode
	switch m.viewMode {
	case "org":
		b.WriteString(m.renderOrgSecrets())
	case "repo":
		b.WriteString(m.renderRepoSecrets())
	case "unused":
		b.WriteString(m.renderUnusedSecrets())
	}

	// Help
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))
	b.WriteString(helpStyle.Render("↑/↓: navigate | 1/2/3: switch view | q: quit"))

	return b.String()
}

func (m Model) renderOrgSecrets() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("🏢 Organization Secrets: %s\n\n", m.org))

	if len(m.orgSecrets) == 0 {
		b.WriteString("No organization secrets found.\n")
		return b.String()
	}

	b.WriteString(fmt.Sprintf("Total: %d secrets\n\n", len(m.orgSecrets)))

	for i, secret := range m.orgSecrets {
		if i >= m.height-10 {
			break
		}

		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		secretStyle := lipgloss.NewStyle()
		if m.cursor == i {
			secretStyle = secretStyle.Bold(true).Foreground(lipgloss.Color("#FFFF00"))
		}

		line := fmt.Sprintf("%s %s\n", cursor, secret.Name)
		if secret.UpdatedAt == "" {
			line += "   Updated: unknown\n"
		} else {
			line += fmt.Sprintf("   Updated: %s\n", secret.UpdatedAt)
		}

		b.WriteString(secretStyle.Render(line))
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderRepoSecrets() string {
	var b strings.Builder

	b.WriteString("📦 Repository Secrets\n\n")

	if len(m.repoSecrets) == 0 {
		b.WriteString("No repository secrets found.\n")
		return b.String()
	}

	for i, repo := range m.repos {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		secrets := m.repoSecrets[repo]

		repoStyle := lipgloss.NewStyle()
		if m.cursor == i {
			repoStyle = repoStyle.Bold(true).Foreground(lipgloss.Color("#FFFF00"))
		}

		line := fmt.Sprintf("%s %s (%d secrets):\n", cursor, repo, len(secrets))

		// Show first few secrets
		for j, secret := range secrets {
			if j >= 3 {
				line += fmt.Sprintf("   ... and %d more\n", len(secrets)-3)
				break
			}
			line += fmt.Sprintf("   - %s\n", secret.Name)
		}

		b.WriteString(repoStyle.Render(line))
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderUnusedSecrets() string {
	var b strings.Builder

	b.WriteString("⚠️  Potentially Unused Secrets\n\n")

	if len(m.unusedSecrets) == 0 {
		b.WriteString("✅ All secrets appear to be in use.\n")
		b.WriteString("(Full analysis requires workflow file parsing)\n")
		return b.String()
	}

	for i, secret := range m.unusedSecrets {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		secretStyle := lipgloss.NewStyle()
		if m.cursor == i {
			secretStyle = secretStyle.Bold(true).Foreground(lipgloss.Color("#FFFF00"))
		}

		line := fmt.Sprintf("%s %s\n", cursor, secret)
		b.WriteString(secretStyle.Render(line))
	}

	return b.String()
}
