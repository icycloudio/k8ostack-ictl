// Package vlan provides mock implementations for testing
package vlan

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

// MockDryRunExecutor mocks the kubectl.DryRunExecutor interface for VLAN testing
type MockDryRunExecutor struct {
	mock.Mock
	dryRun bool
}

// GetNode mocks node existence checking
func (m *MockDryRunExecutor) GetNode(ctx context.Context, nodeName string) (bool, string, error) {
	args := m.Called(ctx, nodeName)
	return args.Bool(0), args.String(1), args.Error(2)
}

// LabelNode mocks node labeling operations
func (m *MockDryRunExecutor) LabelNode(ctx context.Context, nodeName, label string, overwrite bool) (bool, string, error) {
	args := m.Called(ctx, nodeName, label, overwrite)
	return args.Bool(0), args.String(1), args.Error(2)
}

// UnlabelNode mocks node label removal operations
func (m *MockDryRunExecutor) UnlabelNode(ctx context.Context, nodeName, labelKey string) (bool, string, error) {
	args := m.Called(ctx, nodeName, labelKey)
	return args.Bool(0), args.String(1), args.Error(2)
}

// GetNodeLabels mocks node label retrieval
func (m *MockDryRunExecutor) GetNodeLabels(ctx context.Context, nodeName string) (bool, string, error) {
	args := m.Called(ctx, nodeName)
	return args.Bool(0), args.String(1), args.Error(2)
}

// ExecNodeCommand mocks node command execution
func (m *MockDryRunExecutor) ExecNodeCommand(ctx context.Context, nodeName, command string) (bool, string, error) {
	args := m.Called(ctx, nodeName, command)
	return args.Bool(0), args.String(1), args.Error(2)
}

// GetPods mocks pod retrieval with filtering
func (m *MockDryRunExecutor) GetPods(ctx context.Context, fieldSelector, labelSelector string) (bool, string, error) {
	args := m.Called(ctx, fieldSelector, labelSelector)
	return args.Bool(0), args.String(1), args.Error(2)
}

// DeletePod mocks pod deletion operations
func (m *MockDryRunExecutor) DeletePod(ctx context.Context, podName string) (bool, string, error) {
	args := m.Called(ctx, podName)
	return args.Bool(0), args.String(1), args.Error(2)
}

// SetDryRun enables or disables dry-run mode
func (m *MockDryRunExecutor) SetDryRun(enabled bool) {
	m.dryRun = enabled
	m.Called(enabled)
}

// IsDryRun returns whether dry-run mode is enabled
func (m *MockDryRunExecutor) IsDryRun() bool {
	args := m.Called()
	return args.Bool(0)
}

// SetPollingInterval mocks the polling interval configuration
func (m *MockDryRunExecutor) SetPollingInterval(interval time.Duration) {
	m.Called(interval)
}

// MockLogger mocks the kubectl.Logger interface for test output verification
type MockLogger struct {
	mock.Mock
	Messages []LogMessage
}

// LogMessage captures structured log data for assertions
type LogMessage struct {
	Level   string
	Message string
}

// Debug captures debug messages
func (m *MockLogger) Debug(message string) {
	m.Messages = append(m.Messages, LogMessage{"DEBUG", message})
	m.Called(message)
}

// Info captures info messages
func (m *MockLogger) Info(message string) {
	m.Messages = append(m.Messages, LogMessage{"INFO", message})
	m.Called(message)
}

// Warn captures warning messages
func (m *MockLogger) Warn(message string) {
	m.Messages = append(m.Messages, LogMessage{"WARN", message})
	m.Called(message)
}

// Error captures error messages
func (m *MockLogger) Error(message string) {
	m.Messages = append(m.Messages, LogMessage{"ERROR", message})
	m.Called(message)
}

// GetMessages returns all captured messages for test assertions
func (m *MockLogger) GetMessages() []LogMessage {
	return m.Messages
}

// GetMessagesByLevel returns messages filtered by log level
func (m *MockLogger) GetMessagesByLevel(level string) []LogMessage {
	var filtered []LogMessage
	for _, msg := range m.Messages {
		if msg.Level == level {
			filtered = append(filtered, msg)
		}
	}
	return filtered
}

// Clear resets captured messages for fresh test runs
func (m *MockLogger) Clear() {
	m.Messages = []LogMessage{}
}

// NewMockDryRunExecutor creates a new mock executor for testing
func NewMockDryRunExecutor() *MockDryRunExecutor {
	return &MockDryRunExecutor{}
}

// NewMockLogger creates a new mock logger for testing
func NewMockLogger() *MockLogger {
	return &MockLogger{
		Messages: make([]LogMessage, 0),
	}
}
