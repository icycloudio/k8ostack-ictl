// Package testing provides the core business logic for OpenStack network connectivity testing
package testing

import (
	"context"
	"time"

	"k8ostack-ictl/internal/config"
	"k8ostack-ictl/internal/kubectl"
)

// TestResults tracks the results of connectivity test operations
type TestResults struct {
	TotalTests        int
	SuccessfulTests   int
	FailedTests       int
	SkippedTests      int
	TestExecutions    []TestExecution
	NetworkValidation map[string]NetworkHealth // network -> health status
	Errors            []error
	Duration          time.Duration
}

// TestExecution represents a single test execution result
type TestExecution struct {
	TestName        string
	TestType        string
	Source          TestEndpoint
	Target          TestEndpoint
	Status          TestStatus
	Duration        time.Duration
	ExpectedResult  bool
	ActualResult    bool
	ErrorMessage    string
	Details         map[string]interface{}
	Timestamp       time.Time
}

// TestEndpoint represents a network endpoint for testing
type TestEndpoint struct {
	NodeName    string `json:"nodeName,omitempty" yaml:"nodeName,omitempty"`
	NetworkName string `json:"networkName,omitempty" yaml:"networkName,omitempty"`
	IPAddress   string `json:"ipAddress,omitempty" yaml:"ipAddress,omitempty"`
	Interface   string `json:"interface,omitempty" yaml:"interface,omitempty"`
	Port        int    `json:"port,omitempty" yaml:"port,omitempty"`
}

// TestStatus represents the execution status of a test
type TestStatus string

const (
	TestStatusPending    TestStatus = "pending"
	TestStatusRunning    TestStatus = "running"
	TestStatusPassed     TestStatus = "passed"
	TestStatusFailed     TestStatus = "failed"
	TestStatusSkipped    TestStatus = "skipped"
	TestStatusError      TestStatus = "error"
	TestStatusTimeout    TestStatus = "timeout"
)

// NetworkHealth represents the health status of a network
type NetworkHealth struct {
	NetworkName     string
	TotalNodes      int
	HealthyNodes    int
	UnhealthyNodes  []string
	LastChecked     time.Time
	Issues          []string
}

// OpenStackTest represents an enhanced connectivity test for OpenStack
type OpenStackTest struct {
	Name             string            `json:"name" yaml:"name"`
	Description      string            `json:"description,omitempty" yaml:"description,omitempty"`
	Type             OpenStackTestType `json:"type,omitempty" yaml:"type,omitempty"`
	Source           TestEndpoint      `json:"source" yaml:"source"`
	Targets          []TestEndpoint    `json:"targets" yaml:"targets"`
	Timeout          int               `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	ExpectSuccess    bool              `json:"expectSuccess,omitempty" yaml:"expectSuccess,omitempty"`
	Retries          int               `json:"retries,omitempty" yaml:"retries,omitempty"`
	Parallel         bool              `json:"parallel,omitempty" yaml:"parallel,omitempty"`
	Protocol         string            `json:"protocol,omitempty" yaml:"protocol,omitempty"`
	Port             int               `json:"port,omitempty" yaml:"port,omitempty"`
	Tags             []string          `json:"tags,omitempty" yaml:"tags,omitempty"`
	OpenStackService string            `json:"openstackService,omitempty" yaml:"openstackService,omitempty"`
	Command          string            `json:"command,omitempty" yaml:"command,omitempty"`
}

// OpenStackTestType represents different types of OpenStack tests
type OpenStackTestType string

const (
	// Basic connectivity tests
	TestTypeICMP            OpenStackTestType = "icmp"
	TestTypeTCP             OpenStackTestType = "tcp"
	TestTypeUDP             OpenStackTestType = "udp"
	
	// OpenStack service tests
	TestTypeOpenStackAPI    OpenStackTestType = "openstack-api"
	TestTypeDatabase        OpenStackTestType = "database"
	TestTypeMessageQueue    OpenStackTestType = "message-queue"
	TestTypeCeph            OpenStackTestType = "ceph"
	TestTypeKeystone        OpenStackTestType = "keystone"
	TestTypeNova            OpenStackTestType = "nova"
	TestTypeNeutron         OpenStackTestType = "neutron"
	TestTypeGlance          OpenStackTestType = "glance"
	TestTypeCinder          OpenStackTestType = "cinder"
	
	// Network isolation tests
	TestTypeIsolation       OpenStackTestType = "isolation"
	TestTypeNetworkSegment  OpenStackTestType = "network-segment"
	
	// High availability tests
	TestTypeHA              OpenStackTestType = "ha"
	TestTypeLoadBalancer    OpenStackTestType = "load-balancer"
	
	// Custom command tests
	TestTypeCustomCommand   OpenStackTestType = "custom-command"
)

// Service defines the interface for the OpenStack testing service
type Service interface {
	// RunTests executes all tests defined in the configuration
	RunTests(ctx context.Context, config *config.NodeTestConf) (*TestResults, error)

	// StopTests stops any running tests
	StopTests(ctx context.Context, config *config.NodeTestConf) (*TestResults, error)

	// VerifyTests verifies that test infrastructure is ready
	VerifyTests(ctx context.Context, config *config.NodeTestConf) (*TestResults, error)

	// GetCurrentState discovers the current network testing state
	GetCurrentState(ctx context.Context, networks []string) (map[string]NetworkHealth, error)
	
	// RunSingleTest executes a single test for debugging
	RunSingleTest(ctx context.Context, test OpenStackTest, vlanConfig map[string]config.VLANConfig) (*TestExecution, error)
}

// Options contains configuration options for the testing service
type Options struct {
	DryRun            bool
	Verbose           bool
	Parallel          bool
	Retries           int
	OutputFormat      string
	TimeoutDefault    int
	CleanupAfterTests bool
	Logger            kubectl.Logger
}

// TestingService implements the Service interface
type TestingService struct {
	kubectl kubectl.DryRunExecutor
	options Options
}

// NewService creates a new OpenStack testing service
func NewService(kubectl kubectl.DryRunExecutor, options Options) Service {
	return &TestingService{
		kubectl: kubectl,
		options: options,
	}
}

// OpenStackServiceEndpoints defines common OpenStack service endpoints
var OpenStackServiceEndpoints = map[string]int{
	"keystone":       5000,
	"nova":          8774,
	"neutron":       9696,
	"glance":        9292,
	"cinder":        8776,
	"heat":          8004,
	"ceilometer":    8777,
	"mysql":         3306,
	"mariadb":       3306,
	"rabbitmq":      5672,
	"memcached":     11211,
	"ceph-mon":      6789,
	"ceph-osd":      6800,
	"ceph-mgr":      7000,
}

// OpenStackNetworkProfiles defines network testing profiles
var OpenStackNetworkProfiles = map[string][]OpenStackTest{
	"control-plane": {
		{
			Name:             "keystone-api",
			Type:             TestTypeKeystone,
			Protocol:         "tcp",
			Port:             5000,
			OpenStackService: "keystone",
			ExpectSuccess:    true,
		},
		{
			Name:             "database-connectivity",
			Type:             TestTypeDatabase,
			Protocol:         "tcp",
			Port:             3306,
			OpenStackService: "mysql",
			ExpectSuccess:    true,
		},
		{
			Name:             "message-queue",
			Type:             TestTypeMessageQueue,
			Protocol:         "tcp",
			Port:             5672,
			OpenStackService: "rabbitmq",
			ExpectSuccess:    true,
		},
	},
	"compute": {
		{
			Name:             "nova-api",
			Type:             TestTypeNova,
			Protocol:         "tcp",
			Port:             8774,
			OpenStackService: "nova",
			ExpectSuccess:    true,
		},
		{
			Name:             "neutron-api",
			Type:             TestTypeNeutron,
			Protocol:         "tcp",
			Port:             9696,
			OpenStackService: "neutron",
			ExpectSuccess:    true,
		},
		{
			Name:             "ceph-storage",
			Type:             TestTypeCeph,
			Protocol:         "tcp",
			Port:             6789,
			OpenStackService: "ceph-mon",
			ExpectSuccess:    true,
		},
	},
	"storage": {
		{
			Name:             "ceph-cluster",
			Type:             TestTypeCeph,
			Protocol:         "tcp",
			Port:             6789,
			OpenStackService: "ceph-mon",
			ExpectSuccess:    true,
		},
		{
			Name:             "ceph-replication",
			Type:             TestTypeCeph,
			Protocol:         "tcp",
			Port:             6800,
			OpenStackService: "ceph-osd",
			ExpectSuccess:    true,
		},
	},
}
