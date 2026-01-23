package orphans

import (
	"testing"
	"time"

	"github.com/KyleKing/gh-sweep/internal/github"
)

func TestDetector_ClassifyBranch_MergedPR(t *testing.T) {
	opts := DefaultScanOptions()
	detector := NewDetector(opts)

	repo := github.Repository{
		Name:          "test-repo",
		FullName:      "owner/test-repo",
		Owner:         "owner",
		DefaultBranch: "main",
	}

	branch := github.Branch{
		Name:           "feature-branch",
		SHA:            "abc123",
		Protected:      false,
		LastCommitDate: time.Now().Add(-24 * time.Hour),
	}

	mergedAt := time.Now().Add(-12 * time.Hour)
	prs := []github.PullRequest{
		{
			Number:   1,
			Title:    "Feature PR",
			State:    "closed",
			Head:     github.PRRef{Ref: "feature-branch"},
			MergedAt: &mergedAt,
		},
	}

	orphan := detector.ClassifyBranch(repo, branch, prs)

	if orphan == nil {
		t.Fatal("expected orphan, got nil")
	}

	if orphan.Type != OrphanTypeMergedPR {
		t.Errorf("expected type %s, got %s", OrphanTypeMergedPR, orphan.Type)
	}

	if orphan.PRNumber == nil || *orphan.PRNumber != 1 {
		t.Errorf("expected PR number 1, got %v", orphan.PRNumber)
	}
}

func TestDetector_ClassifyBranch_ClosedPR(t *testing.T) {
	opts := DefaultScanOptions()
	detector := NewDetector(opts)

	repo := github.Repository{
		Name:          "test-repo",
		FullName:      "owner/test-repo",
		Owner:         "owner",
		DefaultBranch: "main",
	}

	branch := github.Branch{
		Name:           "feature-branch",
		SHA:            "abc123",
		Protected:      false,
		LastCommitDate: time.Now().Add(-24 * time.Hour),
	}

	closedAt := time.Now().Add(-12 * time.Hour)
	prs := []github.PullRequest{
		{
			Number:   2,
			Title:    "Closed PR",
			State:    "closed",
			Head:     github.PRRef{Ref: "feature-branch"},
			MergedAt: nil,
			ClosedAt: &closedAt,
		},
	}

	orphan := detector.ClassifyBranch(repo, branch, prs)

	if orphan == nil {
		t.Fatal("expected orphan, got nil")
	}

	if orphan.Type != OrphanTypeClosedPR {
		t.Errorf("expected type %s, got %s", OrphanTypeClosedPR, orphan.Type)
	}
}

func TestDetector_ClassifyBranch_OpenPR(t *testing.T) {
	opts := DefaultScanOptions()
	detector := NewDetector(opts)

	repo := github.Repository{
		Name:          "test-repo",
		FullName:      "owner/test-repo",
		Owner:         "owner",
		DefaultBranch: "main",
	}

	branch := github.Branch{
		Name:           "feature-branch",
		SHA:            "abc123",
		Protected:      false,
		LastCommitDate: time.Now().Add(-24 * time.Hour),
	}

	prs := []github.PullRequest{
		{
			Number: 3,
			Title:  "Open PR",
			State:  "open",
			Head:   github.PRRef{Ref: "feature-branch"},
		},
	}

	orphan := detector.ClassifyBranch(repo, branch, prs)

	if orphan != nil {
		t.Errorf("expected nil for open PR, got %+v", orphan)
	}
}

func TestDetector_ClassifyBranch_Stale(t *testing.T) {
	opts := DefaultScanOptions()
	opts.StaleDaysThreshold = 7
	detector := NewDetector(opts)

	repo := github.Repository{
		Name:          "test-repo",
		FullName:      "owner/test-repo",
		Owner:         "owner",
		DefaultBranch: "main",
	}

	branch := github.Branch{
		Name:           "old-branch",
		SHA:            "abc123",
		Protected:      false,
		LastCommitDate: time.Now().Add(-14 * 24 * time.Hour),
	}

	orphan := detector.ClassifyBranch(repo, branch, nil)

	if orphan == nil {
		t.Fatal("expected orphan, got nil")
	}

	if orphan.Type != OrphanTypeStale {
		t.Errorf("expected type %s, got %s", OrphanTypeStale, orphan.Type)
	}

	if orphan.DaysSinceActivity < 14 {
		t.Errorf("expected at least 14 days since activity, got %d", orphan.DaysSinceActivity)
	}
}

func TestDetector_ClassifyBranch_RecentNoPR(t *testing.T) {
	opts := DefaultScanOptions()
	opts.StaleDaysThreshold = 7
	opts.IncludeRecentNoPR = true
	detector := NewDetector(opts)

	repo := github.Repository{
		Name:          "test-repo",
		FullName:      "owner/test-repo",
		Owner:         "owner",
		DefaultBranch: "main",
	}

	branch := github.Branch{
		Name:           "recent-branch",
		SHA:            "abc123",
		Protected:      false,
		LastCommitDate: time.Now().Add(-2 * 24 * time.Hour),
	}

	orphan := detector.ClassifyBranch(repo, branch, nil)

	if orphan == nil {
		t.Fatal("expected orphan, got nil")
	}

	if orphan.Type != OrphanTypeRecentNoPR {
		t.Errorf("expected type %s, got %s", OrphanTypeRecentNoPR, orphan.Type)
	}
}

func TestDetector_ClassifyBranch_RecentNoPR_Disabled(t *testing.T) {
	opts := DefaultScanOptions()
	opts.StaleDaysThreshold = 7
	opts.IncludeRecentNoPR = false
	detector := NewDetector(opts)

	repo := github.Repository{
		Name:          "test-repo",
		FullName:      "owner/test-repo",
		Owner:         "owner",
		DefaultBranch: "main",
	}

	branch := github.Branch{
		Name:           "recent-branch",
		SHA:            "abc123",
		Protected:      false,
		LastCommitDate: time.Now().Add(-2 * 24 * time.Hour),
	}

	orphan := detector.ClassifyBranch(repo, branch, nil)

	if orphan != nil {
		t.Errorf("expected nil for recent branch without IncludeRecentNoPR, got %+v", orphan)
	}
}

func TestDetector_ClassifyBranch_ExcludePattern(t *testing.T) {
	opts := DefaultScanOptions()
	detector := NewDetector(opts)

	repo := github.Repository{
		Name:          "test-repo",
		FullName:      "owner/test-repo",
		Owner:         "owner",
		DefaultBranch: "main",
	}

	branch := github.Branch{
		Name:           "main",
		SHA:            "abc123",
		Protected:      false,
		LastCommitDate: time.Now().Add(-30 * 24 * time.Hour),
	}

	orphan := detector.ClassifyBranch(repo, branch, nil)

	if orphan != nil {
		t.Errorf("expected nil for excluded branch 'main', got %+v", orphan)
	}
}

func TestDetector_ClassifyBranch_ExcludeWildcard(t *testing.T) {
	opts := DefaultScanOptions()
	detector := NewDetector(opts)

	repo := github.Repository{
		Name:          "test-repo",
		FullName:      "owner/test-repo",
		Owner:         "owner",
		DefaultBranch: "main",
	}

	branch := github.Branch{
		Name:           "release/v1.0",
		SHA:            "abc123",
		Protected:      false,
		LastCommitDate: time.Now().Add(-30 * 24 * time.Hour),
	}

	orphan := detector.ClassifyBranch(repo, branch, nil)

	if orphan != nil {
		t.Errorf("expected nil for excluded branch 'release/*', got %+v", orphan)
	}
}

func TestDetector_ClassifyBranch_ProtectedSkipped(t *testing.T) {
	opts := DefaultScanOptions()
	opts.IncludeProtected = false
	detector := NewDetector(opts)

	repo := github.Repository{
		Name:          "test-repo",
		FullName:      "owner/test-repo",
		Owner:         "owner",
		DefaultBranch: "main",
	}

	branch := github.Branch{
		Name:           "feature-branch",
		SHA:            "abc123",
		Protected:      true,
		LastCommitDate: time.Now().Add(-30 * 24 * time.Hour),
	}

	orphan := detector.ClassifyBranch(repo, branch, nil)

	if orphan != nil {
		t.Errorf("expected nil for protected branch, got %+v", orphan)
	}
}

func TestDetector_ClassifyBranch_ProtectedIncluded(t *testing.T) {
	opts := DefaultScanOptions()
	opts.IncludeProtected = true
	opts.ExcludePatterns = []string{}
	detector := NewDetector(opts)

	repo := github.Repository{
		Name:          "test-repo",
		FullName:      "owner/test-repo",
		Owner:         "owner",
		DefaultBranch: "main",
	}

	branch := github.Branch{
		Name:           "feature-branch",
		SHA:            "abc123",
		Protected:      true,
		LastCommitDate: time.Now().Add(-30 * 24 * time.Hour),
	}

	orphan := detector.ClassifyBranch(repo, branch, nil)

	if orphan == nil {
		t.Fatal("expected orphan for protected branch with IncludeProtected, got nil")
	}

	if !orphan.Protected {
		t.Error("expected orphan.Protected to be true")
	}
}

func TestOrphanType_Label(t *testing.T) {
	tests := []struct {
		orphanType OrphanType
		expected   string
	}{
		{OrphanTypeMergedPR, "Merged PR"},
		{OrphanTypeClosedPR, "Closed PR"},
		{OrphanTypeStale, "Stale"},
		{OrphanTypeRecentNoPR, "Recent (no PR)"},
	}

	for _, tt := range tests {
		t.Run(string(tt.orphanType), func(t *testing.T) {
			if got := tt.orphanType.Label(); got != tt.expected {
				t.Errorf("Label() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestOrphanedBranch_Key(t *testing.T) {
	orphan := OrphanedBranch{
		Repository: "owner/repo",
		BranchName: "feature",
	}

	expected := "owner/repo/feature"
	if got := orphan.Key(); got != expected {
		t.Errorf("Key() = %s, want %s", got, expected)
	}
}

func TestDefaultScanOptions(t *testing.T) {
	opts := DefaultScanOptions()

	if opts.StaleDaysThreshold != 7 {
		t.Errorf("StaleDaysThreshold = %d, want 7", opts.StaleDaysThreshold)
	}

	if opts.IncludeRecentNoPR {
		t.Error("IncludeRecentNoPR should be false by default")
	}

	if opts.IncludeProtected {
		t.Error("IncludeProtected should be false by default")
	}

	if opts.Concurrency != 5 {
		t.Errorf("Concurrency = %d, want 5", opts.Concurrency)
	}

	expectedExcludes := []string{"main", "master", "develop", "release/*", "hotfix/*"}
	if len(opts.ExcludePatterns) != len(expectedExcludes) {
		t.Errorf("ExcludePatterns length = %d, want %d", len(opts.ExcludePatterns), len(expectedExcludes))
	}
}
