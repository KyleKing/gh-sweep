package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func setupTestRepo(t *testing.T) string {
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git
	configName := exec.Command("git", "config", "user.name", "Test User")
	configName.Dir = tmpDir
	configName.Run()

	configEmail := exec.Command("git", "config", "user.email", "test@example.com")
	configEmail.Dir = tmpDir
	configEmail.Run()

	// Disable commit signing for tests
	configSign := exec.Command("git", "config", "commit.gpgsign", "false")
	configSign.Dir = tmpDir
	configSign.Run()

	// Create initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	return tmpDir
}

func TestListBranches(t *testing.T) {
	repoPath := setupTestRepo(t)
	repo := NewLocalRepo(repoPath)

	branches, err := repo.ListBranches()
	if err != nil {
		t.Fatalf("Failed to list branches: %v", err)
	}

	if len(branches) == 0 {
		t.Fatal("Expected at least one branch")
	}

	// Should have master or main branch
	found := false
	for _, b := range branches {
		if b.Name == "master" || b.Name == "main" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find master or main branch")
	}
}

func TestGetCurrentBranch(t *testing.T) {
	repoPath := setupTestRepo(t)
	repo := NewLocalRepo(repoPath)

	branch, err := repo.GetCurrentBranch()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}

	if branch == "" {
		t.Error("Expected non-empty branch name")
	}
}

func TestIsInsideWorkTree(t *testing.T) {
	repoPath := setupTestRepo(t)
	repo := NewLocalRepo(repoPath)

	if !repo.IsInsideWorkTree() {
		t.Error("Expected to be inside work tree")
	}

	// Test with non-repo directory
	tmpDir := t.TempDir()
	nonRepo := NewLocalRepo(tmpDir)

	if nonRepo.IsInsideWorkTree() {
		t.Error("Expected NOT to be inside work tree")
	}
}

func TestGetDefaultBranch(t *testing.T) {
	repoPath := setupTestRepo(t)
	repo := NewLocalRepo(repoPath)

	defaultBranch, err := repo.GetDefaultBranch()
	if err != nil {
		t.Fatalf("Failed to get default branch: %v", err)
	}

	if defaultBranch != "master" && defaultBranch != "main" {
		t.Errorf("Expected default branch to be master or main, got %s", defaultBranch)
	}
}
