// Package vlan provides the core business logic for VLAN configuration operations
package vlan

import (
	"context"

	"k8ostack-ictl/internal/config"
	"k8ostack-ictl/internal/kubectl"
)

// OperationResults tracks the results of VLAN configuration operations
type OperationResults struct {
	TotalNodes      int
	SuccessfulNodes int
	FailedNodes     []string
	ConfiguredVLANs map[string][]VLANInterfaceInfo // node -> VLAN interfaces configured
	Errors          []error
}

// VLANInterfaceInfo represents information about a configured VLAN interface
type VLANInterfaceInfo struct {
	VLANName      string // e.g., "management", "storage"
	VLANId        int    // e.g., 100, 200
	Interface     string // e.g., "eth0.100", "eth1.300"
	IPAddress     string // e.g., "192.168.100.15/24"
	PhysInterface string // e.g., "eth0", "eth1"
	Subnet        string // e.g., "192.168.100.0/24"
}

// Service defines the interface for the VLAN configuration service
type Service interface {
	// ConfigureVLANs configures all VLANs defined in the configuration
	ConfigureVLANs(ctx context.Context, config *config.NodeVLANConf) (*OperationResults, error)

	// RemoveVLANs removes all VLANs defined in the configuration
	RemoveVLANs(ctx context.Context, config *config.NodeVLANConf) (*OperationResults, error)

	// VerifyVLANs checks if VLANs are configured correctly
	VerifyVLANs(ctx context.Context, config *config.NodeVLANConf) (*OperationResults, error)

	// GetCurrentState discovers the current VLAN configuration state
	GetCurrentState(ctx context.Context, nodes []string) (map[string][]VLANInterfaceInfo, error)
}

// Options contains configuration options for the VLAN service
type Options struct {
	DryRun               bool
	Verbose              bool
	ValidateConnectivity bool
	PersistentConfig     bool
	DefaultInterface     string
	Logger               kubectl.Logger
}

// VLANService implements the Service interface
type VLANService struct {
	kubectl kubectl.DryRunExecutor
	options Options
}

// NewService creates a new VLAN configuration service
func NewService(kubectl kubectl.DryRunExecutor, options Options) Service {
	return &VLANService{
		kubectl: kubectl,
		options: options,
	}
}

// NodeVLANState represents the VLAN configuration state for a single node
type NodeVLANState struct {
	NodeName       string
	VLANInterfaces []VLANInterfaceInfo
	ConfigStatus   string // "configured", "missing", "partial", "error"
	ErrorMessage   string
}

// VLANOperation represents a single VLAN configuration operation
type VLANOperation struct {
	NodeName     string
	VLANName     string
	VLANConfig   config.VLANConfig
	IPAddress    string
	Interface    string
	Operation    string // "create", "delete", "verify"
	Success      bool
	ErrorMessage string
}
