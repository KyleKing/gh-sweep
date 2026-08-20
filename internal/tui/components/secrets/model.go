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
	"github.com/KyleKing/gh-sweep/internal/tui/scroll"
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

	var header strings.Builder

	// Header
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Primary)

	header.WriteString(titleStyle.Render("🔐 Secrets Audit (Read-Only)"))
	header.WriteString("\n\n")

	// View mode tabs
	activeTab := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Warning)

	inactiveTab := lipgloss.NewStyle().
		Foreground(theme.Current().Muted)

	if m.viewMode == viewModeOrg {
		header.WriteString(activeTab.Render("[1] Organization"))
	} else {
		header.WriteString(inactiveTab.Render("[1] Organization"))
	}
	header.WriteString("  ")
	if m.viewMode == viewModeRepo {
		header.WriteString(activeTab.Render("[2] Repository"))
	} else {
		header.WriteString(inactiveTab.Render("[2] Repository"))
	}
	header.WriteString("  ")
	if m.viewMode == viewModeUnused {
		header.WriteString(activeTab.Render("[3] Unused"))
	} else {
		header.WriteString(inactiveTab.Render("[3] Unused"))
	}
	header.WriteString("\n\n")

	if m.showHelp {
		return renderHelp(&header)
	}

	var footer strings.Builder
	footer.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	footer.WriteString(helpStyle.Render("↑/↓: navigate | 1/2/3: switch view | ?: help | q: quit"))

	preamble, lines, cursorLine := m.currentContent()
	header.WriteString(preamble)

	headerLines := strings.Count(header.String(), "\n")
	footerLines := strings.Count(footer.String(), "\n")
	available := m.height - headerLines - footerLines

	renderSecretList(&header, lines, cursorLine, available)
	header.WriteString(footer.String())

	return header.String()
}

// currentContent returns the header text above the list for the active view
// mode, the list rendered as one line per row, and the row index the cursor
// is on among those lines.
func (m Model) currentContent() (string, []string, int) {
	switch m.viewMode {
	case viewModeRepo:
		return m.repoSecretsContent()
	case viewModeUnused:
		return m.unusedSecretsContent()
	default:
		return m.orgSecretsContent()
	}
}

// renderSecretList windows lines to the visible range around cursorLine,
// adding "more above"/"more below" hints when the list overflows available.
func renderSecretList(b *strings.Builder, lines []string, cursorLine, available int) {
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

// orgSecretsContent renders each org secret, and its "Updated" line, as its
// own two lines so the caller can window by line rather than by item.
func (m Model) orgSecretsContent() (string, []string, int) {
	var preamble strings.Builder

	fmt.Fprintf(&preamble, "🏢 Organization Secrets: %s\n\n", m.org)

	if len(m.orgSecrets) == 0 {
		preamble.WriteString("No organization secrets found.\n")
		return preamble.String(), nil, 0
	}

	fmt.Fprintf(&preamble, "Total: %d secrets\n\n", len(m.orgSecrets))

	var lines []string

	cursorLine := 0

	for i, secret := range m.orgSecrets {
		if i == m.cursor {
			cursorLine = len(lines)
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

		lines = append(lines, strings.Split(secretStyle.Render(line), "\n")...)
	}

	return preamble.String(), lines, cursorLine
}

func (m Model) repoSecretsContent() (string, []string, int) {
	var preamble strings.Builder

	preamble.WriteString("📦 Repository Secrets\n\n")

	if len(m.repoSecrets) == 0 {
		preamble.WriteString("No repository secrets found.\n")
		return preamble.String(), nil, 0
	}

	var lines []string

	cursorLine := 0

	for i, repo := range m.repos {
		if i == m.cursor {
			cursorLine = len(lines)
		}

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

		lines = append(lines, strings.Split(repoStyle.Render(line), "\n")...)
	}

	return preamble.String(), lines, cursorLine
}

func (m Model) unusedSecretsContent() (string, []string, int) {
	var preamble strings.Builder

	preamble.WriteString("⚠️  Potentially Unused Secrets\n\n")
	preamble.WriteString("No ${{ secrets.NAME }} reference found in any scanned repo's workflow files.\n\n")

	if len(m.unusedSecrets) == 0 {
		preamble.WriteString("✅ All secrets appear to be in use.\n")

		return preamble.String(), nil, 0
	}

	var lines []string

	cursorLine := 0

	for i, secret := range m.unusedSecrets {
		if i == m.cursor {
			cursorLine = len(lines)
		}

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

		lines = append(lines, strings.Split(secretStyle.Render(line), "\n")...)
	}

	return preamble.String(), lines, cursorLine
}
