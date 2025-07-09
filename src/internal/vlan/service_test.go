// Package vlan provides tests for the VLAN service
package vlan

import (
	"context"
	"testing"

	"k8ostack-ictl/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestNewService tests the creation of a new VLAN service
func TestNewService(t *testing.T) {
	mockKubectl := NewMockDryRunExecutor()
	mockLogger := NewMockLogger()

	options := Options{
		DryRun:               true,
		Verbose:              true,
		ValidateConnectivity: true,
		PersistentConfig:     false,
		DefaultInterface:     "eth0",
		Logger:               mockLogger,
	}

	service := NewService(mockKubectl, options)

	assert.NotNil(t, service)
	assert.IsType(t, &VLANService{}, service)

	// Type assertion to access internal fields for testing
	vlanService := service.(*VLANService)
	assert.Equal(t, mockKubectl, vlanService.kubectl)
	assert.Equal(t, options, vlanService.options)
}

// TestVLANService_ConfigureVLANs tests VLAN configuration
func TestVLANService_ConfigureVLANs(t *testing.T) {
	tests := []struct {
		name                 string
		description          string
		vlanConfig           *config.NodeVLANConf
		mockSetupFunc        func(*MockDryRunExecutor, *MockLogger)
		expectedTotalNodes   int
		expectedSuccessNodes int
		expectedFailedNodes  []string
		shouldError          bool
	}{
		{
			name:        "successful_single_vlan_configuration",
			description: "Successfully configures one VLAN on one node",
			vlanConfig: &config.NodeVLANConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeVLANConf",
				Metadata: config.Metadata{
					Name: "test-vlans",
				},
				Spec: config.NodeVLANSpec{
					VLANs: map[string]config.VLANConfig{
						"management": {
							ID:        100,
							Subnet:    "192.168.100.0/24",
							Interface: "eth0",
							NodeMapping: map[string]string{
								"node1": "192.168.100.10/24",
							},
						},
					},
				},
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", false).Return()
				mockKubectl.On("GetNode", mock.Anything, "node1").Return(true, "node/node1", nil)
				mockKubectl.On("ExecNodeCommand", mock.Anything, "node1", mock.AnythingOfType("string")).
					Return(true, "VLAN configured", nil)
				// Mock the cleanup calls
				mockKubectl.On("GetPods", mock.Anything, "", "").Return(true, "", nil)
				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Debug", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Error", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectedTotalNodes:   1,
			expectedSuccessNodes: 1,
			expectedFailedNodes:  nil,
			shouldError:          false,
		},
		{
			name:        "node_not_found_failure",
			description: "Handles failure when node doesn't exist",
			vlanConfig: &config.NodeVLANConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeVLANConf",
				Metadata: config.Metadata{
					Name: "test-vlans",
				},
				Spec: config.NodeVLANSpec{
					VLANs: map[string]config.VLANConfig{
						"management": {
							ID:        100,
							Subnet:    "192.168.100.0/24",
							Interface: "eth0",
							NodeMapping: map[string]string{
								"nonexistent-node": "192.168.100.10/24",
							},
						},
					},
				},
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", false).Return()
				mockKubectl.On("GetNode", mock.Anything, "nonexistent-node").Return(false, "", nil)
				// Mock the cleanup calls
				mockKubectl.On("GetPods", mock.Anything, "", "").Return(true, "", nil)
				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Debug", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Error", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectedTotalNodes:   1,
			expectedSuccessNodes: 0,
			expectedFailedNodes:  []string{"nonexistent-node"},
			shouldError:          false,
		},
		{
			name:        "invalid_ip_address_format",
			description: "Handles invalid IP address formats",
			vlanConfig: &config.NodeVLANConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeVLANConf",
				Metadata: config.Metadata{
					Name: "test-vlans",
				},
				Spec: config.NodeVLANSpec{
					VLANs: map[string]config.VLANConfig{
						"management": {
							ID:        100,
							Subnet:    "192.168.100.0/24",
							Interface: "eth0",
							NodeMapping: map[string]string{
								"node1": "invalid-ip-format",
							},
						},
					},
				},
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", false).Return()
				mockKubectl.On("GetNode", mock.Anything, "node1").Return(true, "node/node1", nil)
				// Mock the cleanup calls
				mockKubectl.On("GetPods", mock.Anything, "", "").Return(true, "", nil)
				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Debug", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Error", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectedTotalNodes:   1,
			expectedSuccessNodes: 0,
			expectedFailedNodes:  []string{"node1"},
			shouldError:          false,
		},
		{
			name:        "empty_vlan_configuration",
			description: "Handles empty VLAN configuration",
			vlanConfig: &config.NodeVLANConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeVLANConf",
				Metadata: config.Metadata{
					Name: "empty-vlans",
				},
				Spec: config.NodeVLANSpec{
					VLANs: map[string]config.VLANConfig{},
				},
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", false).Return()
				// Mock the cleanup calls (even for empty config, cleanup is still called)
				mockKubectl.On("GetPods", mock.Anything, "", "").Return(true, "", nil)
				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Debug", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectedTotalNodes:   0,
			expectedSuccessNodes: 0,
			expectedFailedNodes:  nil,
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
				DryRun:               false,
				ValidateConnectivity: true,
				DefaultInterface:     "eth0",
				Logger:               mockLogger,
			})

			// When: Configure VLANs
			result, err := service.ConfigureVLANs(context.Background(), tt.vlanConfig)

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

// TestVLANService_RemoveVLANs tests VLAN removal
func TestVLANService_RemoveVLANs(t *testing.T) {
	t.Run("successful_vlan_removal", func(t *testing.T) {
		// Given: VLAN configuration and mocks
		mockKubectl := NewMockDryRunExecutor()
		mockLogger := NewMockLogger()

		vlanConfig := &config.NodeVLANConf{
			APIVersion: "openstack.kictl.icycloud.io/v1",
			Kind:       "NodeVLANConf",
			Metadata: config.Metadata{
				Name: "test-vlans",
			},
			Spec: config.NodeVLANSpec{
				VLANs: map[string]config.VLANConfig{
					"management": {
						ID:        100,
						Subnet:    "192.168.100.0/24",
						Interface: "eth0",
						NodeMapping: map[string]string{
							"node1": "192.168.100.10/24",
						},
					},
				},
			},
		}

		mockKubectl.On("SetDryRun", false).Return()
		mockKubectl.On("GetNode", mock.Anything, "node1").Return(true, "node/node1", nil)
		mockKubectl.On("ExecNodeCommand", mock.Anything, "node1", mock.AnythingOfType("string")).
			Return(true, "VLAN interface removed", nil)
		// Mock the cleanup calls
		mockKubectl.On("GetPods", mock.Anything, "", "").Return(true, "", nil)
		mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
		mockLogger.On("Debug", mock.AnythingOfType("string")).Return().Maybe()
		mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()

		service := NewService(mockKubectl, Options{
			DryRun:               false,
			ValidateConnectivity: true,
			DefaultInterface:     "eth0",
			Logger:               mockLogger,
		})

		// When: Remove VLANs
		result, err := service.RemoveVLANs(context.Background(), vlanConfig)

		// Then: Verify removal
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, result.TotalNodes)
		assert.Equal(t, 1, result.SuccessfulNodes)
		assert.Empty(t, result.FailedNodes)

		mockKubectl.AssertExpectations(t)
	})
}

// TestVLANService_VerifyVLANs tests VLAN verification
func TestVLANService_VerifyVLANs(t *testing.T) {
	t.Run("successful_vlan_verification", func(t *testing.T) {
		// Given: VLAN configuration and mocks
		mockKubectl := NewMockDryRunExecutor()
		mockLogger := NewMockLogger()

		vlanConfig := &config.NodeVLANConf{
			APIVersion: "openstack.kictl.icycloud.io/v1",
			Kind:       "NodeVLANConf",
			Metadata: config.Metadata{
				Name: "test-vlans",
			},
			Spec: config.NodeVLANSpec{
				VLANs: map[string]config.VLANConfig{
					"management": {
						ID:        100,
						Subnet:    "192.168.100.0/24",
						Interface: "eth0",
						NodeMapping: map[string]string{
							"node1": "192.168.100.10/24",
						},
					},
				},
			},
		}

		mockKubectl.On("SetDryRun", true).Return()
		mockKubectl.On("GetNode", mock.Anything, "node1").Return(true, "node/node1", nil)
		mockKubectl.On("ExecNodeCommand", mock.Anything, "node1", mock.AnythingOfType("string")).
			Return(true, "eth0.100: interface exists", nil)
		// Mock the cleanup calls
		mockKubectl.On("GetPods", mock.Anything, "", "").Return(true, "", nil)
		mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
		mockLogger.On("Debug", mock.AnythingOfType("string")).Return().Maybe()
		mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()

		service := NewService(mockKubectl, Options{
			DryRun:               true,
			ValidateConnectivity: true,
			DefaultInterface:     "eth0",
			Logger:               mockLogger,
		})

		// When: Verify VLANs
		result, err := service.VerifyVLANs(context.Background(), vlanConfig)

		// Then: Verify verification results
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, result.TotalNodes)
		assert.Equal(t, 1, result.SuccessfulNodes)
		assert.Empty(t, result.FailedNodes)

		mockKubectl.AssertExpectations(t)
	})
}

// TestVLANService_GetCurrentState tests current state discovery
func TestVLANService_GetCurrentState(t *testing.T) {
	t.Run("successful_state_discovery", func(t *testing.T) {
		// Given: Service and node list
		mockKubectl := NewMockDryRunExecutor()
		mockLogger := NewMockLogger()

		nodes := []string{"node1", "node2"}

		mockKubectl.On("ExecNodeCommand", mock.Anything, "node1", mock.AnythingOfType("string")).
			Return(true, "eth0.100\neth1.200", nil)
		mockKubectl.On("ExecNodeCommand", mock.Anything, "node2", mock.AnythingOfType("string")).
			Return(true, "eth0.300", nil)

		service := NewService(mockKubectl, Options{
			Logger: mockLogger,
		})

		// When: Get current state
		state, err := service.GetCurrentState(context.Background(), nodes)

		// Then: Verify state discovery
		assert.NoError(t, err)
		assert.NotNil(t, state)
		assert.Len(t, state, 2)
		assert.Contains(t, state, "node1")
		assert.Contains(t, state, "node2")

		mockKubectl.AssertExpectations(t)
	})

	t.Run("failure_during_discovery", func(t *testing.T) {
		// Given: Service and node list with failure
		mockKubectl := NewMockDryRunExecutor()
		mockLogger := NewMockLogger()

		nodes := []string{"failing-node"}

		mockKubectl.On("ExecNodeCommand", mock.Anything, "failing-node", mock.AnythingOfType("string")).
			Return(false, "", assert.AnError)

		service := NewService(mockKubectl, Options{
			Logger: mockLogger,
		})

		// When: Get current state
		state, err := service.GetCurrentState(context.Background(), nodes)

		// Then: Verify error handling
		assert.Error(t, err)
		assert.Nil(t, state)
		assert.Contains(t, err.Error(), "failed to discover VLANs on node failing-node")

		mockKubectl.AssertExpectations(t)
	})
}

// TestVLANService_DryRunMode tests dry-run functionality
func TestVLANService_DryRunMode(t *testing.T) {
	t.Run("dry_run_enabled", func(t *testing.T) {
		// Given: Service in dry-run mode
		mockKubectl := NewMockDryRunExecutor()
		mockLogger := NewMockLogger()

		vlanConfig := &config.NodeVLANConf{
			APIVersion: "openstack.kictl.icycloud.io/v1",
			Kind:       "NodeVLANConf",
			Metadata: config.Metadata{
				Name: "dry-run-test",
			},
			Spec: config.NodeVLANSpec{
				VLANs: map[string]config.VLANConfig{
					"management": {
						ID:        100,
						Subnet:    "192.168.100.0/24",
						Interface: "eth0",
						NodeMapping: map[string]string{
							"node1": "192.168.100.10/24",
						},
					},
				},
			},
		}

		mockKubectl.On("SetDryRun", true).Return()
		mockKubectl.On("GetNode", mock.Anything, "node1").Return(true, "node/node1", nil)
		mockKubectl.On("ExecNodeCommand", mock.Anything, "node1", mock.AnythingOfType("string")).
			Return(true, "DRY RUN: Would configure VLAN", nil)
		// Mock the cleanup calls
		mockKubectl.On("GetPods", mock.Anything, "", "").Return(true, "", nil)
		mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
		mockLogger.On("Debug", mock.AnythingOfType("string")).Return().Maybe()
		mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()

		service := NewService(mockKubectl, Options{
			DryRun:               true,
			ValidateConnectivity: true,
			DefaultInterface:     "eth0",
			Logger:               mockLogger,
		})

		// When: Configure VLANs in dry-run mode
		result, err := service.ConfigureVLANs(context.Background(), vlanConfig)

		// Then: Verify dry-run execution
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, result.TotalNodes)
		assert.Equal(t, 1, result.SuccessfulNodes)

		mockKubectl.AssertExpectations(t)
	})
}

// TestVLANService_Options tests various service options
func TestVLANService_Options(t *testing.T) {
	tests := []struct {
		name        string
		options     Options
		description string
	}{
		{
			name: "minimal_options",
			options: Options{
				Logger: NewMockLogger(),
			},
			description: "Service with minimal options",
		},
		{
			name: "full_options",
			options: Options{
				DryRun:               true,
				Verbose:              true,
				ValidateConnectivity: true,
				PersistentConfig:     true,
				DefaultInterface:     "eth1",
				Logger:               NewMockLogger(),
			},
			description: "Service with all options enabled",
		},
		{
			name: "custom_interface",
			options: Options{
				DefaultInterface: "ens192",
				Logger:           NewMockLogger(),
			},
			description: "Service with custom default interface",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockKubectl := NewMockDryRunExecutor()

			// When: Create service with options
			service := NewService(mockKubectl, tt.options)

			// Then: Verify service creation
			assert.NotNil(t, service, "Service should be created successfully")

			// Type assertion to verify options are set correctly
			vlanService := service.(*VLANService)
			assert.Equal(t, tt.options, vlanService.options, "Options should match")
		})
	}
}

// TestVLANInterfaceInfo tests the VLANInterfaceInfo struct
func TestVLANInterfaceInfo(t *testing.T) {
	t.Run("vlan_interface_info_creation", func(t *testing.T) {
		// Given: VLAN interface information
		vlanInfo := VLANInterfaceInfo{
			VLANName:      "management",
			VLANId:        100,
			Interface:     "eth0.100",
			IPAddress:     "192.168.100.10/24",
			PhysInterface: "eth0",
			Subnet:        "192.168.100.0/24",
		}

		// Then: Verify all fields are set correctly
		assert.Equal(t, "management", vlanInfo.VLANName)
		assert.Equal(t, 100, vlanInfo.VLANId)
		assert.Equal(t, "eth0.100", vlanInfo.Interface)
		assert.Equal(t, "192.168.100.10/24", vlanInfo.IPAddress)
		assert.Equal(t, "eth0", vlanInfo.PhysInterface)
		assert.Equal(t, "192.168.100.0/24", vlanInfo.Subnet)
	})
}

// TestOperationResults tests the OperationResults struct
func TestOperationResults(t *testing.T) {
	t.Run("operation_results_creation", func(t *testing.T) {
		// Given: Operation results
		results := &OperationResults{
			TotalNodes:      3,
			SuccessfulNodes: 2,
			FailedNodes:     []string{"node3"},
			ConfiguredVLANs: map[string][]VLANInterfaceInfo{
				"node1": {
					{VLANName: "management", VLANId: 100, Interface: "eth0.100"},
				},
				"node2": {
					{VLANName: "storage", VLANId: 200, Interface: "eth1.200"},
				},
			},
			Errors: []error{assert.AnError},
		}

		// Then: Verify all fields are set correctly
		assert.Equal(t, 3, results.TotalNodes)
		assert.Equal(t, 2, results.SuccessfulNodes)
		assert.Equal(t, []string{"node3"}, results.FailedNodes)
		assert.Len(t, results.ConfiguredVLANs, 2)
		assert.Contains(t, results.ConfiguredVLANs, "node1")
		assert.Contains(t, results.ConfiguredVLANs, "node2")
		assert.Len(t, results.Errors, 1)
	})
}
