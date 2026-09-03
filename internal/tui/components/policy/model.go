// Package policy is the TUI view for the gh-sweep policy command: it edits a
// config.PolicyConfig, previews the drift that produces against live repos,
// applies it on demand, and saves edits back to the policy file.
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
	"github.com/KyleKing/gh-sweep/internal/tui/scroll"
	"github.com/KyleKing/gh-sweep/internal/tui/theme"
)

const (
	helpKeyWidth = 16
	keyEsc       = "esc"
)

// Model represents the policy editor/diff/apply TUI state.
type Model struct {
	cfg          *config.PolicyConfig
	policyPath   string
	applyOpts    policy.ApplyOptions
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

	// diffFocus is entered with enter/l on a repo with drift, to move a
	// second cursor over that repo's individual diffs and edit one.
	diffFocus  bool
	diffCursor int

	// editing holds the in-progress text prompt for a non-boolean field; a
	// boolean field toggles immediately without one.
	editing    bool
	editBuffer string
	editDiff   policy.Diff
}

// NewModel creates a new policy model for cfg, saving edits to policyPath.
func NewModel(cfg *config.PolicyConfig, policyPath string, applyOpts policy.ApplyOptions) Model {
	return Model{cfg: cfg, policyPath: policyPath, applyOpts: applyOpts, loading: true}
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

type policySavedMsg struct {
	err error
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
		m.diffFocus = false
		m.diffCursor = 0

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

	case policySavedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Failed to save policy: %v", msg.err)
			return m, nil
		}

		m.statusMsg = "Saved " + m.policyPath

		return m, nil

	case tea.KeyPressMsg:
		if m.editing {
			return m.handleEditKeys(msg)
		}

		if m.confirmApply {
			return m.handleConfirmKeys(msg)
		}

		if m.showHelp {
			if msg.String() == "?" || msg.String() == keyEsc {
				m.showHelp = false
			}

			return m, nil
		}

		if m.diffFocus {
			return m.handleDiffFocusKeys(msg)
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

	case "a":
		return m.handleApply()

	case "r":
		return m.reload()

	case "s":
		return m.saveCmd()

	case "enter", "l":
		drift := m.currentDrift()
		if drift != nil && drift.Err == nil && len(drift.Diffs) > 0 {
			m.diffFocus = true
			m.diffCursor = 0
		}

	default:
		m.handleNavKeys(msg)
	}

	return m, nil
}

// handleNavKeys moves the repo-list cursor: g/G jump to the ends, up/down
// and j/k step by one. Every other key is a no-op.
func (m *Model) handleNavKeys(msg tea.KeyPressMsg) {
	switch msg.String() {
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
	}
}

func (m Model) reload() (Model, tea.Cmd) {
	m.loading = true
	m.report = nil
	m.err = nil
	m.cursor = 0

	return m, m.loadReport
}

func (m Model) saveCmd() (Model, tea.Cmd) {
	if m.cfg == nil {
		return m, nil
	}

	cfg := m.cfg
	path := m.policyPath

	return m, func() tea.Msg {
		return policySavedMsg{err: cfg.SavePolicy(path)}
	}
}

// handleDiffFocusKeys navigates the selected repo's individual diffs and
// starts editing the one under the cursor.
func (m Model) handleDiffFocusKeys(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	drift := m.currentDrift()
	if drift == nil {
		m.diffFocus = false
		return m, nil
	}

	switch msg.String() {
	case keyEsc, "h":
		m.diffFocus = false

	case "up", "k":
		if m.diffCursor > 0 {
			m.diffCursor--
		}

	case "down", "j":
		if m.diffCursor < len(drift.Diffs)-1 {
			m.diffCursor++
		}

	case "e":
		return m.startEdit(drift.Diffs[m.diffCursor])
	}

	return m, nil
}

// startEdit either toggles a boolean field immediately or opens a text
// prompt seeded with its currently declared value, for every other kind.
func (m Model) startEdit(diff policy.Diff) (Model, tea.Cmd) {
	switch fieldKindOf(diff.Domain, diff.Field) {
	case kindBool:
		return m.commitEdit(diff, toggledBool(diff.Desired))

	case kindInt, kindString, kindStringSlice:
		m.editing = true
		m.editDiff = diff
		m.editBuffer = diff.Desired

		return m, nil

	default:
		m.statusMsg = fmt.Sprintf("%s/%s is not editable here", diff.Domain, diff.Field)

		return m, nil
	}
}

func (m Model) handleEditKeys(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc:
		m.editing = false
		m.editBuffer = ""

		return m, nil

	case "enter":
		diff := m.editDiff
		value := m.editBuffer
		m.editing = false
		m.editBuffer = ""

		return m.commitEdit(diff, value)

	case "backspace":
		if m.editBuffer != "" {
			runes := []rune(m.editBuffer)
			m.editBuffer = string(runes[:len(runes)-1])
		}

	default:
		if runes := []rune(msg.String()); len(runes) == 1 {
			m.editBuffer += msg.String()
		}
	}

	return m, nil
}

// commitEdit writes value into cfg and re-evaluates against GitHub so the
// resulting diff is visible in the same view, without a separate
// `policy --list` run.
func (m Model) commitEdit(diff policy.Diff, value string) (Model, tea.Cmd) {
	if err := applyEdit(m.cfg, diff.Domain, diff.Field, value); err != nil {
		m.statusMsg = err.Error()
		return m, nil
	}

	m.diffFocus = false
	m.statusMsg = fmt.Sprintf(
		"Set %s/%s = %s (unsaved; press s to write %s)", diff.Domain, diff.Field, value, m.policyPath,
	)

	reload, cmd := m.reload()

	return reload, cmd
}

func (m Model) handleConfirmKeys(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		return m.executeApply()
	case "n", "N", keyEsc:
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
	applyOpts := m.applyOpts
	target := *drift
	m.confirmApply = false

	return m, func() tea.Msg {
		ctx := context.Background()
		client, err := github.NewClient(ctx)
		if err != nil {
			return applyResultMsg{result: policy.ApplyResult{Repository: target.Repository, Err: err}}
		}

		return applyResultMsg{result: policy.Apply(client, cfg, target, applyOpts)}
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
	b.WriteString(titleStyle.Render("Policy: " + m.policyPath))
	b.WriteString("\n\n")

	if m.showHelp {
		return renderHelp(&b)
	}

	if m.report == nil || len(m.report.Repos) == 0 {
		b.WriteString("No repositories in policy.\n")
		return b.String()
	}

	var footer strings.Builder
	m.viewFooter(&footer)

	headerLines := strings.Count(b.String(), "\n")
	footerLines := strings.Count(footer.String(), "\n")
	available := m.height - headerLines - footerLines

	m.renderDriftList(&b, available)
	b.WriteString(footer.String())

	return b.String()
}

func (m Model) renderDriftList(b *strings.Builder, available int) {
	lines, cursorLine := m.buildDriftLines()

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

// buildDriftLines renders each repo's drift summary, and its expanded diff
// detail lines when it holds the cursor, as one line each so the caller can
// window by line rather than by item, returning the lines and the cursor
// row's index among them.
func (m Model) buildDriftLines() ([]string, int) {
	var lines []string

	cursorLine := 0

	for i, drift := range m.report.Repos {
		if i == m.cursor && !m.diffFocus {
			cursorLine = len(lines)
		}

		lines = append(lines, m.renderDriftLine(i, drift))

		if m.cursor == i && drift.Err == nil && len(drift.Diffs) > 0 {
			for d, diff := range drift.Diffs {
				if m.diffFocus && d == m.diffCursor {
					cursorLine = len(lines)
				}

				lines = append(lines, m.renderDiffLine(d, diff))
			}
		}
	}

	return lines, cursorLine
}

func (m Model) renderDriftLine(i int, drift policy.RepoDrift) string {
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
		return lineStyle.Render(fmt.Sprintf("%s %s: ERROR %v", cursor, drift.Repository, drift.Err))
	case len(drift.Diffs) == 0:
		okStyle := lineStyle.Foreground(theme.Current().Success)
		return okStyle.Render(fmt.Sprintf("%s %s: in sync", cursor, drift.Repository))
	default:
		return lineStyle.Render(fmt.Sprintf("%s %s: %d field(s) drifted", cursor, drift.Repository, len(drift.Diffs)))
	}
}

func (m Model) renderDiffLine(i int, d policy.Diff) string {
	cursor := " "
	lineStyle := lipgloss.NewStyle()

	if m.diffFocus && m.diffCursor == i {
		cursor = ">"
		lineStyle = lineStyle.Bold(true).Foreground(theme.Current().Accent)
	}

	return lineStyle.Render(fmt.Sprintf("    %s [%s] %s: %s -> %s", cursor, d.Domain, d.Field, d.Current, d.Desired))
}

func (m Model) viewFooter(b *strings.Builder) {
	if m.editing {
		b.WriteString("\n")
		editStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Accent)
		b.WriteString(editStyle.Render(fmt.Sprintf(
			"%s/%s = %s_", m.editDiff.Domain, m.editDiff.Field, m.editBuffer,
		)))
		b.WriteString("\n")
	}

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

	switch {
	case m.editing:
		b.WriteString(helpStyle.Render("enter: confirm | esc: cancel"))
	case m.diffFocus:
		b.WriteString(helpStyle.Render("↑/↓: navigate | e: edit | esc: back | ?: help | q: quit"))
	default:
		b.WriteString(helpStyle.Render(
			"↑/↓: navigate | enter: edit fields | a: apply | s: save | r: refresh | ?: help | q: quit",
		))
	}
}

func renderHelp(b *strings.Builder) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Primary)
	b.WriteString(titleStyle.Render("Keybindings"))
	b.WriteString("\n\n")

	bindings := [][2]string{
		{"j/k, up/down", "move the cursor"},
		{"g / G", "jump to top / bottom"},
		{"enter / l", "drill into a drifted repo's fields"},
		{"e", "edit the field under the cursor (bool toggles, others prompt)"},
		{"esc / h", "back out of field view"},
		{"a", "apply drift on the selected repo"},
		{"s", "save the edited policy to its file"},
		{"r", "refresh (re-evaluates against GitHub)"},
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
