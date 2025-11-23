package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/KyleKing/gh-sweep/internal/github"
)

// ExportFormat represents the export format
type ExportFormat string

const (
	FormatCSV  ExportFormat = "csv"
	FormatJSON ExportFormat = "json"
)

// ExportWorkflowStats exports workflow statistics to a file
func ExportWorkflowStats(stats *github.WorkflowRunStats, format ExportFormat, outputPath string) error {
	switch format {
	case FormatCSV:
		return exportStatsCSV(stats, outputPath)
	case FormatJSON:
		return exportStatsJSON(stats, outputPath)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func exportStatsCSV(stats *github.WorkflowRunStats, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Header
	writer.Write([]string{"Metric", "Value"})

	// Data
	writer.Write([]string{"Total Runs", fmt.Sprintf("%d", stats.TotalRuns)})
	writer.Write([]string{"Success Rate", fmt.Sprintf("%.2f%%", stats.SuccessRate)})
	writer.Write([]string{"Failure Count", fmt.Sprintf("%d", stats.FailureCount)})
	writer.Write([]string{"Avg Duration", stats.AvgDuration.String()})

	return nil
}

func exportStatsJSON(stats *github.WorkflowRunStats, outputPath string) error {
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// ExportComments exports comments to a file
func ExportComments(comments []github.Comment, format ExportFormat, outputPath string) error {
	switch format {
	case FormatCSV:
		return exportCommentsCSV(comments, outputPath)
	case FormatJSON:
		return exportCommentsJSON(comments, outputPath)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func exportCommentsCSV(comments []github.Comment, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Header
	writer.Write([]string{"Repository", "PR", "Author", "Path", "Line", "Body", "Created"})

	// Data
	for _, c := range comments {
		writer.Write([]string{
			c.Repository,
			fmt.Sprintf("%d", c.PRNumber),
			c.Author,
			c.Path,
			fmt.Sprintf("%d", c.Line),
			c.Body,
			c.CreatedAt.Format(time.RFC3339),
		})
	}

	return nil
}

func exportCommentsJSON(comments []github.Comment, outputPath string) error {
	data, err := json.MarshalIndent(comments, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// ExportProtectionRules exports protection rules to a file
func ExportProtectionRules(rules []*github.ProtectionRule, format ExportFormat, outputPath string) error {
	switch format {
	case FormatCSV:
		return exportProtectionCSV(rules, outputPath)
	case FormatJSON:
		return exportProtectionJSON(rules, outputPath)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func exportProtectionCSV(rules []*github.ProtectionRule, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Header
	writer.Write([]string{"Repository", "Branch", "Required Reviews", "Code Owner Reviews", "Enforce Admins"})

	// Data
	for _, rule := range rules {
		writer.Write([]string{
			rule.Repository,
			rule.Branch,
			fmt.Sprintf("%d", rule.RequiredReviews),
			fmt.Sprintf("%v", rule.RequireCodeOwnerReviews),
			fmt.Sprintf("%v", rule.EnforceAdmins),
		})
	}

	return nil
}

func exportProtectionJSON(rules []*github.ProtectionRule, outputPath string) error {
	data, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
