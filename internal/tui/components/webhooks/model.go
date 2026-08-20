// Package webhooks is the TUI view that lists repository webhooks and their
// recent delivery health.
package webhooks

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
	repoPartsCount       = 2
	helpKeyWidth         = 16
	maxWebhooksShown     = 3
	unhealthySuccessRate = 80
	degradedSuccessRate  = 95
)

// Model represents the webhook management TUI state.
type Model struct {
	repos    []string
	webhooks map[string][]github.Webhook             // repo -> webhooks
	health   map[string]map[int]github.WebhookHealth // repo -> webhook ID -> health
	cursor   int
	width    int
	height   int
	loading  bool
	err      error
	showHelp bool
}

// NewModel creates a new webhook management model.
func NewModel(repos []string) Model {
	return Model{
		repos:    repos,
		webhooks: make(map[string][]github.Webhook),
		health:   make(map[string]map[int]github.WebhookHealth),
		loading:  true,
	}
}

type webhooksLoadedMsg struct {
	webhooks map[string][]github.Webhook
	health   map[string]map[int]github.WebhookHealth
	err      error
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return m.loadWebhooks
}

func (m Model) loadWebhooks() tea.Msg {
	// Create GitHub client
	ctx := context.Background()
	client, err := github.NewClient(ctx)
	if err != nil {
		return webhooksLoadedMsg{
			webhooks: make(map[string][]github.Webhook),
			health:   make(map[string]map[int]github.WebhookHealth),
			err:      fmt.Errorf("failed to create GitHub client: %w", err),
		}
	}

	// Load webhooks for each repo
	webhooks := make(map[string][]github.Webhook)
	health := make(map[string]map[int]github.WebhookHealth)

	for _, repoStr := range m.repos {
		owner, repo, ok := splitRepo(repoStr)
		if !ok {
			continue
		}

		// Load webhooks
		repoWebhooks, err := client.ListWebhooks(owner, repo)
		if err != nil {
			// Skip repos on error
			continue
		}
		webhooks[repoStr] = repoWebhooks

		// Load health metrics for each webhook
		repoHealth := make(map[int]github.WebhookHealth)
		for _, webhook := range repoWebhooks {
			deliveries, err := client.ListWebhookDeliveries(owner, repo, webhook.ID)
			if err != nil {
				// Skip health metrics on error
				continue
			}
			webhookHealth := github.AnalyzeWebhookHealth(deliveries)
			repoHealth[webhook.ID] = webhookHealth
		}
		health[repoStr] = repoHealth
	}

	return webhooksLoadedMsg{
		webhooks: webhooks,
		health:   health,
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

	case webhooksLoadedMsg:
		m.loading = false
		m.webhooks = msg.webhooks
		m.health = msg.health
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
		if len(m.repos) > 0 {
			m.cursor = len(m.repos) - 1
		}

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.repos)-1 {
			m.cursor++
		}
	}

	return m, nil
}

// View renders the model.
func (m Model) View() string {
	if m.loading {
		return "Loading webhooks...\n"
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var header strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Primary)

	header.WriteString(titleStyle.Render("🔔 Webhooks"))
	header.WriteString("\n\n")

	if m.showHelp {
		return renderHelp(&header)
	}

	var footer strings.Builder
	footer.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	footer.WriteString(helpStyle.Render("↑/↓: navigate | ?: help | q: quit"))

	headerLines := strings.Count(header.String(), "\n")
	footerLines := strings.Count(footer.String(), "\n")
	available := m.height - headerLines - footerLines

	var b strings.Builder
	b.WriteString(header.String())

	if len(m.webhooks) == 0 {
		b.WriteString("No webhooks found.\n")
	} else {
		m.renderWebhookList(&b, available)
	}

	b.WriteString(footer.String())

	return b.String()
}

func (m Model) renderWebhookList(b *strings.Builder, available int) {
	lines, cursorLine := m.buildWebhookLines()

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

// buildWebhookLines renders each repository's block of webhook lines as
// entries in a flat slice so the caller can window by line rather than by
// repository, returning the lines and the cursor row's index among them.
func (m Model) buildWebhookLines() ([]string, int) {
	var body strings.Builder

	cursorLine := 0

	for i, repo := range m.repos {
		if i == m.cursor {
			cursorLine = strings.Count(body.String(), "\n")
		}

		m.renderWebhookRepo(&body, repo, i)
	}

	lines := strings.Split(body.String(), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return lines, cursorLine
}

func (m Model) renderWebhookRepo(b *strings.Builder, repo string, i int) {
	cursor := " "
	if m.cursor == i {
		cursor = ">"
	}

	repoStyle := lipgloss.NewStyle()
	if m.cursor == i {
		repoStyle = repoStyle.Bold(true).Foreground(theme.Current().Warning)
	}

	webhooks := m.webhooks[repo]
	line := fmt.Sprintf("%s %s (%d webhooks):\n", cursor, repo, len(webhooks))
	line += m.renderRepoWebhooks(repo, webhooks)

	b.WriteString(repoStyle.Render(line))
	b.WriteString("\n")
}

func (m Model) renderRepoWebhooks(repo string, webhooks []github.Webhook) string {
	var b strings.Builder

	for j, webhook := range webhooks {
		if j >= maxWebhooksShown {
			fmt.Fprintf(&b, "   ... and %d more\n", len(webhooks)-maxWebhooksShown)
			break
		}

		fmt.Fprintf(&b, "   ID: %d | %s\n", webhook.ID, webhook.URL)
		fmt.Fprintf(&b, "   Events: %s\n", strings.Join(webhook.Events, ", "))

		if health, ok := m.health[repo][webhook.ID]; ok {
			b.WriteString(renderWebhookHealth(health))
		}
	}

	return b.String()
}

func renderWebhookHealth(health github.WebhookHealth) string {
	statusColor := theme.Current().Success
	if health.SuccessRate < unhealthySuccessRate {
		statusColor = theme.Current().Error
	} else if health.SuccessRate < degradedSuccessRate {
		statusColor = theme.Current().Warning
	}

	healthStyle := lipgloss.NewStyle().Foreground(statusColor)

	return healthStyle.Render(fmt.Sprintf(
		"   Health: %.1f%% success | Avg: %dms | Total: %d\n",
		health.SuccessRate,
		health.AvgDuration,
		health.TotalDeliveries,
	))
}

func renderHelp(b *strings.Builder) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Primary)
	b.WriteString(titleStyle.Render("Keybindings"))
	b.WriteString("\n\n")

	bindings := [][2]string{
		{"j/k, up/down", "move the cursor"},
		{"g / G", "jump to top / bottom"},
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
