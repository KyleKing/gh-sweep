package webhooks

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/tui/theme"
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
		parts := strings.Split(repoStr, "/")
		if len(parts) != 2 {
			continue
		}
		owner, repo := parts[0], parts[1]

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

// Update handles messages.
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

	var b strings.Builder

	// Header
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Primary)

	b.WriteString(titleStyle.Render("🔔 Webhooks"))
	b.WriteString("\n\n")

	if m.showHelp {
		return m.renderHelp(&b)
	}

	// Webhook list by repository
	if len(m.webhooks) == 0 {
		b.WriteString("No webhooks found.\n")
	} else {
		for i, repo := range m.repos {
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

			// Show first few webhooks
			var lineSb172 strings.Builder
			for j, webhook := range webhooks {
				if j >= 3 {
					fmt.Fprintf(&lineSb172, "   ... and %d more\n", len(webhooks)-3)
					break
				}

				fmt.Fprintf(&lineSb172, "   ID: %d | %s\n", webhook.ID, webhook.URL)
				fmt.Fprintf(&lineSb172, "   Events: %s\n", strings.Join(webhook.Events, ", "))

				// Add health metrics if available
				if repoHealth, ok := m.health[repo]; ok {
					if health, ok := repoHealth[webhook.ID]; ok {
						statusColor := theme.Current().Success
						if health.SuccessRate < 80 {
							statusColor = theme.Current().Error
						} else if health.SuccessRate < 95 {
							statusColor = theme.Current().Warning
						}

						healthStyle := lipgloss.NewStyle().Foreground(statusColor)
						healthLine := fmt.Sprintf(
							"   Health: %.1f%% success | Avg: %dms | Total: %d\n",
							health.SuccessRate,
							health.AvgDuration,
							health.TotalDeliveries,
						)
						lineSb172.WriteString(healthStyle.Render(healthLine))
					}
				}
			}
			line += lineSb172.String()

			b.WriteString(repoStyle.Render(line))
			b.WriteString("\n")
		}
	}

	// Help
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	b.WriteString(helpStyle.Render("↑/↓: navigate | ?: help | q: quit"))

	return b.String()
}

func (m Model) renderHelp(b *strings.Builder) string {
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
		keyStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Warning).Width(16)
		fmt.Fprintf(b, "%s %s\n", keyStyle.Render(binding[0]), binding[1])
	}

	b.WriteString("\nPress '?' or 'esc' to close\n")

	return b.String()
}
