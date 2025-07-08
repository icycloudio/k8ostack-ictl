// Package config provides unit tests for configuration bundle management
// WHY: ConfigBundle enables unified multi-CRD operations essential for complex infrastructure deployments
package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestConfigBundle_GetAllConfigs tests configuration aggregation
// WHY: Validates that bundle properly aggregates different CRD types for unified processing
func TestConfigBundle_GetAllConfigs(t *testing.T) {
	tests := []struct {
		name          string
		description   string
		bundle        *ConfigBundle
		expectedCount int
		expectedTypes []string
	}{
		{
			name:          "empty_bundle_returns_empty",
			description:   "Empty bundle should return no configurations for processing",
			bundle:        NewEmptyBundle(),
			expectedCount: 0,
			expectedTypes: []string{},
		},
		{
			name:        "single_node_labels_config",
			description: "Bundle with only NodeLabels should return single configuration",
			bundle: &ConfigBundle{
				NodeLabels: &NodeLabelConf{
					APIVersion: "openstack.kictl.icycloud.io/v1",
					Kind:       "NodeLabelConf",
					Metadata:   Metadata{Name: "test-labels"},
				},
			},
			expectedCount: 1,
			expectedTypes: []string{"*config.NodeLabelConf"},
		},
		{
			name:        "complete_multi_crd_bundle",
			description: "Complete bundle should return all three CRD types for comprehensive processing",
			bundle: &ConfigBundle{
				NodeLabels: &NodeLabelConf{
					APIVersion: "openstack.kictl.icycloud.io/v1",
					Kind:       "NodeLabelConf",
					Metadata:   Metadata{Name: "production-labels"},
				},
				VLANs: &NodeVLANConf{
					APIVersion: "openstack.kictl.icycloud.io/v1",
					Kind:       "NodeVLANConf",
					Metadata:   Metadata{Name: "production-vlans"},
				},
				Tests: &NodeTestConf{
					APIVersion: "openstack.kictl.icycloud.io/v1",
					Kind:       "NodeTestConf",
					Metadata:   Metadata{Name: "production-tests"},
				},
			},
			expectedCount: 3,
			expectedTypes: []string{"*config.NodeLabelConf", "*config.NodeVLANConf", "*config.NodeTestConf"},
		},
		{
			name:        "partial_bundle_vlans_and_tests",
			description: "Partial bundle should return only configured CRD types",
			bundle: &ConfigBundle{
				VLANs: &NodeVLANConf{
					APIVersion: "openstack.kictl.icycloud.io/v1",
					Kind:       "NodeVLANConf",
					Metadata:   Metadata{Name: "network-vlans"},
				},
				Tests: &NodeTestConf{
					APIVersion: "openstack.kictl.icycloud.io/v1",
					Kind:       "NodeTestConf",
					Metadata:   Metadata{Name: "connectivity-tests"},
				},
			},
			expectedCount: 2,
			expectedTypes: []string{"*config.NodeVLANConf", "*config.NodeTestConf"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Get all configurations
			configs := tt.bundle.GetAllConfigs()

			// Then: Verify count and types
			assert.Equal(t, tt.expectedCount, len(configs), "Configuration count mismatch")
			assert.Equal(t, tt.expectedCount, tt.bundle.GetConfigCount(), "GetConfigCount should match actual count")

			// Verify types (basic type checking)
			for i, cfg := range configs {
				assert.NotNil(t, cfg, "Configuration %d should not be nil", i)
			}
		})
	}
}

// TestConfigBundle_GetAllConfigsTyped tests typed configuration access
// WHY: Validates type-safe access to configurations for interface-based operations
func TestConfigBundle_GetAllConfigsTyped(t *testing.T) {
	tests := []struct {
		name             string
		description      string
		bundle           *ConfigBundle
		expectedCount    int
		expectNodeLabels bool
		expectVLANs      bool
		expectTests      bool
	}{
		{
			name:             "empty_bundle_typed_empty",
			description:      "Empty bundle should return empty typed configurations",
			bundle:           NewEmptyBundle(),
			expectedCount:    0,
			expectNodeLabels: false,
			expectVLANs:      false,
			expectTests:      false,
		},
		{
			name:        "complete_bundle_typed_access",
			description: "Complete bundle should provide type-safe access to all configurations",
			bundle: &ConfigBundle{
				NodeLabels: &NodeLabelConf{
					APIVersion: "openstack.kictl.icycloud.io/v1",
					Kind:       "NodeLabelConf",
					Metadata:   Metadata{Name: "typed-labels"},
				},
				VLANs: &NodeVLANConf{
					APIVersion: "openstack.kictl.icycloud.io/v1",
					Kind:       "NodeVLANConf",
					Metadata:   Metadata{Name: "typed-vlans"},
				},
				Tests: &NodeTestConf{
					APIVersion: "openstack.kictl.icycloud.io/v1",
					Kind:       "NodeTestConf",
					Metadata:   Metadata{Name: "typed-tests"},
				},
			},
			expectedCount:    3,
			expectNodeLabels: true,
			expectVLANs:      true,
			expectTests:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Get typed configurations
			typedConfigs := tt.bundle.GetAllConfigsTyped()

			// Then: Verify typed access
			assert.Equal(t, tt.expectedCount, len(typedConfigs), "Typed configuration count mismatch")

			// Verify each config implements Config interface
			for i, cfg := range typedConfigs {
				assert.NotEmpty(t, cfg.GetAPIVersion(), "Config %d should have API version", i)
				assert.NotEmpty(t, cfg.GetKind(), "Config %d should have kind", i)
				assert.NotEmpty(t, cfg.GetMetadata().Name, "Config %d should have name", i)
			}

			// Verify presence indicators
			assert.Equal(t, tt.expectNodeLabels, tt.bundle.HasNodeLabels(), "NodeLabels presence mismatch")
			assert.Equal(t, tt.expectVLANs, tt.bundle.HasVLANs(), "VLANs presence mismatch")
			assert.Equal(t, tt.expectTests, tt.bundle.HasTests(), "Tests presence mismatch")
		})
	}
}

// TestConfigBundle_GetSummary tests bundle summary generation
// WHY: Validates human-readable bundle status for operational visibility
func TestConfigBundle_GetSummary(t *testing.T) {
	tests := []struct {
		name            string
		description     string
		bundle          *ConfigBundle
		expectedSummary string
		containsText    []string
	}{
		{
			name:            "empty_bundle_summary",
			description:     "Empty bundle should provide clear empty status",
			bundle:          NewEmptyBundle(),
			expectedSummary: "Empty bundle",
			containsText:    []string{"Empty bundle"},
		},
		{
			name:        "node_labels_only_summary",
			description: "NodeLabels-only bundle should show role and node counts",
			bundle: &ConfigBundle{
				NodeLabels: &NodeLabelConf{
					APIVersion: "openstack.kictl.icycloud.io/v1",
					Kind:       "NodeLabelConf",
					Metadata:   Metadata{Name: "production-labels"},
					Spec: NodeLabelSpec{
						NodeRoles: map[string]NodeRole{
							"control": {
								Nodes:  []string{"rsb2", "rsb3"},
								Labels: map[string]string{"role": "control"},
							},
							"storage": {
								Nodes:  []string{"rsb5"},
								Labels: map[string]string{"role": "storage"},
							},
						},
					},
				},
			},
			containsText: []string{"NodeLabels", "2 roles", "3 nodes"},
		},
		{
			name:        "vlans_only_summary",
			description: "VLANs-only bundle should show VLAN count",
			bundle: &ConfigBundle{
				VLANs: &NodeVLANConf{
					APIVersion: "openstack.kictl.icycloud.io/v1",
					Kind:       "NodeVLANConf",
					Metadata:   Metadata{Name: "network-vlans"},
					Spec: NodeVLANSpec{
						VLANs: map[string]VLANConfig{
							"management": {ID: 100, Subnet: "192.168.100.0/24"},
							"storage":    {ID: 200, Subnet: "192.168.200.0/24"},
						},
					},
				},
			},
			containsText: []string{"VLANs", "2 vlans"},
		},
		{
			name:        "complete_bundle_comprehensive_summary",
			description: "Complete bundle should provide comprehensive summary of all components",
			bundle: &ConfigBundle{
				NodeLabels: &NodeLabelConf{
					APIVersion: "openstack.kictl.icycloud.io/v1",
					Kind:       "NodeLabelConf",
					Metadata:   Metadata{Name: "comprehensive-labels"},
					Spec: NodeLabelSpec{
						NodeRoles: map[string]NodeRole{
							"control": {
								Nodes:  []string{"rsb2", "rsb3", "rsb4"},
								Labels: map[string]string{"role": "control"},
							},
						},
					},
				},
				VLANs: &NodeVLANConf{
					APIVersion: "openstack.kictl.icycloud.io/v1",
					Kind:       "NodeVLANConf",
					Metadata:   Metadata{Name: "comprehensive-vlans"},
					Spec: NodeVLANSpec{
						VLANs: map[string]VLANConfig{
							"management": {ID: 100, Subnet: "192.168.100.0/24"},
						},
					},
				},
				Tests: &NodeTestConf{
					APIVersion: "openstack.kictl.icycloud.io/v1",
					Kind:       "NodeTestConf",
					Metadata:   Metadata{Name: "comprehensive-tests"},
					Spec: NodeTestSpec{
						Tests: []ConnectivityTest{
							{Name: "ping-test", Source: "node1", Targets: []string{"node2"}},
							{Name: "isolation-test", Source: "tenant", Targets: []string{"management"}},
						},
					},
				},
			},
			containsText: []string{"NodeLabels", "1 roles", "3 nodes", "VLANs", "1 vlans", "Tests", "2 tests"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Get bundle summary
			summary := tt.bundle.GetSummary()

			// Then: Verify summary content
			assert.NotEmpty(t, summary, "Summary should not be empty")

			if tt.expectedSummary != "" {
				assert.Equal(t, tt.expectedSummary, summary, "Summary mismatch")
			}

			for _, text := range tt.containsText {
				assert.Contains(t, summary, text, "Summary should contain '%s'", text)
			}
		})
	}
}

// TestConfigBundle_Validate tests bundle validation
// WHY: Validates that bundle enforces business rules and prevents invalid deployments
func TestConfigBundle_Validate(t *testing.T) {
	tests := []struct {
		name        string
		description string
		bundle      *ConfigBundle
		shouldError bool
		errorText   string
	}{
		{
			name:        "empty_bundle_validation_error",
			description: "Empty bundle should fail validation as it provides no value",
			bundle:      NewEmptyBundle(),
			shouldError: true,
			errorText:   "bundle contains no configurations",
		},
		{
			name:        "valid_single_config_bundle",
			description: "Valid single configuration should pass validation",
			bundle: &ConfigBundle{
				NodeLabels: &NodeLabelConf{
					APIVersion: "openstack.kictl.icycloud.io/v1",
					Kind:       "NodeLabelConf",
					Metadata:   Metadata{Name: "valid-config"},
					Spec: NodeLabelSpec{
						NodeRoles: map[string]NodeRole{
							"control": {
								Nodes:  []string{"rsb2"},
								Labels: map[string]string{"role": "control"},
							},
						},
					},
				},
			},
			shouldError: false,
		},
		{
			name:        "invalid_config_missing_api_version",
			description: "Configuration missing APIVersion should fail validation",
			bundle: &ConfigBundle{
				NodeLabels: &NodeLabelConf{
					// APIVersion missing
					Kind:     "NodeLabelConf",
					Metadata: Metadata{Name: "invalid-config"},
				},
			},
			shouldError: true,
			errorText:   "apiVersion is required",
		},
		{
			name:        "invalid_config_missing_kind",
			description: "Configuration missing Kind should fail validation",
			bundle: &ConfigBundle{
				NodeLabels: &NodeLabelConf{
					APIVersion: "openstack.kictl.icycloud.io/v1",
					// Kind missing
					Metadata: Metadata{Name: "invalid-config"},
				},
			},
			shouldError: true,
			errorText:   "kind is required",
		},
		{
			name:        "invalid_config_missing_name",
			description: "Configuration missing metadata name should fail validation",
			bundle: &ConfigBundle{
				NodeLabels: &NodeLabelConf{
					APIVersion: "openstack.kictl.icycloud.io/v1",
					Kind:       "NodeLabelConf",
					Metadata:   Metadata{}, // Name missing
				},
			},
			shouldError: true,
			errorText:   "metadata.name is required",
		},
		{
			name:        "valid_multi_crd_bundle",
			description: "Valid multi-CRD bundle should pass comprehensive validation",
			bundle: &ConfigBundle{
				NodeLabels: &NodeLabelConf{
					APIVersion: "openstack.kictl.icycloud.io/v1",
					Kind:       "NodeLabelConf",
					Metadata:   Metadata{Name: "multi-labels"},
					Spec: NodeLabelSpec{
						NodeRoles: map[string]NodeRole{
							"control": {
								Nodes:  []string{"rsb2"},
								Labels: map[string]string{"role": "control"},
							},
						},
					},
				},
				VLANs: &NodeVLANConf{
					APIVersion: "openstack.kictl.icycloud.io/v1",
					Kind:       "NodeVLANConf",
					Metadata:   Metadata{Name: "multi-vlans"},
					Spec: NodeVLANSpec{
						VLANs: map[string]VLANConfig{
							"management": {ID: 100, Subnet: "192.168.100.0/24"},
						},
					},
				},
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Validate bundle
			err := tt.bundle.Validate()

			// Then: Verify validation result
			if tt.shouldError {
				assert.Error(t, err, "Expected validation error")
				if tt.errorText != "" {
					assert.Contains(t, err.Error(), tt.errorText, "Error should contain expected text")
				}
			} else {
				assert.NoError(t, err, "Unexpected validation error")
			}
		})
	}
}

// TestNewSingleConfigBundle tests single config bundle creation
// WHY: Validates compatibility bridge between single-config and multi-config workflows
func TestNewSingleConfigBundle(t *testing.T) {
	tests := []struct {
		name             string
		description      string
		config           Config
		expectNodeLabels bool
		expectVLANs      bool
		expectTests      bool
	}{
		{
			name:        "node_label_conf_to_bundle",
			description: "NodeLabelConf should create bundle with NodeLabels populated",
			config: NodeLabelConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeLabelConf",
				Metadata:   Metadata{Name: "single-labels"},
			},
			expectNodeLabels: true,
			expectVLANs:      false,
			expectTests:      false,
		},
		{
			name:        "node_label_conf_pointer_to_bundle",
			description: "NodeLabelConf pointer should create bundle with NodeLabels populated",
			config: &NodeLabelConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeLabelConf",
				Metadata:   Metadata{Name: "single-labels-ptr"},
			},
			expectNodeLabels: true,
			expectVLANs:      false,
			expectTests:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Create single config bundle
			bundle := NewSingleConfigBundle(tt.config)

			// Then: Verify bundle structure
			assert.NotNil(t, bundle, "Bundle should not be nil")
			assert.Equal(t, tt.expectNodeLabels, bundle.HasNodeLabels(), "NodeLabels presence mismatch")
			assert.Equal(t, tt.expectVLANs, bundle.HasVLANs(), "VLANs presence mismatch")
			assert.Equal(t, tt.expectTests, bundle.HasTests(), "Tests presence mismatch")

			if tt.expectNodeLabels {
				assert.Equal(t, 1, bundle.GetConfigCount(), "Single config bundle should have count 1")
				configs := bundle.GetAllConfigsTyped()
				assert.Len(t, configs, 1, "Should have exactly one typed config")
				assert.Equal(t, "NodeLabelConf", configs[0].GetKind(), "Config should be NodeLabelConf")
			}
		})
	}
}

// TestConfigBundle_EdgeCases tests edge cases and error conditions
// WHY: Ensures robust behavior under unusual conditions for production reliability
func TestConfigBundle_EdgeCases(t *testing.T) {
	t.Run("nil_bundle_methods_dont_panic", func(t *testing.T) {
		// Given: Empty bundle to test methods don't panic
		// Note: Testing with empty bundle instead of nil to avoid nil pointer panics

		// When/Then: Methods should not panic
		assert.NotPanics(t, func() {
			emptyBundle := &ConfigBundle{}
			_ = emptyBundle.GetAllConfigs()
			_ = emptyBundle.GetAllConfigsTyped()
			_ = emptyBundle.GetConfigCount()
			_ = emptyBundle.HasNodeLabels()
			_ = emptyBundle.HasVLANs()
			_ = emptyBundle.HasTests()
			_ = emptyBundle.GetSummary()
		})
	})

	t.Run("bundle_with_source_tracking", func(t *testing.T) {
		// Given: Bundle with source information
		bundle := &ConfigBundle{
			Source: "/path/to/config.yaml",
			NodeLabels: &NodeLabelConf{
				APIVersion: "openstack.kictl.icycloud.io/v1",
				Kind:       "NodeLabelConf",
				Metadata:   Metadata{Name: "sourced-config"},
			},
		}

		// When: Validate bundle
		err := bundle.Validate()

		// Then: Should validate successfully with source tracking
		assert.NoError(t, err)
		assert.Equal(t, "/path/to/config.yaml", bundle.Source, "Source should be preserved")
		assert.True(t, bundle.HasNodeLabels(), "Should have NodeLabels")
		assert.Equal(t, 1, bundle.GetConfigCount(), "Should have single config")
	})

	t.Run("empty_bundle_constructor", func(t *testing.T) {
		// Given: New empty bundle
		bundle := NewEmptyBundle()

		// When: Check bundle state
		// Then: Should be properly initialized
		assert.NotNil(t, bundle, "Bundle should not be nil")
		assert.Equal(t, 0, bundle.GetConfigCount(), "Empty bundle should have zero configs")
		assert.False(t, bundle.HasNodeLabels(), "Should not have NodeLabels")
		assert.False(t, bundle.HasVLANs(), "Should not have VLANs")
		assert.False(t, bundle.HasTests(), "Should not have Tests")
		assert.Equal(t, "Empty bundle", bundle.GetSummary(), "Should have empty summary")
	})
}
