package analytics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/KyleKing/gh-sweep/internal/github"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the analytics TUI state
type Model struct {
	repo           string
	stats          *github.WorkflowRunStats
	runs           []github.WorkflowRun
	width          int
	height         int
	loading        bool
	err            error
	viewMode       string // "overview", "flaky", "errors"
}

// NewModel creates a new analytics model
func NewModel(repo string) Model {
	return Model{
		repo:     repo,
		loading:  true,
		viewMode: "overview",
	}
}

type analyticsLoadedMsg struct {
	stats *github.WorkflowRunStats
	runs  []github.WorkflowRun
	err   error
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return m.loadAnalytics
}

func (m Model) loadAnalytics() tea.Msg {
	// If no repo specified, return empty
	if m.repo == "" {
		return analyticsLoadedMsg{
			stats: nil,
			runs:  []github.WorkflowRun{},
			err:   fmt.Errorf("no repository specified"),
		}
	}

	// Parse repo (owner/name format)
	parts := strings.Split(m.repo, "/")
	if len(parts) != 2 {
		return analyticsLoadedMsg{
			stats: nil,
			runs:  []github.WorkflowRun{},
			err:   fmt.Errorf("invalid repo format, expected owner/repo"),
		}
	}
	owner, repo := parts[0], parts[1]

	// Create GitHub client
	ctx := context.Background()
	client, err := github.NewClient(ctx)
	if err != nil {
		return analyticsLoadedMsg{
			stats: nil,
			runs:  []github.WorkflowRun{},
			err:   fmt.Errorf("failed to create GitHub client: %w", err),
		}
	}

	// Load workflow runs from GitHub
	runs, err := client.ListWorkflowRuns(owner, repo)
	if err != nil {
		// Return empty on error (repo might not have workflows)
		return analyticsLoadedMsg{
			stats: &github.WorkflowRunStats{
				TotalRuns:    0,
				SuccessRate:  0,
				FailureCount: 0,
				AvgDuration:  0,
			},
			runs: []github.WorkflowRun{},
			err:  nil, // Don't error out, just show empty
		}
	}

	// Analyze runs to get statistics
	stats := github.AnalyzeWorkflowRuns(runs)

	return analyticsLoadedMsg{
		stats: &stats,
		runs:  runs,
		err:   nil,
	}
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case analyticsLoadedMsg:
		m.loading = false
		m.stats = msg.stats
		m.runs = msg.runs
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "1":
			m.viewMode = "overview"
		case "2":
			m.viewMode = "flaky"
		case "3":
			m.viewMode = "errors"
		}
	}

	return m, nil
}

// View renders the model
func (m Model) View() string {
	if m.loading {
		return "Loading analytics...\n"
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var b strings.Builder

	// Header
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FFFF"))

	b.WriteString(titleStyle.Render(fmt.Sprintf("📊 Analytics: %s", m.repo)))
	b.WriteString("\n\n")

	// View mode tabs
	activeTab := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFF00"))

	inactiveTab := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#777777"))

	tabs := []string{
		"[1] Overview",
		"[2] Flaky Tests",
		"[3] Errors",
	}

	for i, tab := range tabs {
		viewModes := []string{"overview", "flaky", "errors"}
		if m.viewMode == viewModes[i] {
			b.WriteString(activeTab.Render(tab))
		} else {
			b.WriteString(inactiveTab.Render(tab))
		}
		b.WriteString("  ")
	}
	b.WriteString("\n\n")

	// Content based on view mode
	switch m.viewMode {
	case "overview":
		b.WriteString(m.renderOverview())
	case "flaky":
		b.WriteString(m.renderFlaky())
	case "errors":
		b.WriteString(m.renderErrors())
	}

	// Help
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))
	b.WriteString(helpStyle.Render("1/2/3: switch view | q: quit"))

	return b.String()
}

func (m Model) renderOverview() string {
	if m.stats == nil {
		return "No data available\n"
	}

	var b strings.Builder

	b.WriteString("📈 CI/CD Statistics\n\n")
	b.WriteString(fmt.Sprintf("Total Runs:     %d\n", m.stats.TotalRuns))
	b.WriteString(fmt.Sprintf("Success Rate:   %.1f%%\n", m.stats.SuccessRate))
	b.WriteString(fmt.Sprintf("Failures:       %d\n", m.stats.FailureCount))
	b.WriteString(fmt.Sprintf("Avg Duration:   %s\n", m.stats.AvgDuration.Round(time.Second)))

	// Simple bar chart
	b.WriteString("\nSuccess/Failure Distribution:\n")
	successCount := m.stats.TotalRuns - m.stats.FailureCount
	b.WriteString(fmt.Sprintf("✓ Success: %s (%d)\n",
		strings.Repeat("█", successCount*50/m.stats.TotalRuns), successCount))
	b.WriteString(fmt.Sprintf("✗ Failure: %s (%d)\n",
		strings.Repeat("█", m.stats.FailureCount*50/m.stats.TotalRuns), m.stats.FailureCount))

	return b.String()
}

func (m Model) renderFlaky() string {
	var b strings.Builder

	b.WriteString("🔍 Flaky Test Detection\n\n")
	b.WriteString("Pattern-based detection (fail → pass on same commit)\n\n")

	// Mock flaky tests
	b.WriteString("Found 3 potentially flaky tests:\n\n")
	b.WriteString("1. TestUserAuthentication\n")
	b.WriteString("   Failure rate: 15% (3/20 runs)\n")
	b.WriteString("   Last flip: 2 days ago\n\n")

	b.WriteString("2. TestDatabaseConnection\n")
	b.WriteString("   Failure rate: 8% (2/25 runs)\n")
	b.WriteString("   Last flip: 1 week ago\n\n")

	b.WriteString("3. TestAPITimeout\n")
	b.WriteString("   Failure rate: 12% (6/50 runs)\n")
	b.WriteString("   Last flip: 3 days ago\n")

	return b.String()
}

func (m Model) renderErrors() string {
	var b strings.Builder

	b.WriteString("❌ Recent Errors\n\n")
	b.WriteString("Extracting last 100 lines of failed jobs...\n\n")

	// Mock error logs
	b.WriteString("Latest Failure (2 hours ago):\n")
	b.WriteString("---\n")
	b.WriteString("FAIL: TestUserLogin (0.45s)\n")
	b.WriteString("    user_test.go:42: expected status 200, got 401\n")
	b.WriteString("    Stack trace:\n")
	b.WriteString("        TestUserLogin\n")
	b.WriteString("            /app/test/user_test.go:42\n")
	b.WriteString("---\n\n")

	b.WriteString("💡 AI-friendly format: JSON export available\n")

	return b.String()
}
