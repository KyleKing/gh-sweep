package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// LocalRepo represents a local Git repository
type LocalRepo struct {
	Path string
}

// BranchInfo represents information about a branch
type BranchInfo struct {
	Name           string
	SHA            string
	Ahead          int
	Behind         int
	LastCommitDate time.Time
	LastCommitMsg  string
}

// NewLocalRepo creates a new local repository handle
func NewLocalRepo(path string) *LocalRepo {
	return &LocalRepo{Path: path}
}

// ListBranches lists all local branches
func (r *LocalRepo) ListBranches() ([]BranchInfo, error) {
	// Run: git for-each-ref --format='%(refname:short)|%(objectname)|%(committerdate:iso8601)|%(subject)' refs/heads
	cmd := exec.Command("git", "for-each-ref",
		"--format=%(refname:short)|%(objectname)|%(committerdate:iso8601)|%(subject)",
		"refs/heads")
	cmd.Dir = r.Path

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	branches := make([]BranchInfo, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) != 4 {
			continue
		}

		date, _ := time.Parse("2006-01-02 15:04:05 -0700", parts[2])

		branches = append(branches, BranchInfo{
			Name:           parts[0],
			SHA:            parts[1],
			LastCommitDate: date,
			LastCommitMsg:  parts[3],
		})
	}

	return branches, nil
}

// GetCurrentBranch returns the current branch name
func (r *LocalRepo) GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = r.Path

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(out.String()), nil
}

// CompareBranches compares two branches and returns ahead/behind counts
func (r *LocalRepo) CompareBranches(base, head string) (ahead, behind int, err error) {
	// Run: git rev-list --left-right --count base...head
	cmd := exec.Command("git", "rev-list", "--left-right", "--count", fmt.Sprintf("%s...%s", base, head))
	cmd.Dir = r.Path

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return 0, 0, fmt.Errorf("failed to compare branches: %w", err)
	}

	// Output format: "behind\tahead\n"
	parts := strings.Fields(strings.TrimSpace(out.String()))
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected git output: %s", out.String())
	}

	// Parse counts
	fmt.Sscanf(parts[0], "%d", &behind)
	fmt.Sscanf(parts[1], "%d", &ahead)

	return ahead, behind, nil
}

// DeleteBranch deletes a branch locally
func (r *LocalRepo) DeleteBranch(branch string, force bool) error {
	args := []string{"branch"}
	if force {
		args = append(args, "-D")
	} else {
		args = append(args, "-d")
	}
	args = append(args, branch)

	cmd := exec.Command("git", args...)
	cmd.Dir = r.Path

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete branch %s: %w", branch, err)
	}

	return nil
}

// GetMergeBase returns the merge base of two branches
func (r *LocalRepo) GetMergeBase(branch1, branch2 string) (string, error) {
	cmd := exec.Command("git", "merge-base", branch1, branch2)
	cmd.Dir = r.Path

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get merge base: %w", err)
	}

	return strings.TrimSpace(out.String()), nil
}

// GetDefaultBranch attempts to get the default branch (main or master)
func (r *LocalRepo) GetDefaultBranch() (string, error) {
	// Try to get from remote
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	cmd.Dir = r.Path

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err == nil {
		// Format: refs/remotes/origin/main
		ref := strings.TrimSpace(out.String())
		parts := strings.Split(ref, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1], nil
		}
	}

	// Fallback: check if main or master exists
	branches, err := r.ListBranches()
	if err != nil {
		return "", err
	}

	for _, b := range branches {
		if b.Name == "main" {
			return "main", nil
		}
		if b.Name == "master" {
			return "master", nil
		}
	}

	// Last resort: return first branch
	if len(branches) > 0 {
		return branches[0].Name, nil
	}

	return "", fmt.Errorf("no branches found")
}

// IsInsideWorkTree checks if the path is inside a Git repository
func (r *LocalRepo) IsInsideWorkTree() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = r.Path

	return cmd.Run() == nil
}
