package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Logger handles detailed logging of all operations
type Logger struct {
	logFile    *os.File
	logDir     string
	startTime  time.Time
	operations []Operation
}

// Operation represents a single operation performed by the tool
type Operation struct {
	Timestamp time.Time
	Step      string
	Action    string
	Details   string
	Success   bool
	Error     string
	CommitSHA string
	OldValue  string
	NewValue  string
}

// New creates a new logger instance
// The nsha directory is created in the user's home directory
func New(repoPath string) (*Logger, error) {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Create nsha directory in user's home directory
	nshaDir := filepath.Join(homeDir, "nsha")
	err = os.MkdirAll(nshaDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create nsha directory: %w", err)
	}

	// Create timestamped subdirectory for this run
	timestamp := time.Now().Format("20060102-150405")
	runDir := filepath.Join(nshaDir, timestamp)
	err = os.MkdirAll(runDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create run directory: %w", err)
	}

	// Create log file
	logPath := filepath.Join(runDir, "nsha.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	logger := &Logger{
		logFile:    logFile,
		logDir:     runDir,
		startTime:  time.Now(),
		operations: make([]Operation, 0),
	}

	// Write header
	logger.writeHeader()

	return logger, nil
}

// writeHeader writes the log file header
func (l *Logger) writeHeader() {
	header := fmt.Sprintf(`╔═══════════════════════════════════════════════════════════╗
║                NSHA - Detailed Operation Log              ║
╚═══════════════════════════════════════════════════════════╝

Start Time: %s
Log Directory: %s

`, l.startTime.Format("2006-01-02 15:04:05"), l.logDir)

	l.logFile.WriteString(header)
	l.logFile.Sync()
}

// LogStep logs a major step in the process
func (l *Logger) LogStep(step, description string) {
	timestamp := time.Now()
	msg := fmt.Sprintf("\n[%s] STEP: %s - %s\n",
		timestamp.Format("15:04:05"), step, description)

	l.logFile.WriteString(msg)
	l.logFile.Sync()

	op := Operation{
		Timestamp: timestamp,
		Step:      step,
		Action:    "STEP",
		Details:   description,
		Success:   true,
	}
	l.operations = append(l.operations, op)
}

// LogAction logs a specific action
func (l *Logger) LogAction(step, action, details string) {
	timestamp := time.Now()
	msg := fmt.Sprintf("[%s]   ACTION: %s - %s\n",
		timestamp.Format("15:04:05"), action, details)

	l.logFile.WriteString(msg)
	l.logFile.Sync()

	op := Operation{
		Timestamp: timestamp,
		Step:      step,
		Action:    action,
		Details:   details,
		Success:   true,
	}
	l.operations = append(l.operations, op)
}

// LogChange logs a change with before/after values
func (l *Logger) LogChange(step, action, commitSHA, oldValue, newValue string) {
	timestamp := time.Now()
	msg := fmt.Sprintf("[%s]   CHANGE: %s\n", timestamp.Format("15:04:05"), action)
	if commitSHA != "" {
		msg += fmt.Sprintf("            Commit: %s\n", commitSHA)
	}
	msg += fmt.Sprintf("            Before: %s\n", oldValue)
	msg += fmt.Sprintf("            After:  %s\n", newValue)

	l.logFile.WriteString(msg)
	l.logFile.Sync()

	op := Operation{
		Timestamp: timestamp,
		Step:      step,
		Action:    action,
		CommitSHA: commitSHA,
		OldValue:  oldValue,
		NewValue:  newValue,
		Success:   true,
	}
	l.operations = append(l.operations, op)
}

// LogError logs an error
func (l *Logger) LogError(step, action, details, errorMsg string) {
	timestamp := time.Now()
	msg := fmt.Sprintf("[%s]   ERROR: %s - %s\n",
		timestamp.Format("15:04:05"), action, details)
	msg += fmt.Sprintf("            Error: %s\n", errorMsg)

	l.logFile.WriteString(msg)
	l.logFile.Sync()

	op := Operation{
		Timestamp: timestamp,
		Step:      step,
		Action:    action,
		Details:   details,
		Success:   false,
		Error:     errorMsg,
	}
	l.operations = append(l.operations, op)
}

// GetLogDir returns the directory where logs and backups are stored
func (l *Logger) GetLogDir() string {
	return l.logDir
}

// GetOperations returns all logged operations
func (l *Logger) GetOperations() []Operation {
	return l.operations
}

// Close closes the log file and writes summary
func (l *Logger) Close() error {
	duration := time.Since(l.startTime)

	footer := fmt.Sprintf(`
═══════════════════════════════════════════════════════════
End Time: %s
Duration: %s
Total Operations: %d
═══════════════════════════════════════════════════════════
`, time.Now().Format("2006-01-02 15:04:05"), duration, len(l.operations))

	l.logFile.WriteString(footer)
	l.logFile.Sync()

	return l.logFile.Close()
}

// LogInfo logs informational message
func (l *Logger) LogInfo(step, message string) {
	timestamp := time.Now()
	msg := fmt.Sprintf("[%s]   INFO: %s\n", timestamp.Format("15:04:05"), message)

	l.logFile.WriteString(msg)
	l.logFile.Sync()
}

// LogWarning logs a warning message
func (l *Logger) LogWarning(step, message string) {
	timestamp := time.Now()
	msg := fmt.Sprintf("[%s]   WARNING: %s\n", timestamp.Format("15:04:05"), message)

	l.logFile.WriteString(msg)
	l.logFile.Sync()
}
