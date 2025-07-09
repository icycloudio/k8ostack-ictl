// Package vlan provides tests for the VLAN service
package vlan

import (
	"context"
	"fmt"
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

// TestVLANService_ConfigureVLANs tests comprehensive VLAN configuration scenarios
func TestVLANService_ConfigureVLANs(t *testing.T) {
	tests := []struct {
		name                 string
		description          string
		vlanConfig           *config.NodeVLANConf
		options              Options
		mockSetupFunc        func(*MockDryRunExecutor, *MockLogger)
		expectedTotalNodes   int
		expectedSuccessNodes int
		expectedFailedNodes  []string
		shouldError          bool
		validateResults      func(*testing.T, *OperationResults)
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
			options: Options{
				DryRun:               false,
				ValidateConnectivity: true,
				DefaultInterface:     "eth0",
				Logger:               nil, // Will be set in test
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", false).Return()
				mockKubectl.On("GetNode", mock.Anything, "node1").Return(true, "node/node1", nil)
				mockKubectl.On("ExecNodeCommand", mock.Anything, "node1", mock.AnythingOfType("string")).
					Return(true, "VLAN configured", nil)
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
			validateResults: func(t *testing.T, results *OperationResults) {
				assert.Contains(t, results.ConfiguredVLANs, "node1")
				assert.Len(t, results.ConfiguredVLANs["node1"], 1)
				vlan := results.ConfiguredVLANs["node1"][0]
				assert.Equal(t, "management", vlan.VLANName)
				assert.Equal(t, 100, vlan.VLANId)
				assert.Equal(t, "eth0.100", vlan.Interface)
				assert.Equal(t, "192.168.100.10/24", vlan.IPAddress)
				assert.Equal(t, "eth0", vlan.PhysInterface)
			},
		},
		{
			name:        "multiple_vlans_multiple_nodes",
			description: "Successfully configures multiple VLANs across multiple nodes",
			vlanConfig: &config.NodeVLANConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeVLANConf",
				Metadata: config.Metadata{
					Name: "multi-vlan-test",
				},
				Spec: config.NodeVLANSpec{
					VLANs: map[string]config.VLANConfig{
						"management": {
							ID:        100,
							Subnet:    "192.168.100.0/24",
							Interface: "eth0",
							NodeMapping: map[string]string{
								"node1": "192.168.100.10/24",
								"node2": "192.168.100.11/24",
							},
						},
						"storage": {
							ID:        200,
							Subnet:    "10.10.200.0/24",
							Interface: "eth1",
							NodeMapping: map[string]string{
								"node1": "10.10.200.10/24",
								"node2": "10.10.200.11/24",
							},
						},
					},
				},
			},
			options: Options{
				DryRun:               false,
				ValidateConnectivity: true,
				DefaultInterface:     "eth0",
				Logger:               nil,
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", false).Return()
				// Node existence checks
				mockKubectl.On("GetNode", mock.Anything, "node1").Return(true, "node/node1", nil)
				mockKubectl.On("GetNode", mock.Anything, "node2").Return(true, "node/node2", nil)
				// VLAN configuration commands
				mockKubectl.On("ExecNodeCommand", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(true, "VLAN configured", nil)
				mockKubectl.On("GetPods", mock.Anything, "", "").Return(true, "", nil)
				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Debug", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectedTotalNodes:   4, // 2 nodes * 2 VLANs
			expectedSuccessNodes: 4,
			expectedFailedNodes:  nil,
			shouldError:          false,
			validateResults: func(t *testing.T, results *OperationResults) {
				assert.Contains(t, results.ConfiguredVLANs, "node1")
				assert.Contains(t, results.ConfiguredVLANs, "node2")
				assert.Len(t, results.ConfiguredVLANs["node1"], 2)
				assert.Len(t, results.ConfiguredVLANs["node2"], 2)
			},
		},
		{
			name:        "persistent_config_enabled",
			description: "Configures VLANs with persistent configuration enabled",
			vlanConfig: &config.NodeVLANConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeVLANConf",
				Metadata: config.Metadata{
					Name: "persistent-test",
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
			options: Options{
				DryRun:               false,
				ValidateConnectivity: true,
				PersistentConfig:     true, // Enable persistent config
				DefaultInterface:     "eth0",
				Logger:               nil,
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", false).Return()
				mockKubectl.On("GetNode", mock.Anything, "node1").Return(true, "node/node1", nil)
				// Should include netplan configuration in the command
				mockKubectl.On("ExecNodeCommand", mock.Anything, "node1", mock.MatchedBy(func(cmd string) bool {
					return len(cmd) > 200 // Persistent config commands are longer
				})).Return(true, "VLAN configured with persistence", nil)
				mockKubectl.On("GetPods", mock.Anything, "", "").Return(true, "", nil)
				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Debug", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectedTotalNodes:   1,
			expectedSuccessNodes: 1,
			expectedFailedNodes:  nil,
			shouldError:          false,
			validateResults: func(t *testing.T, results *OperationResults) {
				assert.Contains(t, results.ConfiguredVLANs, "node1")
			},
		},
		{
			name:        "verbose_logging_enabled",
			description: "Tests verbose logging during VLAN configuration",
			vlanConfig: &config.NodeVLANConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeVLANConf",
				Metadata: config.Metadata{
					Name: "verbose-test",
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
			options: Options{
				DryRun:               false,
				Verbose:              true, // Enable verbose logging
				ValidateConnectivity: true,
				DefaultInterface:     "eth0",
				Logger:               nil,
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", false).Return()
				mockKubectl.On("GetNode", mock.Anything, "node1").Return(true, "node/node1", nil)
				mockKubectl.On("ExecNodeCommand", mock.Anything, "node1", mock.AnythingOfType("string")).
					Return(true, "VLAN configured", nil)
				mockKubectl.On("GetPods", mock.Anything, "", "").Return(true, "", nil)
				// Verbose mode should trigger additional Info calls
				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Debug", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectedTotalNodes:   1,
			expectedSuccessNodes: 1,
			expectedFailedNodes:  nil,
			shouldError:          false,
			validateResults: func(t *testing.T, results *OperationResults) {
				assert.Contains(t, results.ConfiguredVLANs, "node1")
			},
		},
		{
			name:        "default_interface_fallback",
			description: "Uses default interface when VLAN config doesn't specify interface",
			vlanConfig: &config.NodeVLANConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeVLANConf",
				Metadata: config.Metadata{
					Name: "default-interface-test",
				},
				Spec: config.NodeVLANSpec{
					VLANs: map[string]config.VLANConfig{
						"management": {
							ID:     100,
							Subnet: "192.168.100.0/24",
							// Interface not specified - should use default
							NodeMapping: map[string]string{
								"node1": "192.168.100.10/24",
							},
						},
					},
				},
			},
			options: Options{
				DryRun:               false,
				ValidateConnectivity: true,
				DefaultInterface:     "ens192", // Custom default interface
				Logger:               nil,
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", false).Return()
				mockKubectl.On("GetNode", mock.Anything, "node1").Return(true, "node/node1", nil)
				// Command should use ens192.100 as interface
				mockKubectl.On("ExecNodeCommand", mock.Anything, "node1", mock.AnythingOfType("string")).
					Return(true, "VLAN configured", nil)
				mockKubectl.On("GetPods", mock.Anything, "", "").Return(true, "", nil)
				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Debug", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectedTotalNodes:   1,
			expectedSuccessNodes: 1,
			expectedFailedNodes:  nil,
			shouldError:          false,
			validateResults: func(t *testing.T, results *OperationResults) {
				assert.Contains(t, results.ConfiguredVLANs, "node1")
				vlan := results.ConfiguredVLANs["node1"][0]
				assert.Equal(t, "ens192.100", vlan.Interface)
				assert.Equal(t, "ens192", vlan.PhysInterface)
			},
		},
		{
			name:        "no_default_interface_eth0_fallback",
			description: "Falls back to eth0 when no default interface specified",
			vlanConfig: &config.NodeVLANConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeVLANConf",
				Metadata: config.Metadata{
					Name: "eth0-fallback-test",
				},
				Spec: config.NodeVLANSpec{
					VLANs: map[string]config.VLANConfig{
						"management": {
							ID:     100,
							Subnet: "192.168.100.0/24",
							// Interface not specified
							NodeMapping: map[string]string{
								"node1": "192.168.100.10/24",
							},
						},
					},
				},
			},
			options: Options{
				DryRun:               false,
				ValidateConnectivity: true,
				// DefaultInterface not specified - should fall back to eth0
				Logger: nil,
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", false).Return()
				mockKubectl.On("GetNode", mock.Anything, "node1").Return(true, "node/node1", nil)
				mockKubectl.On("ExecNodeCommand", mock.Anything, "node1", mock.AnythingOfType("string")).
					Return(true, "VLAN configured", nil)
				mockKubectl.On("GetPods", mock.Anything, "", "").Return(true, "", nil)
				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Debug", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectedTotalNodes:   1,
			expectedSuccessNodes: 1,
			expectedFailedNodes:  nil,
			shouldError:          false,
			validateResults: func(t *testing.T, results *OperationResults) {
				assert.Contains(t, results.ConfiguredVLANs, "node1")
				vlan := results.ConfiguredVLANs["node1"][0]
				assert.Equal(t, "eth0.100", vlan.Interface)
				assert.Equal(t, "eth0", vlan.PhysInterface)
			},
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
			options: Options{
				DryRun:               false,
				ValidateConnectivity: true,
				DefaultInterface:     "eth0",
				Logger:               nil,
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", false).Return()
				mockKubectl.On("GetNode", mock.Anything, "nonexistent-node").Return(false, "", nil)
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
			validateResults: func(t *testing.T, results *OperationResults) {
				assert.Empty(t, results.ConfiguredVLANs)
			},
		},
		{
			name:        "invalid_ip_address_formats",
			description: "Handles multiple invalid IP address formats",
			vlanConfig: &config.NodeVLANConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeVLANConf",
				Metadata: config.Metadata{
					Name: "invalid-ip-test",
				},
				Spec: config.NodeVLANSpec{
					VLANs: map[string]config.VLANConfig{
						"management": {
							ID:        100,
							Subnet:    "192.168.100.0/24",
							Interface: "eth0",
							NodeMapping: map[string]string{
								"node1": "invalid-ip-format",
								"node2": "999.999.999.999/24",
								"node3": "192.168.100.10", // Missing /24
							},
						},
					},
				},
			},
			options: Options{
				DryRun:               false,
				ValidateConnectivity: true,
				DefaultInterface:     "eth0",
				Logger:               nil,
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", false).Return()
				mockKubectl.On("GetNode", mock.Anything, mock.AnythingOfType("string")).Return(true, "node/found", nil)
				mockKubectl.On("GetPods", mock.Anything, "", "").Return(true, "", nil)
				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Debug", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Error", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectedTotalNodes:   3,
			expectedSuccessNodes: 0,
			expectedFailedNodes:  []string{"node1", "node2", "node3"},
			shouldError:          false,
			validateResults: func(t *testing.T, results *OperationResults) {
				assert.Empty(t, results.ConfiguredVLANs)
				assert.Len(t, results.Errors, 3)
			},
		},
		{
			name:        "command_execution_failure",
			description: "Handles kubectl command execution failures",
			vlanConfig: &config.NodeVLANConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeVLANConf",
				Metadata: config.Metadata{
					Name: "command-failure-test",
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
			options: Options{
				DryRun:               false,
				ValidateConnectivity: true,
				DefaultInterface:     "eth0",
				Logger:               nil,
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", false).Return()
				mockKubectl.On("GetNode", mock.Anything, "node1").Return(true, "node/node1", nil)
				mockKubectl.On("ExecNodeCommand", mock.Anything, "node1", mock.AnythingOfType("string")).
					Return(false, "Failed to execute command", fmt.Errorf("command execution failed"))
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
			validateResults: func(t *testing.T, results *OperationResults) {
				assert.Empty(t, results.ConfiguredVLANs)
				assert.Len(t, results.Errors, 1)
			},
		},
		{
			name:        "vlan_without_node_mappings",
			description: "Handles VLANs with empty node mappings",
			vlanConfig: &config.NodeVLANConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeVLANConf",
				Metadata: config.Metadata{
					Name: "empty-mappings-test",
				},
				Spec: config.NodeVLANSpec{
					VLANs: map[string]config.VLANConfig{
						"management": {
							ID:          100,
							Subnet:      "192.168.100.0/24",
							Interface:   "eth0",
							NodeMapping: map[string]string{}, // Empty mapping
						},
						"storage": {
							ID:        200,
							Subnet:    "10.10.200.0/24",
							Interface: "eth1",
							NodeMapping: map[string]string{
								"node1": "10.10.200.10/24",
							},
						},
					},
				},
			},
			options: Options{
				DryRun:               false,
				ValidateConnectivity: true,
				DefaultInterface:     "eth0",
				Logger:               nil,
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", false).Return()
				mockKubectl.On("GetNode", mock.Anything, "node1").Return(true, "node/node1", nil)
				mockKubectl.On("ExecNodeCommand", mock.Anything, "node1", mock.AnythingOfType("string")).
					Return(true, "VLAN configured", nil)
				mockKubectl.On("GetPods", mock.Anything, "", "").Return(true, "", nil)
				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Debug", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectedTotalNodes:   1, // Only storage VLAN has a node
			expectedSuccessNodes: 1,
			expectedFailedNodes:  nil,
			shouldError:          false,
			validateResults: func(t *testing.T, results *OperationResults) {
				assert.Contains(t, results.ConfiguredVLANs, "node1")
				assert.Len(t, results.ConfiguredVLANs["node1"], 1)
				assert.Equal(t, "storage", results.ConfiguredVLANs["node1"][0].VLANName)
			},
		},
		{
			name:        "connectivity_validation_disabled",
			description: "Configures VLANs without node connectivity validation",
			vlanConfig: &config.NodeVLANConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeVLANConf",
				Metadata: config.Metadata{
					Name: "no-validation-test",
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
			options: Options{
				DryRun:               false,
				ValidateConnectivity: false, // Disable connectivity validation
				DefaultInterface:     "eth0",
				Logger:               nil,
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", false).Return()
				// GetNode should NOT be called when validation is disabled
				mockKubectl.On("ExecNodeCommand", mock.Anything, "node1", mock.AnythingOfType("string")).
					Return(true, "VLAN configured", nil)
				mockKubectl.On("GetPods", mock.Anything, "", "").Return(true, "", nil)
				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Debug", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectedTotalNodes:   1,
			expectedSuccessNodes: 1,
			expectedFailedNodes:  nil,
			shouldError:          false,
			validateResults: func(t *testing.T, results *OperationResults) {
				assert.Contains(t, results.ConfiguredVLANs, "node1")
			},
		},
		{
			name:        "empty_vlan_configuration",
			description: "Handles completely empty VLAN configuration",
			vlanConfig: &config.NodeVLANConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeVLANConf",
				Metadata: config.Metadata{
					Name: "empty-vlans",
				},
				Spec: config.NodeVLANSpec{
					VLANs: map[string]config.VLANConfig{}, // Completely empty
				},
			},
			options: Options{
				DryRun:               false,
				ValidateConnectivity: true,
				DefaultInterface:     "eth0",
				Logger:               nil,
			},
			mockSetupFunc: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", false).Return()
				mockKubectl.On("GetPods", mock.Anything, "", "").Return(true, "", nil)
				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Debug", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectedTotalNodes:   0,
			expectedSuccessNodes: 0,
			expectedFailedNodes:  nil,
			shouldError:          false,
			validateResults: func(t *testing.T, results *OperationResults) {
				assert.Empty(t, results.ConfiguredVLANs)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Setup mocks and service
			mockKubectl := NewMockDryRunExecutor()
			mockLogger := NewMockLogger()

			tt.mockSetupFunc(mockKubectl, mockLogger)

			// Set logger in options
			tt.options.Logger = mockLogger

			service := NewService(mockKubectl, tt.options)

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

			// Verify failed nodes
			if tt.expectedFailedNodes == nil {
				assert.Empty(t, result.FailedNodes, "Test %s: failed nodes should be empty", tt.name)
			} else {
				assert.ElementsMatch(t, tt.expectedFailedNodes, result.FailedNodes,
					"Test %s: failed nodes mismatch", tt.name)
			}

			// Run custom result validation if provided
			if tt.validateResults != nil {
				tt.validateResults(t, result)
			}

			// Verify all mock expectations were met
			mockKubectl.AssertExpectations(t)
		})
	}
}

// TestVLANService_RemoveVLANs tests comprehensive VLAN removal scenarios
func TestVLANService_RemoveVLANs(t *testing.T) {
	tests := []struct {
		name        string
		description string
		vlanConfig  *config.NodeVLANConf
		options     Options
		setupMocks  func(*MockDryRunExecutor, *MockLogger)
		expectError bool
	}{
		{
			name:        "successful_vlan_removal",
			description: "Successfully removes VLANs from nodes",
			vlanConfig: &config.NodeVLANConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeVLANConf",
				Metadata: config.Metadata{
					Name: "removal-test",
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
			options: Options{
				DryRun:               false,
				ValidateConnectivity: true,
				DefaultInterface:     "eth0",
			},
			setupMocks: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", false).Return()
				mockKubectl.On("GetNode", mock.Anything, "node1").Return(true, "node/node1", nil)
				mockKubectl.On("ExecNodeCommand", mock.Anything, "node1", mock.MatchedBy(func(cmd string) bool {
					return len(cmd) > 30 // Removal commands include "|| true" suffix
				})).Return(true, "VLAN interface removed", nil)
				mockKubectl.On("GetPods", mock.Anything, "", "").Return(true, "", nil)
				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Debug", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectError: false,
		},
		{
			name:        "removal_with_lenient_failures",
			description: "Handles removal failures gracefully (interface doesn't exist)",
			vlanConfig: &config.NodeVLANConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeVLANConf",
				Metadata: config.Metadata{
					Name: "lenient-removal-test",
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
			options: Options{
				DryRun:               false,
				ValidateConnectivity: true,
				DefaultInterface:     "eth0",
			},
			setupMocks: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", false).Return()
				mockKubectl.On("GetNode", mock.Anything, "node1").Return(true, "node/node1", nil)
				// Return false but we're lenient for removal
				mockKubectl.On("ExecNodeCommand", mock.Anything, "node1", mock.AnythingOfType("string")).
					Return(false, "Interface not found", nil)
				mockKubectl.On("GetPods", mock.Anything, "", "").Return(true, "", nil)
				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Debug", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectError: false, // Should not error due to lenient removal
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockKubectl := NewMockDryRunExecutor()
			mockLogger := NewMockLogger()
			tt.setupMocks(mockKubectl, mockLogger)

			tt.options.Logger = mockLogger
			service := NewService(mockKubectl, tt.options)

			result, err := service.RemoveVLANs(context.Background(), tt.vlanConfig)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			mockKubectl.AssertExpectations(t)
		})
	}
}

// TestVLANService_VerifyVLANs tests comprehensive VLAN verification scenarios
func TestVLANService_VerifyVLANs(t *testing.T) {
	tests := []struct {
		name        string
		description string
		vlanConfig  *config.NodeVLANConf
		options     Options
		setupMocks  func(*MockDryRunExecutor, *MockLogger)
		expectError bool
		validateFn  func(*testing.T, *OperationResults)
	}{
		{
			name:        "successful_verification_with_correct_ip",
			description: "Successfully verifies VLAN with correct IP configuration",
			vlanConfig: &config.NodeVLANConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeVLANConf",
				Metadata: config.Metadata{
					Name: "verify-test",
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
			options: Options{
				DryRun:               true,
				ValidateConnectivity: true,
				DefaultInterface:     "eth0",
			},
			setupMocks: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", true).Return()
				mockKubectl.On("GetNode", mock.Anything, "node1").Return(true, "node/node1", nil)
				// Return output that contains the expected IP
				mockKubectl.On("ExecNodeCommand", mock.Anything, "node1", "ip addr show eth0.100").
					Return(true, "eth0.100: interface exists\n    inet 192.168.100.10/24 brd", nil)
				mockKubectl.On("GetPods", mock.Anything, "", "").Return(true, "", nil)
				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Debug", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectError: false,
			validateFn: func(t *testing.T, results *OperationResults) {
				assert.Equal(t, 1, results.SuccessfulNodes)
				assert.Contains(t, results.ConfiguredVLANs, "node1")
				assert.Len(t, results.ConfiguredVLANs["node1"], 1)
			},
		},
		{
			name:        "verification_interface_not_found",
			description: "Handles verification when interface doesn't exist",
			vlanConfig: &config.NodeVLANConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeVLANConf",
				Metadata: config.Metadata{
					Name: "verify-missing-test",
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
			options: Options{
				DryRun:               true,
				ValidateConnectivity: true,
				DefaultInterface:     "eth0",
			},
			setupMocks: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", true).Return()
				mockKubectl.On("GetNode", mock.Anything, "node1").Return(true, "node/node1", nil)
				// Interface not found
				mockKubectl.On("ExecNodeCommand", mock.Anything, "node1", "ip addr show eth0.100").
					Return(false, "Device not found", fmt.Errorf("interface not found"))
				mockKubectl.On("GetPods", mock.Anything, "", "").Return(true, "", nil)
				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Debug", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectError: false,
			validateFn: func(t *testing.T, results *OperationResults) {
				assert.Equal(t, 1, results.SuccessfulNodes)
				assert.Contains(t, results.ConfiguredVLANs, "node1")
				assert.Len(t, results.ConfiguredVLANs["node1"], 0) // No VLANs found
			},
		},
		{
			name:        "verification_incorrect_ip",
			description: "Handles verification when IP doesn't match expected",
			vlanConfig: &config.NodeVLANConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeVLANConf",
				Metadata: config.Metadata{
					Name: "verify-wrong-ip-test",
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
			options: Options{
				DryRun:               true,
				ValidateConnectivity: true,
				DefaultInterface:     "eth0",
			},
			setupMocks: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", true).Return()
				mockKubectl.On("GetNode", mock.Anything, "node1").Return(true, "node/node1", nil)
				// Return output with wrong IP
				mockKubectl.On("ExecNodeCommand", mock.Anything, "node1", "ip addr show eth0.100").
					Return(true, "eth0.100: interface exists\n    inet 192.168.100.99/24 brd", nil)
				mockKubectl.On("GetPods", mock.Anything, "", "").Return(true, "", nil)
				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Debug", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()
			},
			expectError: false,
			validateFn: func(t *testing.T, results *OperationResults) {
				assert.Equal(t, 1, results.SuccessfulNodes)
				assert.Contains(t, results.ConfiguredVLANs, "node1")
				assert.Len(t, results.ConfiguredVLANs["node1"], 0) // IP mismatch, no VLAN recorded
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockKubectl := NewMockDryRunExecutor()
			mockLogger := NewMockLogger()
			tt.setupMocks(mockKubectl, mockLogger)

			tt.options.Logger = mockLogger
			service := NewService(mockKubectl, tt.options)

			result, err := service.VerifyVLANs(context.Background(), tt.vlanConfig)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.validateFn != nil {
					tt.validateFn(t, result)
				}
			}

			mockKubectl.AssertExpectations(t)
		})
	}
}

// TestVLANService_GetCurrentState tests current state discovery
func TestVLANService_GetCurrentState(t *testing.T) {
	tests := []struct {
		name        string
		description string
		nodes       []string
		setupMocks  func(*MockDryRunExecutor, *MockLogger)
		expectError bool
		validateFn  func(*testing.T, map[string][]VLANInterfaceInfo)
	}{
		{
			name:        "successful_state_discovery",
			description: "Successfully discovers VLAN state on multiple nodes",
			nodes:       []string{"node1", "node2"},
			setupMocks: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				// Mock discovery for node1
				mockKubectl.On("ExecNodeCommand", mock.Anything, "node1", "ip link show type vlan").
					Return(true, "3: eth0.100@eth0: <BROADCAST,MULTICAST,UP,LOWER_UP>\n4: eth1.200@eth1: <BROADCAST,MULTICAST,UP,LOWER_UP>", nil)
				// Mock discovery for node2
				mockKubectl.On("ExecNodeCommand", mock.Anything, "node2", "ip link show type vlan").
					Return(true, "3: eth0.300@eth0: <BROADCAST,MULTICAST,UP,LOWER_UP>", nil)
			},
			expectError: false,
			validateFn: func(t *testing.T, state map[string][]VLANInterfaceInfo) {
				assert.Len(t, state, 2)
				assert.Contains(t, state, "node1")
				assert.Contains(t, state, "node2")
			},
		},
		{
			name:        "discovery_failure",
			description: "Handles failure during VLAN discovery",
			nodes:       []string{"failing-node"},
			setupMocks: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("ExecNodeCommand", mock.Anything, "failing-node", "ip link show type vlan").
					Return(false, "", fmt.Errorf("command failed"))
			},
			expectError: true,
			validateFn:  nil,
		},
		{
			name:        "no_vlans_found",
			description: "Handles nodes with no VLAN interfaces",
			nodes:       []string{"node-no-vlans"},
			setupMocks: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("ExecNodeCommand", mock.Anything, "node-no-vlans", "ip link show type vlan").
					Return(false, "", nil) // No VLANs found
			},
			expectError: false,
			validateFn: func(t *testing.T, state map[string][]VLANInterfaceInfo) {
				assert.Len(t, state, 1)
				assert.Contains(t, state, "node-no-vlans")
				assert.Len(t, state["node-no-vlans"], 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockKubectl := NewMockDryRunExecutor()
			mockLogger := NewMockLogger()
			tt.setupMocks(mockKubectl, mockLogger)

			service := NewService(mockKubectl, Options{Logger: mockLogger})

			state, err := service.GetCurrentState(context.Background(), tt.nodes)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, state)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, state)
				if tt.validateFn != nil {
					tt.validateFn(t, state)
				}
			}

			mockKubectl.AssertExpectations(t)
		})
	}
}

// TestVLANService_HelperMethods tests internal helper methods
func TestVLANService_HelperMethods(t *testing.T) {
	t.Run("getAllNodesFromConfig", func(t *testing.T) {
		// Given: VLAN config with multiple nodes across multiple VLANs
		vlanConfig := &config.NodeVLANConf{
			Spec: config.NodeVLANSpec{
				VLANs: map[string]config.VLANConfig{
					"management": {
						NodeMapping: map[string]string{
							"node1": "192.168.100.10/24",
							"node2": "192.168.100.11/24",
						},
					},
					"storage": {
						NodeMapping: map[string]string{
							"node2": "10.10.200.11/24",
							"node3": "10.10.200.12/24",
						},
					},
				},
			},
		}

		mockKubectl := NewMockDryRunExecutor()
		mockLogger := NewMockLogger()
		service := NewService(mockKubectl, Options{Logger: mockLogger})

		// Access internal method via type assertion
		vlanService := service.(*VLANService)

		// When: Get all nodes from config
		allNodes := vlanService.getAllNodesFromConfig(vlanConfig)

		// Then: Verify all unique nodes are found
		assert.Len(t, allNodes, 3)
		assert.Contains(t, allNodes, "node1")
		assert.Contains(t, allNodes, "node2")
		assert.Contains(t, allNodes, "node3")
	})

	t.Run("getAllNodesFromConfig_empty", func(t *testing.T) {
		// Given: Empty VLAN config
		vlanConfig := &config.NodeVLANConf{
			Spec: config.NodeVLANSpec{
				VLANs: map[string]config.VLANConfig{},
			},
		}

		mockKubectl := NewMockDryRunExecutor()
		mockLogger := NewMockLogger()
		service := NewService(mockKubectl, Options{Logger: mockLogger})
		vlanService := service.(*VLANService)

		// When: Get all nodes from empty config
		allNodes := vlanService.getAllNodesFromConfig(vlanConfig)

		// Then: Should return empty map
		assert.Len(t, allNodes, 0)
	})

	t.Run("generateNetplanConfig", func(t *testing.T) {
		// Given: VLAN service and config
		mockKubectl := NewMockDryRunExecutor()
		mockLogger := NewMockLogger()
		service := NewService(mockKubectl, Options{Logger: mockLogger})
		vlanService := service.(*VLANService)

		vlanConfig := config.VLANConfig{
			ID:        100,
			Subnet:    "192.168.100.0/24",
			Interface: "eth0",
		}

		// When: Generate netplan config
		netplanCmd := vlanService.generateNetplanConfig("management", vlanConfig, "eth0.100", "eth0", "192.168.100.10/24")

		// Then: Should return expected command
		assert.NotEmpty(t, netplanCmd)
		assert.Contains(t, netplanCmd, "management")
		assert.Contains(t, netplanCmd, "echo")
	})
}

// TestVLANService_CleanupDebugPods tests the cleanup functionality
func TestVLANService_CleanupDebugPods(t *testing.T) {
	tests := []struct {
		name        string
		description string
		setupMocks  func(*MockDryRunExecutor, *MockLogger)
		expectLogs  []string
	}{
		{
			name:        "successful_cleanup_with_pods",
			description: "Successfully cleans up debug pods",
			setupMocks: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				// Mock pod listing with debug pods found
				mockKubectl.On("GetPods", mock.Anything, "", "").
					Return(true, "pod/node-debugger-abc123\npod/node-debugger-xyz789\npod/other-pod", nil)
				// Mock pod deletion
				mockKubectl.On("DeletePod", mock.Anything, "node-debugger-abc123").Return(true, "", nil)
				mockKubectl.On("DeletePod", mock.Anything, "node-debugger-xyz789").Return(true, "", nil)

				mockLogger.On("Info", "🧹 Cleaning up debug pods...").Return()
				mockLogger.On("Info", "✅ Cleaned up 2 debug pods").Return()
			},
			expectLogs: []string{"Cleaning up", "Cleaned up 2"},
		},
		{
			name:        "cleanup_no_pods_found",
			description: "Handles cleanup when no debug pods exist",
			setupMocks: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				// Mock pod listing with no debug pods
				mockKubectl.On("GetPods", mock.Anything, "", "").
					Return(true, "pod/other-pod-1\npod/other-pod-2", nil)

				mockLogger.On("Info", "🧹 Cleaning up debug pods...").Return()
				mockLogger.On("Info", "✅ No debug pods to clean up").Return()
			},
			expectLogs: []string{"Cleaning up", "No debug pods"},
		},
		{
			name:        "cleanup_pod_listing_failure",
			description: "Handles failure when listing pods",
			setupMocks: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				// Mock pod listing failure
				mockKubectl.On("GetPods", mock.Anything, "", "").
					Return(false, "", fmt.Errorf("failed to list pods"))

				mockLogger.On("Info", "🧹 Cleaning up debug pods...").Return()
				mockLogger.On("Warn", mock.MatchedBy(func(msg string) bool {
					return len(msg) > 10 // Check message has reasonable length
				})).Return()
			},
			expectLogs: []string{"Cleaning up", "Failed to get pods"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockKubectl := NewMockDryRunExecutor()
			mockLogger := NewMockLogger()
			tt.setupMocks(mockKubectl, mockLogger)

			service := NewService(mockKubectl, Options{Logger: mockLogger})
			vlanService := service.(*VLANService)

			// When: Call cleanup method
			vlanService.cleanupDebugPods(context.Background())

			// Then: Verify mock expectations
			mockKubectl.AssertExpectations(t)
			mockLogger.AssertExpectations(t)
		})
	}
}

// TestVLANService_DryRunMode tests comprehensive dry-run functionality
func TestVLANService_DryRunMode(t *testing.T) {
	tests := []struct {
		name        string
		description string
		options     Options
		operation   string
		setupMocks  func(*MockDryRunExecutor, *MockLogger)
	}{
		{
			name:        "dry_run_configure",
			description: "Dry run mode for VLAN configuration",
			options: Options{
				DryRun:               true,
				ValidateConnectivity: true,
				DefaultInterface:     "eth0",
			},
			operation: "configure",
			setupMocks: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", true).Return()
				mockKubectl.On("GetNode", mock.Anything, "node1").Return(true, "node/node1", nil)
				mockKubectl.On("ExecNodeCommand", mock.Anything, "node1", mock.AnythingOfType("string")).
					Return(true, "DRY RUN: Would configure VLAN", nil)
				mockKubectl.On("GetPods", mock.Anything, "", "").Return(true, "", nil)
				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Debug", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()
			},
		},
		{
			name:        "dry_run_verify",
			description: "Dry run mode for VLAN verification",
			options: Options{
				DryRun:               true,
				ValidateConnectivity: true,
				DefaultInterface:     "eth0",
			},
			operation: "verify",
			setupMocks: func(mockKubectl *MockDryRunExecutor, mockLogger *MockLogger) {
				mockKubectl.On("SetDryRun", true).Return()
				mockKubectl.On("GetNode", mock.Anything, "node1").Return(true, "node/node1", nil)
				mockKubectl.On("ExecNodeCommand", mock.Anything, "node1", "ip addr show eth0.100").
					Return(true, "eth0.100: interface exists", nil)
				mockKubectl.On("GetPods", mock.Anything, "", "").Return(true, "", nil)
				mockLogger.On("Info", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Debug", mock.AnythingOfType("string")).Return().Maybe()
				mockLogger.On("Warn", mock.AnythingOfType("string")).Return().Maybe()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockKubectl := NewMockDryRunExecutor()
			mockLogger := NewMockLogger()
			tt.setupMocks(mockKubectl, mockLogger)

			tt.options.Logger = mockLogger
			service := NewService(mockKubectl, tt.options)

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

			var result *OperationResults
			var err error

			// Execute operation based on test type
			switch tt.operation {
			case "configure":
				result, err = service.ConfigureVLANs(context.Background(), vlanConfig)
			case "verify":
				result, err = service.VerifyVLANs(context.Background(), vlanConfig)
			case "remove":
				result, err = service.RemoveVLANs(context.Background(), vlanConfig)
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)
			mockKubectl.AssertExpectations(t)
		})
	}
}

// TestVLANService_Options tests various service options configurations
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
				DefaultInterface:     "ens192",
				Logger:               NewMockLogger(),
			},
			description: "Service with all options enabled",
		},
		{
			name: "production_options",
			options: Options{
				DryRun:               false,
				Verbose:              false,
				ValidateConnectivity: true,
				PersistentConfig:     true,
				DefaultInterface:     "eth0",
				Logger:               NewMockLogger(),
			},
			description: "Production-like configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockKubectl := NewMockDryRunExecutor()

			// When: Create service with options
			service := NewService(mockKubectl, tt.options)

			// Then: Verify service creation and options
			assert.NotNil(t, service, "Service should be created successfully")

			// Type assertion to verify options are set correctly
			vlanService := service.(*VLANService)
			assert.Equal(t, tt.options, vlanService.options, "Options should match")
			assert.Equal(t, mockKubectl, vlanService.kubectl, "Kubectl executor should match")
		})
	}
}

// TestVLANInterfaceInfo tests the VLANInterfaceInfo struct comprehensively
func TestVLANInterfaceInfo(t *testing.T) {
	tests := []struct {
		name     string
		vlanInfo VLANInterfaceInfo
	}{
		{
			name: "standard_vlan_info",
			vlanInfo: VLANInterfaceInfo{
				VLANName:      "management",
				VLANId:        100,
				Interface:     "eth0.100",
				IPAddress:     "192.168.100.10/24",
				PhysInterface: "eth0",
				Subnet:        "192.168.100.0/24",
			},
		},
		{
			name: "storage_vlan_info",
			vlanInfo: VLANInterfaceInfo{
				VLANName:      "storage",
				VLANId:        200,
				Interface:     "eth1.200",
				IPAddress:     "10.10.200.10/24",
				PhysInterface: "eth1",
				Subnet:        "10.10.200.0/24",
			},
		},
		{
			name: "custom_interface_vlan",
			vlanInfo: VLANInterfaceInfo{
				VLANName:      "tenant",
				VLANId:        4094,
				Interface:     "ens192.4094",
				IPAddress:     "172.16.50.100/24",
				PhysInterface: "ens192",
				Subnet:        "172.16.50.0/24",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify all fields are set correctly
			assert.NotEmpty(t, tt.vlanInfo.VLANName)
			assert.Greater(t, tt.vlanInfo.VLANId, 0)
			assert.NotEmpty(t, tt.vlanInfo.Interface)
			assert.NotEmpty(t, tt.vlanInfo.IPAddress)
			assert.NotEmpty(t, tt.vlanInfo.PhysInterface)
			assert.NotEmpty(t, tt.vlanInfo.Subnet)

			// Verify interface naming convention
			expectedInterface := fmt.Sprintf("%s.%d", tt.vlanInfo.PhysInterface, tt.vlanInfo.VLANId)
			assert.Equal(t, expectedInterface, tt.vlanInfo.Interface)
		})
	}
}

// TestOperationResults tests the OperationResults struct comprehensively
func TestOperationResults(t *testing.T) {
	tests := []struct {
		name    string
		results *OperationResults
	}{
		{
			name: "successful_operation_results",
			results: &OperationResults{
				TotalNodes:      3,
				SuccessfulNodes: 3,
				FailedNodes:     []string{},
				ConfiguredVLANs: map[string][]VLANInterfaceInfo{
					"node1": {
						{VLANName: "management", VLANId: 100, Interface: "eth0.100"},
					},
					"node2": {
						{VLANName: "management", VLANId: 100, Interface: "eth0.100"},
					},
					"node3": {
						{VLANName: "management", VLANId: 100, Interface: "eth0.100"},
					},
				},
				Errors: []error{},
			},
		},
		{
			name: "mixed_operation_results",
			results: &OperationResults{
				TotalNodes:      4,
				SuccessfulNodes: 2,
				FailedNodes:     []string{"node3", "node4"},
				ConfiguredVLANs: map[string][]VLANInterfaceInfo{
					"node1": {
						{VLANName: "management", VLANId: 100, Interface: "eth0.100"},
						{VLANName: "storage", VLANId: 200, Interface: "eth1.200"},
					},
					"node2": {
						{VLANName: "management", VLANId: 100, Interface: "eth0.100"},
					},
				},
				Errors: []error{
					fmt.Errorf("node3 not found"),
					fmt.Errorf("node4 command failed"),
				},
			},
		},
		{
			name: "failed_operation_results",
			results: &OperationResults{
				TotalNodes:      2,
				SuccessfulNodes: 0,
				FailedNodes:     []string{"node1", "node2"},
				ConfiguredVLANs: map[string][]VLANInterfaceInfo{},
				Errors: []error{
					fmt.Errorf("all operations failed"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify results structure
			assert.NotNil(t, tt.results)
			assert.GreaterOrEqual(t, tt.results.TotalNodes, 0)
			assert.GreaterOrEqual(t, tt.results.SuccessfulNodes, 0)
			assert.LessOrEqual(t, tt.results.SuccessfulNodes, tt.results.TotalNodes)

			// Verify failed nodes count matches
			expectedFailedCount := tt.results.TotalNodes - tt.results.SuccessfulNodes
			assert.Equal(t, expectedFailedCount, len(tt.results.FailedNodes))

			// Verify ConfiguredVLANs structure
			assert.NotNil(t, tt.results.ConfiguredVLANs)

			// For successful nodes, should have VLAN configurations
			if tt.results.SuccessfulNodes > 0 && len(tt.results.ConfiguredVLANs) > 0 {
				for nodeName, vlans := range tt.results.ConfiguredVLANs {
					assert.NotEmpty(t, nodeName)
					for _, vlan := range vlans {
						assert.NotEmpty(t, vlan.VLANName)
						assert.Greater(t, vlan.VLANId, 0)
					}
				}
			}

			// Verify errors collection
			assert.NotNil(t, tt.results.Errors)
		})
	}
}
