package watching

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/cli/browser"

	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/tui/theme"
)

type Model struct {
	username    string
	repos       []github.RepoWatchInfo
	cursor      int
	width       int
	height      int
	loading     bool
	err         error
	viewMode    string
	selected    map[int]bool
	statusMsg   string
	searching   bool
	searchQuery string
	showHelp    bool
}

func NewModel() Model {
	return Model{
		selected: make(map[int]bool),
		loading:  true,
		viewMode: "unwatched",
	}
}

type dataLoadedMsg struct {
	username string
	repos    []github.RepoWatchInfo
	err      error
}

type watchResultMsg struct {
	repo string
	err  error
}

type unwatchResultMsg struct {
	repo string
	err  error
}

type ignoreResultMsg struct {
	repo string
	err  error
}

type openResultMsg struct {
	repo string
	err  error
}

func (m Model) Init() tea.Cmd {
	return m.loadData
}

func (m Model) loadData() tea.Msg {
	client, err := github.NewGQLClient()
	if err != nil {
		return dataLoadedMsg{err: fmt.Errorf("failed to create GitHub GraphQL client: %w", err)}
	}

	username, repos, err := client.ListViewerRepoWatchInfo()
	if err != nil {
		return dataLoadedMsg{err: fmt.Errorf("failed to list repo watch info: %w", err)}
	}

	return dataLoadedMsg{username: username, repos: repos, err: nil}
}

func (m Model) watchRepo(repo github.RepoWatchInfo) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		client, err := github.NewClient(ctx)
		if err != nil {
			return watchResultMsg{repo: repo.FullName, err: err}
		}

		if _, err := client.SetRepoSubscription(repo.Owner, repo.Name, true, false); err != nil {
			return watchResultMsg{repo: repo.FullName, err: err}
		}

		return watchResultMsg{repo: repo.FullName, err: nil}
	}
}

func (m Model) openRepo(repo github.RepoWatchInfo) tea.Cmd {
	return func() tea.Msg {
		err := browser.OpenURL("https://github.com/" + repo.FullName)
		return openResultMsg{repo: repo.FullName, err: err}
	}
}

func (m Model) unwatchRepo(repo github.RepoWatchInfo) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		client, err := github.NewClient(ctx)
		if err != nil {
			return unwatchResultMsg{repo: repo.FullName, err: err}
		}

		if err := client.DeleteRepoSubscription(repo.Owner, repo.Name); err != nil {
			return unwatchResultMsg{repo: repo.FullName, err: err}
		}

		return unwatchResultMsg{repo: repo.FullName, err: nil}
	}
}

func (m Model) ignoreRepo(repo github.RepoWatchInfo) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		client, err := github.NewClient(ctx)
		if err != nil {
			return ignoreResultMsg{repo: repo.FullName, err: err}
		}

		if _, err := client.SetRepoSubscription(repo.Owner, repo.Name, false, true); err != nil {
			return ignoreResultMsg{repo: repo.FullName, err: err}
		}

		return ignoreResultMsg{repo: repo.FullName, err: nil}
	}
}

func (m Model) setState(fullName string, state github.WatchState) {
	for i := range m.repos {
		if m.repos[i].FullName == fullName {
			m.repos[i].State = state
			return
		}
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		return m, nil

	case dataLoadedMsg:
		m.loading = false
		m.username = msg.username
		m.repos = msg.repos
		m.err = msg.err

		return m, nil

	case watchResultMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Failed to watch %s: %v", msg.repo, msg.err)
		} else {
			m.statusMsg = "Watching " + msg.repo
			m.setState(msg.repo, github.WatchStateSubscribed)
		}

		return m, nil

	case unwatchResultMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Failed to unwatch %s: %v", msg.repo, msg.err)
		} else {
			m.statusMsg = "Unwatched " + msg.repo
			m.setState(msg.repo, github.WatchStateDefault)
		}

		return m, nil

	case ignoreResultMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Failed to ignore %s: %v", msg.repo, msg.err)
		} else {
			m.statusMsg = "Ignoring " + msg.repo
			m.setState(msg.repo, github.WatchStateIgnored)
		}

		return m, nil

	case openResultMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Failed to open %s: %v", msg.repo, msg.err)
		} else {
			m.statusMsg = "Opened " + msg.repo
		}

		return m, nil

	case tea.KeyPressMsg:
		if m.searching {
			return m.handleSearchKeys(msg)
		}

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

		case "/":
			m.searching = true

		case "g":
			m.cursor = 0

		case "G":
			if filtered := m.getFilteredRepos(); len(filtered) > 0 {
				m.cursor = len(filtered) - 1
			}

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			filtered := m.getFilteredRepos()
			if m.cursor < len(filtered)-1 {
				m.cursor++
			}

		case "1":
			m.viewMode = "unwatched"
			m.cursor = 0
			m.selected = make(map[int]bool)

		case "2":
			m.viewMode = "watched"
			m.cursor = 0
			m.selected = make(map[int]bool)

		case "3":
			m.viewMode = "ignored"
			m.cursor = 0
			m.selected = make(map[int]bool)

		case "4":
			m.viewMode = "all"
			m.cursor = 0
			m.selected = make(map[int]bool)

		case "space":
			m.selected[m.cursor] = !m.selected[m.cursor]

		case "I":
			filtered := m.getFilteredRepos()
			for idx := range filtered {
				m.selected[idx] = !m.selected[idx]
			}

		case "w":
			return m.handleWatch()

		case "u":
			return m.handleUnwatch()

		case "i":
			return m.handleIgnore()

		case "o":
			if filtered := m.getFilteredRepos(); m.cursor < len(filtered) {
				return m, m.openRepo(filtered[m.cursor])
			}
		}
	}

	return m, nil
}

func (m Model) getFilteredRepos() []github.RepoWatchInfo {
	query := strings.ToLower(m.searchQuery)

	var filtered []github.RepoWatchInfo
	for _, repo := range m.repos {
		switch m.viewMode {
		case "unwatched":
			if repo.State != github.WatchStateDefault {
				continue
			}
		case "watched":
			if repo.State != github.WatchStateSubscribed {
				continue
			}
		case "ignored":
			if repo.State != github.WatchStateIgnored {
				continue
			}
		case "all":
		default:
			continue
		}

		if query != "" && !strings.Contains(strings.ToLower(repo.FullName), query) {
			continue
		}

		filtered = append(filtered, repo)
	}

	return filtered
}

func (m Model) handleSearchKeys(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc":
		m.searching = false
		m.searchQuery = ""
		m.cursor = 0

	case "enter":
		m.searching = false

	case "backspace":
		if len(m.searchQuery) > 0 {
			runes := []rune(m.searchQuery)
			m.searchQuery = string(runes[:len(runes)-1])
			m.cursor = 0
		}

	default:
		if key := msg.String(); len([]rune(key)) == 1 {
			m.searchQuery += key
			m.cursor = 0
		}
	}

	return m, nil
}

func (m Model) handleWatch() (Model, tea.Cmd) {
	filtered := m.getFilteredRepos()
	var cmds []tea.Cmd

	hasSelection := false
	for idx := range m.selected {
		if m.selected[idx] && idx < len(filtered) {
			hasSelection = true
			cmds = append(cmds, m.watchRepo(filtered[idx]))
		}
	}

	if !hasSelection && m.cursor < len(filtered) {
		cmds = append(cmds, m.watchRepo(filtered[m.cursor]))
	}

	m.selected = make(map[int]bool)

	return m, tea.Batch(cmds...)
}

func (m Model) handleUnwatch() (Model, tea.Cmd) {
	filtered := m.getFilteredRepos()
	var cmds []tea.Cmd

	hasSelection := false
	for idx := range m.selected {
		if m.selected[idx] && idx < len(filtered) {
			hasSelection = true
			cmds = append(cmds, m.unwatchRepo(filtered[idx]))
		}
	}

	if !hasSelection && m.cursor < len(filtered) {
		cmds = append(cmds, m.unwatchRepo(filtered[m.cursor]))
	}

	m.selected = make(map[int]bool)

	return m, tea.Batch(cmds...)
}

func (m Model) handleIgnore() (Model, tea.Cmd) {
	filtered := m.getFilteredRepos()
	var cmds []tea.Cmd

	hasSelection := false
	for idx := range m.selected {
		if m.selected[idx] && idx < len(filtered) {
			hasSelection = true
			cmds = append(cmds, m.ignoreRepo(filtered[idx]))
		}
	}

	if !hasSelection && m.cursor < len(filtered) {
		cmds = append(cmds, m.ignoreRepo(filtered[m.cursor]))
	}

	m.selected = make(map[int]bool)

	return m, tea.Batch(cmds...)
}

func formatDate(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}

	return t.Format("2006-01-02")
}

func repoMetadata(repo github.RepoWatchInfo) string {
	var tags []string
	if repo.IsArchived {
		tags = append(tags, "archived")
	}
	if repo.IsFork {
		tags = append(tags, "fork")
	}
	if !repo.ViewerCanSubscribe {
		tags = append(tags, "cannot subscribe")
	}

	tags = append(tags, "pushed "+formatDate(repo.PushedAt))

	return strings.Join(tags, " | ")
}

func (m Model) View() string {
	if m.loading {
		return "Loading watch status...\n"
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Primary)

	b.WriteString(titleStyle.Render("Watch Status Audit"))
	b.WriteString("\n")
	fmt.Fprintf(&b, "User: %s\n\n", m.username)

	if m.showHelp {
		return m.renderHelp(&b)
	}

	activeTab := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Warning)

	inactiveTab := lipgloss.NewStyle().
		Foreground(theme.Current().Muted)

	tab := func(label, mode string) string {
		if m.viewMode == mode {
			return activeTab.Render(label)
		}

		return inactiveTab.Render(label)
	}

	b.WriteString(tab("[1] Default", "unwatched"))
	b.WriteString("  ")
	b.WriteString(tab("[2] Watched", "watched"))
	b.WriteString("  ")
	b.WriteString(tab("[3] Ignored", "ignored"))
	b.WriteString("  ")
	b.WriteString(tab("[4] All", "all"))
	b.WriteString("\n\n")

	if m.searching || m.searchQuery != "" {
		searchStyle := lipgloss.NewStyle().Foreground(theme.Current().Warning)
		fmt.Fprintf(&b, "%s %s\n\n", searchStyle.Render("Search:"), m.searchQuery)
	}

	filtered := m.getFilteredRepos()

	if len(filtered) == 0 {
		b.WriteString("No repositories in this view.\n")
	} else {
		for i, repo := range filtered {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}

			selectMark := " "
			if m.selected[i] {
				selectMark = "*"
			}

			status := "default (participating/@mentions)"
			statusStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)
			switch repo.State {
			case github.WatchStateSubscribed:
				status = "all activity"
				statusStyle = lipgloss.NewStyle().Foreground(theme.Current().Success)
			case github.WatchStateIgnored:
				status = "ignored"
				statusStyle = lipgloss.NewStyle().Foreground(theme.Current().Warning)
			}

			lineStyle := lipgloss.NewStyle()
			if m.cursor == i {
				lineStyle = lineStyle.Bold(true).Foreground(theme.Current().Warning)
			}

			line := fmt.Sprintf("%s%s %s ", cursor, selectMark, repo.FullName)
			b.WriteString(lineStyle.Render(line))
			b.WriteString(statusStyle.Render(fmt.Sprintf("[%s]", status)))

			metaStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)
			b.WriteString(" ")
			b.WriteString(metaStyle.Render(repoMetadata(repo)))
			b.WriteString("\n")
		}
	}

	if m.statusMsg != "" {
		b.WriteString("\n")
		statusStyle := lipgloss.NewStyle().Foreground(theme.Current().Primary)
		b.WriteString(statusStyle.Render(m.statusMsg))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)
	b.WriteString(
		helpStyle.Render(
			"j/k: navigate | space: select | I: invert selection | w: watch all activity | u: unwatch (default) | " +
				"i: ignore | o: open on github.com | 1/2/3/4: view mode | ?: help | esc: back",
		),
	)
	b.WriteString("\n")
	b.WriteString(
		helpStyle.Render(
			"GitHub's API can't see or set \"Custom\" per-notification-type settings; manage those on github.com.",
		),
	)

	return b.String()
}

func (m Model) renderHelp(b *strings.Builder) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Primary)
	b.WriteString(titleStyle.Render("Keybindings"))
	b.WriteString("\n\n")

	bindings := [][2]string{
		{"j/k, up/down", "move the cursor"},
		{"g / G", "jump to top / bottom"},
		{"space", "toggle selection on the cursor row"},
		{"I", "invert selection"},
		{"/", "search repository name"},
		{"1-4", "default, watched, ignored, all"},
		{"w / u / i", "watch all activity / unwatch / ignore"},
		{"o", "open on github.com"},
		{"?", "toggle this help"},
		{"esc / q", "back / quit"},
	}

	for _, binding := range bindings {
		keyStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Current().Warning).Width(16)
		fmt.Fprintf(b, "%s %s\n", keyStyle.Render(binding[0]), binding[1])
	}

	b.WriteString("\nPress '?' or 'esc' to close\n")

	return b.String()
}
