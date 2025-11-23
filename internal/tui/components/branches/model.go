package branches

import (
	"context"
	"fmt"
	"strings"

	"github.com/KyleKing/gh-sweep/internal/git"
	"github.com/KyleKing/gh-sweep/internal/github"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the branch management TUI state
type Model struct {
	repo         string
	branches     []github.BranchWithComparison
	selected     map[int]bool
	cursor       int
	width        int
	height       int
	loading      bool
	err          error
	baseBranch   string
	showTree     bool
}

// NewModel creates a new branch management model
func NewModel(repo, baseBranch string) Model {
	return Model{
		repo:       repo,
		baseBranch: baseBranch,
		selected:   make(map[int]bool),
		loading:    true,
	}
}

type branchesLoadedMsg struct {
	branches []github.BranchWithComparison
	err      error
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return m.loadBranches
}

func (m Model) loadBranches() tea.Msg {
	// If no repo specified, return empty
	if m.repo == "" {
		return branchesLoadedMsg{
			branches: []github.BranchWithComparison{},
			err:      fmt.Errorf("no repository specified"),
		}
	}

	// Parse repo (owner/name format)
	parts := strings.Split(m.repo, "/")
	if len(parts) != 2 {
		return branchesLoadedMsg{
			branches: []github.BranchWithComparison{},
			err:      fmt.Errorf("invalid repo format, expected owner/repo"),
		}
	}
	owner, repo := parts[0], parts[1]

	// Create GitHub client
	ctx := context.Background()
	client, err := github.NewClient(ctx)
	if err != nil {
		return branchesLoadedMsg{
			branches: []github.BranchWithComparison{},
			err:      fmt.Errorf("failed to create GitHub client: %w", err),
		}
	}

	// Load branches from GitHub
	branches, err := client.ListBranches(owner, repo)
	if err != nil {
		return branchesLoadedMsg{
			branches: []github.BranchWithComparison{},
			err:      fmt.Errorf("failed to load branches: %w", err),
		}
	}

	// Use default base branch if not specified
	baseBranch := m.baseBranch
	if baseBranch == "" {
		baseBranch = "main"
	}

	// Build comparison info for each branch
	branchesWithComparison := make([]github.BranchWithComparison, 0, len(branches))
	for _, branch := range branches {
		// Skip comparison for base branch
		if branch.Name != baseBranch {
			ahead, behind, _ := client.CompareBranches(owner, repo, baseBranch, branch.Name)
			branch.Ahead = ahead
			branch.Behind = behind
		}

		branchesWithComparison = append(branchesWithComparison, github.BranchWithComparison{
			Branch:     branch,
			ComparedTo: baseBranch,
		})
	}

	return branchesLoadedMsg{
		branches: branchesWithComparison,
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

	case branchesLoadedMsg:
		m.loading = false
		m.branches = msg.branches
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
			if m.cursor < len(m.branches)-1 {
				m.cursor++
			}

		case " ": // Space to select
			m.selected[m.cursor] = !m.selected[m.cursor]

		case "a": // Select all
			for i := range m.branches {
				m.selected[i] = true
			}

		case "n": // Select none
			m.selected = make(map[int]bool)

		case "t": // Toggle tree view
			m.showTree = !m.showTree

		case "d": // Delete selected
			// TODO: Implement delete confirmation
			return m, nil
		}
	}

	return m, nil
}

// View renders the model
func (m Model) View() string {
	if m.loading {
		return "Loading branches...\n"
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var b strings.Builder

	// Header
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FFFF"))

	b.WriteString(titleStyle.Render(fmt.Sprintf("📋 Branches for %s", m.repo)))
	b.WriteString("\n\n")

	// Branch list
	if len(m.branches) == 0 {
		b.WriteString("No branches found.\n")
	} else {
		for i, branch := range m.branches {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}

			checkbox := "[ ]"
			if m.selected[i] {
				checkbox = "[✓]"
			}

			aheadBehind := fmt.Sprintf("↑%d ↓%d", branch.Ahead, branch.Behind)

			line := fmt.Sprintf("%s %s %s %s",
				cursor,
				checkbox,
				branch.Name,
				aheadBehind,
			)

			if m.cursor == i {
				selectedStyle := lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("#FFFF00"))
				b.WriteString(selectedStyle.Render(line))
			} else {
				b.WriteString(line)
			}
			b.WriteString("\n")
		}
	}

	// Help
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))
	b.WriteString(helpStyle.Render("↑/↓: navigate | space: select | a: all | n: none | t: tree | d: delete | q: quit"))

	return b.String()
}

// GetLocalBranches loads branches from local Git repository
func GetLocalBranches(repoPath string) ([]git.BranchInfo, error) {
	repo := git.NewLocalRepo(repoPath)
	return repo.ListBranches()
}
