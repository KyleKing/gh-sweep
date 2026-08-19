// Package secrets is the TUI view that audits organization and repository
// secrets and flags ones that appear unused.
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

const (
	repoPartsCount  = 2
	helpKeyWidth    = 16
	maxPreviewItems = 3

	viewModeOrg    = "org"
	viewModeRepo   = "repo"
	viewModeUnused = "unused"
)

// Model represents the secrets audit TUI state.
type Model struct {
	org           string
	repos         []string
	orgSecrets    []github.Secret
	repoSecrets   map[string][]github.Secret
	unusedSecrets []github.SecretUsage
	cursor        int
	width         int
	height        int
	loading       bool
	err           error
	viewMode      string // viewModeOrg, viewModeRepo, viewModeUnused
	showHelp      bool
}

// NewModel creates a new secrets audit model.
func NewModel(org string, repos []string) Model {
	return Model{
		org:         org,
		repos:       repos,
		repoSecrets: make(map[string][]github.Secret),
		loading:     true,
		viewMode:    viewModeOrg,
	}
}

type secretsLoadedMsg struct {
	orgSecrets    []github.Secret
	repoSecrets   map[string][]github.Secret
	unusedSecrets []github.SecretUsage
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
			unusedSecrets: []github.SecretUsage{},
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

	// Load repository secrets and scan each repo's workflows for references
	repoSecrets := make(map[string][]github.Secret)
	workflowRefs := make(map[string][]string)

	for _, repoStr := range m.repos {
		parts := strings.Split(repoStr, "/")
		if len(parts) != repoPartsCount {
			continue
		}
		owner, repo := parts[0], parts[1]

		secrets, err := client.ListRepoSecrets(owner, repo)
		if err != nil {
			// Skip repos on error
			continue
		}

		repoSecrets[repoStr] = secrets

		refs, err := client.ScanRepoSecretRefs(owner, repo)
		if err != nil {
			// A repo's workflows failing to scan doesn't invalidate refs
			// already found in other repos.
			continue
		}

		for name, paths := range refs {
			workflowRefs[name] = append(workflowRefs[name], paths...)
		}
	}

	allSecrets := make([]github.Secret, 0, len(orgSecrets)+len(repoSecrets))
	allSecrets = append(allSecrets, orgSecrets...)
	for _, secrets := range repoSecrets {
		allSecrets = append(allSecrets, secrets...)
	}

	unusedSecrets := make([]github.SecretUsage, 0)
	for _, usage := range github.DetectUnusedSecrets(allSecrets, workflowRefs) {
		if usage.Unused {
			unusedSecrets = append(unusedSecrets, usage)
		}
	}

	return secretsLoadedMsg{
		orgSecrets:    orgSecrets,
		repoSecrets:   repoSecrets,
		unusedSecrets: unusedSecrets,
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

	case secretsLoadedMsg:
		m.loading = false
		m.orgSecrets = msg.orgSecrets
		m.repoSecrets = msg.repoSecrets
		m.unusedSecrets = msg.unusedSecrets
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

	case "1", "2", "3":
		m.viewMode = viewModeForKey(msg.String())
		m.cursor = 0
	}

	return m, nil
}

func viewModeForKey(key string) string {
	switch key {
	case "2":
		return viewModeRepo
	case "3":
		return viewModeUnused
	default:
		return viewModeOrg
	}
}

func (m Model) maxCursor() int {
	switch m.viewMode {
	case viewModeRepo:
		return len(m.repos) - 1
	case viewModeUnused:
		return len(m.unusedSecrets) - 1
	default:
		return len(m.orgSecrets) - 1
	}
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

	if m.viewMode == viewModeOrg {
		b.WriteString(activeTab.Render("[1] Organization"))
	} else {
		b.WriteString(inactiveTab.Render("[1] Organization"))
	}
	b.WriteString("  ")
	if m.viewMode == viewModeRepo {
		b.WriteString(activeTab.Render("[2] Repository"))
	} else {
		b.WriteString(inactiveTab.Render("[2] Repository"))
	}
	b.WriteString("  ")
	if m.viewMode == viewModeUnused {
		b.WriteString(activeTab.Render("[3] Unused"))
	} else {
		b.WriteString(inactiveTab.Render("[3] Unused"))
	}
	b.WriteString("\n\n")

	if m.showHelp {
		return renderHelp(&b)
	}

	// Content based on view mode
	switch m.viewMode {
	case viewModeOrg:
		b.WriteString(m.renderOrgSecrets())
	case viewModeRepo:
		b.WriteString(m.renderRepoSecrets())
	case viewModeUnused:
		b.WriteString(m.renderUnusedSecrets())
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
		{"1/2/3", "switch view: organization / repository / unused"},
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
			if j >= maxPreviewItems {
				fmt.Fprintf(&lineSb288, "   ... and %d more\n", len(secrets)-maxPreviewItems)
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
	b.WriteString("No ${{ secrets.NAME }} reference found in any scanned repo's workflow files.\n\n")

	if len(m.unusedSecrets) == 0 {
		b.WriteString("✅ All secrets appear to be in use.\n")

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

		line := fmt.Sprintf("%s %s (%s)\n", cursor, secret.Name, secret.Scope)
		if secret.Repository != "" {
			line = fmt.Sprintf("%s %s (%s, %s)\n", cursor, secret.Name, secret.Scope, secret.Repository)
		}

		b.WriteString(secretStyle.Render(line))
	}

	return b.String()
}
