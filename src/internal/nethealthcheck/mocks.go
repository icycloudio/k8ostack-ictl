package nethealthcheck

import (
	"context"
	"fmt"
	"time"

	"k8ostack-ictl/internal/config"
)

// MockNetHealthCheckService implements the Service interface for testing
type MockNetHealthCheckService struct {
	shouldFail   bool
	testResults  *TestResults
	networkState map[string]NetworkHealth
}

// NewMockService creates a new mock network health check service
func NewMockService(shouldFail bool) Service {
	return &MockNetHealthCheckService{
		shouldFail: shouldFail,
		testResults: &TestResults{
			TotalTests:        3,
			SuccessfulTests:   2,
			FailedTests:       1,
			SkippedTests:      0,
			TestExecutions:    []TestExecution{},
			NetworkValidation: make(map[string]NetworkHealth),
			Errors:            []error{},
			Duration:          5 * time.Second,
		},
		networkState: make(map[string]NetworkHealth),
	}
}

// RunTests mock implementation
func (m *MockNetHealthCheckService) RunTests(ctx context.Context, config *config.NodeTestConf) (*TestResults, error) {
	if m.shouldFail {
		return nil, &NetworkTestError{
			TestName:    "mock-test",
			Operation:   "run",
			Reason:      "mock failure",
			OriginalErr: nil,
		}
	}

	// Simulate test executions
	for i, test := range config.Spec.Tests {
		execution := TestExecution{
			TestName:      test.Name,
			TestType:      "ping",
			SourceNode:    "rsb7",
			TargetNode:    "rsb2",
			SourceNetwork: test.Source,
			TargetNetwork: test.Targets[0],
			Protocol:      "icmp",
			Port:          0,
			Service:       "",
			ExpectSuccess: test.ExpectSuccess,
			ActualSuccess: i%2 == 0, // Alternate success/failure
			Duration:      time.Duration(i+1) * time.Second,
			Output:        "Mock test output",
			ErrorMessage:  "",
		}

		if !execution.ActualSuccess {
			execution.ErrorMessage = "Mock test failure"
		}

		m.testResults.TestExecutions = append(m.testResults.TestExecutions, execution)
	}

	return m.testResults, nil
}

// StopTests mock implementation
func (m *MockNetHealthCheckService) StopTests(ctx context.Context, config *config.NodeTestConf) (*TestResults, error) {
	if m.shouldFail {
		return nil, &NetworkTestError{
			TestName:    "mock-test",
			Operation:   "stop",
			Reason:      "mock failure",
			OriginalErr: nil,
		}
	}
	return &TestResults{}, nil
}

// VerifyTests mock implementation
func (m *MockNetHealthCheckService) VerifyTests(ctx context.Context, config *config.NodeTestConf) (*TestResults, error) {
	if m.shouldFail {
		return nil, &NetworkTestError{
			TestName:    "mock-test",
			Operation:   "verify",
			Reason:      "mock failure",
			OriginalErr: nil,
		}
	}
	return m.testResults, nil
}

// GetCurrentState mock implementation
func (m *MockNetHealthCheckService) GetCurrentState(ctx context.Context, networks []string) (map[string]NetworkHealth, error) {
	if m.shouldFail {
		return nil, &NetworkTestError{
			TestName:    "state-discovery",
			Operation:   "get-state",
			Reason:      "mock failure",
			OriginalErr: nil,
		}
	}

	state := make(map[string]NetworkHealth)
	for _, network := range networks {
		state[network] = NetworkHealth{
			NetworkName:     network,
			Subnet:          "10.100.0.0/24",
			HealthyNodes:    []string{"rsb2", "rsb3", "rsb4"},
			UnhealthyNodes:  []string{},
			ServiceStatus:   map[string]bool{"keystone": true, "nova": true},
			IsolationStatus: map[string]bool{"storage": true, "tenant": true},
			OverallHealth:   "healthy",
		}
	}

	return state, nil
}

// NetworkTestError represents a network testing error
type NetworkTestError struct {
	TestName    string
	Operation   string
	Reason      string
	OriginalErr error
}

func (e *NetworkTestError) Error() string {
	if e.OriginalErr != nil {
		return fmt.Sprintf("network test %s failed during %s: %s (caused by: %v)", 
			e.TestName, e.Operation, e.Reason, e.OriginalErr)
	}
	return fmt.Sprintf("network test %s failed during %s: %s", 
		e.TestName, e.Operation, e.Reason)
}

func (e *NetworkTestError) Unwrap() error {
	return e.OriginalErr
}
