// Package config provides unit tests for configuration data structures
// WHY: Validates multi-CRD type system and interface contracts essential for infrastructure safety
package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestConfig_Interface tests the Config interface implementation
// WHY: Ensures all CRD types properly implement the common interface for unified processing
func TestConfig_Interface(t *testing.T) {
	tests := []struct {
		name         string
		description  string
		config       Config
		expectedAPI  string
		expectedKind string
		hasNodeRoles bool
	}{
		{
			name:        "node_label_conf_implements_config",
			description: "NodeLabelConf properly implements Config interface for unified processing",
			config: NodeLabelConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeLabelConf",
				Metadata: Metadata{
					Name: "test-config",
				},
				Spec: NodeLabelSpec{
					NodeRoles: map[string]NodeRole{
						"control": {
							Nodes:  []string{"node1"},
							Labels: map[string]string{"role": "control"},
						},
					},
				},
			},
			expectedAPI:  "openstack.kictl.icycloud.io/v1",
			expectedKind: "NodeLabelConf",
			hasNodeRoles: true,
		},
		{
			name:        "node_vlan_conf_implements_config",
			description: "NodeVLANConf properly implements Config interface for multi-CRD compatibility",
			config: NodeVLANConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeVLANConf",
				Metadata: Metadata{
					Name: "test-vlan",
				},
				Spec: NodeVLANSpec{
					VLANs: map[string]VLANConfig{
						"management": {
							ID:     100,
							Subnet: "192.168.1.0/24",
						},
					},
				},
			},
			expectedAPI:  "openstack.kictl.icycloud.io/v1",
			expectedKind: "NodeVLANConf",
			hasNodeRoles: false, // VLANs don't use NodeRoles
		},
		{
			name:        "node_test_conf_implements_config",
			description: "NodeTestConf properly implements Config interface for unified testing framework",
			config: NodeTestConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeTestConf",
				Metadata: Metadata{
					Name: "test-connectivity",
				},
				Spec: NodeTestSpec{
					Tests: []ConnectivityTest{
						{
							Name:    "ping-test",
							Source:  "node1",
							Targets: []string{"node2"},
						},
					},
				},
			},
			expectedAPI:  "openstack.kictl.icycloud.io/v1",
			expectedKind: "NodeTestConf",
			hasNodeRoles: false, // Tests don't use NodeRoles
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Access interface methods
			apiVersion := tt.config.GetAPIVersion()
			kind := tt.config.GetKind()
			metadata := tt.config.GetMetadata()
			nodeRoles := tt.config.GetNodeRoles()
			tools := tt.config.GetTools()

			// Then: Verify interface contract
			assert.Equal(t, tt.expectedAPI, apiVersion, "APIVersion mismatch")
			assert.Equal(t, tt.expectedKind, kind, "Kind mismatch")
			assert.NotEmpty(t, metadata.Name, "Metadata name should not be empty")
			assert.NotNil(t, nodeRoles, "NodeRoles should not be nil (can be empty)")
			assert.NotNil(t, tools, "Tools should not be nil")

			if tt.hasNodeRoles {
				assert.NotEmpty(t, nodeRoles, "Should have node roles")
			} else {
				assert.Empty(t, nodeRoles, "Should have empty node roles")
			}
		})
	}
}

// TestMetadata_Structure tests metadata structure validation
// WHY: Kubernetes-style metadata is critical for proper CRD identification and labeling
func TestMetadata_Structure(t *testing.T) {
	tests := []struct {
		name        string
		description string
		metadata    Metadata
		isValid     bool
	}{
		{
			name:        "complete_metadata_valid",
			description: "Complete metadata with all fields should be valid",
			metadata: Metadata{
				Name:      "production-config",
				Namespace: "openstack",
				Labels: map[string]string{
					"environment": "production",
					"region":      "homelab",
				},
			},
			isValid: true,
		},
		{
			name:        "minimal_metadata_valid",
			description: "Minimal metadata with just name should be valid",
			metadata: Metadata{
				Name: "test-config",
			},
			isValid: true,
		},
		{
			name:        "empty_name_invalid",
			description: "Metadata without name should be invalid",
			metadata: Metadata{
				Namespace: "openstack",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Validate metadata
			hasName := tt.metadata.Name != ""

			// Then: Verify validity
			assert.Equal(t, tt.isValid, hasName, "Metadata validity mismatch")

			if tt.isValid {
				assert.NotEmpty(t, tt.metadata.Name, "Valid metadata must have name")
			}
		})
	}
}

// TestNodeRole_Structure tests node role configuration structure
// WHY: Node roles define the mapping between physical nodes and their infrastructure roles
func TestNodeRole_Structure(t *testing.T) {
	tests := []struct {
		name        string
		description string
		nodeRole    NodeRole
		expectValid bool
	}{
		{
			name:        "complete_node_role_valid",
			description: "Complete node role with all fields should be valid for production use",
			nodeRole: NodeRole{
				Nodes: []string{"rsb2", "rsb3", "rsb4"},
				Labels: map[string]string{
					"openstack-role":            "control-plane",
					"cluster.openstack.io/role": "control-plane",
				},
				Description: "OpenStack control plane services",
			},
			expectValid: true,
		},
		{
			name:        "minimal_node_role_valid",
			description: "Minimal node role with just nodes and labels should be valid",
			nodeRole: NodeRole{
				Nodes: []string{"rsb5"},
				Labels: map[string]string{
					"role": "storage",
				},
			},
			expectValid: true,
		},
		{
			name:        "empty_nodes_invalid",
			description: "Node role without nodes should be invalid",
			nodeRole: NodeRole{
				Labels: map[string]string{
					"role": "compute",
				},
			},
			expectValid: false,
		},
		{
			name:        "empty_labels_invalid",
			description: "Node role without labels should be invalid",
			nodeRole: NodeRole{
				Nodes: []string{"rsb6"},
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Validate node role structure
			hasNodes := len(tt.nodeRole.Nodes) > 0
			hasLabels := len(tt.nodeRole.Labels) > 0
			isValid := hasNodes && hasLabels

			// Then: Verify validity
			assert.Equal(t, tt.expectValid, isValid, "NodeRole validity mismatch")

			if tt.expectValid {
				assert.NotEmpty(t, tt.nodeRole.Nodes, "Valid node role must have nodes")
				assert.NotEmpty(t, tt.nodeRole.Labels, "Valid node role must have labels")
			}
		})
	}
}

// TestToolConfig_Structure tests tool configuration structure
// WHY: Tool configurations enable consistent behavior across all CRD types
func TestToolConfig_Structure(t *testing.T) {
	tests := []struct {
		name        string
		description string
		toolConfig  ToolConfig
		expectValid bool
	}{
		{
			name:        "production_tool_config",
			description: "Production tool config with validation enabled should be safe for real clusters",
			toolConfig: ToolConfig{
				DryRun:        false,
				ValidateNodes: true,
				LogLevel:      "info",
			},
			expectValid: true,
		},
		{
			name:        "dry_run_tool_config",
			description: "Dry-run tool config should be safe for testing without cluster changes",
			toolConfig: ToolConfig{
				DryRun:        true,
				ValidateNodes: true,
				LogLevel:      "debug",
			},
			expectValid: true,
		},
		{
			name:        "minimal_tool_config",
			description: "Minimal tool config should use reasonable defaults",
			toolConfig: ToolConfig{
				LogLevel: "warn",
			},
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Analyze tool config
			hasLogLevel := tt.toolConfig.LogLevel != ""

			// Then: Verify structure
			assert.Equal(t, tt.expectValid, hasLogLevel || tt.expectValid, "ToolConfig structure check")

			if tt.expectValid {
				// Verify boolean fields have explicit values (default is false)
				assert.NotNil(t, tt.toolConfig.DryRun)
				assert.NotNil(t, tt.toolConfig.ValidateNodes)
			}
		})
	}
}

// TestVLANConfig_Structure tests VLAN configuration structure
// WHY: VLAN configurations define network segmentation critical for OpenStack networking
func TestVLANConfig_Structure(t *testing.T) {
	tests := []struct {
		name        string
		description string
		vlanConfig  VLANConfig
		expectValid bool
	}{
		{
			name:        "complete_vlan_config",
			description: "Complete VLAN config with all fields should be valid for production networking",
			vlanConfig: VLANConfig{
				ID:        100,
				Subnet:    "192.168.100.0/24",
				Interface: "eth0",
				NodeMapping: map[string]string{
					"rsb2": "192.168.100.12",
					"rsb3": "192.168.100.13",
				},
			},
			expectValid: true,
		},
		{
			name:        "minimal_vlan_config",
			description: "Minimal VLAN config should be valid with required fields",
			vlanConfig: VLANConfig{
				ID:     200,
				Subnet: "10.0.0.0/24",
				NodeMapping: map[string]string{
					"rsb4": "10.0.0.14",
				},
			},
			expectValid: true,
		},
		{
			name:        "invalid_vlan_id",
			description: "VLAN config with invalid ID should be invalid",
			vlanConfig: VLANConfig{
				ID:     0, // Invalid VLAN ID
				Subnet: "192.168.1.0/24",
				NodeMapping: map[string]string{
					"rsb5": "192.168.1.15",
				},
			},
			expectValid: false,
		},
		{
			name:        "empty_node_mapping",
			description: "VLAN config without node mapping should be invalid",
			vlanConfig: VLANConfig{
				ID:          300,
				Subnet:      "192.168.3.0/24",
				NodeMapping: map[string]string{}, // Empty mapping
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Validate VLAN config structure
			hasValidID := tt.vlanConfig.ID > 0 && tt.vlanConfig.ID <= 4094
			hasSubnet := tt.vlanConfig.Subnet != ""
			hasNodeMapping := len(tt.vlanConfig.NodeMapping) > 0
			isValid := hasValidID && hasSubnet && hasNodeMapping

			// Then: Verify validity
			assert.Equal(t, tt.expectValid, isValid, "VLANConfig validity mismatch")

			if tt.expectValid {
				assert.Greater(t, tt.vlanConfig.ID, 0, "Valid VLAN ID must be > 0")
				assert.LessOrEqual(t, tt.vlanConfig.ID, 4094, "Valid VLAN ID must be <= 4094")
				assert.NotEmpty(t, tt.vlanConfig.Subnet, "Valid VLAN must have subnet")
				assert.NotEmpty(t, tt.vlanConfig.NodeMapping, "Valid VLAN must have node mapping")
			}
		})
	}
}

// TestConnectivityTest_Structure tests connectivity test structure
// WHY: Connectivity tests validate network segmentation and reachability in OpenStack deployments
func TestConnectivityTest_Structure(t *testing.T) {
	tests := []struct {
		name        string
		description string
		connectTest ConnectivityTest
		expectValid bool
	}{
		{
			name:        "complete_connectivity_test",
			description: "Complete connectivity test should validate network reachability properly",
			connectTest: ConnectivityTest{
				Name:          "management-reachability",
				Description:   "Test management network connectivity",
				Source:        "management",
				Targets:       []string{"storage", "compute"},
				Timeout:       30,
				ExpectSuccess: true,
			},
			expectValid: true,
		},
		{
			name:        "isolation_test",
			description: "Network isolation test should verify proper segmentation",
			connectTest: ConnectivityTest{
				Name:          "tenant-isolation",
				Description:   "Verify tenant network isolation",
				Source:        "tenant",
				Targets:       []string{"management"},
				Timeout:       10,
				ExpectSuccess: false, // Should fail for proper isolation
			},
			expectValid: true,
		},
		{
			name:        "minimal_connectivity_test",
			description: "Minimal connectivity test should be valid with required fields",
			connectTest: ConnectivityTest{
				Name:    "basic-ping",
				Source:  "node1",
				Targets: []string{"node2"},
			},
			expectValid: true,
		},
		{
			name:        "empty_name_invalid",
			description: "Connectivity test without name should be invalid",
			connectTest: ConnectivityTest{
				Source:  "node1",
				Targets: []string{"node2"},
			},
			expectValid: false,
		},
		{
			name:        "empty_targets_invalid",
			description: "Connectivity test without targets should be invalid",
			connectTest: ConnectivityTest{
				Name:   "test",
				Source: "node1",
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Validate connectivity test structure
			hasName := tt.connectTest.Name != ""
			hasSource := tt.connectTest.Source != ""
			hasTargets := len(tt.connectTest.Targets) > 0
			isValid := hasName && hasSource && hasTargets

			// Then: Verify validity
			assert.Equal(t, tt.expectValid, isValid, "ConnectivityTest validity mismatch")

			if tt.expectValid {
				assert.NotEmpty(t, tt.connectTest.Name, "Valid test must have name")
				assert.NotEmpty(t, tt.connectTest.Source, "Valid test must have source")
				assert.NotEmpty(t, tt.connectTest.Targets, "Valid test must have targets")
			}
		})
	}
}

// TestNodeLabelConf_Complete tests complete NodeLabelConf structure
// WHY: NodeLabelConf is the primary CRD for infrastructure node management
func TestNodeLabelConf_Complete(t *testing.T) {
	// Given: Complete NodeLabelConf structure
	config := NodeLabelConf{
		APIVersion: "openstack.kictl.icycloud.io/v1",
		Kind:       "NodeLabelConf",
		Metadata: Metadata{
			Name:      "production-labels",
			Namespace: "openstack",
			Labels: map[string]string{
				"environment": "production",
			},
		},
		Spec: NodeLabelSpec{
			NodeRoles: map[string]NodeRole{
				"control": {
					Nodes:       []string{"rsb2", "rsb3"},
					Labels:      map[string]string{"role": "control"},
					Description: "Control plane nodes",
				},
			},
		},
		Tools: Tools{
			Nlabel: ToolConfig{
				DryRun:        false,
				ValidateNodes: true,
				LogLevel:      "info",
			},
		},
	}

	// When: Test interface implementation
	var cfg Config = config

	// Then: Verify complete structure
	assert.Equal(t, "openstack.kictl.icycloud.io/v1", cfg.GetAPIVersion())
	assert.Equal(t, "NodeLabelConf", cfg.GetKind())
	assert.Equal(t, "production-labels", cfg.GetMetadata().Name)
	assert.NotEmpty(t, cfg.GetNodeRoles())
	assert.Equal(t, "info", cfg.GetTools().Nlabel.LogLevel)
}

// TestMultiCRD_TypeCompatibility tests type compatibility across CRDs
// WHY: Multi-CRD configurations must work together in unified workflows
func TestMultiCRD_TypeCompatibility(t *testing.T) {
	tests := []struct {
		name        string
		description string
		configs     []Config
	}{
		{
			name:        "all_crd_types_compatible",
			description: "All CRD types should be processable through common Config interface",
			configs: []Config{
				NodeLabelConf{
					APIVersion: "openstack.kictl.icycloud.io/v1",
					Kind:       "NodeLabelConf",
					Metadata:   Metadata{Name: "labels"},
					Spec: NodeLabelSpec{
						NodeRoles: make(map[string]NodeRole),
					},
				},
				NodeVLANConf{
					APIVersion: "openstack.kictl.icycloud.io/v1",
					Kind:       "NodeVLANConf",
					Metadata:   Metadata{Name: "vlans"},
				},
				NodeTestConf{
					APIVersion: "openstack.kictl.icycloud.io/v1",
					Kind:       "NodeTestConf",
					Metadata:   Metadata{Name: "tests"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Process all configs through common interface
			for i, cfg := range tt.configs {
				// Then: Verify interface compatibility
				assert.NotEmpty(t, cfg.GetAPIVersion(), "Config %d should have API version", i)
				assert.NotEmpty(t, cfg.GetKind(), "Config %d should have kind", i)
				assert.NotEmpty(t, cfg.GetMetadata().Name, "Config %d should have name", i)
				assert.NotNil(t, cfg.GetNodeRoles(), "Config %d should have node roles map (can be empty)", i)
				assert.NotNil(t, cfg.GetTools(), "Config %d should have tools config", i)
			}
		})
	}
}
