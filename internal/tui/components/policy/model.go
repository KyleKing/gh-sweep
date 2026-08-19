// Package policy is the TUI view for the gh-sweep policy command: it shows
// drift between a config.PolicyConfig and live repos, and applies it on demand.
package policy

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/KyleKing/gh-sweep/internal/config"
	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/policy"
	"github.com/KyleKing/gh-sweep/internal/tui/theme"
)

const helpKeyWidth = 16

// Model represents the policy diff/apply TUI state.
type Model struct {
	cfg          *config.PolicyConfig
	report       *policy.Report
	cursor       int
	width        int
	height       int
	loading      bool
	err          error
	confirmApply bool
	applyTarget  string
	statusMsg    string
	showHelp     bool
}

// NewModel creates a new policy model for cfg.
func NewModel(cfg *config.PolicyConfig) Model {
	return Model{cfg: cfg, loading: true}
}

// NewModelWithConfigError creates a model that immediately surfaces a policy
// load failure (e.g. no .gh-sweep-policy.yaml found) instead of fetching anything.
func NewModelWithConfigError(err error) Model {
	return Model{err: err}
}

type reportLoadedMsg struct {
	report *policy.Report
	err    error
}

type applyResultMsg struct {
	result policy.ApplyResult
}

// Init loads the drift report, unless the model was constructed with a
// config-load error already recorded.
func (m Model) Init() tea.Cmd {
	if m.cfg == nil {
		return nil
	}

	return m.loadReport
}

func (m Model) loadReport() tea.Msg {
	ctx := context.Background()
	client, err := github.NewClient(ctx)
	if err != nil {
		return reportLoadedMsg{err: fmt.Errorf("failed to create GitHub client: %w", err)}
	}

	return reportLoadedMsg{report: policy.Evaluate(client, m.cfg)}
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

	case reportLoadedMsg:
		m.loading = false
		m.report = msg.report
		m.err = msg.err

		return m, nil

	case applyResultMsg:
		m.confirmApply = false
		if msg.result.Err != nil {
			m.statusMsg = fmt.Sprintf("Failed to apply %s: %v", msg.result.Repository, msg.result.Err)
			return m, nil
		}

		m.statusMsg = fmt.Sprintf("Applied %s: %v", msg.result.Repository, msg.result.Applied)
		m.clearDiffs(msg.result.Repository)

		return m, nil

	case tea.KeyPressMsg:
		if m.confirmApply {
			return m.handleConfirmKeys(msg)
		}

		if m.showHelp {
			if msg.String() == "?" || msg.String() == "esc" {
				m.showHelp = false
			}

			return m, nil
		}

		return m.handleKeys(msg)
	}

	return m, nil
}

func (m Model) handleKeys(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "?":
		m.showHelp = true

	case "g":
		m.cursor = 0

	case "G":
		if m.report != nil && len(m.report.Repos) > 0 {
			m.cursor = len(m.report.Repos) - 1
		}

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.report != nil && m.cursor < len(m.report.Repos)-1 {
			m.cursor++
		}

	case "a":
		return m.handleApply()

	case "r":
		m.loading = true
		m.report = nil
		m.err = nil
		m.cursor = 0

		return m, m.loadReport
	}

	return m, nil
}

func (m Model) handleConfirmKeys(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		return m.executeApply()
	case "n", "N", "esc":
		m.confirmApply = false
		m.applyTarget = ""
		m.statusMsg = "Apply canceled"
	}

	return m, nil
}

func (m Model) handleApply() (Model, tea.Cmd) {
	drift := m.currentDrift()
	if drift == nil || len(drift.Diffs) == 0 {
		m.statusMsg = "No drift on the selected repo"
		return m, nil
	}

	m.confirmApply = true
	m.applyTarget = drift.Repository

	return m, nil
}

func (m Model) executeApply() (Model, tea.Cmd) {
	drift := m.currentDrift()
	if drift == nil {
		m.confirmApply = false
		return m, nil
	}

	cfg := m.cfg
	target := *drift
	m.confirmApply = false

	return m, func() tea.Msg {
		ctx := context.Background()
		client, err := github.NewClient(ctx)
		if err != nil {
			return applyResultMsg{result: policy.ApplyResult{Repository: target.Repository, Err: err}}
		}

		return applyResultMsg{result: policy.Apply(client, cfg, target)}
	}
}

func (m Model) currentDrift() *policy.RepoDrift {
	if m.report == nil || m.cursor >= len(m.report.Repos) {
		return nil
	}

	return &m.report.Repos[m.cursor]
}

func (m *Model) clearDiffs(repository string) {
	if m.report == nil {
		return
	}

	for i := range m.report.Repos {
		if m.report.Repos[i].Repository == repository {
			m.report.Repos[i].Diffs = nil
			return
		}
	}
}

// View renders the model.
func (m Model) View() string {
	if m.loading {
		return "Evaluating policy against repositories...\n"
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Primary)
	b.WriteString(titleStyle.Render("Policy Drift"))
	b.WriteString("\n\n")

	if m.showHelp {
		return renderHelp(&b)
	}

	if m.report == nil || len(m.report.Repos) == 0 {
		b.WriteString("No repositories in policy.\n")
		return b.String()
	}

	for i, drift := range m.report.Repos {
		m.viewDriftLine(&b, i, drift)
	}

	m.viewFooter(&b)

	return b.String()
}

func (m Model) viewDriftLine(b *strings.Builder, i int, drift policy.RepoDrift) {
	cursor := " "
	if m.cursor == i {
		cursor = ">"
	}

	lineStyle := lipgloss.NewStyle()
	if m.cursor == i {
		lineStyle = lineStyle.Bold(true).Foreground(theme.Current().Warning)
	}

	switch {
	case drift.Err != nil:
		b.WriteString(lineStyle.Render(fmt.Sprintf("%s %s: ERROR %v\n", cursor, drift.Repository, drift.Err)))
	case len(drift.Diffs) == 0:
		okStyle := lineStyle.Foreground(theme.Current().Success)
		b.WriteString(okStyle.Render(fmt.Sprintf("%s %s: in sync\n", cursor, drift.Repository)))
	default:
		line := fmt.Sprintf("%s %s: %d field(s) drifted\n", cursor, drift.Repository, len(drift.Diffs))
		b.WriteString(lineStyle.Render(line))
		if m.cursor == i {
			for _, d := range drift.Diffs {
				fmt.Fprintf(b, "     [%s] %s: %s -> %s\n", d.Domain, d.Field, d.Current, d.Desired)
			}
		}
	}
}

func (m Model) viewFooter(b *strings.Builder) {
	if m.confirmApply {
		b.WriteString("\n")
		warnStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Error)
		b.WriteString(warnStyle.Render(fmt.Sprintf("Apply drift to %s? (y/n)", m.applyTarget)))
		b.WriteString("\n")
	}

	if m.statusMsg != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(theme.Current().Muted).Render(m.statusMsg))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	b.WriteString(helpStyle.Render("↑/↓: navigate | a: apply selected | r: refresh | ?: help | q: quit"))
}

func renderHelp(b *strings.Builder) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Primary)
	b.WriteString(titleStyle.Render("Keybindings"))
	b.WriteString("\n\n")

	bindings := [][2]string{
		{"j/k, up/down", "move the cursor"},
		{"g / G", "jump to top / bottom"},
		{"a", "apply drift on the selected repo"},
		{"r", "refresh"},
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
