// Package nethealthcheck provides unit tests for role-based node discovery
package nethealthcheck

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDryRunExecutor provides a mock implementation for testing
type MockDryRunExecutor struct {
	mock.Mock
}

func (m *MockDryRunExecutor) GetNode(ctx context.Context, nodeName string) (bool, string, error) {
	args := m.Called(ctx, nodeName)
	return args.Bool(0), args.String(1), args.Error(2)
}

func (m *MockDryRunExecutor) LabelNode(ctx context.Context, nodeName, label string, overwrite bool) (bool, string, error) {
	args := m.Called(ctx, nodeName, label, overwrite)
	return args.Bool(0), args.String(1), args.Error(2)
}

func (m *MockDryRunExecutor) UnlabelNode(ctx context.Context, nodeName, labelKey string) (bool, string, error) {
	args := m.Called(ctx, nodeName, labelKey)
	return args.Bool(0), args.String(1), args.Error(2)
}

func (m *MockDryRunExecutor) GetNodeLabels(ctx context.Context, nodeName string) (bool, string, error) {
	args := m.Called(ctx, nodeName)
	return args.Bool(0), args.String(1), args.Error(2)
}

func (m *MockDryRunExecutor) ExecNodeCommand(ctx context.Context, nodeName, command string) (bool, string, error) {
	args := m.Called(ctx, nodeName, command)
	return args.Bool(0), args.String(1), args.Error(2)
}

func (m *MockDryRunExecutor) GetPods(ctx context.Context, fieldSelector, labelSelector string) (bool, string, error) {
	args := m.Called(ctx, fieldSelector, labelSelector)
	return args.Bool(0), args.String(1), args.Error(2)
}

func (m *MockDryRunExecutor) DeletePod(ctx context.Context, podName string) (bool, string, error) {
	args := m.Called(ctx, podName)
	return args.Bool(0), args.String(1), args.Error(2)
}

func (m *MockDryRunExecutor) GetAllNodes(ctx context.Context) (bool, string, error) {
	args := m.Called(ctx)
	return args.Bool(0), args.String(1), args.Error(2)
}

func (m *MockDryRunExecutor) GetNodesByLabel(ctx context.Context, labelSelector string) (bool, string, error) {
	args := m.Called(ctx, labelSelector)
	return args.Bool(0), args.String(1), args.Error(2)
}

func (m *MockDryRunExecutor) GetNodeRole(ctx context.Context, nodeName string) (string, error) {
	args := m.Called(ctx, nodeName)
	return args.String(0), args.Error(1)
}

func (m *MockDryRunExecutor) DiscoverClusterState(ctx context.Context) (map[string]interface{}, error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockDryRunExecutor) DiscoverNodeVLANs(ctx context.Context, nodeName string) (bool, string, error) {
	args := m.Called(ctx, nodeName)
	return args.Bool(0), args.String(1), args.Error(2)
}

func (m *MockDryRunExecutor) DiscoverAllVLANs(ctx context.Context) (map[string]string, error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockDryRunExecutor) GetNodeNetworkInfo(ctx context.Context, nodeName string) (bool, string, error) {
	args := m.Called(ctx, nodeName)
	return args.Bool(0), args.String(1), args.Error(2)
}

func (m *MockDryRunExecutor) GetNodeHardwareInfo(ctx context.Context, nodeName string) (bool, string, error) {
	args := m.Called(ctx, nodeName)
	return args.Bool(0), args.String(1), args.Error(2)
}

func (m *MockDryRunExecutor) SetDryRun(enabled bool) {
	m.Called(enabled)
}

func (m *MockDryRunExecutor) IsDryRun() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockDryRunExecutor) SetPollingInterval(interval interface{}) {
	m.Called(interval)
}

// MockLogger for testing
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(message string) {
	m.Called(message)
}

func (m *MockLogger) Info(message string) {
	m.Called(message)
}

func (m *MockLogger) Warn(message string) {
	m.Called(message)
}

func (m *MockLogger) Error(message string) {
	m.Called(message)
}

// TestRoleBasedNodeDiscovery tests the new role-based node selection logic
func TestRoleBasedNodeDiscovery(t *testing.T) {
	tests := []struct {
		name           string
		networkName    string
		expectedRole   string
		nodeList       string
		nodeRoles      map[string]string
		expectedNodes  []string
		shouldError    bool
	}{
		{
			name:         "storage_network_selects_storage_nodes",
			networkName:  "storage",
			expectedRole: "storage",
			nodeList:     "node/rsb2\nnode/rsb3\nnode/rsb4\nnode/rsb5\nnode/rsb6\nnode/rsb7\nnode/rsb8",
			nodeRoles: map[string]string{
				"rsb2": "control-plane",
				"rsb3": "control-plane", 
				"rsb4": "control-plane",
				"rsb5": "storage",
				"rsb6": "storage",
				"rsb7": "compute",
				"rsb8": "compute",
			},
			expectedNodes: []string{"rsb5", "rsb6"},
			shouldError:   false,
		},
		{
			name:         "api_network_selects_control_plane_nodes",
			networkName:  "api",
			expectedRole: "control-plane",
			nodeList:     "node/rsb2\nnode/rsb3\nnode/rsb4\nnode/rsb5\nnode/rsb6",
			nodeRoles: map[string]string{
				"rsb2": "control-plane",
				"rsb3": "control-plane",
				"rsb4": "control-plane",
				"rsb5": "storage",
				"rsb6": "storage",
			},
			expectedNodes: []string{"rsb2", "rsb3", "rsb4"},
			shouldError:   false,
		},
		{
			name:         "tenant_network_selects_compute_nodes",
			networkName:  "tenant",
			expectedRole: "compute",
			nodeList:     "node/rsb5\nnode/rsb6\nnode/rsb7\nnode/rsb8",
			nodeRoles: map[string]string{
				"rsb5": "storage",
				"rsb6": "storage",
				"rsb7": "compute",
				"rsb8": "compute",
			},
			expectedNodes: []string{"rsb7", "rsb8"},
			shouldError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockKubectl := &MockDryRunExecutor{}
			mockLogger := &MockLogger{}

			// Mock GetAllNodes
			mockKubectl.On("GetAllNodes", mock.Anything).Return(true, tt.nodeList, nil)

			// Mock GetNodeRole for each node
			for nodeName, role := range tt.nodeRoles {
				mockKubectl.On("GetNodeRole", mock.Anything, nodeName).Return(role, nil)
			}

			// Allow flexible logging
			mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
			mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()

			// Create service
			service := &NetHealthCheckService{
				kubectl: mockKubectl,
				options: Options{
					Logger: mockLogger,
				},
			}

			// When: Get nodes for network
			nodes, err := service.getNodesForNetwork(tt.networkName)

			// Then: Verify results
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.ElementsMatch(t, tt.expectedNodes, nodes, 
					"Expected nodes %v but got %v for network %s", tt.expectedNodes, nodes, tt.networkName)
			}

			// Verify all mock expectations
			mockKubectl.AssertExpectations(t)
		})
	}
}

// TestNetworkRoleMapping tests the network to role mapping logic
func TestNetworkRoleMapping(t *testing.T) {
	mockKubectl := &MockDryRunExecutor{}
	mockLogger := &MockLogger{}

	service := &NetHealthCheckService{
		kubectl: mockKubectl,
		options: Options{
			Logger: mockLogger,
		},
	}

	tests := []struct {
		networkName    string
		expectVLANCall bool
	}{
		{"storage", false},      // Should use role-based
		{"api", false},          // Should use role-based  
		{"tenant", false},       // Should use role-based
		{"management", false},   // Should use "all" nodes
		{"unknown", true},       // Should fallback to VLAN-based
	}

	for _, tt := range tests {
		t.Run("network_"+tt.networkName, func(t *testing.T) {
			// Reset mocks for each test
			mockKubectl.ExpectedCalls = nil
			mockLogger.ExpectedCalls = nil

			if tt.expectVLANCall {
				// For unknown networks, service should use VLAN config (will fail since we don't set it)
				mockLogger.On("Warn", mock.MatchedBy(func(msg string) bool {
					return strings.Contains(msg, "Unknown network")
				})).Return()
			} else {
				if tt.networkName == "management" {
					// Management network gets all nodes
					mockKubectl.On("GetAllNodes", mock.Anything).Return(true, "node/rsb2\nnode/rsb5", nil)
				} else {
					// Other networks use role-based discovery
					mockKubectl.On("GetAllNodes", mock.Anything).Return(true, "node/rsb2\nnode/rsb5", nil)
					mockKubectl.On("GetNodeRole", mock.Anything, "rsb2").Return("control-plane", nil)
					mockKubectl.On("GetNodeRole", mock.Anything, "rsb5").Return("storage", nil)
					mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				}
			}

			// Call the function
			_, err := service.getNodesForNetwork(tt.networkName)

			if tt.expectVLANCall {
				// Should error since we don't have VLAN config
				assert.Error(t, err)
			} else {
				// Should succeed with role-based discovery
				assert.NoError(t, err)
			}
		})
	}
}
