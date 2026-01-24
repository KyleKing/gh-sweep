package ghaperf

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/KyleKing/gh-sweep/internal/cache"
	"github.com/KyleKing/gh-sweep/internal/github"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type viewMode int

const (
	viewOverview viewMode = iota
	viewWorkflows
	viewJobs
	viewBranches
)

type Model struct {
	repo       string
	owner      string
	repoName   string
	width      int
	height     int
	loading    bool
	err        error
	viewMode   viewMode
	cursor     int
	scrollTop  int
	maxVisible int

	workflows        []github.WorkflowFile
	selectedWorkflow string
	filterBranch     string
	filterDays       int
	cacheOnly        bool

	runs          []github.RunTiming
	workflowStats map[string]*github.WorkflowStats
	jobStats      map[string]*github.JobStats
	branchStats   map[string]*github.BranchStats
	baseBranch    string

	cacheManager *cache.GHAPerfCacheManager
	cachedCount  int
	newCount     int
}

func NewModel(repo string, opts ...Option) Model {
	parts := strings.Split(repo, "/")
	owner, repoName := "", ""
	if len(parts) == 2 {
		owner, repoName = parts[0], parts[1]
	}

	m := Model{
		repo:        repo,
		owner:       owner,
		repoName:    repoName,
		loading:     true,
		viewMode:    viewOverview,
		filterDays:  30,
		baseBranch:  "main",
		maxVisible:  15,
	}

	for _, opt := range opts {
		opt(&m)
	}

	return m
}

type Option func(*Model)

func WithBranch(branch string) Option {
	return func(m *Model) {
		m.filterBranch = branch
	}
}

func WithDays(days int) Option {
	return func(m *Model) {
		m.filterDays = days
	}
}

func WithWorkflow(workflow string) Option {
	return func(m *Model) {
		m.selectedWorkflow = workflow
	}
}

func WithCacheOnly(cacheOnly bool) Option {
	return func(m *Model) {
		m.cacheOnly = cacheOnly
	}
}

func WithBaseBranch(branch string) Option {
	return func(m *Model) {
		m.baseBranch = branch
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

func (m Model) Init() tea.Cmd {
	return m.loadData
}

func (m Model) loadData() tea.Msg {
	if m.owner == "" || m.repoName == "" {
		return dataLoadedMsg{err: fmt.Errorf("invalid repo format, expected owner/repo")}
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
	var allRuns []github.RunTiming
	var newCount int

	if m.cacheOnly {
		allRuns = cachedData.Runs
	} else {
		cachedIDs := make(map[int]bool)
		for _, r := range cachedData.Runs {
			cachedIDs[r.RunID] = true
		}

		since := time.Now().AddDate(0, 0, -m.filterDays)
		opts := github.FetchWorkflowRunsOptions{
			WorkflowFile: m.selectedWorkflow,
			Branch:       m.filterBranch,
			Limit:        100,
			CreatedAfter: since,
		}

		newRuns, err := client.FetchWorkflowRunsWithDetails(m.owner, m.repoName, opts)
		if err != nil {
			if cachedCount > 0 {
				allRuns = cachedData.Runs
			} else {
				return dataLoadedMsg{err: fmt.Errorf("failed to fetch workflow runs: %w", err)}
			}
		} else {
			var actuallyNew []github.RunTiming
			for _, r := range newRuns {
				if !cachedIDs[r.RunID] {
					actuallyNew = append(actuallyNew, r)
				}
			}
			newCount = len(actuallyNew)

			allRuns = cacheManager.MergeRuns(cachedData.Runs, newRuns)

			if newCount > 0 {
				cachedData.Runs = allRuns
				_ = cacheManager.Save(m.owner, m.repoName, cachedData)
			}
		}
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

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.maxVisible = msg.Height - 12
		if m.maxVisible < 5 {
			m.maxVisible = 5
		}
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

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "1":
			m.viewMode = viewOverview
			m.cursor = 0
			m.scrollTop = 0
		case "2":
			m.viewMode = viewWorkflows
			m.cursor = 0
			m.scrollTop = 0
		case "3":
			m.viewMode = viewJobs
			m.cursor = 0
			m.scrollTop = 0
		case "4":
			m.viewMode = viewBranches
			m.cursor = 0
			m.scrollTop = 0

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.scrollTop {
					m.scrollTop = m.cursor
				}
			}
		case "down", "j":
			maxCursor := m.getMaxCursor()
			if m.cursor < maxCursor {
				m.cursor++
				if m.cursor >= m.scrollTop+m.maxVisible {
					m.scrollTop = m.cursor - m.maxVisible + 1
				}
			}

		case "r":
			m.loading = true
			m.cacheOnly = false
			return m, m.loadData
		}
	}

	return m, nil
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

func (m Model) View() string {
	if m.loading {
		return "Loading GHA performance data...\n"
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FFFF"))

	b.WriteString(titleStyle.Render(fmt.Sprintf("GHA Performance: %s", m.repo)))
	b.WriteString("\n")

	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#777777"))

	b.WriteString(subtitleStyle.Render(fmt.Sprintf(
		"Last %d days | %d runs (%d cached, %d new)",
		m.filterDays, len(m.runs), m.cachedCount, m.newCount)))
	b.WriteString("\n\n")

	activeTab := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFF00"))

	inactiveTab := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#777777"))

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
			b.WriteString(activeTab.Render(tab.label))
		} else {
			b.WriteString(inactiveTab.Render(tab.label))
		}
		b.WriteString("  ")
	}
	b.WriteString("\n\n")

	switch m.viewMode {
	case viewOverview:
		b.WriteString(m.renderOverview())
	case viewWorkflows:
		b.WriteString(m.renderWorkflows())
	case viewJobs:
		b.WriteString(m.renderJobs())
	case viewBranches:
		b.WriteString(m.renderBranches())
	}

	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))
	b.WriteString(helpStyle.Render("1-4: views | j/k: navigate | r: refresh | esc: back | q: quit"))

	return b.String()
}

func (m Model) renderOverview() string {
	var b strings.Builder

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF"))

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FF00"))

	b.WriteString(sectionStyle.Render("Summary"))
	b.WriteString("\n")

	totalRuns := len(m.runs)
	var successCount, failureCount int
	var totalDuration time.Duration

	for _, r := range m.runs {
		totalDuration += r.Duration
		switch r.Conclusion {
		case "success":
			successCount++
		case "failure":
			failureCount++
		}
	}

	successRate := float64(0)
	avgDuration := time.Duration(0)
	if totalRuns > 0 {
		successRate = float64(successCount) / float64(totalRuns) * 100
		avgDuration = totalDuration / time.Duration(totalRuns)
	}

	b.WriteString(fmt.Sprintf("  Total Runs:     %s\n", valueStyle.Render(fmt.Sprintf("%d", totalRuns))))
	b.WriteString(fmt.Sprintf("  Success Rate:   %s\n", valueStyle.Render(fmt.Sprintf("%.1f%%", successRate))))
	b.WriteString(fmt.Sprintf("  Failures:       %s\n", valueStyle.Render(fmt.Sprintf("%d", failureCount))))
	b.WriteString(fmt.Sprintf("  Avg Duration:   %s\n", valueStyle.Render(github.FormatDuration(avgDuration))))
	b.WriteString(fmt.Sprintf("  Workflows:      %s\n", valueStyle.Render(fmt.Sprintf("%d", len(m.workflowStats)))))
	b.WriteString(fmt.Sprintf("  Branches:       %s\n", valueStyle.Render(fmt.Sprintf("%d", len(m.branchStats)))))

	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Recent Runs"))
	b.WriteString("\n")

	displayRuns := m.runs
	if len(displayRuns) > 10 {
		displayRuns = displayRuns[:10]
	}

	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	failureStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))

	for _, r := range displayRuns {
		status := successStyle.Render("OK")
		if r.Conclusion != "success" {
			status = failureStyle.Render("FAIL")
		}

		workflow := r.Workflow
		if len(workflow) > 30 {
			workflow = workflow[:27] + "..."
		}

		b.WriteString(fmt.Sprintf("  %s %-30s %-15s %s\n",
			status,
			workflow,
			r.Branch,
			github.FormatDuration(r.Duration)))
	}

	return b.String()
}

func (m Model) renderWorkflows() string {
	var b strings.Builder

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF"))

	b.WriteString(sectionStyle.Render("Workflow Performance"))
	b.WriteString("\n\n")

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#777777"))

	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-35s %8s %8s %8s %8s %8s\n",
		"Workflow", "Runs", "Avg", "Min", "Max", "Success")))

	var workflows []*github.WorkflowStats
	for _, ws := range m.workflowStats {
		workflows = append(workflows, ws)
	}
	sort.Slice(workflows, func(i, j int) bool {
		return workflows[i].AvgDuration > workflows[j].AvgDuration
	})

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#333333"))

	for i, ws := range workflows {
		if i < m.scrollTop || i >= m.scrollTop+m.maxVisible {
			continue
		}

		name := ws.Workflow
		if len(name) > 35 {
			name = name[:32] + "..."
		}

		line := fmt.Sprintf("  %-35s %8d %8s %8s %8s %7.0f%%",
			name,
			ws.TotalRuns,
			github.FormatDuration(ws.AvgDuration),
			github.FormatDuration(ws.MinDuration),
			github.FormatDuration(ws.MaxDuration),
			ws.SuccessRate)

		if i == m.cursor {
			b.WriteString(selectedStyle.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderJobs() string {
	var b strings.Builder

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF"))

	b.WriteString(sectionStyle.Render("Job Performance (Top by Avg Duration)"))
	b.WriteString("\n\n")

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#777777"))

	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-50s %8s %8s %8s %8s\n",
		"Job", "Runs", "Avg", "Min", "Max")))

	jobs := github.GetTopJobsByDuration(m.jobStats, 0)

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#333333"))

	for i, js := range jobs {
		if i < m.scrollTop || i >= m.scrollTop+m.maxVisible {
			continue
		}

		name := js.WorkflowJob
		if len(name) > 50 {
			name = name[:47] + "..."
		}

		line := fmt.Sprintf("  %-50s %8d %8s %8s %8s",
			name,
			js.TotalRuns,
			github.FormatDuration(js.AvgDuration),
			github.FormatDuration(js.MinDuration),
			github.FormatDuration(js.MaxDuration))

		if i == m.cursor {
			b.WriteString(selectedStyle.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderBranches() string {
	var b strings.Builder

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF"))

	b.WriteString(sectionStyle.Render(fmt.Sprintf("Performance by Branch (vs %s)", m.baseBranch)))
	b.WriteString("\n\n")

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#777777"))

	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-30s %8s %10s %12s\n",
		"Branch", "Runs", "Avg", "Delta")))

	var branches []*github.BranchStats
	for _, bs := range m.branchStats {
		branches = append(branches, bs)
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

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#333333"))

	fasterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	slowerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))

	for i, bs := range branches {
		if i < m.scrollTop || i >= m.scrollTop+m.maxVisible {
			continue
		}

		name := bs.Branch
		if bs.Branch == m.baseBranch {
			name = "[BASE] " + name
		}
		if len(name) > 30 {
			name = name[:27] + "..."
		}

		delta := ""
		if bs.Branch != m.baseBranch && bs.DeltaVsBasePct != 0 {
			sign := "+"
			style := slowerStyle
			if bs.DeltaVsBasePct < 0 {
				sign = ""
				style = fasterStyle
			}
			delta = style.Render(fmt.Sprintf("%s%.0f%%", sign, bs.DeltaVsBasePct))
		}

		line := fmt.Sprintf("  %-30s %8d %10s %12s",
			name,
			bs.TotalRuns,
			github.FormatDuration(bs.AvgDuration),
			delta)

		if i == m.cursor {
			b.WriteString(selectedStyle.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	return b.String()
}
