package orphans_test

import (
	"testing"
	"time"

	"github.com/KyleKing/gh-sweep/internal/github"
	"github.com/KyleKing/gh-sweep/internal/orphans"
)

func TestDetector_ClassifyBranch_MergedPR(t *testing.T) {
	t.Parallel()

	opts := orphans.DefaultScanOptions()
	detector := orphans.NewDetector(opts)

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

	if orphan.Type != orphans.OrphanTypeMergedPR {
		t.Errorf("expected type %s, got %s", orphans.OrphanTypeMergedPR, orphan.Type)
	}

	if orphan.PRNumber == nil || *orphan.PRNumber != 1 {
		t.Errorf("expected PR number 1, got %v", orphan.PRNumber)
	}
}

func TestDetector_ClassifyBranch_ClosedPR(t *testing.T) {
	t.Parallel()

	opts := orphans.DefaultScanOptions()
	detector := orphans.NewDetector(opts)

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

	if orphan.Type != orphans.OrphanTypeClosedPR {
		t.Errorf("expected type %s, got %s", orphans.OrphanTypeClosedPR, orphan.Type)
	}
}

func TestDetector_ClassifyBranch_OpenPR(t *testing.T) {
	t.Parallel()

	opts := orphans.DefaultScanOptions()
	detector := orphans.NewDetector(opts)

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
	t.Parallel()

	opts := orphans.DefaultScanOptions()
	opts.StaleDaysThreshold = 7
	detector := orphans.NewDetector(opts)

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

	if orphan.Type != orphans.OrphanTypeStale {
		t.Errorf("expected type %s, got %s", orphans.OrphanTypeStale, orphan.Type)
	}

	if orphan.DaysSinceActivity < 14 {
		t.Errorf("expected at least 14 days since activity, got %d", orphan.DaysSinceActivity)
	}
}

func TestDetector_ClassifyBranch_RecentNoPR(t *testing.T) {
	t.Parallel()

	opts := orphans.DefaultScanOptions()
	opts.StaleDaysThreshold = 7
	opts.IncludeRecentNoPR = true
	detector := orphans.NewDetector(opts)

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

	if orphan.Type != orphans.OrphanTypeRecentNoPR {
		t.Errorf("expected type %s, got %s", orphans.OrphanTypeRecentNoPR, orphan.Type)
	}
}

func TestDetector_ClassifyBranch_RecentNoPR_Disabled(t *testing.T) {
	t.Parallel()

	opts := orphans.DefaultScanOptions()
	opts.StaleDaysThreshold = 7
	opts.IncludeRecentNoPR = false
	detector := orphans.NewDetector(opts)

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

func TestDetector_ClassifyBranch_Excluded(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		branchName string
	}{
		{"literal pattern", "main"},
		{"wildcard pattern", "release/v1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := orphans.DefaultScanOptions()
			detector := orphans.NewDetector(opts)

			repo := github.Repository{
				Name:          "test-repo",
				FullName:      "owner/test-repo",
				Owner:         "owner",
				DefaultBranch: "main",
			}

			branch := github.Branch{
				Name:           tt.branchName,
				SHA:            "abc123",
				Protected:      false,
				LastCommitDate: time.Now().Add(-30 * 24 * time.Hour),
			}

			orphan := detector.ClassifyBranch(repo, branch, nil)

			if orphan != nil {
				t.Errorf("expected nil for excluded branch %q, got %+v", tt.branchName, orphan)
			}
		})
	}
}

func TestDetector_ClassifyBranch_ProtectedSkipped(t *testing.T) {
	t.Parallel()

	opts := orphans.DefaultScanOptions()
	opts.IncludeProtected = false
	detector := orphans.NewDetector(opts)

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
	t.Parallel()

	opts := orphans.DefaultScanOptions()
	opts.IncludeProtected = true
	opts.ExcludePatterns = []string{}
	detector := orphans.NewDetector(opts)

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
	t.Parallel()

	tests := []struct {
		orphanType orphans.OrphanType
		expected   string
	}{
		{orphans.OrphanTypeMergedPR, "Merged PR"},
		{orphans.OrphanTypeClosedPR, "Closed PR"},
		{orphans.OrphanTypeStale, "Stale"},
		{orphans.OrphanTypeRecentNoPR, "Recent (no PR)"},
	}

	for _, tt := range tests {
		t.Run(string(tt.orphanType), func(t *testing.T) {
			t.Parallel()

			if got := tt.orphanType.Label(); got != tt.expected {
				t.Errorf("Label() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestOrphanedBranch_Key(t *testing.T) {
	t.Parallel()

	orphan := orphans.OrphanedBranch{
		Repository: "owner/repo",
		BranchName: "feature",
	}

	expected := "owner/repo/feature"
	if got := orphan.Key(); got != expected {
		t.Errorf("Key() = %s, want %s", got, expected)
	}
}

func TestDefaultScanOptions(t *testing.T) {
	t.Parallel()

	opts := orphans.DefaultScanOptions()

	if opts.StaleDaysThreshold != 30 {
		t.Errorf("StaleDaysThreshold = %d, want 30", opts.StaleDaysThreshold)
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
		t.Errorf(
			"ExcludePatterns length = %d, want %d",
			len(opts.ExcludePatterns),
			len(expectedExcludes),
		)
	}
}

// MinAgeDays must spare a recently merged branch, which StaleDaysThreshold
// does not: that threshold only classifies branches with no PR at all.
func TestDetector_ClassifyBranch_MinAgeDays(t *testing.T) {
	t.Parallel()

	mergedAt := time.Now().Add(-12 * time.Hour)
	prs := []github.PullRequest{{
		Number:   1,
		Title:    "Feature PR",
		State:    "closed",
		Head:     github.PRRef{Ref: "feature-branch"},
		MergedAt: &mergedAt,
	}}

	repo := github.Repository{
		Name:          "test-repo",
		FullName:      "owner/test-repo",
		Owner:         "owner",
		DefaultBranch: "main",
	}

	tests := []struct {
		name       string
		ageDays    int
		minAgeDays int
		wantOrphan bool
	}{
		{name: "merged yesterday, guard at 30 days", ageDays: 1, minAgeDays: 30},
		{name: "merged 45 days ago, guard at 30 days", ageDays: 45, minAgeDays: 30, wantOrphan: true},
		{name: "merged yesterday, no guard", ageDays: 1, wantOrphan: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			opts := orphans.DefaultScanOptions()
			opts.MinAgeDays = tc.minAgeDays
			detector := orphans.NewDetector(opts)

			branch := github.Branch{
				Name:           "feature-branch",
				SHA:            "abc123",
				LastCommitDate: time.Now().AddDate(0, 0, -tc.ageDays),
			}

			orphan := detector.ClassifyBranch(repo, branch, prs)
			if got := orphan != nil; got != tc.wantOrphan {
				t.Fatalf("ClassifyBranch() orphan = %v, want %v", got, tc.wantOrphan)
			}
		})
	}
}
