// Package nethealthcheck provides the core business logic for network connectivity testing operations
package nethealthcheck

import (
	"context"
	"time"

	"k8ostack-ictl/internal/config"
	"k8ostack-ictl/internal/kubectl"
)

// TestResults tracks the results of network connectivity testing operations
type TestResults struct {
	TotalTests        int
	SuccessfulTests   int
	FailedTests       int
	SkippedTests      int
	TestExecutions    []TestExecution
	NetworkValidation map[string]NetworkHealth
	Errors            []error
	Duration          time.Duration
}

// TestExecution represents information about a single test execution
type TestExecution struct {
	TestName       string        // e.g., "keystone-api-connectivity"
	TestType       string        // e.g., "openstack-api", "ping", "tcp"
	SourceNode     string        // e.g., "rsb7"
	TargetNode     string        // e.g., "rsb2"
	SourceNetwork  string        // e.g., "management"
	TargetNetwork  string        // e.g., "api"
	Protocol       string        // e.g., "tcp", "udp", "icmp"
	Port           int           // e.g., 5000 for Keystone
	Service        string        // e.g., "keystone", "nova", "ceph-mon"
	ExpectSuccess  bool          // Whether this test should succeed
	ActualSuccess  bool          // Whether this test actually succeeded
	Duration       time.Duration // How long the test took
	Output         string        // Command output or response
	ErrorMessage   string        // Error details if failed
}

// NetworkHealth represents the health status of a network segment
type NetworkHealth struct {
	NetworkName     string                // e.g., "management", "storage"
	Subnet          string                // e.g., "10.100.0.0/24"
	HealthyNodes    []string              // Nodes that are reachable
	UnhealthyNodes  []string              // Nodes that are not reachable
	ServiceStatus   map[string]bool       // Service -> healthy status
	IsolationStatus map[string]bool       // Target network -> properly isolated
	OverallHealth   string                // "healthy", "degraded", "unhealthy"
}

// Service defines the interface for the network health check service
type Service interface {
	// RunTests executes all tests defined in the configuration
	RunTests(ctx context.Context, config *config.NodeTestConf) (*TestResults, error)

	// StopTests stops any running tests
	StopTests(ctx context.Context, config *config.NodeTestConf) (*TestResults, error)

	// VerifyTests checks if test infrastructure is configured correctly
	VerifyTests(ctx context.Context, config *config.NodeTestConf) (*TestResults, error)

	// GetCurrentState discovers the current network health state
	GetCurrentState(ctx context.Context, networks []string) (map[string]NetworkHealth, error)
}

// Options contains configuration options for the network health check service
type Options struct {
	DryRun               bool
	Verbose              bool
	Parallel             bool
	Retries              int
	OutputFormat         string        // "summary", "detailed", "json"
	TimeoutDefault       int           // Default timeout in seconds
	CleanupAfterTests    bool
	OpenstackProfiles    []string      // e.g., ["control-plane", "compute", "storage"]
	Logger               kubectl.Logger
	TestDelay            time.Duration // For testing - can be set to 0 to skip sleep
}

// NetHealthCheckService implements the Service interface
type NetHealthCheckService struct {
	kubectl kubectl.DryRunExecutor
	options Options
	vlanConfig *config.NodeVLANConf // For network-to-IP mapping
}

// NewService creates a new network health check service
func NewService(kubectl kubectl.DryRunExecutor, options Options) Service {
	return &NetHealthCheckService{
		kubectl: kubectl,
		options: options,
	}
}

// NewServiceWithVLAN creates a new network health check service with VLAN configuration
func NewServiceWithVLAN(kubectl kubectl.DryRunExecutor, options Options, vlanConfig *config.NodeVLANConf) Service {
	return &NetHealthCheckService{
		kubectl:    kubectl,
		options:    options,
		vlanConfig: vlanConfig,
	}
}

// TestOperation represents a single network test operation
type TestOperation struct {
	TestName     string
	TestType     string
	SourceNode   string
	TargetNode   string
	TestConfig   config.ConnectivityTest
	Command      string
	Operation    string // "execute", "verify", "cleanup"
	Success      bool
	ErrorMessage string
	Duration     time.Duration
}

// Enhanced configuration types for more sophisticated testing
type EnhancedTestConfig struct {
	Name         string              `json:"name" yaml:"name"`
	Description  string              `json:"description,omitempty" yaml:"description,omitempty"`
	Type         string              `json:"type" yaml:"type"` // "ping", "tcp", "openstack-api", "ceph", "isolation"
	Source       TestEndpoint        `json:"source" yaml:"source"`
	Targets      []TestEndpoint      `json:"targets" yaml:"targets"`
	Protocol     string              `json:"protocol,omitempty" yaml:"protocol,omitempty"`
	Port         int                 `json:"port,omitempty" yaml:"port,omitempty"`
	Service      string              `json:"service,omitempty" yaml:"service,omitempty"`
	Timeout      int                 `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Retries      int                 `json:"retries,omitempty" yaml:"retries,omitempty"`
	ExpectSuccess bool               `json:"expectSuccess,omitempty" yaml:"expectSuccess,omitempty"`
	Tags         []string            `json:"tags,omitempty" yaml:"tags,omitempty"`
	CustomCommand string             `json:"customCommand,omitempty" yaml:"customCommand,omitempty"`
}

// TestEndpoint represents a source or target for testing
type TestEndpoint struct {
	NetworkName string   `json:"networkName" yaml:"networkName"`
	NodeNames   []string `json:"nodeNames" yaml:"nodeNames"`
	Port        int      `json:"port,omitempty" yaml:"port,omitempty"`
	Service     string   `json:"service,omitempty" yaml:"service,omitempty"`
}

// OpenStackServiceTest represents specific OpenStack service tests
type OpenStackServiceTest struct {
	Service  string `json:"service" yaml:"service"`   // "keystone", "nova", "neutron", etc.
	Port     int    `json:"port" yaml:"port"`         // Service port
	Endpoint string `json:"endpoint" yaml:"endpoint"` // API endpoint path
	Type     string `json:"type" yaml:"type"`         // "tcp", "http", "https"
}
