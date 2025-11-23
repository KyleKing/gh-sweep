package github

import (
	"sort"
	"time"
)

// FlakyTest represents a test that exhibits flaky behavior
type FlakyTest struct {
	Name         string
	FailureRate  float64
	FirstFailure time.Time
	LastFlip     time.Time
	FlipCount    int
	TotalRuns    int
	FailureCount int
	Pattern      string // "same-commit-flip", "intermittent", "consistent"
}

// TestRun represents a single test execution
type TestRun struct {
	Name       string
	Status     string // "success", "failure", "skipped"
	CommitSHA  string
	Timestamp  time.Time
	Duration   time.Duration
	Repository string
	WorkflowID int
}

// FlakyDetectionConfig configures flaky test detection
type FlakyDetectionConfig struct {
	MinFlips         int     // Minimum flips to be considered flaky
	MinFailureRate   float64 // Minimum failure rate (0.0-1.0)
	TimeWindow       time.Duration
	SameCommitOnly   bool // Only detect same-commit flips
	IncludeSkipped   bool // Include skipped tests in analysis
}

// DefaultFlakyConfig returns sensible defaults
func DefaultFlakyConfig() FlakyDetectionConfig {
	return FlakyDetectionConfig{
		MinFlips:       2,
		MinFailureRate: 0.1, // 10%
		TimeWindow:     7 * 24 * time.Hour, // 7 days
		SameCommitOnly: false,
		IncludeSkipped: false,
	}
}

// DetectFlakyTests identifies flaky tests from test runs
// Pure function: no side effects, deterministic output
func DetectFlakyTests(runs []TestRun, config FlakyDetectionConfig) []FlakyTest {
	// Group runs by test name
	grouped := groupByTestName(runs)

	// Analyze each test for flaky behavior
	flaky := make([]FlakyTest, 0)
	for name, testRuns := range grouped {
		if test := analyzeFlakyPattern(name, testRuns, config); test != nil {
			flaky = append(flaky, *test)
		}
	}

	// Sort by failure rate (descending)
	sort.Slice(flaky, func(i, j int) bool {
		return flaky[i].FailureRate > flaky[j].FailureRate
	})

	return flaky
}

// groupByTestName groups test runs by test name
// Pure function for composition
func groupByTestName(runs []TestRun) map[string][]TestRun {
	grouped := make(map[string][]TestRun)
	for _, run := range runs {
		grouped[run.Name] = append(grouped[run.Name], run)
	}

	// Sort each group by timestamp
	for name := range grouped {
		sort.Slice(grouped[name], func(i, j int) bool {
			return grouped[name][i].Timestamp.Before(grouped[name][j].Timestamp)
		})
	}

	return grouped
}

// analyzeFlakyPattern analyzes a single test for flaky behavior
// Pure function: returns nil if not flaky
func analyzeFlakyPattern(name string, runs []TestRun, config FlakyDetectionConfig) *FlakyTest {
	if len(runs) < 2 {
		return nil // Need at least 2 runs to detect flakiness
	}

	// Filter by time window
	cutoff := time.Now().Add(-config.TimeWindow)
	filtered := filterByTime(runs, cutoff)
	if len(filtered) < 2 {
		return nil
	}

	// Calculate statistics
	stats := calculateTestStats(filtered, config.IncludeSkipped)

	// Detect flips (status changes)
	flips := detectFlips(filtered, config.SameCommitOnly)

	// Check if meets flaky criteria
	// Same-commit flips are always considered flaky (strong signal)
	if flips.sameCommitFlips > 0 {
		// Pass through - same commit flips are always flaky
	} else if flips.count < config.MinFlips || stats.failureRate < config.MinFailureRate {
		return nil
	}

	// Determine pattern
	pattern := classifyPattern(stats, flips, config)

	return &FlakyTest{
		Name:         name,
		FailureRate:  stats.failureRate,
		FirstFailure: stats.firstFailure,
		LastFlip:     flips.lastFlip,
		FlipCount:    flips.count,
		TotalRuns:    stats.totalRuns,
		FailureCount: stats.failureCount,
		Pattern:      pattern,
	}
}

// testStats holds calculated statistics
type testStats struct {
	totalRuns    int
	failureCount int
	failureRate  float64
	firstFailure time.Time
}

// calculateTestStats computes statistics for a test
// Pure function for composition
func calculateTestStats(runs []TestRun, includeSkipped bool) testStats {
	stats := testStats{totalRuns: len(runs)}

	for i, run := range runs {
		if run.Status == "failure" {
			stats.failureCount++
			if stats.firstFailure.IsZero() || run.Timestamp.Before(stats.firstFailure) {
				stats.firstFailure = run.Timestamp
			}
		}

		// Optionally exclude skipped from count
		if !includeSkipped && run.Status == "skipped" {
			stats.totalRuns--
		}

		// Keep timestamp of first occurrence
		if i == 0 && run.Status == "failure" {
			stats.firstFailure = run.Timestamp
		}
	}

	if stats.totalRuns > 0 {
		stats.failureRate = float64(stats.failureCount) / float64(stats.totalRuns)
	}

	return stats
}

// flipDetection holds flip analysis results
type flipDetection struct {
	count    int
	lastFlip time.Time
	sameCommitFlips int
}

// detectFlips identifies status changes between runs
// Pure function: no side effects
func detectFlips(runs []TestRun, sameCommitOnly bool) flipDetection {
	detection := flipDetection{}

	for i := 1; i < len(runs); i++ {
		prev, curr := runs[i-1], runs[i]

		// Skip if both are skipped
		if prev.Status == "skipped" && curr.Status == "skipped" {
			continue
		}

		// Detect flip
		if prev.Status != curr.Status {
			// Skip skipped transitions unless they're meaningful
			if prev.Status == "skipped" || curr.Status == "skipped" {
				continue
			}

			detection.count++
			detection.lastFlip = curr.Timestamp

			// Track same-commit flips (strong signal of flakiness)
			if prev.CommitSHA == curr.CommitSHA {
				detection.sameCommitFlips++
			}
		}
	}

	// If same-commit-only mode, only count those
	if sameCommitOnly {
		detection.count = detection.sameCommitFlips
	}

	return detection
}

// classifyPattern determines the flaky pattern type
// Pure function for pattern classification
func classifyPattern(stats testStats, flips flipDetection, config FlakyDetectionConfig) string {
	// Same-commit flip = strongest signal
	if flips.sameCommitFlips > 0 {
		return "same-commit-flip"
	}

	// High failure rate but inconsistent = intermittent
	if stats.failureRate > 0.3 && stats.failureRate < 0.7 {
		return "intermittent"
	}

	// Low failure rate = occasional
	if stats.failureRate < 0.3 {
		return "occasional"
	}

	// High failure rate = consistent failure (not really "flaky")
	return "consistent"
}

// filterByTime filters test runs within a time window
// Pure function for composition
func filterByTime(runs []TestRun, cutoff time.Time) []TestRun {
	filtered := make([]TestRun, 0, len(runs))
	for _, run := range runs {
		if run.Timestamp.After(cutoff) {
			filtered = append(filtered, run)
		}
	}
	return filtered
}

// FilterByRepository creates a filter for specific repositories
// Higher-order function returning a filter predicate
func FilterByRepository(repos ...string) func(TestRun) bool {
	repoSet := make(map[string]bool)
	for _, r := range repos {
		repoSet[r] = true
	}

	return func(run TestRun) bool {
		return repoSet[run.Repository]
	}
}

// FilterByCommit creates a filter for specific commits
// Higher-order function for functional composition
func FilterByCommit(commits ...string) func(TestRun) bool {
	commitSet := make(map[string]bool)
	for _, c := range commits {
		commitSet[c] = true
	}

	return func(run TestRun) bool {
		return commitSet[run.CommitSHA]
	}
}

// ApplyFilters applies a list of filters to test runs
// Functional composition helper
func ApplyFilters(runs []TestRun, filters ...func(TestRun) bool) []TestRun {
	filtered := make([]TestRun, 0, len(runs))

	for _, run := range runs {
		include := true
		for _, filter := range filters {
			if !filter(run) {
				include = false
				break
			}
		}
		if include {
			filtered = append(filtered, run)
		}
	}

	return filtered
}
