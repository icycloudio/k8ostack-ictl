// Package kubectl provides interfaces and implementations for Kubernetes operations
package kubectl

import "context"

// Executor defines the interface for executing kubectl commands
type Executor interface {
	// GetNode retrieves information about a specific node
	GetNode(ctx context.Context, nodeName string) (bool, string, error)

	// LabelNode applies a label to a node
	LabelNode(ctx context.Context, nodeName, label string, overwrite bool) (bool, string, error)

	// UnlabelNode removes a label from a node
	UnlabelNode(ctx context.Context, nodeName, labelKey string) (bool, string, error)

	// GetNodeLabels retrieves all labels for a specific node
	GetNodeLabels(ctx context.Context, nodeName string) (bool, string, error)
}

// DryRunExecutor wraps an Executor to provide dry-run capabilities
type DryRunExecutor interface {
	Executor

	// SetDryRun enables or disables dry-run mode
	SetDryRun(enabled bool)

	// IsDryRun returns whether dry-run mode is enabled
	IsDryRun() bool
}

// Logger defines the interface for logging kubectl operations
type Logger interface {
	Debug(message string)
	Info(message string)
	Warn(message string)
	Error(message string)
}
