package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rahul/nsha/pkg/git"
	"github.com/rahul/nsha/pkg/logger"
)

// ReportData contains all data for generating a report
type ReportData struct {
	RepoPath      string
	StartTime     time.Time
	EndTime       time.Time
	InitialIssues []git.Issue
	FinalIssues   []git.Issue
	Operations    []logger.Operation
	BackupPath    string
	Success       bool
	ErrorMessage  string
}

// GenerateReport creates comprehensive reports
func GenerateReport(data *ReportData, logDir string) error {
	// Generate comprehensive report (combines summary and detailed analysis)
	comprehensiveReport := generateComprehensiveReport(data)
	reportPath := filepath.Join(logDir, "report.txt")
	err := os.WriteFile(reportPath, []byte(comprehensiveReport), 0644)
	if err != nil {
		return fmt.Errorf("failed to write comprehensive report: %w", err)
	}

	// Generate changes summary
	changesReport := generateChangesReport(data)
	changesPath := filepath.Join(logDir, "changes-summary.txt")
	err = os.WriteFile(changesPath, []byte(changesReport), 0644)
	if err != nil {
		return fmt.Errorf("failed to write changes report: %w", err)
	}

	return nil
}

// generateComprehensiveReport creates a comprehensive report combining summary and detailed analysis
func generateComprehensiveReport(data *ReportData) string {
	var sb strings.Builder

	sb.WriteString("╔═══════════════════════════════════════════════════════════╗\n")
	sb.WriteString("║              NSHA - Execution Report                      ║\n")
	sb.WriteString("╚═══════════════════════════════════════════════════════════╝\n\n")

	// Summary
	sb.WriteString("SUMMARY\n")
	sb.WriteString("═══════════════════════════════════════════════════════════\n")
	sb.WriteString(fmt.Sprintf("Repository: %s\n", data.RepoPath))
	sb.WriteString(fmt.Sprintf("Start Time: %s\n", data.StartTime.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("End Time:   %s\n", data.EndTime.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("Duration:   %s\n", data.EndTime.Sub(data.StartTime)))
	sb.WriteString(fmt.Sprintf("Status:     %s\n", getStatusString(data.Success)))
	if data.ErrorMessage != "" {
		sb.WriteString(fmt.Sprintf("Error:      %s\n", data.ErrorMessage))
	}
	sb.WriteString("\n")

	// Backup Information
	sb.WriteString("BACKUP INFORMATION\n")
	sb.WriteString("═══════════════════════════════════════════════════════════\n")
	if data.BackupPath != "" {
		sb.WriteString(fmt.Sprintf("Backup Location: %s\n", data.BackupPath))
		sb.WriteString("Status: ✅ Backup created successfully\n")
		sb.WriteString("\nTo restore from backup, see backup-info.txt in the backup directory.\n")
	} else {
		sb.WriteString("Status: ⚠️  No backup created\n")
	}
	sb.WriteString("\n")

	// Issues Found
	sb.WriteString("ISSUES ANALYSIS\n")
	sb.WriteString("═══════════════════════════════════════════════════════════\n")
	sb.WriteString(fmt.Sprintf("Initial Issues Found: %d\n", len(data.InitialIssues)))
	sb.WriteString(fmt.Sprintf("Final Issues Remaining: %d\n", len(data.FinalIssues)))
	sb.WriteString(fmt.Sprintf("Issues Fixed: %d\n", len(data.InitialIssues)-len(data.FinalIssues)))

	if len(data.InitialIssues) > 0 {
		successRate := float64(len(data.InitialIssues)-len(data.FinalIssues)) / float64(len(data.InitialIssues)) * 100
		sb.WriteString(fmt.Sprintf("Success Rate: %.1f%%\n", successRate))
	}
	sb.WriteString("\n")

	// Initial Issues Detail
	if len(data.InitialIssues) > 0 {
		sb.WriteString("INITIAL ISSUES DETECTED\n")
		sb.WriteString("═══════════════════════════════════════════════════════════\n")
		issueTypes := categorizeIssues(data.InitialIssues)
		for issueType, issues := range issueTypes {
			sb.WriteString(fmt.Sprintf("\n%s (%d issues):\n", issueType, len(issues)))
			for i, issue := range issues {
				sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, issue.String()))
			}
		}
		sb.WriteString("\n")
	}

	// Remaining Issues
	if len(data.FinalIssues) > 0 {
		sb.WriteString("REMAINING ISSUES\n")
		sb.WriteString("═══════════════════════════════════════════════════════════\n")
		for i, issue := range data.FinalIssues {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, issue.String()))
		}
		sb.WriteString("\n")
	}

	// Operations Summary
	sb.WriteString("OPERATIONS PERFORMED\n")
	sb.WriteString("═══════════════════════════════════════════════════════════\n")
	sb.WriteString(fmt.Sprintf("Total Operations: %d\n", len(data.Operations)))

	successCount := 0
	errorCount := 0
	for _, op := range data.Operations {
		if op.Success {
			successCount++
		} else {
			errorCount++
		}
	}
	sb.WriteString(fmt.Sprintf("Successful: %d\n", successCount))
	sb.WriteString(fmt.Sprintf("Errors: %d\n", errorCount))
	sb.WriteString("\n")

	// Add detailed analysis section
	sb.WriteString("\n")
	sb.WriteString("╔═══════════════════════════════════════════════════════════╗\n")
	sb.WriteString("║              DETAILED OPERATION ANALYSIS                  ║\n")
	sb.WriteString("╚═══════════════════════════════════════════════════════════╝\n\n")

	sb.WriteString("This section contains detailed information about every operation\n")
	sb.WriteString("performed by NSHA, including all commit SHAs, changes made, and\n")
	sb.WriteString("before/after states.\n\n")

	// Group operations by step
	stepOps := make(map[string][]logger.Operation)
	for _, op := range data.Operations {
		stepOps[op.Step] = append(stepOps[op.Step], op)
	}

	// Write operations by step
	for step, ops := range stepOps {
		sb.WriteString(fmt.Sprintf("\n%s\n", strings.Repeat("═", 63)))
		sb.WriteString(fmt.Sprintf("STEP: %s\n", step))
		sb.WriteString(fmt.Sprintf("%s\n\n", strings.Repeat("═", 63)))

		for _, op := range ops {
			sb.WriteString(fmt.Sprintf("[%s] %s\n", op.Timestamp.Format("15:04:05"), op.Action))

			if op.Details != "" {
				sb.WriteString(fmt.Sprintf("  Details: %s\n", op.Details))
			}

			if op.CommitSHA != "" {
				sb.WriteString(fmt.Sprintf("  Commit SHA: %s\n", op.CommitSHA))
			}

			if op.OldValue != "" {
				sb.WriteString(fmt.Sprintf("  Previous State: %s\n", op.OldValue))
			}

			if op.NewValue != "" {
				sb.WriteString(fmt.Sprintf("  New State: %s\n", op.NewValue))
			}

			if !op.Success && op.Error != "" {
				sb.WriteString(fmt.Sprintf("  ❌ ERROR: %s\n", op.Error))
			} else if op.Success {
				sb.WriteString("  ✅ Success\n")
			}

			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// generateChangesReport creates a summary of all changes made
func generateChangesReport(data *ReportData) string {
	var sb strings.Builder

	sb.WriteString("╔═══════════════════════════════════════════════════════════╗\n")
	sb.WriteString("║           NSHA - Changes Summary Report                  ║\n")
	sb.WriteString("╚═══════════════════════════════════════════════════════════╝\n\n")

	// Collect all changes
	var changes []logger.Operation
	for _, op := range data.Operations {
		if op.OldValue != "" && op.NewValue != "" {
			changes = append(changes, op)
		}
	}

	sb.WriteString(fmt.Sprintf("Total Changes Made: %d\n\n", len(changes)))

	if len(changes) == 0 {
		sb.WriteString("No changes were made to the repository.\n")
		return sb.String()
	}

	sb.WriteString("DETAILED CHANGES\n")
	sb.WriteString("═══════════════════════════════════════════════════════════\n\n")

	for i, change := range changes {
		sb.WriteString(fmt.Sprintf("Change #%d: %s\n", i+1, change.Action))
		sb.WriteString(fmt.Sprintf("  Time: %s\n", change.Timestamp.Format("2006-01-02 15:04:05")))

		if change.CommitSHA != "" {
			sb.WriteString(fmt.Sprintf("  Commit: %s\n", change.CommitSHA))
		}

		sb.WriteString(fmt.Sprintf("  Before: %s\n", change.OldValue))
		sb.WriteString(fmt.Sprintf("  After:  %s\n", change.NewValue))

		if change.Details != "" {
			sb.WriteString(fmt.Sprintf("  Details: %s\n", change.Details))
		}

		sb.WriteString("\n")
	}

	// Summary by type
	sb.WriteString("\nSUMMARY BY CHANGE TYPE\n")
	sb.WriteString("═══════════════════════════════════════════════════════════\n")

	changeTypes := make(map[string]int)
	for _, change := range changes {
		changeTypes[change.Action]++
	}

	for changeType, count := range changeTypes {
		sb.WriteString(fmt.Sprintf("  %s: %d\n", changeType, count))
	}

	return sb.String()
}

// Helper functions

func getStatusString(success bool) string {
	if success {
		return "✅ SUCCESS"
	}
	return "❌ FAILED"
}

func categorizeIssues(issues []git.Issue) map[string][]git.Issue {
	categories := make(map[string][]git.Issue)

	for _, issue := range issues {
		category := string(issue.Type)
		categories[category] = append(categories[category], issue)
	}

	return categories
}
