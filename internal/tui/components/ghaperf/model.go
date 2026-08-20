// Package ghaperf is the TUI view for GitHub Actions run-timing data: an
// overview, per-workflow and per-job durations, and branch-vs-baseline
// comparisons.
package ghaperf

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/KyleKing/gh-sweep/internal/cache"
	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/tui/scroll"
	"github.com/KyleKing/gh-sweep/internal/tui/theme"
)

type viewMode int

const (
	viewOverview viewMode = iota
	viewWorkflows
	viewJobs
	viewBranches
)

const (
	repoPartsCount = 2

	defaultBaseBranch          = "main"
	defaultFilterDays          = 30
	defaultRegressionThreshold = 20.0
	defaultRunsFetchLimit      = 100

	helpKeyWidth      = 16
	percentMultiplier = 100.0
	recentRunsLimit   = 10

	workflowNameTruncateLen   = 30
	workflowColumnTruncateLen = 35
	jobNameTruncateLen        = 50
	branchNameTruncateLen     = 30

	conclusionSuccess = "success"

	trendWidth = 20
	barWidth   = 20
)

var errInvalidRepoFormat = errors.New("invalid repo format, expected owner/repo")

// Model is the ghaperf TUI state: the loaded run data, the active tab, and
// the cursor/scroll position within it.
type Model struct {
	repo     string
	owner    string
	repoName string
	width    int
	height   int
	loading  bool
	err      error
	viewMode viewMode
	cursor   int

	workflows           []github.WorkflowFile
	selectedWorkflow    string
	filterBranch        string
	filterDays          int
	cacheOnly           bool
	regressionThreshold float64

	runs          []github.RunTiming
	workflowStats map[string]*github.WorkflowStats
	jobStats      map[string]*github.JobStats
	branchStats   map[string]*github.BranchStats
	baseBranch    string

	cachedCount int
	newCount    int

	showHelp bool
}

// NewModel creates a ghaperf model for repo ("owner/repo"). An invalid repo
// format is validated at load time, not here.
func NewModel(repo string, opts ...Option) Model {
	parts := strings.Split(repo, "/")
	owner, repoName := "", ""
	if len(parts) == repoPartsCount {
		owner, repoName = parts[0], parts[1]
	}

	m := Model{
		repo:                repo,
		owner:               owner,
		repoName:            repoName,
		loading:             true,
		viewMode:            viewOverview,
		filterDays:          defaultFilterDays,
		baseBranch:          defaultBaseBranch,
		regressionThreshold: defaultRegressionThreshold,
	}

	for _, opt := range opts {
		opt(&m)
	}

	return m
}

// Option configures a Model built by NewModel.
type Option func(*Model)

// WithBranch filters runs to a single branch.
func WithBranch(branch string) Option {
	return func(m *Model) {
		m.filterBranch = branch
	}
}

// WithDays sets how many trailing days of runs to load.
func WithDays(days int) Option {
	return func(m *Model) {
		m.filterDays = days
	}
}

// WithWorkflow filters runs to a single workflow file.
func WithWorkflow(workflow string) Option {
	return func(m *Model) {
		m.selectedWorkflow = workflow
	}
}

// WithCacheOnly restricts loading to cached data, skipping the GitHub fetch.
func WithCacheOnly(cacheOnly bool) Option {
	return func(m *Model) {
		m.cacheOnly = cacheOnly
	}
}

// WithBaseBranch sets the branch other branches are compared against in the
// Branches view.
func WithBaseBranch(branch string) Option {
	return func(m *Model) {
		m.baseBranch = branch
	}
}

// WithRegressionThreshold sets the percent slowdown against baseline that
// marks a branch as regressed in the Branches view.
func WithRegressionThreshold(threshold float64) Option {
	return func(m *Model) {
		m.regressionThreshold = threshold
	}
}

type dataLoadedMsg struct {
	runs          []github.RunTiming
	workflows     []github.WorkflowFile
	workflowStats map[string]*github.WorkflowStats
	jobStats      map[string]*github.JobStats
	branchStats   map[string]*github.BranchStats
	cachedCount   int
	newCount      int
	err           error
}

// Init starts the initial data load.
func (m Model) Init() tea.Cmd {
	return m.loadData
}

func (m Model) loadData() tea.Msg {
	if m.owner == "" || m.repoName == "" {
		return dataLoadedMsg{err: errInvalidRepoFormat}
	}

	cacheManager, err := cache.NewGHAPerfCacheManager("")
	if err != nil {
		return dataLoadedMsg{err: fmt.Errorf("failed to create cache manager: %w", err)}
	}

	ctx := context.Background()
	client, err := github.NewClient(ctx)
	if err != nil {
		return dataLoadedMsg{err: fmt.Errorf("failed to create GitHub client: %w", err)}
	}

	workflows, err := client.ListWorkflows(m.owner, m.repoName)
	if err != nil {
		workflows = []github.WorkflowFile{}
	}

	cachedData, err := cacheManager.Load(m.owner, m.repoName)
	if err != nil {
		return dataLoadedMsg{err: fmt.Errorf("failed to load cache: %w", err)}
	}

	cachedCount := len(cachedData.Runs)

	allRuns, newCount, err := m.fetchRuns(client, cacheManager, cachedData)
	if err != nil {
		return dataLoadedMsg{err: err}
	}

	if m.filterBranch != "" {
		allRuns = github.FilterRunsByBranch(allRuns, m.filterBranch)
	}

	since := time.Now().AddDate(0, 0, -m.filterDays)
	allRuns = github.FilterRunsByTimeRange(allRuns, since, time.Time{})

	github.SortRunsByDate(allRuns, false)

	workflowStats := github.ComputeWorkflowStats(allRuns)
	jobStats := github.ComputeJobStats(allRuns)
	branchStats := github.ComputeBranchStats(allRuns, m.baseBranch)

	return dataLoadedMsg{
		runs:          allRuns,
		workflows:     workflows,
		workflowStats: workflowStats,
		jobStats:      jobStats,
		branchStats:   branchStats,
		cachedCount:   cachedCount,
		newCount:      newCount,
	}
}

// fetchRuns returns the runs to render (cached, freshly fetched and merged,
// or a cached fallback on fetch failure) and how many of them are new since
// the last cache write.
func (m Model) fetchRuns(
	client *github.Client,
	cacheManager *cache.GHAPerfCacheManager,
	cachedData *cache.GHAPerfCache,
) ([]github.RunTiming, int, error) {
	if m.cacheOnly {
		return cachedData.Runs, 0, nil
	}

	cachedIDs := make(map[int]bool, len(cachedData.Runs))
	for i := range cachedData.Runs {
		cachedIDs[cachedData.Runs[i].RunID] = true
	}

	since := time.Now().AddDate(0, 0, -m.filterDays)
	opts := github.FetchWorkflowRunsOptions{
		WorkflowFile: m.selectedWorkflow,
		Branch:       m.filterBranch,
		Limit:        defaultRunsFetchLimit,
		CreatedAfter: since,
	}

	newRuns, err := client.FetchWorkflowRunsWithDetails(m.owner, m.repoName, opts)
	if err != nil {
		if len(cachedData.Runs) == 0 {
			return nil, 0, fmt.Errorf("failed to fetch workflow runs: %w", err)
		}

		return cachedData.Runs, 0, nil
	}

	newCount := countNewRuns(newRuns, cachedIDs)
	allRuns := cache.MergeRuns(cachedData.Runs, newRuns)

	if newCount > 0 {
		cachedData.Runs = allRuns
		//nolint:errcheck // cache write failures must not block rendering fetched data
		_ = cacheManager.Save(m.owner, m.repoName, cachedData)
	}

	return allRuns, newCount, nil
}

func countNewRuns(newRuns []github.RunTiming, cachedIDs map[int]bool) int {
	count := 0
	for i := range newRuns {
		if !cachedIDs[newRuns[i].RunID] {
			count++
		}
	}

	return count
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

	case dataLoadedMsg:
		m.loading = false
		m.err = msg.err
		m.runs = msg.runs
		m.workflows = msg.workflows
		m.workflowStats = msg.workflowStats
		m.jobStats = msg.jobStats
		m.branchStats = msg.branchStats
		m.cachedCount = msg.cachedCount
		m.newCount = msg.newCount

		return m, nil

	case tea.KeyPressMsg:
		return m.updateKeyMsg(msg)
	}

	return m, nil
}

var tabKeyModes = map[string]viewMode{
	"1": viewOverview,
	"2": viewWorkflows,
	"3": viewJobs,
	"4": viewBranches,
}

func (m Model) updateKeyMsg(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	if m.showHelp {
		if msg.String() == "?" || msg.String() == "esc" {
			m.showHelp = false
		}

		return m, nil
	}

	key := msg.String()

	if mode, ok := tabKeyModes[key]; ok {
		m.switchView(mode)

		return m, nil
	}

	switch key {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "?":
		m.showHelp = true

	case "g":
		m.cursor = 0

	case "G":
		m.jumpToBottom()

	case "up", "k":
		m.moveCursorUp()
	case "down", "j":
		m.moveCursorDown()

	case "r":
		m.loading = true
		m.cacheOnly = false

		return m, m.loadData
	}

	return m, nil
}

func (m *Model) switchView(mode viewMode) {
	m.viewMode = mode
	m.cursor = 0
}

func (m *Model) jumpToBottom() {
	maxCursor := m.getMaxCursor()
	if maxCursor >= 0 {
		m.cursor = maxCursor
	}
}

func (m *Model) moveCursorUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *Model) moveCursorDown() {
	maxCursor := m.getMaxCursor()
	if m.cursor < maxCursor {
		m.cursor++
	}
}

func (m Model) getMaxCursor() int {
	switch m.viewMode {
	case viewWorkflows:
		return len(m.workflowStats) - 1
	case viewJobs:
		return len(m.jobStats) - 1
	case viewBranches:
		return len(m.branchStats) - 1
	default:
		return len(m.runs) - 1
	}
}

// View renders the model.
func (m Model) View() string {
	if m.loading {
		return "Loading GHA performance data...\n"
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var header strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Primary)

	header.WriteString(titleStyle.Render("GHA Performance: " + m.repo))
	header.WriteString("\n")

	subtitleStyle := lipgloss.NewStyle().
		Foreground(theme.Current().Muted)

	header.WriteString(subtitleStyle.Render(fmt.Sprintf(
		"Last %d days | %d runs (%d cached, %d new)",
		m.filterDays, len(m.runs), m.cachedCount, m.newCount)))
	header.WriteString("\n\n")

	if m.showHelp {
		return renderHelp(&header)
	}

	activeTab := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Warning)

	inactiveTab := lipgloss.NewStyle().
		Foreground(theme.Current().Muted)

	tabs := []struct {
		label string
		mode  viewMode
	}{
		{"[1] Overview", viewOverview},
		{"[2] Workflows", viewWorkflows},
		{"[3] Jobs", viewJobs},
		{"[4] Branches", viewBranches},
	}

	for _, tab := range tabs {
		if m.viewMode == tab.mode {
			header.WriteString(activeTab.Render(tab.label))
		} else {
			header.WriteString(inactiveTab.Render(tab.label))
		}
		header.WriteString("  ")
	}
	header.WriteString("\n\n")

	var footer strings.Builder
	footer.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	footer.WriteString(helpStyle.Render("1-4: views | j/k: navigate | r: refresh | ?: help | esc: back | q: quit"))

	if m.viewMode == viewOverview {
		return header.String() + m.renderOverview() + footer.String()
	}

	lines, cursorLine := m.buildListLines(&header)

	headerLines := strings.Count(header.String(), "\n")
	footerLines := strings.Count(footer.String(), "\n")
	available := m.height - headerLines - footerLines

	var body strings.Builder
	renderScrollList(&body, lines, cursorLine, available)

	return header.String() + body.String() + footer.String()
}

// buildListLines writes the section title and column header for the active
// tabbed view into header, then returns one rendered row per item alongside
// the cursor row's index among them, so View can window the rows by line.
func (m Model) buildListLines(header *strings.Builder) ([]string, int) {
	switch m.viewMode {
	case viewJobs:
		return m.buildJobLines(header)
	case viewBranches:
		return m.buildBranchLines(header)
	default:
		return m.buildWorkflowLines(header)
	}
}

func renderScrollList(b *strings.Builder, lines []string, cursorLine, available int) {
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
		{"1-4", "switch views: overview, workflows, jobs, branches"},
		{"j/k, up/down", "move the cursor"},
		{"g / G", "jump to top / bottom"},
		{"r", "refresh"},
		{"?", "toggle this help"},
		{"esc / q", "back / quit"},
	}

	for _, binding := range bindings {
		keyStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Warning).Width(helpKeyWidth)
		fmt.Fprintf(b, "%s %s\n", keyStyle.Render(binding[0]), binding[1])
	}

	b.WriteString("\nPress '?' or 'esc' to close\n")

	return b.String()
}

func (m Model) renderOverview() string {
	var b strings.Builder

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Text)

	valueStyle := lipgloss.NewStyle().
		Foreground(theme.Current().Success)

	b.WriteString(sectionStyle.Render("Summary"))
	b.WriteString("\n")

	totalRuns := len(m.runs)
	var successCount, failureCount int
	var totalDuration time.Duration

	for i := range m.runs {
		totalDuration += m.runs[i].Duration
		switch m.runs[i].Conclusion {
		case conclusionSuccess:
			successCount++
		case "failure":
			failureCount++
		}
	}

	successRate := float64(0)
	avgDuration := time.Duration(0)
	if totalRuns > 0 {
		successRate = float64(successCount) / float64(totalRuns) * percentMultiplier
		avgDuration = totalDuration / time.Duration(totalRuns)
	}

	fmt.Fprintf(&b, "  Total Runs:     %s\n", valueStyle.Render(strconv.Itoa(totalRuns)))
	fmt.Fprintf(&b, "  Success Rate:   %s\n", valueStyle.Render(fmt.Sprintf("%.1f%%", successRate)))
	fmt.Fprintf(&b, "  Failures:       %s\n", valueStyle.Render(strconv.Itoa(failureCount)))
	fmt.Fprintf(&b, "  Avg Duration:   %s\n", valueStyle.Render(github.FormatDuration(avgDuration)))
	fmt.Fprintf(&b, "  Workflows:      %s\n", valueStyle.Render(strconv.Itoa(len(m.workflowStats))))
	fmt.Fprintf(&b, "  Branches:       %s\n", valueStyle.Render(strconv.Itoa(len(m.branchStats))))

	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Recent Runs"))
	b.WriteString("\n")

	displayRuns := m.runs
	if len(displayRuns) > recentRunsLimit {
		displayRuns = displayRuns[:recentRunsLimit]
	}

	successStyle := lipgloss.NewStyle().Foreground(theme.Current().Success)
	failureStyle := lipgloss.NewStyle().Foreground(theme.Current().Error)

	for i := range displayRuns {
		r := &displayRuns[i]

		status := successStyle.Render("OK")
		if r.Conclusion != conclusionSuccess {
			status = failureStyle.Render("FAIL")
		}

		workflow := r.Workflow
		if len(workflow) > workflowNameTruncateLen {
			workflow = workflow[:workflowNameTruncateLen-3] + "..."
		}

		fmt.Fprintf(&b, "  %s %-30s %-15s %s\n",
			status,
			workflow,
			r.Branch,
			github.FormatDuration(r.Duration))
	}

	return b.String()
}

// runDurationsFor returns workflow's run durations in seconds, oldest first,
// for the trend sparkline.
func (m Model) runDurationsFor(workflow string) []float64 {
	var runs []github.RunTiming
	for i := range m.runs {
		if m.runs[i].Workflow == workflow {
			runs = append(runs, m.runs[i])
		}
	}

	sort.Slice(runs, func(i, j int) bool {
		return runs[i].CreatedAt.Before(runs[j].CreatedAt)
	})

	durations := make([]float64, len(runs))
	for i := range runs {
		durations[i] = runs[i].DurationSeconds
	}

	return durations
}

// buildWorkflowLines writes the Workflows tab's section title and column
// header into header, then returns one rendered row per workflow alongside
// the cursor row's index among them.
func (m Model) buildWorkflowLines(header *strings.Builder) ([]string, int) {
	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Text)

	header.WriteString(sectionStyle.Render("Workflow Performance"))
	header.WriteString("\n\n")

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Muted)

	header.WriteString(headerStyle.Render(fmt.Sprintf("  %-35s %8s %8s %8s %8s %8s  %s\n",
		"Workflow", "Runs", "Avg", "Min", "Max", "Success", "Trend")))

	workflows := make([]*github.WorkflowStats, 0, len(m.workflowStats))
	for _, ws := range m.workflowStats {
		workflows = append(workflows, ws)
	}
	sort.Slice(workflows, func(i, j int) bool {
		return workflows[i].AvgDuration > workflows[j].AvgDuration
	})

	selectedStyle := lipgloss.NewStyle().
		Background(theme.Current().Secondary)

	lines := make([]string, len(workflows))
	cursorLine := 0

	for i, ws := range workflows {
		name := ws.Workflow
		if len(name) > workflowColumnTruncateLen {
			name = name[:workflowColumnTruncateLen-3] + "..."
		}

		trend := sparkline(m.runDurationsFor(ws.Workflow), trendWidth)

		line := fmt.Sprintf("  %-35s %8d %8s %8s %8s %7.0f%%  %s",
			name,
			ws.TotalRuns,
			github.FormatDuration(ws.AvgDuration),
			github.FormatDuration(ws.MinDuration),
			github.FormatDuration(ws.MaxDuration),
			ws.SuccessRate,
			trend)

		if i == m.cursor {
			cursorLine = i
			line = selectedStyle.Render(line)
		}

		lines[i] = line
	}

	return lines, cursorLine
}

// buildJobLines writes the Jobs tab's section title and column header into
// header, then returns one rendered row per job alongside the cursor row's
// index among them.
func (m Model) buildJobLines(header *strings.Builder) ([]string, int) {
	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Text)

	header.WriteString(sectionStyle.Render("Job Performance (Top by Avg Duration)"))
	header.WriteString("\n\n")

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Muted)

	header.WriteString(headerStyle.Render(fmt.Sprintf("  %-50s %8s %8s %8s %8s\n",
		"Job", "Runs", "Avg", "Min", "Max")))

	jobs := github.GetTopJobsByDuration(m.jobStats, 0)

	selectedStyle := lipgloss.NewStyle().
		Background(theme.Current().Secondary)

	lines := make([]string, len(jobs))
	cursorLine := 0

	for i, js := range jobs {
		name := js.WorkflowJob
		if len(name) > jobNameTruncateLen {
			name = name[:jobNameTruncateLen-3] + "..."
		}

		line := fmt.Sprintf("  %-50s %8d %8s %8s %8s",
			name,
			js.TotalRuns,
			github.FormatDuration(js.AvgDuration),
			github.FormatDuration(js.MinDuration),
			github.FormatDuration(js.MaxDuration))

		if i == m.cursor {
			cursorLine = i
			line = selectedStyle.Render(line)
		}

		lines[i] = line
	}

	return lines, cursorLine
}

// buildBranchLines writes the Branches tab's section title and column header
// into header, then returns one rendered row per branch alongside the
// cursor row's index among them.
func (m Model) buildBranchLines(header *strings.Builder) ([]string, int) {
	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Text)

	header.WriteString(sectionStyle.Render(fmt.Sprintf("Performance by Branch (vs %s)", m.baseBranch)))
	header.WriteString("\n\n")

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Muted)

	header.WriteString(headerStyle.Render(fmt.Sprintf("  %-30s %8s %10s %12s  %s\n",
		"Branch", "Runs", "Avg", "Delta", "Relative")))

	branches, maxAvg := m.sortedBranchStats()

	selectedStyle := lipgloss.NewStyle().
		Background(theme.Current().Secondary)

	lines := make([]string, len(branches))
	cursorLine := 0

	for i, bs := range branches {
		line := m.branchRowLine(bs, maxAvg)

		if i == m.cursor {
			cursorLine = i
			line = selectedStyle.Render(line)
		}

		lines[i] = line
	}

	return lines, cursorLine
}

// sortedBranchStats returns m.branchStats as a slice with the baseline
// branch first and the rest ordered slowest-first, alongside the slowest
// average duration for scaling the comparison bar.
func (m Model) sortedBranchStats() ([]*github.BranchStats, time.Duration) {
	branches := make([]*github.BranchStats, 0, len(m.branchStats))
	var maxAvg time.Duration
	for _, bs := range m.branchStats {
		branches = append(branches, bs)
		if bs.AvgDuration > maxAvg {
			maxAvg = bs.AvgDuration
		}
	}

	sort.Slice(branches, func(i, j int) bool {
		if branches[i].Branch == m.baseBranch {
			return true
		}
		if branches[j].Branch == m.baseBranch {
			return false
		}

		return branches[i].AvgDuration > branches[j].AvgDuration
	})

	return branches, maxAvg
}

func (m Model) branchRowLine(bs *github.BranchStats, maxAvg time.Duration) string {
	name := bs.Branch
	if bs.Branch == m.baseBranch {
		name = "[BASE] " + name
	}
	if len(name) > branchNameTruncateLen {
		name = name[:branchNameTruncateLen-3] + "..."
	}

	return fmt.Sprintf("  %-30s %8d %10s %12s  %s",
		name,
		bs.TotalRuns,
		github.FormatDuration(bs.AvgDuration),
		m.branchDelta(bs),
		bar(float64(bs.AvgDuration), float64(maxAvg), barWidth))
}

func (m Model) branchDelta(bs *github.BranchStats) string {
	if bs.Branch == m.baseBranch || bs.DeltaVsBasePct == 0 {
		return ""
	}

	fasterStyle := lipgloss.NewStyle().Foreground(theme.Current().Success)
	slowerStyle := lipgloss.NewStyle().Foreground(theme.Current().Error)

	sign := "+"
	style := slowerStyle
	if bs.DeltaVsBasePct < 0 {
		sign = ""
		style = fasterStyle
	}

	marker := ""
	if bs.DeltaVsBasePct >= m.regressionThreshold {
		marker = " ⚠"
	}

	return style.Render(fmt.Sprintf("%s%.0f%%%s", sign, bs.DeltaVsBasePct, marker))
}
