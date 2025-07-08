// Package labeler provides the core business logic for node labeling operations
package labeler

import (
	"context"

	"k8ostack-ictl/internal/config"
	"k8ostack-ictl/internal/kubectl"
)

// OperationResults tracks the results of labeling operations
type OperationResults struct {
	TotalNodes      int
	SuccessfulNodes int
	FailedNodes     []string
	AppliedLabels   map[string][]string // node -> labels applied
	Errors          []error
}

// Service defines the interface for the labeling service
type Service interface {
	// ApplyLabels applies all labels defined in the configuration
	ApplyLabels(ctx context.Context, config config.Config) (*OperationResults, error)

	// RemoveLabels removes all labels defined in the configuration
	RemoveLabels(ctx context.Context, config config.Config) (*OperationResults, error)

	// VerifyLabels checks if labels are applied correctly
	VerifyLabels(ctx context.Context, config config.Config) (*OperationResults, error)

	// GetCurrentState discovers the current labeling state
	GetCurrentState(ctx context.Context, nodes []string) (map[string]map[string]string, error)
}

// Options contains configuration options for the labeling service
type Options struct {
	DryRun        bool
	Verbose       bool
	ValidateNodes bool
	Logger        kubectl.Logger
}

// LabelingService implements the Service interface
type LabelingService struct {
	kubectl kubectl.DryRunExecutor
	options Options
}

// NewService creates a new labeling service
func NewService(kubectl kubectl.DryRunExecutor, options Options) Service {
	return &LabelingService{
		kubectl: kubectl,
		options: options,
	}
}
