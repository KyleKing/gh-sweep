package github

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// JobLog represents a GitHub Actions job log
type JobLog struct {
	JobID      int
	JobName    string
	WorkflowID int
	Repository string
	Conclusion string
	Lines      []string
	Timestamp  time.Time
}

// ErrorContext represents extracted error information
type ErrorContext struct {
	Repository   string    `json:"repository"`
	WorkflowName string    `json:"workflow_name"`
	JobName      string    `json:"job_name"`
	StepName     string    `json:"step_name,omitempty"`
	Conclusion   string    `json:"conclusion"`
	Timestamp    time.Time `json:"timestamp"`
	ErrorLines   []string  `json:"error_lines"`
	Context      []string  `json:"context_lines,omitempty"`
	ErrorType    string    `json:"error_type,omitempty"`
	Summary      string    `json:"summary"`
}

// LogExtractionConfig configures log extraction behavior
type LogExtractionConfig struct {
	TailLines       int      // Number of lines from end of log
	ContextLines    int      // Additional context lines around errors
	FilterNoise     bool     // Remove timestamps, ANSI codes
	ExtractStackTrace bool   // Include full stack traces
	IncludeSuccess  bool     // Include successful runs
	ErrorPatterns   []string // Custom regex patterns for errors
}

// DefaultLogConfig returns sensible defaults for log extraction
func DefaultLogConfig() LogExtractionConfig {
	return LogExtractionConfig{
		TailLines:       100,
		ContextLines:    5,
		FilterNoise:     true,
		ExtractStackTrace: false, // Usually too verbose
		IncludeSuccess:  false,
		ErrorPatterns: []string{
			`(?i)error:`,
			`(?i)failed:`,
			`(?i)fatal:`,
			`(?i)exception:`,
			`(?i)panic:`,
		},
	}
}

// ExtractErrorContext extracts actionable error information from job logs
// Pure function: deterministic, no side effects
func ExtractErrorContext(log JobLog, workflow string, config LogExtractionConfig) *ErrorContext {
	// Skip successful runs if not included
	if !config.IncludeSuccess && log.Conclusion == "success" {
		return nil
	}

	// Extract tail lines
	tailLines := extractTail(log.Lines, config.TailLines)

	// Filter noise if requested
	if config.FilterNoise {
		tailLines = filterNoise(tailLines)
	}

	// Identify error lines
	errorLines := identifyErrors(tailLines, config.ErrorPatterns)

	// Extract context around errors
	contextLines := extractContext(tailLines, errorLines, config.ContextLines)

	// Classify error type
	errorType := classifyError(errorLines)

	// Generate summary
	summary := generateSummary(log, errorType, len(errorLines))

	return &ErrorContext{
		Repository:   log.Repository,
		WorkflowName: workflow,
		JobName:      log.JobName,
		Conclusion:   log.Conclusion,
		Timestamp:    log.Timestamp,
		ErrorLines:   errorLines,
		Context:      contextLines,
		ErrorType:    errorType,
		Summary:      summary,
	}
}

// extractTail returns the last N lines from a log
// Pure function for composition
func extractTail(lines []string, n int) []string {
	if len(lines) <= n {
		return lines
	}
	return lines[len(lines)-n:]
}

// filterNoise removes common noise from log lines
// Pure function: transforms input without side effects
func filterNoise(lines []string) []string {
	filtered := make([]string, 0, len(lines))

	// Regex patterns for noise
	timestampPattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}`)
	ansiPattern := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	githubActionPattern := regexp.MustCompile(`^##\[.*?\]`)

	for _, line := range lines {
		// Remove ANSI codes
		clean := ansiPattern.ReplaceAllString(line, "")

		// Remove GitHub Actions annotations
		clean = githubActionPattern.ReplaceAllString(clean, "")

		// Remove timestamp prefix
		clean = timestampPattern.ReplaceAllString(clean, "")

		// Trim whitespace
		clean = strings.TrimSpace(clean)

		// Skip empty lines
		if clean == "" {
			continue
		}

		filtered = append(filtered, clean)
	}

	return filtered
}

// identifyErrors finds lines matching error patterns
// Pure function: returns indices of error lines
func identifyErrors(lines []string, patterns []string) []string {
	errors := make([]string, 0)

	// Compile patterns
	regexes := make([]*regexp.Regexp, len(patterns))
	for i, p := range patterns {
		regexes[i] = regexp.MustCompile(p)
	}

	for _, line := range lines {
		for _, re := range regexes {
			if re.MatchString(line) {
				errors = append(errors, line)
				break
			}
		}
	}

	return errors
}

// extractContext extracts context lines around errors
// Pure function for composition
func extractContext(allLines, errorLines []string, contextSize int) []string {
	if contextSize == 0 {
		return []string{}
	}

	// Build set of error lines for quick lookup
	errorSet := make(map[string]bool)
	for _, err := range errorLines {
		errorSet[err] = true
	}

	context := make([]string, 0)
	for i, line := range allLines {
		if errorSet[line] {
			// Extract context before and after
			start := max(0, i-contextSize)
			end := min(len(allLines), i+contextSize+1)

			for j := start; j < end; j++ {
				if !errorSet[allLines[j]] && !contains(context, allLines[j]) {
					context = append(context, allLines[j])
				}
			}
		}
	}

	return context
}

// classifyError determines error type from error lines
// Pure function for error classification
func classifyError(errorLines []string) string {
	if len(errorLines) == 0 {
		return "unknown"
	}

	// Join all errors for pattern matching
	allErrors := strings.ToLower(strings.Join(errorLines, " "))

	// Classification patterns (order matters - more specific first)
	// Check build errors first (more specific than panic)
	if containsAny(allErrors, "build failed", "compilation error", "syntax error") {
		return "build-error"
	}

	if containsAny(allErrors, "test failed", "assertion", "expected", "but got") {
		return "test-failure"
	}

	if containsAny(allErrors, "panic:", "segmentation fault") {
		return "panic"
	}

	if containsAny(allErrors, "timeout", "timed out", "deadline exceeded") {
		return "timeout"
	}

	if containsAny(allErrors, "dependency", "module not found", "modulenotfounderror", "import error", "cannot find") {
		return "dependency"
	}

	if containsAny(allErrors, "lint", "style", "formatting") {
		return "lint-error"
	}

	if containsAny(allErrors, "connection refused", "network", "dns", "unreachable") {
		return "network"
	}

	if containsAny(allErrors, "permission denied", "access denied", "forbidden") {
		return "permission"
	}

	if containsAny(allErrors, "out of memory", "oom", "memory exhausted") {
		return "out-of-memory"
	}

	return "generic-error"
}

// containsAny checks if the text contains any of the keywords
func containsAny(text string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

// generateSummary creates a human-readable summary
// Pure function for summary generation
func generateSummary(log JobLog, errorType string, errorCount int) string {
	if errorCount == 0 {
		return fmt.Sprintf("%s job '%s' failed without clear error messages",
			log.Repository, log.JobName)
	}

	return fmt.Sprintf("%s job '%s' failed with %d %s error(s)",
		log.Repository, log.JobName, errorCount, errorType)
}

// FormatAsJSON formats error context as JSON for AI consumption
// Pure function: serializes to JSON
func FormatAsJSON(contexts []*ErrorContext) (string, error) {
	data, err := json.MarshalIndent(contexts, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
}

// FormatAsMarkdown formats error context as Markdown for AI consumption
// Pure function: generates Markdown string
func FormatAsMarkdown(contexts []*ErrorContext) string {
	var sb strings.Builder

	sb.WriteString("# GitHub Actions Error Report\n\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC3339)))

	for i, ctx := range contexts {
		sb.WriteString(fmt.Sprintf("## %d. %s\n\n", i+1, ctx.Summary))

		sb.WriteString("**Details:**\n")
		sb.WriteString(fmt.Sprintf("- Repository: `%s`\n", ctx.Repository))
		sb.WriteString(fmt.Sprintf("- Workflow: `%s`\n", ctx.WorkflowName))
		sb.WriteString(fmt.Sprintf("- Job: `%s`\n", ctx.JobName))
		if ctx.StepName != "" {
			sb.WriteString(fmt.Sprintf("- Step: `%s`\n", ctx.StepName))
		}
		sb.WriteString(fmt.Sprintf("- Error Type: `%s`\n", ctx.ErrorType))
		sb.WriteString(fmt.Sprintf("- Timestamp: `%s`\n", ctx.Timestamp.Format(time.RFC3339)))
		sb.WriteString("\n")

		if len(ctx.ErrorLines) > 0 {
			sb.WriteString("**Error Messages:**\n```\n")
			for _, line := range ctx.ErrorLines {
				sb.WriteString(line + "\n")
			}
			sb.WriteString("```\n\n")
		}

		if len(ctx.Context) > 0 {
			sb.WriteString("**Context:**\n```\n")
			for _, line := range ctx.Context {
				sb.WriteString(line + "\n")
			}
			sb.WriteString("```\n\n")
		}

		sb.WriteString("---\n\n")
	}

	return sb.String()
}

// BatchExtractErrors extracts errors from multiple logs
// Pure function: maps over logs
func BatchExtractErrors(logs []JobLog, workflow string, config LogExtractionConfig) []*ErrorContext {
	contexts := make([]*ErrorContext, 0, len(logs))

	for _, log := range logs {
		if ctx := ExtractErrorContext(log, workflow, config); ctx != nil {
			contexts = append(contexts, ctx)
		}
	}

	return contexts
}

// Helper functions

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
