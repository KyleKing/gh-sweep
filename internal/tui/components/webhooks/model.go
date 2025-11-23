package webhooks

import (
	"context"
	"fmt"
	"strings"

	"github.com/KyleKing/gh-sweep/internal/github"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the webhook management TUI state
type Model struct {
	repos    []string
	webhooks map[string][]github.Webhook              // repo -> webhooks
	health   map[string]map[int]github.WebhookHealth // repo -> webhook ID -> health
	cursor   int
	width    int
	height   int
	loading  bool
	err      error
}

// NewModel creates a new webhook management model
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

// Init initializes the model
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

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

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

// View renders the model
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
		Foreground(lipgloss.Color("#00FFFF"))

	b.WriteString(titleStyle.Render("🔔 Webhooks"))
	b.WriteString("\n\n")

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
				repoStyle = repoStyle.Bold(true).Foreground(lipgloss.Color("#FFFF00"))
			}

			webhooks := m.webhooks[repo]
			line := fmt.Sprintf("%s %s (%d webhooks):\n", cursor, repo, len(webhooks))

			// Show first few webhooks
			for j, webhook := range webhooks {
				if j >= 3 {
					line += fmt.Sprintf("   ... and %d more\n", len(webhooks)-3)
					break
				}

				line += fmt.Sprintf("   ID: %d | %s\n", webhook.ID, webhook.URL)
				line += fmt.Sprintf("   Events: %s\n", strings.Join(webhook.Events, ", "))

				// Add health metrics if available
				if repoHealth, ok := m.health[repo]; ok {
					if health, ok := repoHealth[webhook.ID]; ok {
						statusColor := "#00FF00" // green for healthy
						if health.SuccessRate < 80 {
							statusColor = "#FF0000" // red for unhealthy
						} else if health.SuccessRate < 95 {
							statusColor = "#FFFF00" // yellow for warning
						}

						healthStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor))
						healthLine := fmt.Sprintf("   Health: %.1f%% success | Avg: %dms | Total: %d\n",
							health.SuccessRate,
							health.AvgDuration,
							health.TotalDeliveries)
						line += healthStyle.Render(healthLine)
					}
				}
			}

			b.WriteString(repoStyle.Render(line))
			b.WriteString("\n")
		}
	}

	// Help
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))
	b.WriteString(helpStyle.Render("↑/↓: navigate | q: quit"))

	return b.String()
}
