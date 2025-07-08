// Package labeler provides unit tests for the labeling service
// WHY: This test suite validates core business logic without external dependencies
package labeler

import (
	"context"
	"fmt"
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

// TestLabelingService_RemoveLabels tests label removal operations
// WHY: Validates that labels are correctly removed with proper error handling
func TestLabelingService_RemoveLabels(t *testing.T) {
	tests := []struct {
		name            string
		description     string
		nodeConfig      map[string]config.NodeRole
		mockSetupFunc   func(*MockDryRunExecutor, *MockLogger)
		expectedSuccess int
		expectedFailure int
		shouldError     bool
	}{
		{
			name:        "successful_label_removal",
			description: "Successfully removes labels from nodes",
			nodeConfig: map[string]config.NodeRole{
				"cleanup": {
					Nodes:       []string{"rsb2"},
					Labels:      map[string]string{"temp.io/label": "remove-me"},
					Description: "Temporary labels to remove",
				},
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", false).Return()
				mockKubectl.On("GetNode", mock.Anything, "rsb2").Return(true, "node/rsb2", nil)
				mockKubectl.On("UnlabelNode", mock.Anything, "rsb2", "temp.io/label").
					Return(true, "label removed", nil)
				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectedSuccess: 1,
			expectedFailure: 0,
			shouldError:     false,
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
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NotNil(t, result)
			assert.Equal(t, tt.expectedSuccess, result.SuccessfulNodes)
			assert.Equal(t, tt.expectedFailure, len(result.FailedNodes))

			mockKubectl.AssertExpectations(t)
		})
	}
}

// TestLabelingService_DryRunMode tests dry-run functionality
// WHY: Validates that dry-run mode works correctly without making actual changes
func TestLabelingService_DryRunMode(t *testing.T) {
	// Given: Setup for dry-run test
	mockKubectl := NewMockDryRunExecutor()
	mockLogger := NewMockLogger()

	// Mock dry-run mode setup
	mockKubectl.On("SetDryRun", true).Return()
	mockKubectl.On("GetNode", mock.Anything, "rsb2").Return(true, "node/rsb2", nil)
	mockKubectl.On("LabelNode", mock.Anything, "rsb2", "test.io/dry-run=true", true).
		Return(true, "DRY RUN: would label node", nil)
	mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()

	service := NewService(mockKubectl, Options{
		DryRun:        true,
		ValidateNodes: true,
		Logger:        mockLogger,
	})

	testConfig := &config.NodeLabelConf{
		APIVersion: "openstack.kictl.icycloud.io/v1",
		Kind:       "NodeLabelConf",
		Metadata: config.Metadata{
			Name: "dry-run-test",
		},
		Spec: config.NodeLabelSpec{
			NodeRoles: map[string]config.NodeRole{
				"test": {
					Nodes:  []string{"rsb2"},
					Labels: map[string]string{"test.io/dry-run": "true"},
				},
			},
		},
	}

	// When: Apply labels in dry-run mode
	result, err := service.ApplyLabels(context.Background(), testConfig)

	// Then: Verify dry-run behavior
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.TotalNodes)
	assert.Equal(t, 1, result.SuccessfulNodes)

	mockKubectl.AssertExpectations(t)
}

// TestLabelingService_VerifyLabels tests label verification functionality
// WHY: Validates that label verification correctly identifies applied and missing labels
func TestLabelingService_VerifyLabels(t *testing.T) {
	tests := []struct {
		name             string
		description      string
		nodeConfig       map[string]config.NodeRole
		mockSetupFunc    func(*MockDryRunExecutor, *MockLogger)
		expectedTotal    int
		expectedSuccess  int
		expectedFailed   []string
		expectedVerified map[string][]string
		shouldError      bool
	}{
		{
			name:        "all_labels_verified_successfully",
			description: "All expected labels are found on all nodes",
			nodeConfig: map[string]config.NodeRole{
				"control": {
					Nodes:  []string{"rsb2"},
					Labels: map[string]string{"role": "master", "zone": "us-west"},
				},
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				// Mock successful label verification
				mockKubectl.On("GetNodeLabels", mock.Anything, "rsb2").Return(
					true, "role=master,zone=us-west,other=value", nil)

				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectedTotal:   1,
			expectedSuccess: 1,
			expectedFailed:  nil,
			expectedVerified: map[string][]string{
				"rsb2": {"role=master", "zone=us-west"},
			},
			shouldError: false,
		},
		{
			name:        "partial_labels_missing",
			description: "Some labels are missing from nodes",
			nodeConfig: map[string]config.NodeRole{
				"worker": {
					Nodes:  []string{"worker1"},
					Labels: map[string]string{"role": "worker", "missing": "label"},
				},
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				// Mock partial label verification (only 'role' found, 'missing' not found)
				mockKubectl.On("GetNodeLabels", mock.Anything, "worker1").Return(
					true, "role=worker,other=existing", nil)

				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectedTotal:   1,
			expectedSuccess: 0,
			expectedFailed:  []string{"worker1"},
			expectedVerified: map[string][]string{
				"worker1": {"role=worker"},
			},
			shouldError: false,
		},
		{
			name:        "node_labels_retrieval_failure",
			description: "Failed to retrieve labels from node",
			nodeConfig: map[string]config.NodeRole{
				"failed": {
					Nodes:  []string{"bad-node"},
					Labels: map[string]string{"test": "value"},
				},
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				// Mock failure to get node labels
				mockKubectl.On("GetNodeLabels", mock.Anything, "bad-node").Return(
					false, "", fmt.Errorf("node not accessible"))

				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Error", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectedTotal:    1,
			expectedSuccess:  0,
			expectedFailed:   []string{"bad-node"},
			expectedVerified: map[string][]string{},
			shouldError:      false,
		},
		{
			name:        "multiple_nodes_mixed_verification",
			description: "Mixed success and failure across multiple nodes",
			nodeConfig: map[string]config.NodeRole{
				"mixed": {
					Nodes:  []string{"good-node", "partial-node", "bad-node"},
					Labels: map[string]string{"env": "prod", "tier": "backend"},
				},
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				// Good node: all labels present
				mockKubectl.On("GetNodeLabels", mock.Anything, "good-node").Return(
					true, "env=prod,tier=backend,extra=label", nil)

				// Partial node: only one label present
				mockKubectl.On("GetNodeLabels", mock.Anything, "partial-node").Return(
					true, "env=prod,different=value", nil)

				// Bad node: retrieval fails
				mockKubectl.On("GetNodeLabels", mock.Anything, "bad-node").Return(
					false, "", fmt.Errorf("network error"))

				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Error", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectedTotal:   3,
			expectedSuccess: 1,
			expectedFailed:  []string{"partial-node", "bad-node"},
			expectedVerified: map[string][]string{
				"good-node":    {"env=prod", "tier=backend"},
				"partial-node": {"env=prod"},
			},
			shouldError: false,
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

			testConfig := &config.NodeLabelConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeLabelConf",
				Metadata:   config.Metadata{Name: "verify-test"},
				Spec:       config.NodeLabelSpec{NodeRoles: tt.nodeConfig},
			}

			// When: Verify labels
			result, err := service.VerifyLabels(context.Background(), testConfig)

			// Then: Validate results
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NotNil(t, result)
			assert.Equal(t, tt.expectedTotal, result.TotalNodes)
			assert.Equal(t, tt.expectedSuccess, result.SuccessfulNodes)

			if tt.expectedFailed == nil {
				assert.Empty(t, result.FailedNodes)
			} else {
				assert.ElementsMatch(t, tt.expectedFailed, result.FailedNodes)
			}

			for node, expectedLabels := range tt.expectedVerified {
				actualLabels := result.AppliedLabels[node]
				assert.ElementsMatch(t, expectedLabels, actualLabels,
					"Labels mismatch for node %s", node)
			}

			mockKubectl.AssertExpectations(t)
		})
	}
}

// TestLabelingService_GetCurrentState tests current state discovery
// WHY: Validates that current labeling state can be accurately discovered
func TestLabelingService_GetCurrentState(t *testing.T) {
	tests := []struct {
		name          string
		description   string
		nodes         []string
		mockSetupFunc func(*MockDryRunExecutor, *MockLogger)
		expectedState map[string]map[string]string
		shouldError   bool
		errorContains string
	}{
		{
			name:        "successful_state_discovery",
			description: "Successfully discovers current state for all nodes",
			nodes:       []string{"node1", "node2"},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("GetNodeLabels", mock.Anything, "node1").Return(
					true, "", nil)
				mockKubectl.On("GetNodeLabels", mock.Anything, "node2").Return(
					true, "", nil)
			},
			expectedState: map[string]map[string]string{
				"node1": {},
				"node2": {},
			},
			shouldError: false,
		},
		{
			name:        "node_access_failure",
			description: "Fails when unable to access node",
			nodes:       []string{"accessible", "inaccessible"},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("GetNodeLabels", mock.Anything, "accessible").Return(
					true, "", nil)
				mockKubectl.On("GetNodeLabels", mock.Anything, "inaccessible").Return(
					false, "", fmt.Errorf("node inaccessible: connection refused"))
			},
			expectedState: nil,
			shouldError:   true,
			errorContains: "failed to get labels for node inaccessible",
		},
		{
			name:        "empty_node_list",
			description: "Handles empty node list gracefully",
			nodes:       []string{},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				// No mock setup needed
			},
			expectedState: map[string]map[string]string{},
			shouldError:   false,
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

			// Then: Validate results
			if tt.shouldError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, state)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedState, state)
			}

			mockKubectl.AssertExpectations(t)
		})
	}
}

// TestLabelingService_ErrorHandling tests comprehensive error scenarios
// WHY: Validates robust error handling across all service methods
func TestLabelingService_ErrorHandling(t *testing.T) {
	t.Run("apply_with_kubectl_errors", func(t *testing.T) {
		// Given: Service with kubectl that returns errors
		mockKubectl := NewMockDryRunExecutor()
		mockLogger := NewMockLogger()

		mockKubectl.On("SetDryRun", false).Return()
		mockKubectl.On("GetNode", mock.Anything, "error-node").Return(
			false, "", fmt.Errorf("kubectl error: cluster unreachable"))

		mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
		mockLogger.On("Error", mock.AnythingOfType("string")).Return().Maybe()
		mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()

		service := NewService(mockKubectl, Options{
			ValidateNodes: true,
			Logger:        mockLogger,
		})

		testConfig := &config.NodeLabelConf{
			APIVersion: "openstack.kictl.icycloud.io/v1",
			Kind:       "NodeLabelConf",
			Metadata:   config.Metadata{Name: "error-test"},
			Spec: config.NodeLabelSpec{
				NodeRoles: map[string]config.NodeRole{
					"test": {
						Nodes:  []string{"error-node"},
						Labels: map[string]string{"test": "label"},
					},
				},
			},
		}

		// When: Apply labels
		result, err := service.ApplyLabels(context.Background(), testConfig)

		// Then: Should handle error gracefully
		assert.NoError(t, err, "Service should not return error for individual node failures")
		assert.NotNil(t, result)
		assert.Equal(t, 1, result.TotalNodes)
		assert.Equal(t, 0, result.SuccessfulNodes)
		assert.Contains(t, result.FailedNodes, "error-node")
		assert.Len(t, result.Errors, 1)

		mockKubectl.AssertExpectations(t)
	})

	t.Run("remove_with_unlabel_errors", func(t *testing.T) {
		// Given: Service with kubectl that fails on unlabel
		mockKubectl := NewMockDryRunExecutor()
		mockLogger := NewMockLogger()

		mockKubectl.On("SetDryRun", false).Return()
		mockKubectl.On("GetNode", mock.Anything, "problem-node").Return(true, "node exists", nil)
		mockKubectl.On("UnlabelNode", mock.Anything, "problem-node", "problematic-label").Return(
			false, "", fmt.Errorf("permission denied"))

		mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
		mockLogger.On("Error", mock.AnythingOfType("string")).Return().Maybe()
		mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()

		service := NewService(mockKubectl, Options{
			ValidateNodes: true,
			Logger:        mockLogger,
		})

		testConfig := &config.NodeLabelConf{
			APIVersion: "openstack.kictl.icycloud.io/v1",
			Kind:       "NodeLabelConf",
			Metadata:   config.Metadata{Name: "remove-error-test"},
			Spec: config.NodeLabelSpec{
				NodeRoles: map[string]config.NodeRole{
					"test": {
						Nodes:  []string{"problem-node"},
						Labels: map[string]string{"problematic-label": "value"},
					},
				},
			},
		}

		// When: Remove labels
		result, err := service.RemoveLabels(context.Background(), testConfig)

		// Then: Should handle error gracefully
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, result.TotalNodes)
		assert.Equal(t, 0, result.SuccessfulNodes)
		assert.Len(t, result.Errors, 1)

		mockKubectl.AssertExpectations(t)
	})
}

// TestLabelingService_ComplexScenarios tests real-world complex scenarios
// WHY: Validates service behavior in production-like scenarios with multiple roles and operations
func TestLabelingService_ComplexScenarios(t *testing.T) {
	t.Run("multi_role_large_cluster", func(t *testing.T) {
		// Given: Large cluster with multiple roles
		mockKubectl := NewMockDryRunExecutor()
		mockLogger := NewMockLogger()

		mockKubectl.On("SetDryRun", false).Return()

		// Setup control plane nodes
		for _, node := range []string{"master1", "master2", "master3"} {
			mockKubectl.On("GetNode", mock.Anything, node).Return(true, "node exists", nil)
			mockKubectl.On("LabelNode", mock.Anything, node, "role=master", true).Return(
				true, "labeled", nil)
			mockKubectl.On("LabelNode", mock.Anything, node, "tier=control", true).Return(
				true, "labeled", nil)
		}

		// Setup worker nodes
		for _, node := range []string{"worker1", "worker2", "worker3", "worker4"} {
			mockKubectl.On("GetNode", mock.Anything, node).Return(true, "node exists", nil)
			mockKubectl.On("LabelNode", mock.Anything, node, "role=worker", true).Return(
				true, "labeled", nil)
			mockKubectl.On("LabelNode", mock.Anything, node, "tier=compute", true).Return(
				true, "labeled", nil)
		}

		mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
		mockLogger.On("Error", mock.AnythingOfType("string")).Return().Maybe()
		mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()

		service := NewService(mockKubectl, Options{
			ValidateNodes: true,
			Logger:        mockLogger,
		})

		testConfig := &config.NodeLabelConf{
			APIVersion: "openstack.kictl.icycloud.io/v1",
			Kind:       "NodeLabelConf",
			Metadata:   config.Metadata{Name: "large-cluster"},
			Spec: config.NodeLabelSpec{
				NodeRoles: map[string]config.NodeRole{
					"control_plane": {
						Nodes:       []string{"master1", "master2", "master3"},
						Labels:      map[string]string{"role": "master", "tier": "control"},
						Description: "Kubernetes control plane nodes",
					},
					"worker_nodes": {
						Nodes:       []string{"worker1", "worker2", "worker3", "worker4"},
						Labels:      map[string]string{"role": "worker", "tier": "compute"},
						Description: "Kubernetes worker nodes",
					},
				},
			},
		}

		// When: Apply labels to entire cluster
		result, err := service.ApplyLabels(context.Background(), testConfig)

		// Then: All operations should succeed
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 7, result.TotalNodes)
		assert.Equal(t, 7, result.SuccessfulNodes)
		assert.Empty(t, result.FailedNodes)
		assert.Empty(t, result.Errors)

		mockKubectl.AssertExpectations(t)
	})

	t.Run("mixed_success_failure_large_scale", func(t *testing.T) {
		// Given: Large-scale operation with mixed results
		mockKubectl := NewMockDryRunExecutor()
		mockLogger := NewMockLogger()

		mockKubectl.On("SetDryRun", false).Return()

		// Successful nodes
		successNodes := []string{"good1", "good2", "good3"}
		for _, node := range successNodes {
			mockKubectl.On("GetNode", mock.Anything, node).Return(true, "exists", nil)
			mockKubectl.On("LabelNode", mock.Anything, node, "env=production", true).Return(
				true, "labeled", nil)
		}

		// Failed nodes (various failure types)
		mockKubectl.On("GetNode", mock.Anything, "missing-node").Return(false, "", nil)

		mockKubectl.On("GetNode", mock.Anything, "kubectl-error").Return(true, "exists", nil)
		mockKubectl.On("LabelNode", mock.Anything, "kubectl-error", "env=production", true).Return(
			false, "", fmt.Errorf("permission denied"))

		mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
		mockLogger.On("Error", mock.AnythingOfType("string")).Return().Maybe()
		mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()

		service := NewService(mockKubectl, Options{
			ValidateNodes: true,
			Logger:        mockLogger,
		})

		testConfig := &config.NodeLabelConf{
			APIVersion: "openstack.kictl.icycloud.io/v1",
			Kind:       "NodeLabelConf",
			Metadata:   config.Metadata{Name: "mixed-results"},
			Spec: config.NodeLabelSpec{
				NodeRoles: map[string]config.NodeRole{
					"production": {
						Nodes:  []string{"good1", "good2", "good3", "missing-node", "kubectl-error"},
						Labels: map[string]string{"env": "production"},
					},
				},
			},
		}

		// When: Apply labels
		result, err := service.ApplyLabels(context.Background(), testConfig)

		// Then: Should have mixed results
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 5, result.TotalNodes)
		assert.Equal(t, 3, result.SuccessfulNodes)
		assert.ElementsMatch(t, []string{"missing-node", "kubectl-error"}, result.FailedNodes)
		assert.Len(t, result.Errors, 1) // Only kubectl-error should have an actual error

		mockKubectl.AssertExpectations(t)
	})
}

// TestLabelingService_Configuration tests service configuration and options
// WHY: Validates that service configuration options work correctly
func TestLabelingService_Configuration(t *testing.T) {
	t.Run("dry_run_mode_configuration", func(t *testing.T) {
		// Given: Service configured in dry-run mode
		mockKubectl := NewMockDryRunExecutor()
		mockLogger := NewMockLogger()

		mockKubectl.On("SetDryRun", true).Return()
		mockKubectl.On("GetNode", mock.Anything, "test-node").Return(true, "exists", nil)
		mockKubectl.On("LabelNode", mock.Anything, "test-node", "test=label", true).Return(
			true, "would label", nil)

		mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()

		service := NewService(mockKubectl, Options{
			DryRun:        true,
			ValidateNodes: true,
			Logger:        mockLogger,
		})

		testConfig := &config.NodeLabelConf{
			APIVersion: "openstack.kictl.icycloud.io/v1",
			Kind:       "NodeLabelConf",
			Metadata:   config.Metadata{Name: "dry-run-test"},
			Spec: config.NodeLabelSpec{
				NodeRoles: map[string]config.NodeRole{
					"test": {
						Nodes:  []string{"test-node"},
						Labels: map[string]string{"test": "label"},
					},
				},
			},
		}

		// When: Apply labels in dry-run mode
		result, err := service.ApplyLabels(context.Background(), testConfig)

		// Then: Should set dry-run mode correctly
		assert.NoError(t, err)
		assert.Equal(t, 1, result.SuccessfulNodes)

		mockKubectl.AssertExpectations(t)
	})

	t.Run("validation_disabled_configuration", func(t *testing.T) {
		// Given: Service with node validation disabled
		mockKubectl := NewMockDryRunExecutor()
		mockLogger := NewMockLogger()

		mockKubectl.On("SetDryRun", false).Return()
		// Note: No GetNode calls should be made when validation is disabled
		mockKubectl.On("LabelNode", mock.Anything, "unvalidated-node", "test=label", true).Return(
			true, "labeled", nil)

		mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()

		service := NewService(mockKubectl, Options{
			DryRun:        false,
			ValidateNodes: false, // Disabled
			Logger:        mockLogger,
		})

		testConfig := &config.NodeLabelConf{
			APIVersion: "openstack.kictl.icycloud.io/v1",
			Kind:       "NodeLabelConf",
			Metadata:   config.Metadata{Name: "no-validation"},
			Spec: config.NodeLabelSpec{
				NodeRoles: map[string]config.NodeRole{
					"test": {
						Nodes:  []string{"unvalidated-node"},
						Labels: map[string]string{"test": "label"},
					},
				},
			},
		}

		// When: Apply labels without validation
		result, err := service.ApplyLabels(context.Background(), testConfig)

		// Then: Should skip validation
		assert.NoError(t, err)
		assert.Equal(t, 1, result.SuccessfulNodes)

		mockKubectl.AssertExpectations(t)
	})
}

// TestNewService tests service creation
// WHY: Validates that service factory function works correctly
func TestNewService(t *testing.T) {
	t.Run("service_creation", func(t *testing.T) {
		// Given: Mock dependencies
		mockKubectl := NewMockDryRunExecutor()
		mockLogger := NewMockLogger()

		options := Options{
			DryRun:        true,
			Verbose:       true,
			ValidateNodes: true,
			Logger:        mockLogger,
		}

		// When: Create service
		service := NewService(mockKubectl, options)

		// Then: Should create valid service
		assert.NotNil(t, service)
		assert.Implements(t, (*Service)(nil), service)

		// Should be able to cast to concrete type for testing
		concreteService, ok := service.(*LabelingService)
		assert.True(t, ok)
		assert.NotNil(t, concreteService)
	})
}

// TestOperationResults tests the results structure
// WHY: Validates that operation results are properly structured and accessible
func TestOperationResults(t *testing.T) {
	t.Run("results_structure", func(t *testing.T) {
		// Given: Create operation results
		results := &OperationResults{
			TotalNodes:      5,
			SuccessfulNodes: 3,
			FailedNodes:     []string{"node1", "node2"},
			AppliedLabels: map[string][]string{
				"node3": {"label1=value1", "label2=value2"},
				"node4": {"label3=value3"},
			},
			Errors: []error{fmt.Errorf("test error")},
		}

		// When: Access fields
		// Then: All fields should be accessible and correct
		assert.Equal(t, 5, results.TotalNodes)
		assert.Equal(t, 3, results.SuccessfulNodes)
		assert.ElementsMatch(t, []string{"node1", "node2"}, results.FailedNodes)
		assert.Len(t, results.AppliedLabels, 2)
		assert.Len(t, results.Errors, 1)

		// Test applied labels structure
		assert.ElementsMatch(t, []string{"label1=value1", "label2=value2"}, results.AppliedLabels["node3"])
		assert.ElementsMatch(t, []string{"label3=value3"}, results.AppliedLabels["node4"])
	})
}
