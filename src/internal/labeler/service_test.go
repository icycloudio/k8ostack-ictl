// Package labeler provides unit tests for the labeling service
// WHY: This test suite validates core business logic without external dependencies
package labeler

import (
	"context"
	"testing"

	"k8ostack-ictl/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestLabelingService_ApplyLabels tests the core label application business logic
// WHY: Validates that labels are correctly applied to nodes with proper error handling
func TestLabelingService_ApplyLabels(t *testing.T) {
	tests := []struct {
		name                 string
		description          string
		nodeConfig           map[string]config.NodeRole
		mockSetupFunc        func(*MockDryRunExecutor, *MockLogger)
		expectedTotalNodes   int
		expectedSuccessNodes int
		expectedFailedNodes  []string
		shouldError          bool
	}{
		{
			name:        "successful_single_node_single_label",
			description: "Successfully applies one label to one node",
			nodeConfig: map[string]config.NodeRole{
				"control_plane": {
					Nodes:       []string{"rsb2"},
					Labels:      map[string]string{"node.openstack.io/control-plane": "true"},
					Description: "Control plane node",
				},
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				// Mock dry-run setting
				mockKubectl.On("SetDryRun", false).Return()

				// Mock successful node validation
				mockKubectl.On("GetNode", mock.Anything, "rsb2").Return(true, "node/rsb2", nil)

				// Mock successful label application
				mockKubectl.On("LabelNode", mock.Anything, "rsb2", "node.openstack.io/control-plane=true", true).
					Return(true, "node/rsb2 labeled", nil)

				// Mock logger calls - we don't need to assert on these for this test
				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectedTotalNodes:   1,
			expectedSuccessNodes: 1,
			expectedFailedNodes:  nil,
			shouldError:          false,
		},
		{
			name:        "multiple_nodes_multiple_labels",
			description: "Successfully applies multiple labels to multiple nodes",
			nodeConfig: map[string]config.NodeRole{
				"control_plane": {
					Nodes: []string{"rsb2", "rsb3"},
					Labels: map[string]string{
						"node.openstack.io/control-plane": "true",
						"topology.kubernetes.io/zone":     "zone-a",
					},
					Description: "Control plane nodes",
				},
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", false).Return()

				// Mock successful operations for both nodes
				for _, node := range []string{"rsb2", "rsb3"} {
					mockKubectl.On("GetNode", mock.Anything, node).Return(true, "node/"+node, nil)

					// Both labels for each node
					mockKubectl.On("LabelNode", mock.Anything, node, "node.openstack.io/control-plane=true", true).
						Return(true, "node/"+node+" labeled", nil)
					mockKubectl.On("LabelNode", mock.Anything, node, "topology.kubernetes.io/zone=zone-a", true).
						Return(true, "node/"+node+" labeled", nil)
				}

				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectedTotalNodes:   2,
			expectedSuccessNodes: 2,
			expectedFailedNodes:  nil,
			shouldError:          false,
		},
		{
			name:        "node_not_found_failure",
			description: "Handles failure when node doesn't exist",
			nodeConfig: map[string]config.NodeRole{
				"worker": {
					Nodes:       []string{"nonexistent-node"},
					Labels:      map[string]string{"node.openstack.io/worker": "true"},
					Description: "Worker node that doesn't exist",
				},
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", false).Return()

				// Mock node not found
				mockKubectl.On("GetNode", mock.Anything, "nonexistent-node").Return(false, "", nil)

				// Logger should capture error and warn messages
				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Error", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectedTotalNodes:   1,
			expectedSuccessNodes: 0,
			expectedFailedNodes:  []string{"nonexistent-node"},
			shouldError:          false,
		},
		{
			name:        "mixed_success_and_failure",
			description: "Handles mix of successful and failed operations",
			nodeConfig: map[string]config.NodeRole{
				"mixed": {
					Nodes:       []string{"good-node", "bad-node"},
					Labels:      map[string]string{"test.io/label": "value"},
					Description: "Mix of good and bad nodes",
				},
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", false).Return()

				// Good node succeeds
				mockKubectl.On("GetNode", mock.Anything, "good-node").Return(true, "node/good-node", nil)
				mockKubectl.On("LabelNode", mock.Anything, "good-node", "test.io/label=value", true).
					Return(true, "node/good-node labeled", nil)

				// Bad node fails
				mockKubectl.On("GetNode", mock.Anything, "bad-node").Return(false, "", nil)

				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Error", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectedTotalNodes:   2,
			expectedSuccessNodes: 1,
			expectedFailedNodes:  []string{"bad-node"},
			shouldError:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Setup mocks and service
			mockKubectl := NewMockDryRunExecutor()
			mockLogger := NewMockLogger()

			tt.mockSetupFunc(mockKubectl, mockLogger)

			service := NewService(mockKubectl, Options{
				DryRun:        false,
				ValidateNodes: true,
				Logger:        mockLogger,
			})

			// Create test configuration
			testConfig := &config.NodeLabelConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeLabelConf",
				Metadata: config.Metadata{
					Name: "test-config",
				},
				Spec: config.NodeLabelSpec{
					NodeRoles: tt.nodeConfig,
				},
			}

			// When: Apply labels
			result, err := service.ApplyLabels(context.Background(), testConfig)

			// Then: Verify results
			if tt.shouldError {
				assert.Error(t, err, "Test %s: expected error but got none", tt.name)
			} else {
				assert.NoError(t, err, "Test %s: unexpected error: %v", tt.name, err)
			}

			assert.NotNil(t, result, "Test %s: result should not be nil", tt.name)
			assert.Equal(t, tt.expectedTotalNodes, result.TotalNodes,
				"Test %s: total nodes mismatch", tt.name)
			assert.Equal(t, tt.expectedSuccessNodes, result.SuccessfulNodes,
				"Test %s: successful nodes mismatch", tt.name)

			if tt.expectedFailedNodes == nil {
				assert.Empty(t, result.FailedNodes, "Test %s: failed nodes should be empty", tt.name)
			} else {
				assert.Equal(t, tt.expectedFailedNodes, result.FailedNodes,
					"Test %s: failed nodes mismatch", tt.name)
			}

			// Verify all mock expectations were met
			mockKubectl.AssertExpectations(t)
		})
	}
}

// TestLabelingService_RemoveLabels tests the label removal functionality
// WHY: Validates that labels are correctly removed from nodes with proper error handling
func TestLabelingService_RemoveLabels(t *testing.T) {
	tests := []struct {
		name                 string
		description          string
		nodeConfig           map[string]config.NodeRole
		mockSetupFunc        func(*MockDryRunExecutor, *MockLogger)
		expectedTotalNodes   int
		expectedSuccessNodes int
		expectedFailedNodes  []string
		shouldError          bool
	}{
		{
			name:        "successful_label_removal",
			description: "Successfully removes labels from nodes",
			nodeConfig: map[string]config.NodeRole{
				"control_plane": {
					Nodes:       []string{"rsb2"},
					Labels:      map[string]string{"node.openstack.io/control-plane": "true"},
					Description: "Control plane node",
				},
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", false).Return()
				mockKubectl.On("GetNode", mock.Anything, "rsb2").Return(true, "node/rsb2", nil)
				mockKubectl.On("UnlabelNode", mock.Anything, "rsb2", "node.openstack.io/control-plane").
					Return(true, "node/rsb2 unlabeled", nil)
				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectedTotalNodes:   1,
			expectedSuccessNodes: 1,
			expectedFailedNodes:  nil,
			shouldError:          false,
		},
		{
			name:        "remove_failure_on_nonexistent_node",
			description: "Handles removal failure when node doesn't exist",
			nodeConfig: map[string]config.NodeRole{
				"worker": {
					Nodes:       []string{"nonexistent-node"},
					Labels:      map[string]string{"node.openstack.io/worker": "true"},
					Description: "Worker node that doesn't exist",
				},
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", false).Return()
				mockKubectl.On("GetNode", mock.Anything, "nonexistent-node").Return(false, "", nil)
				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Error", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectedTotalNodes:   1,
			expectedSuccessNodes: 0,
			expectedFailedNodes:  []string{"nonexistent-node"},
			shouldError:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Setup mocks and service
			mockKubectl := NewMockDryRunExecutor()
			mockLogger := NewMockLogger()

			tt.mockSetupFunc(mockKubectl, mockLogger)

			service := NewService(mockKubectl, Options{
				DryRun:        false,
				ValidateNodes: true,
				Logger:        mockLogger,
			})

			// Create test configuration
			testConfig := &config.NodeLabelConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeLabelConf",
				Metadata: config.Metadata{
					Name: "test-config",
				},
				Spec: config.NodeLabelSpec{
					NodeRoles: tt.nodeConfig,
				},
			}

			// When: Remove labels
			result, err := service.RemoveLabels(context.Background(), testConfig)

			// Then: Verify results
			if tt.shouldError {
				assert.Error(t, err, "Test %s: expected error but got none", tt.name)
			} else {
				assert.NoError(t, err, "Test %s: unexpected error: %v", tt.name, err)
			}

			assert.NotNil(t, result, "Test %s: result should not be nil", tt.name)
			assert.Equal(t, tt.expectedTotalNodes, result.TotalNodes,
				"Test %s: total nodes mismatch", tt.name)
			assert.Equal(t, tt.expectedSuccessNodes, result.SuccessfulNodes,
				"Test %s: successful nodes mismatch", tt.name)

			if tt.expectedFailedNodes == nil {
				assert.Empty(t, result.FailedNodes, "Test %s: failed nodes should be empty", tt.name)
			} else {
				assert.Equal(t, tt.expectedFailedNodes, result.FailedNodes,
					"Test %s: failed nodes mismatch", tt.name)
			}

			// Verify all mock expectations were met
			mockKubectl.AssertExpectations(t)
		})
	}
}

// TestLabelingService_VerifyLabels tests the label verification functionality
// WHY: Validates that label verification works correctly with proper error handling
func TestLabelingService_VerifyLabels(t *testing.T) {
	tests := []struct {
		name                 string
		description          string
		nodeConfig           map[string]config.NodeRole
		mockSetupFunc        func(*MockDryRunExecutor, *MockLogger)
		expectedTotalNodes   int
		expectedSuccessNodes int
		expectedFailedNodes  []string
		shouldError          bool
	}{
		{
			name:        "successful_label_verification",
			description: "Successfully verifies labels on nodes",
			nodeConfig: map[string]config.NodeRole{
				"control_plane": {
					Nodes:       []string{"rsb2"},
					Labels:      map[string]string{"node.openstack.io/control-plane": "true"},
					Description: "Control plane node",
				},
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("GetNodeLabels", mock.Anything, "rsb2").
					Return(true, "node.openstack.io/control-plane=true", nil)
				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectedTotalNodes:   1,
			expectedSuccessNodes: 1,
			expectedFailedNodes:  nil,
			shouldError:          false,
		},
		{
			name:        "verification_failure_missing_label",
			description: "Handles verification failure when label is missing",
			nodeConfig: map[string]config.NodeRole{
				"worker": {
					Nodes:       []string{"rsb3"},
					Labels:      map[string]string{"node.openstack.io/worker": "true"},
					Description: "Worker node",
				},
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("GetNodeLabels", mock.Anything, "rsb3").
					Return(true, "other-label=value", nil) // Missing expected label
				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectedTotalNodes:   1,
			expectedSuccessNodes: 0,
			expectedFailedNodes:  []string{"rsb3"},
			shouldError:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Setup mocks and service
			mockKubectl := NewMockDryRunExecutor()
			mockLogger := NewMockLogger()

			tt.mockSetupFunc(mockKubectl, mockLogger)

			service := NewService(mockKubectl, Options{
				DryRun:        false,
				ValidateNodes: true,
				Logger:        mockLogger,
			})

			// Create test configuration
			testConfig := &config.NodeLabelConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeLabelConf",
				Metadata: config.Metadata{
					Name: "test-config",
				},
				Spec: config.NodeLabelSpec{
					NodeRoles: tt.nodeConfig,
				},
			}

			// When: Verify labels
			result, err := service.VerifyLabels(context.Background(), testConfig)

			// Then: Verify results
			if tt.shouldError {
				assert.Error(t, err, "Test %s: expected error but got none", tt.name)
			} else {
				assert.NoError(t, err, "Test %s: unexpected error: %v", tt.name, err)
			}

			assert.NotNil(t, result, "Test %s: result should not be nil", tt.name)
			assert.Equal(t, tt.expectedTotalNodes, result.TotalNodes,
				"Test %s: total nodes mismatch", tt.name)
			assert.Equal(t, tt.expectedSuccessNodes, result.SuccessfulNodes,
				"Test %s: successful nodes mismatch", tt.name)

			if tt.expectedFailedNodes == nil {
				assert.Empty(t, result.FailedNodes, "Test %s: failed nodes should be empty", tt.name)
			} else {
				assert.Equal(t, tt.expectedFailedNodes, result.FailedNodes,
					"Test %s: failed nodes mismatch", tt.name)
			}

			// Verify all mock expectations were met
			mockKubectl.AssertExpectations(t)
		})
	}
}

// TestLabelingService_GetCurrentState tests the current state discovery functionality
// WHY: Validates that current state discovery works correctly with proper error handling
func TestLabelingService_GetCurrentState(t *testing.T) {
	tests := []struct {
		name          string
		description   string
		nodes         []string
		mockSetupFunc func(*MockDryRunExecutor, *MockLogger)
		expectedState map[string]map[string]string
		shouldError   bool
	}{
		{
			name:        "successful_state_discovery",
			description: "Successfully discovers current state",
			nodes:       []string{"rsb2", "rsb3"},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("GetNodeLabels", mock.Anything, "rsb2").Return(true, "labels", nil)
				mockKubectl.On("GetNodeLabels", mock.Anything, "rsb3").Return(true, "labels", nil)
			},
			expectedState: map[string]map[string]string{
				"rsb2": {},
				"rsb3": {},
			},
			shouldError: false,
		},
		{
			name:        "state_discovery_failure",
			description: "Handles failure during state discovery",
			nodes:       []string{"failing-node"},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("GetNodeLabels", mock.Anything, "failing-node").Return(false, "", assert.AnError)
			},
			expectedState: nil,
			shouldError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Setup mocks and service
			mockKubectl := NewMockDryRunExecutor()
			mockLogger := NewMockLogger()

			tt.mockSetupFunc(mockKubectl, mockLogger)

			service := NewService(mockKubectl, Options{
				Logger: mockLogger,
			})

			// When: Get current state
			state, err := service.GetCurrentState(context.Background(), tt.nodes)

			// Then: Verify results
			if tt.shouldError {
				assert.Error(t, err, "Test %s: expected error but got none", tt.name)
				assert.Nil(t, state, "Test %s: state should be nil on error", tt.name)
			} else {
				assert.NoError(t, err, "Test %s: unexpected error: %v", tt.name, err)
				assert.Equal(t, tt.expectedState, state, "Test %s: state mismatch", tt.name)
			}

			// Verify all mock expectations were met
			mockKubectl.AssertExpectations(t)
		})
	}
}
