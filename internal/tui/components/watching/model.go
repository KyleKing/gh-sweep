package watching

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/tui/theme"
)

type Model struct {
	username      string
	userRepos     []github.RepoBasic
	subscriptions map[string]*github.Subscription
	fetchErrors   map[string]error
	cursor        int
	width         int
	height        int
	loading       bool
	err           error
	viewMode      string
	selected      map[int]bool
	statusMsg     string
}

func NewModel() Model {
	return Model{
		subscriptions: make(map[string]*github.Subscription),
		fetchErrors:   make(map[string]error),
		selected:      make(map[int]bool),
		loading:       true,
		viewMode:      "unwatched",
	}
}

type dataLoadedMsg struct {
	username      string
	userRepos     []github.RepoBasic
	subscriptions map[string]*github.Subscription
	fetchErrors   map[string]error
	err           error
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

func (m Model) Init() tea.Cmd {
	return m.loadData
}

func (m Model) loadData() tea.Msg {
	ctx := context.Background()
	client, err := github.NewClient(ctx)
	if err != nil {
		return dataLoadedMsg{err: fmt.Errorf("failed to create GitHub client: %w", err)}
	}

	username, err := client.GetAuthenticatedUser()
	if err != nil {
		return dataLoadedMsg{err: fmt.Errorf("failed to get authenticated user: %w", err)}
	}

	repos, err := client.ListUserRepos()
	if err != nil {
		return dataLoadedMsg{err: fmt.Errorf("failed to list user repos: %w", err)}
	}

	subscriptions := make(map[string]*github.Subscription)
	fetchErrors := make(map[string]error)
	for _, repo := range repos {
		sub, err := client.GetRepoSubscription(repo.Owner, repo.Name)
		if err != nil {
			fetchErrors[repo.FullName] = err
			continue
		}
		subscriptions[repo.FullName] = sub
	}

	return dataLoadedMsg{
		username:      username,
		userRepos:     repos,
		subscriptions: subscriptions,
		fetchErrors:   fetchErrors,
		err:           nil,
	}
}

func (m Model) watchRepo(repo github.RepoBasic) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		client, err := github.NewClient(ctx)
		if err != nil {
			return watchResultMsg{repo: repo.FullName, err: err}
		}

		sub, err := client.SetRepoSubscription(repo.Owner, repo.Name, true, false)
		if err != nil {
			return watchResultMsg{repo: repo.FullName, err: err}
		}

		m.subscriptions[repo.FullName] = sub

		return watchResultMsg{repo: repo.FullName, err: nil}
	}
}

func (m Model) unwatchRepo(repo github.RepoBasic) tea.Cmd {
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

func (m Model) ignoreRepo(repo github.RepoBasic) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		client, err := github.NewClient(ctx)
		if err != nil {
			return ignoreResultMsg{repo: repo.FullName, err: err}
		}

		sub, err := client.SetRepoSubscription(repo.Owner, repo.Name, false, true)
		if err != nil {
			return ignoreResultMsg{repo: repo.FullName, err: err}
		}

		m.subscriptions[repo.FullName] = sub

		return ignoreResultMsg{repo: repo.FullName, err: nil}
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
		m.userRepos = msg.userRepos
		m.subscriptions = msg.subscriptions
		m.fetchErrors = msg.fetchErrors
		m.err = msg.err

		return m, nil

	case watchResultMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Failed to watch %s: %v", msg.repo, msg.err)
		} else {
			m.statusMsg = "Watching " + msg.repo
			if sub, ok := m.subscriptions[msg.repo]; ok {
				sub.Subscribed = true
				sub.Ignored = false
				sub.State = github.WatchStateSubscribed
			}
		}

		return m, nil

	case unwatchResultMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Failed to unwatch %s: %v", msg.repo, msg.err)
		} else {
			m.statusMsg = "Unwatched " + msg.repo
			if sub, ok := m.subscriptions[msg.repo]; ok {
				sub.Subscribed = false
				sub.Ignored = false
				sub.State = github.WatchStateDefault
			}
		}

		return m, nil

	case ignoreResultMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Failed to ignore %s: %v", msg.repo, msg.err)
		} else {
			m.statusMsg = "Ignoring " + msg.repo
			if sub, ok := m.subscriptions[msg.repo]; ok {
				sub.Subscribed = false
				sub.Ignored = true
				sub.State = github.WatchStateIgnored
			}
		}

		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

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
			m.viewMode = "all"
			m.cursor = 0
			m.selected = make(map[int]bool)

		case "space":
			m.selected[m.cursor] = !m.selected[m.cursor]

		case "w":
			return m.handleWatch()

		case "u":
			return m.handleUnwatch()

		case "i":
			return m.handleIgnore()
		}
	}

	return m, nil
}

func (m Model) getFilteredRepos() []github.RepoBasic {
	var filtered []github.RepoBasic
	for _, repo := range m.userRepos {
		if _, failed := m.fetchErrors[repo.FullName]; failed && m.viewMode != "all" {
			continue
		}

		sub := m.subscriptions[repo.FullName]
		switch m.viewMode {
		case "unwatched":
			if sub != nil && sub.State == github.WatchStateDefault {
				filtered = append(filtered, repo)
			}
		case "watched":
			if sub != nil && sub.State == github.WatchStateSubscribed {
				filtered = append(filtered, repo)
			}
		case "all":
			filtered = append(filtered, repo)
		}
	}

	return filtered
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
	b.WriteString(fmt.Sprintf("User: %s\n\n", m.username))

	activeTab := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current().Warning)

	inactiveTab := lipgloss.NewStyle().
		Foreground(theme.Current().Muted)

	if m.viewMode == "unwatched" {
		b.WriteString(activeTab.Render("[1] Unwatched"))
	} else {
		b.WriteString(inactiveTab.Render("[1] Unwatched"))
	}
	b.WriteString("  ")
	if m.viewMode == "watched" {
		b.WriteString(activeTab.Render("[2] Watched"))
	} else {
		b.WriteString(inactiveTab.Render("[2] Watched"))
	}
	b.WriteString("  ")
	if m.viewMode == "all" {
		b.WriteString(activeTab.Render("[3] All"))
	} else {
		b.WriteString(inactiveTab.Render("[3] All"))
	}
	b.WriteString("\n\n")

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

			sub := m.subscriptions[repo.FullName]
			status := "default (participating/@mentions)"
			statusStyle := lipgloss.NewStyle().Foreground(theme.Current().Muted)
			if fetchErr, failed := m.fetchErrors[repo.FullName]; failed {
				status = fmt.Sprintf("error: %v", fetchErr)
				statusStyle = lipgloss.NewStyle().Foreground(theme.Current().Error)
			} else if sub != nil {
				switch sub.State {
				case github.WatchStateSubscribed:
					status = "all activity"
					statusStyle = lipgloss.NewStyle().Foreground(theme.Current().Success)
				case github.WatchStateIgnored:
					status = "ignored"
					statusStyle = lipgloss.NewStyle().Foreground(theme.Current().Warning)
				}
			}

			lineStyle := lipgloss.NewStyle()
			if m.cursor == i {
				lineStyle = lineStyle.Bold(true).Foreground(theme.Current().Warning)
			}

			line := fmt.Sprintf("%s%s %s ", cursor, selectMark, repo.FullName)
			b.WriteString(lineStyle.Render(line))
			b.WriteString(statusStyle.Render(fmt.Sprintf("[%s]", status)))
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
	b.WriteString(helpStyle.Render("j/k: navigate | space: select | w: watch all activity | u: unwatch (default) | i: ignore | 1/2/3: view mode | esc: back"))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("GitHub's API doesn't expose per-notification-type custom settings; manage those on github.com."))

	return b.String()
}
