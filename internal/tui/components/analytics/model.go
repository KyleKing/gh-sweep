// Package analytics is the TUI view for CI/CD workflow-run statistics: an
// overview tab, flaky-test detection, and extracted failure errors.
package analytics

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/tui/theme"
)

const (
	maxFailedRunsToExtract = 3
	maxErrorLinesShown     = 5
	repoPartsCount         = 2
	percentMultiplier      = 100

	viewModeOverview = "overview"
	viewModeFlaky    = "flaky"
	viewModeErrors   = "errors"

	conclusionFailure = "failure"

	reportFilePerm = 0o600
)

var (
	errNoRepository      = errors.New("no repository specified")
	errInvalidRepoFormat = errors.New("invalid repo format, expected owner/repo")
)

// Model represents the analytics TUI state.
type Model struct {
	repo          string
	stats         *github.WorkflowRunStats
	runs          []github.WorkflowRun
	flaky         []github.FlakyTest
	errorContexts []*github.ErrorContext
	errorsLoaded  bool
	errorsLoading bool
	errorsErr     error
	exportPath    string
	exportErr     error
	width         int
	height        int
	loading       bool
	err           error
	viewMode      string // "overview", "flaky", "errors"
}

// NewModel creates a new analytics model.
func NewModel(repo string) Model {
	return Model{
		repo:     repo,
		loading:  true,
		viewMode: viewModeOverview,
	}
}

type analyticsLoadedMsg struct {
	stats *github.WorkflowRunStats
	runs  []github.WorkflowRun
	flaky []github.FlakyTest
	err   error
}

type errorsLoadedMsg struct {
	contexts []*github.ErrorContext
	err      error
}

type errorsExportedMsg struct {
	path string
	err  error
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return m.loadAnalytics
}

func splitRepo(repo string) (string, string, error) {
	if repo == "" {
		return "", "", errNoRepository
	}

	parts := strings.Split(repo, "/")
	if len(parts) != repoPartsCount {
		return "", "", errInvalidRepoFormat
	}

	return parts[0], parts[1], nil
}

func (m Model) loadAnalytics() tea.Msg {
	owner, repo, err := splitRepo(m.repo)
	if err != nil {
		return analyticsLoadedMsg{err: err}
	}

	ctx := context.Background()
	client, err := github.NewClient(ctx)
	if err != nil {
		return analyticsLoadedMsg{err: fmt.Errorf("failed to create GitHub client: %w", err)}
	}

	runs, err := client.ListWorkflowRuns(owner, repo)
	if err != nil {
		return analyticsLoadedMsg{
			stats: &github.WorkflowRunStats{},
			runs:  []github.WorkflowRun{},
		}
	}

	stats := github.AnalyzeWorkflowRuns(runs)
	flaky := github.DetectFlakyTests(
		github.WorkflowRunsToTestRuns(m.repo, runs),
		github.DefaultFlakyConfig(),
	)

	return analyticsLoadedMsg{
		stats: &stats,
		runs:  runs,
		flaky: flaky,
	}
}

func (m Model) loadErrors() tea.Msg {
	owner, repo, err := splitRepo(m.repo)
	if err != nil {
		return errorsLoadedMsg{err: err}
	}

	ctx := context.Background()
	client, err := github.NewClient(ctx)
	if err != nil {
		return errorsLoadedMsg{err: fmt.Errorf("failed to create GitHub client: %w", err)}
	}

	config := github.DefaultLogConfig()
	contexts := make([]*github.ErrorContext, 0)
	extracted := 0

	for i := range m.runs {
		run := &m.runs[i]
		if run.Conclusion != conclusionFailure {
			continue
		}
		if extracted >= maxFailedRunsToExtract {
			break
		}
		extracted++

		logs, err := client.FetchFailedJobLogs(owner, repo, run.ID)
		if err != nil {
			continue
		}

		contexts = append(contexts, github.BatchExtractErrors(logs, run.Name, config)...)
	}

	return errorsLoadedMsg{contexts: contexts}
}

func (m Model) exportErrors() tea.Msg {
	path := fmt.Sprintf("gha-errors-%s.md", strings.ReplaceAll(m.repo, "/", "-"))
	report := github.FormatAsMarkdown(m.errorContexts)

	if err := os.WriteFile(path, []byte(report), reportFilePerm); err != nil {
		return errorsExportedMsg{err: fmt.Errorf("failed to write error report: %w", err)}
	}

	return errorsExportedMsg{path: path}
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

	case analyticsLoadedMsg:
		m.loading = false
		m.stats = msg.stats
		m.runs = msg.runs
		m.flaky = msg.flaky
		m.err = msg.err

		return m, nil

	case errorsLoadedMsg:
		m.errorsLoading = false
		m.errorsLoaded = true
		m.errorContexts = msg.contexts
		m.errorsErr = msg.err

		return m, nil

	case errorsExportedMsg:
		m.exportPath = msg.path
		m.exportErr = msg.err

		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "1":
		m.viewMode = viewModeOverview

	case "2":
		m.viewMode = viewModeFlaky

	case "3":
		m.viewMode = viewModeErrors
		if !m.errorsLoaded && !m.errorsLoading && !m.loading {
			m.errorsLoading = true
			return m, m.loadErrors
		}

	case "s":
		if m.viewMode == viewModeErrors && m.errorsLoaded && len(m.errorContexts) > 0 {
			return m, m.exportErrors
		}
	}

	return m, nil
}

// View renders the model.
func (m Model) View() string {
	if m.loading {
		return "Loading analytics...\n"
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Primary)

	b.WriteString(titleStyle.Render("📊 Analytics: " + m.repo))
	b.WriteString("\n\n")

	activeTab := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Warning)

	inactiveTab := lipgloss.NewStyle().
		Foreground(theme.Current().Muted)

	tabs := []string{
		"[1] Overview",
		"[2] Flaky Tests",
		"[3] Errors",
	}

	for i, tab := range tabs {
		viewModes := []string{viewModeOverview, viewModeFlaky, viewModeErrors}
		if m.viewMode == viewModes[i] {
			b.WriteString(activeTab.Render(tab))
		} else {
			b.WriteString(inactiveTab.Render(tab))
		}
		b.WriteString("  ")
	}
	b.WriteString("\n\n")

	switch m.viewMode {
	case viewModeOverview:
		b.WriteString(m.renderOverview())
	case viewModeFlaky:
		b.WriteString(m.renderFlaky())
	case viewModeErrors:
		b.WriteString(m.renderErrors())
	}

	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	help := "1/2/3: switch view | q: quit"
	if m.viewMode == viewModeErrors && m.errorsLoaded && len(m.errorContexts) > 0 {
		help = "1/2/3: switch view | s: export markdown | q: quit"
	}
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

func (m Model) renderOverview() string {
	if m.stats == nil || m.stats.TotalRuns == 0 {
		return "No workflow runs found\n"
	}

	var b strings.Builder

	b.WriteString("📈 CI/CD Statistics\n\n")
	fmt.Fprintf(&b, "Total Runs:     %d\n", m.stats.TotalRuns)
	fmt.Fprintf(&b, "Success Rate:   %.1f%%\n", m.stats.SuccessRate)
	fmt.Fprintf(&b, "Failures:       %d\n", m.stats.FailureCount)
	fmt.Fprintf(&b, "Avg Duration:   %s\n", m.stats.AvgDuration.Round(time.Second))

	b.WriteString("\nSuccess/Failure Distribution:\n")
	successCount := m.stats.TotalRuns - m.stats.FailureCount
	fmt.Fprintf(&b, "✓ Success: %s (%d)\n",
		strings.Repeat("█", successCount*50/m.stats.TotalRuns), successCount)
	fmt.Fprintf(&b, "✗ Failure: %s (%d)\n",
		strings.Repeat("█", m.stats.FailureCount*50/m.stats.TotalRuns), m.stats.FailureCount)

	return b.String()
}

func (m Model) renderFlaky() string {
	var b strings.Builder

	b.WriteString("🔍 Flaky Test Detection\n\n")
	b.WriteString("Pattern-based detection over workflow conclusions (fail → pass flips)\n\n")

	if len(m.flaky) == 0 {
		fmt.Fprintf(&b, "No flaky tests detected in last %d runs\n", len(m.runs))
		return b.String()
	}

	fmt.Fprintf(&b, "Found %d potentially flaky workflow(s):\n\n", len(m.flaky))
	for i, test := range m.flaky {
		fmt.Fprintf(&b, "%d. %s\n", i+1, test.Name)
		fmt.Fprintf(&b, "   Failure rate: %.0f%% (%d/%d runs)\n",
			test.FailureRate*percentMultiplier, test.FailureCount, test.TotalRuns)
		fmt.Fprintf(&b, "   Pattern: %s | Flips: %d\n", test.Pattern, test.FlipCount)
		if !test.LastFlip.IsZero() {
			fmt.Fprintf(&b, "   Last flip: %s\n", test.LastFlip.Format("2006-01-02 15:04"))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderErrors() string {
	var b strings.Builder

	b.WriteString("❌ Recent Errors\n\n")

	if m.errorsLoading {
		b.WriteString("Extracting errors from failed runs...\n")
		return b.String()
	}

	if m.errorsErr != nil {
		fmt.Fprintf(&b, "Error: %v\n", m.errorsErr)
		return b.String()
	}

	if len(m.errorContexts) == 0 {
		fmt.Fprintf(&b, "No errors extracted from the last %d failed runs\n",
			min(maxFailedRunsToExtract, countFailedRuns(m.runs)))

		return b.String()
	}

	for i, ctx := range m.errorContexts {
		fmt.Fprintf(&b, "%d. %s\n", i+1, ctx.Summary)
		fmt.Fprintf(&b, "   Workflow: %s | Type: %s | %s\n",
			ctx.WorkflowName, ctx.ErrorType, ctx.Timestamp.Format("2006-01-02 15:04"))

		for j, line := range ctx.ErrorLines {
			if j >= maxErrorLinesShown {
				fmt.Fprintf(&b, "   ... %d more error line(s)\n", len(ctx.ErrorLines)-j)
				break
			}
			b.WriteString("   | " + line + "\n")
		}
		b.WriteString("\n")
	}

	if m.exportErr != nil {
		fmt.Fprintf(&b, "Export failed: %v\n", m.exportErr)
	} else if m.exportPath != "" {
		fmt.Fprintf(&b, "Exported markdown report to %s\n", m.exportPath)
	}

	return b.String()
}

func countFailedRuns(runs []github.WorkflowRun) int {
	count := 0
	for i := range runs {
		if runs[i].Conclusion == conclusionFailure {
			count++
		}
	}

	return count
}
