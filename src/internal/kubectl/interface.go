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

	// ExecNodeCommand executes a command on a specific node
	ExecNodeCommand(ctx context.Context, nodeName, command string) (bool, string, error)
}

// DryRunExecutor extends Executor with dry-run functionality
type DryRunExecutor interface {
	Executor
	SetDryRun(enabled bool)
	IsDryRun() bool
}

// Logger defines the interface for logging kubectl operations
type Logger interface {
	Debug(message string)
	Info(message string)
	Warn(message string)
	Error(message string)
}
