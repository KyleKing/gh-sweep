package secrets

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/tui/theme"
)

// Model represents the secrets audit TUI state.
type Model struct {
	org           string
	repos         []string
	orgSecrets    []github.Secret
	repoSecrets   map[string][]github.Secret
	unusedSecrets []string
	cursor        int
	width         int
	height        int
	loading       bool
	err           error
	viewMode      string // "org", "repo", "unused"
	showHelp      bool
}

// NewModel creates a new secrets audit model.
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

// Init initializes the model.
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

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
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
			maxCursor := len(m.orgSecrets) - 1
			switch m.viewMode {
			case "repo":
				maxCursor = len(m.repos) - 1
			case "unused":
				maxCursor = len(m.unusedSecrets) - 1
			}
			if maxCursor >= 0 {
				m.cursor = maxCursor
			}

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			maxCursor := len(m.orgSecrets) - 1
			switch m.viewMode {
			case "repo":
				maxCursor = len(m.repos) - 1
			case "unused":
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

// View renders the model.
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
		Foreground(theme.Current().Primary)

	b.WriteString(titleStyle.Render("🔐 Secrets Audit (Read-Only)"))
	b.WriteString("\n\n")

	// View mode tabs
	activeTab := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Warning)

	inactiveTab := lipgloss.NewStyle().
		Foreground(theme.Current().Muted)

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

	if m.showHelp {
		return m.renderHelp(&b)
	}

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
	helpStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	b.WriteString(helpStyle.Render("↑/↓: navigate | 1/2/3: switch view | ?: help | q: quit"))

	return b.String()
}

func (m Model) renderHelp(b *strings.Builder) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Primary)
	b.WriteString(titleStyle.Render("Keybindings"))
	b.WriteString("\n\n")

	bindings := [][2]string{
		{"j/k, up/down", "move the cursor"},
		{"g / G", "jump to top / bottom"},
		{"1/2/3", "switch view: organization / repository / unused"},
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

func (m Model) renderOrgSecrets() string {
	var b strings.Builder

	fmt.Fprintf(&b, "🏢 Organization Secrets: %s\n\n", m.org)

	if len(m.orgSecrets) == 0 {
		b.WriteString("No organization secrets found.\n")
		return b.String()
	}

	fmt.Fprintf(&b, "Total: %d secrets\n\n", len(m.orgSecrets))

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
			secretStyle = secretStyle.Bold(true).Foreground(theme.Current().Warning)
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
			repoStyle = repoStyle.Bold(true).Foreground(theme.Current().Warning)
		}

		line := fmt.Sprintf("%s %s (%d secrets):\n", cursor, repo, len(secrets))

		// Show first few secrets
		var lineSb288 strings.Builder
		for j, secret := range secrets {
			if j >= 3 {
				fmt.Fprintf(&lineSb288, "   ... and %d more\n", len(secrets)-3)
				break
			}
			fmt.Fprintf(&lineSb288, "   - %s\n", secret.Name)
		}
		line += lineSb288.String()

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
			secretStyle = secretStyle.Bold(true).Foreground(theme.Current().Warning)
		}

		line := fmt.Sprintf("%s %s\n", cursor, secret)
		b.WriteString(secretStyle.Render(line))
	}

	return b.String()
}
