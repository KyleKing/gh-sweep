package github

import (
	"strings"
	"testing"
	"time"
)

// TestAnalyzeWorkflowRuns tests workflow run statistics
func TestAnalyzeWorkflowRuns(t *testing.T) {
	tests := []struct {
		name            string
		runs            []WorkflowRun
		expectedSuccess float64
		expectedFailures int
	}{
		{
			name: "all successful",
			runs: []WorkflowRun{
				{Conclusion: "success", Duration: 1 * time.Minute},
				{Conclusion: "success", Duration: 2 * time.Minute},
			},
			expectedSuccess:  100.0,
			expectedFailures: 0,
		},
		{
			name: "mixed results",
			runs: []WorkflowRun{
				{Conclusion: "success", Duration: 1 * time.Minute},
				{Conclusion: "failure", Duration: 2 * time.Minute},
				{Conclusion: "success", Duration: 1 * time.Minute},
				{Conclusion: "failure", Duration: 3 * time.Minute},
			},
			expectedSuccess:  50.0,
			expectedFailures: 2,
		},
		{
			name:            "empty runs",
			runs:            []WorkflowRun{},
			expectedSuccess:  0.0,
			expectedFailures: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := AnalyzeWorkflowRuns(tt.runs)

			if stats.SuccessRate != tt.expectedSuccess {
				t.Errorf("Expected success rate %.2f%%, got %.2f%%",
					tt.expectedSuccess, stats.SuccessRate)
			}

			if stats.FailureCount != tt.expectedFailures {
				t.Errorf("Expected %d failures, got %d",
					tt.expectedFailures, stats.FailureCount)
			}

			if stats.TotalRuns != len(tt.runs) {
				t.Errorf("Expected %d total runs, got %d",
					len(tt.runs), stats.TotalRuns)
			}
		})
	}
}

// TestDetectFlakyTests tests flaky test detection
func TestDetectFlakyTests(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		runs          []TestRun
		config        FlakyDetectionConfig
		expectFlaky   bool
		expectedPattern string
	}{
		{
			name: "same commit flip",
			runs: []TestRun{
				{
					Name:      "TestFoo",
					Status:    "failure",
					CommitSHA: "abc123",
					Timestamp: now.Add(-1 * time.Hour),
				},
				{
					Name:      "TestFoo",
					Status:    "success",
					CommitSHA: "abc123",
					Timestamp: now,
				},
			},
			config: FlakyDetectionConfig{
				MinFlips:       1, // Same-commit flips are strong signal, only need 1
				MinFailureRate: 0.1,
				TimeWindow:     7 * 24 * time.Hour,
			},
			expectFlaky:   true,
			expectedPattern: "same-commit-flip",
		},
		{
			name: "intermittent failures",
			runs: []TestRun{
				{Name: "TestBar", Status: "success", CommitSHA: "a", Timestamp: now.Add(-5 * time.Hour)},
				{Name: "TestBar", Status: "failure", CommitSHA: "b", Timestamp: now.Add(-4 * time.Hour)},
				{Name: "TestBar", Status: "success", CommitSHA: "c", Timestamp: now.Add(-3 * time.Hour)},
				{Name: "TestBar", Status: "failure", CommitSHA: "d", Timestamp: now.Add(-2 * time.Hour)},
				{Name: "TestBar", Status: "success", CommitSHA: "e", Timestamp: now.Add(-1 * time.Hour)},
			},
			config:        DefaultFlakyConfig(),
			expectFlaky:   true,
			expectedPattern: "intermittent",
		},
		{
			name: "consistent failure",
			runs: []TestRun{
				{Name: "TestBaz", Status: "failure", CommitSHA: "a", Timestamp: now.Add(-3 * time.Hour)},
				{Name: "TestBaz", Status: "failure", CommitSHA: "b", Timestamp: now.Add(-2 * time.Hour)},
				{Name: "TestBaz", Status: "failure", CommitSHA: "c", Timestamp: now.Add(-1 * time.Hour)},
			},
			config:        DefaultFlakyConfig(),
			expectFlaky:   false, // Too consistent, not flaky
		},
		{
			name: "not enough flips",
			runs: []TestRun{
				{Name: "TestQux", Status: "success", CommitSHA: "a", Timestamp: now.Add(-2 * time.Hour)},
				{Name: "TestQux", Status: "failure", CommitSHA: "b", Timestamp: now.Add(-1 * time.Hour)},
			},
			config: FlakyDetectionConfig{
				MinFlips:       2,
				MinFailureRate: 0.1,
				TimeWindow:     7 * 24 * time.Hour,
			},
			expectFlaky: false, // Only 1 flip, needs 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectFlakyTests(tt.runs, tt.config)

			if tt.expectFlaky {
				if len(result) == 0 {
					t.Fatal("Expected flaky test to be detected, got none")
				}

				flaky := result[0]
				if flaky.Pattern != tt.expectedPattern {
					t.Errorf("Expected pattern '%s', got '%s'",
						tt.expectedPattern, flaky.Pattern)
				}

				// For same-commit-flip, we only need 1 flip (strong signal)
				minExpected := tt.config.MinFlips
				if tt.expectedPattern == "same-commit-flip" {
					minExpected = 1
				}

				if flaky.FlipCount < minExpected {
					t.Errorf("Expected at least %d flips, got %d",
						minExpected, flaky.FlipCount)
				}
			} else {
				if len(result) > 0 {
					t.Errorf("Expected no flaky tests, got %d", len(result))
				}
			}
		})
	}
}

// TestGroupByTestName tests grouping functionality
func TestGroupByTestName(t *testing.T) {
	now := time.Now()

	runs := []TestRun{
		{Name: "TestA", Timestamp: now.Add(-2 * time.Hour)},
		{Name: "TestB", Timestamp: now.Add(-1 * time.Hour)},
		{Name: "TestA", Timestamp: now},
		{Name: "TestA", Timestamp: now.Add(-3 * time.Hour)},
	}

	grouped := groupByTestName(runs)

	if len(grouped) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(grouped))
	}

	if len(grouped["TestA"]) != 3 {
		t.Errorf("Expected 3 runs for TestA, got %d", len(grouped["TestA"]))
	}

	if len(grouped["TestB"]) != 1 {
		t.Errorf("Expected 1 run for TestB, got %d", len(grouped["TestB"]))
	}

	// Verify sorted by timestamp
	testA := grouped["TestA"]
	if !testA[0].Timestamp.Before(testA[1].Timestamp) {
		t.Error("Expected runs to be sorted by timestamp")
	}
}

// TestFilterByTime tests time-based filtering
func TestFilterByTime(t *testing.T) {
	now := time.Now()
	cutoff := now.Add(-2 * time.Hour)

	runs := []TestRun{
		{Name: "Test1", Timestamp: now.Add(-3 * time.Hour)}, // Before cutoff
		{Name: "Test2", Timestamp: now.Add(-1 * time.Hour)}, // After cutoff
		{Name: "Test3", Timestamp: now},                     // After cutoff
	}

	filtered := filterByTime(runs, cutoff)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 runs after cutoff, got %d", len(filtered))
	}
}

// TestApplyFilters tests filter composition
func TestApplyFilters(t *testing.T) {
	runs := []TestRun{
		{Name: "Test1", Repository: "repo-a", CommitSHA: "abc"},
		{Name: "Test2", Repository: "repo-b", CommitSHA: "def"},
		{Name: "Test3", Repository: "repo-a", CommitSHA: "ghi"},
	}

	// Filter by repository
	repoFilter := FilterByRepository("repo-a")
	filtered := ApplyFilters(runs, repoFilter)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 runs for repo-a, got %d", len(filtered))
	}

	// Combine filters
	commitFilter := FilterByCommit("abc")
	filtered = ApplyFilters(runs, repoFilter, commitFilter)

	if len(filtered) != 1 {
		t.Errorf("Expected 1 run matching both filters, got %d", len(filtered))
	}
}

// TestExtractErrorContext tests error extraction
func TestExtractErrorContext(t *testing.T) {
	config := DefaultLogConfig()

	tests := []struct {
		name           string
		log            JobLog
		expectNil      bool
		expectedErrors int
		expectedType   string
	}{
		{
			name: "test failure",
			log: JobLog{
				JobName:    "test",
				Conclusion: "failure",
				Lines: []string{
					"Running tests...",
					"ERROR: Test failed: assertion failed",
					"Expected: 5",
					"But got: 3",
					"Test suite failed",
				},
				Timestamp: time.Now(),
			},
			expectNil:      false,
			expectedErrors: 1, // One line with ERROR
			expectedType:   "test-failure",
		},
		{
			name: "build error",
			log: JobLog{
				JobName:    "build",
				Conclusion: "failure",
				Lines: []string{
					"Building project...",
					"Error: Build failed",
					"fatal error: syntax error on line 42",
					"Compilation terminated",
				},
				Timestamp: time.Now(),
			},
			expectNil:      false,
			expectedErrors: 2, // "Error:" and "fatal error:"
			expectedType:   "build-error",
		},
		{
			name: "successful run",
			log: JobLog{
				JobName:    "test",
				Conclusion: "success",
				Lines: []string{
					"All tests passed!",
				},
				Timestamp: time.Now(),
			},
			expectNil: true, // Success runs skipped by default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := ExtractErrorContext(tt.log, "test-workflow", config)

			if tt.expectNil {
				if ctx != nil {
					t.Error("Expected nil context for successful run")
				}
				return
			}

			if ctx == nil {
				t.Fatal("Expected error context, got nil")
			}

			if len(ctx.ErrorLines) != tt.expectedErrors {
				t.Errorf("Expected %d error lines, got %d: %v",
					tt.expectedErrors, len(ctx.ErrorLines), ctx.ErrorLines)
			}

			if ctx.ErrorType != tt.expectedType {
				t.Errorf("Expected error type '%s', got '%s'",
					tt.expectedType, ctx.ErrorType)
			}

			if ctx.Summary == "" {
				t.Error("Expected non-empty summary")
			}
		})
	}
}

// TestFilterNoise tests log noise filtering
func TestFilterNoise(t *testing.T) {
	lines := []string{
		"2024-01-15T10:30:00 Starting test...",
		"\x1b[32mSUCCESS\x1b[0m",
		"##[group]Running tests",
		"Actual test output",
		"",
		"   ",
	}

	filtered := filterNoise(lines)

	// Should remove timestamp, ANSI codes, GitHub annotations prefixes, empty lines
	// Result: "Starting test...", "SUCCESS", "Running tests", "Actual test output"
	if len(filtered) != 4 {
		t.Errorf("Expected 4 clean lines, got %d: %v", len(filtered), filtered)
	}

	// Check no ANSI codes remain
	for _, line := range filtered {
		if strings.Contains(line, "\x1b") {
			t.Errorf("ANSI codes not removed: %s", line)
		}
	}

	// Check specific cleaned results
	expected := []string{"Starting test...", "SUCCESS", "Running tests", "Actual test output"}
	for i, exp := range expected {
		if filtered[i] != exp {
			t.Errorf("Line %d: expected '%s', got '%s'", i, exp, filtered[i])
		}
	}
}

// TestClassifyError tests error type classification
func TestClassifyError(t *testing.T) {
	tests := []struct {
		name         string
		errorLines   []string
		expectedType string
	}{
		{
			name:         "test failure",
			errorLines:   []string{"assertion failed: expected 5 but got 3"},
			expectedType: "test-failure",
		},
		{
			name:         "build error",
			errorLines:   []string{"compilation error: syntax error on line 42"},
			expectedType: "build-error",
		},
		{
			name:         "timeout",
			errorLines:   []string{"Error: operation timed out"},
			expectedType: "timeout",
		},
		{
			name:         "dependency error",
			errorLines:   []string{"ModuleNotFoundError: No module named 'foo'"},
			expectedType: "dependency",
		},
		{
			name:         "panic",
			errorLines:   []string{"panic: runtime error: index out of range"},
			expectedType: "panic",
		},
		{
			name:         "unknown",
			errorLines:   []string{},
			expectedType: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifyError(tt.errorLines)

			if result != tt.expectedType {
				t.Errorf("Expected error type '%s', got '%s'",
					tt.expectedType, result)
			}
		})
	}
}

// TestFormatAsJSON tests JSON formatting
func TestFormatAsJSON(t *testing.T) {
	contexts := []*ErrorContext{
		{
			Repository:   "owner/repo",
			WorkflowName: "CI",
			JobName:      "test",
			ErrorType:    "test-failure",
			Summary:      "Test failed",
			ErrorLines:   []string{"ERROR: assertion failed"},
			Timestamp:    time.Now(),
		},
	}

	json, err := FormatAsJSON(contexts)
	if err != nil {
		t.Fatalf("Failed to format JSON: %v", err)
	}

	if !strings.Contains(json, "owner/repo") {
		t.Error("JSON should contain repository name")
	}

	if !strings.Contains(json, "test-failure") {
		t.Error("JSON should contain error type")
	}
}

// TestFormatAsMarkdown tests Markdown formatting
func TestFormatAsMarkdown(t *testing.T) {
	contexts := []*ErrorContext{
		{
			Repository:   "owner/repo",
			WorkflowName: "CI",
			JobName:      "test",
			ErrorType:    "test-failure",
			Summary:      "Test failed",
			ErrorLines:   []string{"ERROR: assertion failed"},
			Context:      []string{"Running test suite..."},
			Timestamp:    time.Now(),
		},
	}

	md := FormatAsMarkdown(contexts)

	if !strings.Contains(md, "# GitHub Actions Error Report") {
		t.Error("Markdown should contain title")
	}

	if !strings.Contains(md, "owner/repo") {
		t.Error("Markdown should contain repository")
	}

	if !strings.Contains(md, "```") {
		t.Error("Markdown should contain code blocks")
	}

	if !strings.Contains(md, "ERROR: assertion failed") {
		t.Error("Markdown should contain error lines")
	}
}

// TestBatchExtractErrors tests batch extraction
func TestBatchExtractErrors(t *testing.T) {
	logs := []JobLog{
		{
			JobName:    "test1",
			Conclusion: "failure",
			Lines:      []string{"ERROR: test failed"},
			Timestamp:  time.Now(),
		},
		{
			JobName:    "test2",
			Conclusion: "success",
			Lines:      []string{"All good"},
			Timestamp:  time.Now(),
		},
		{
			JobName:    "test3",
			Conclusion: "failure",
			Lines:      []string{"FATAL: panic occurred"},
			Timestamp:  time.Now(),
		},
	}

	config := DefaultLogConfig()
	contexts := BatchExtractErrors(logs, "CI", config)

	// Should only extract from failed jobs (2 failures, success skipped)
	if len(contexts) != 2 {
		t.Errorf("Expected 2 error contexts, got %d", len(contexts))
	}
}
