package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func setupTestRepo(t *testing.T) string {
	t.Helper()

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
	if err := configName.Run(); err != nil {
		t.Fatalf("Failed to configure user.name: %v", err)
	}

	configEmail := exec.Command("git", "config", "user.email", "test@example.com")
	configEmail.Dir = tmpDir
	if err := configEmail.Run(); err != nil {
		t.Fatalf("Failed to configure user.email: %v", err)
	}

	// Disable commit signing for tests
	configSign := exec.Command("git", "config", "commit.gpgsign", "false")
	configSign.Dir = tmpDir
	if err := configSign.Run(); err != nil {
		t.Fatalf("Failed to disable commit signing: %v", err)
	}

	// Create initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to stage test file: %v", err)
	}

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

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func TestCompareBranches(t *testing.T) {
	repoPath := setupTestRepo(t)
	repo := NewLocalRepo(repoPath)
	base, err := repo.GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch() error = %v", err)
	}

	runGit(t, repoPath, "checkout", "-b", "feature")
	if err := os.WriteFile(filepath.Join(repoPath, "feature.txt"), []byte("feature"), 0o644); err != nil {
		t.Fatalf("failed to write feature file: %v", err)
	}
	runGit(t, repoPath, "add", "feature.txt")
	runGit(t, repoPath, "commit", "-m", "feature commit")

	ahead, behind, err := repo.CompareBranches(base, "feature")
	if err != nil {
		t.Fatalf("CompareBranches() error = %v", err)
	}
	if ahead != 1 || behind != 0 {
		t.Errorf("CompareBranches() = (ahead=%d, behind=%d), want (1, 0)", ahead, behind)
	}
}

func TestGetMergeBase(t *testing.T) {
	repoPath := setupTestRepo(t)
	repo := NewLocalRepo(repoPath)
	base, err := repo.GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch() error = %v", err)
	}

	baseSHA, err := repo.GetMergeBase(base, base)
	if err != nil {
		t.Fatalf("GetMergeBase() error = %v", err)
	}

	runGit(t, repoPath, "checkout", "-b", "feature")
	if err := os.WriteFile(filepath.Join(repoPath, "feature.txt"), []byte("feature"), 0o644); err != nil {
		t.Fatalf("failed to write feature file: %v", err)
	}
	runGit(t, repoPath, "add", "feature.txt")
	runGit(t, repoPath, "commit", "-m", "feature commit")

	mergeBase, err := repo.GetMergeBase(base, "feature")
	if err != nil {
		t.Fatalf("GetMergeBase() error = %v", err)
	}
	if mergeBase != baseSHA {
		t.Errorf("GetMergeBase() = %q, want the base branch's tip %q", mergeBase, baseSHA)
	}
}

func TestDeleteBranch(t *testing.T) {
	repoPath := setupTestRepo(t)
	repo := NewLocalRepo(repoPath)

	runGit(t, repoPath, "branch", "throwaway")

	if err := repo.DeleteBranch("throwaway", false); err != nil {
		t.Fatalf("DeleteBranch() error = %v", err)
	}

	branches, err := repo.ListBranches()
	if err != nil {
		t.Fatalf("ListBranches() error = %v", err)
	}
	for _, b := range branches {
		if b.Name == "throwaway" {
			t.Error("throwaway branch still present after DeleteBranch()")
		}
	}

	if err := repo.DeleteBranch("does-not-exist", false); err == nil {
		t.Error("expected an error deleting a branch that doesn't exist")
	}
}
