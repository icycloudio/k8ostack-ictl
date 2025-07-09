// Package logging provides logging utilities for k8ostack-ictl
package logging

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// FileLogger implements the kubectl.Logger interface with file and console output
type FileLogger struct {
	fileLogger *log.Logger
	logFile    *os.File
	verbose    bool
}

// NewFileLogger creates a new logger that writes to both file and console
func NewFileLogger(logDir string, verbose bool) (*FileLogger, error) {
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("20060102_150405")
	logPath := filepath.Join(logDir, fmt.Sprintf("node_labeling_%s.log", timestamp))

	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	fileLogger := log.New(logFile, "", log.LstdFlags)

	logger := &FileLogger{
		fileLogger: fileLogger,
		logFile:    logFile,
		verbose:    verbose,
	}

	// Log initialization
	fmt.Printf("📝 Logging to: %s\n", logPath)
	logger.Info(fmt.Sprintf("Logging to: %s", logPath))

	return logger, nil
}

// Close closes the log file
func (l *FileLogger) Close() error {
	if l.logFile != nil {
		err := l.logFile.Close()
		l.logFile = nil // Set to nil to prevent double closing
		return err
	}
	return nil
}

// Debug logs debug messages (only in verbose mode)
func (l *FileLogger) Debug(message string) {
	l.fileLogger.Printf("[DEBUG] %s", message)
	if l.verbose {
		fmt.Printf("DEBUG: %s\n", message)
	}
}

// Info logs informational messages
func (l *FileLogger) Info(message string) {
	l.fileLogger.Printf("[INFO] %s", message)
	fmt.Printf("INFO: %s\n", message)
}

// Warn logs warning messages
func (l *FileLogger) Warn(message string) {
	l.fileLogger.Printf("[WARN] %s", message)
	fmt.Printf("WARN: %s\n", message)
}

// Error logs error messages
func (l *FileLogger) Error(message string) {
	l.fileLogger.Printf("[ERROR] %s", message)
	fmt.Printf("ERROR: %s\n", message)
}
