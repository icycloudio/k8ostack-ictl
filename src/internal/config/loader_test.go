// Package config provides unit tests for configuration loading and processing
// WHY: Loader is the critical entry point for all configurations and must handle diverse file formats and edge cases
package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLoadConfig tests single configuration loading
// WHY: Validates the primary config loading path that must work reliably in production
func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		description string
		configData  string
		expectKind  string
		shouldError bool
		errorText   string
	}{
		{
			name:        "valid_node_label_conf_loading",
			description: "Valid NodeLabelConf should load successfully for production use",
			configData: `apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeLabelConf
metadata:
  name: production-labels
  namespace: openstack
spec:
  nodeRoles:
    control:
      nodes: ["rsb2", "rsb3"]
      labels:
        role: "control"
        openstack-role: "control-plane"
    storage:
      nodes: ["rsb5", "rsb6"]
      labels:
        role: "storage"
        ceph-node: "enabled"
tools:
  nlabel:
    dryRun: false
    validateNodes: true
    logLevel: "info"`,
			expectKind:  "NodeLabelConf",
			shouldError: false,
		},
		{
			name:        "minimal_valid_node_label_conf",
			description: "Minimal valid NodeLabelConf should load with defaults applied",
			configData: `apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeLabelConf
metadata:
  name: minimal-config
spec:
  nodeRoles:
    worker:
      nodes: ["rsb7"]
      labels:
        role: "worker"`,
			expectKind:  "NodeLabelConf",
			shouldError: false,
		},
		{
			name:        "unsupported_kind_error",
			description: "Unsupported configuration kind should fail with clear error",
			configData: `apiVersion: openstack.kictl.icycloud.io/v1
kind: UnsupportedConf
metadata:
  name: unsupported`,
			shouldError: true,
			errorText:   "unsupported config kind 'UnsupportedConf'",
		},
		{
			name:        "invalid_yaml_syntax_error",
			description: "Invalid YAML syntax should fail with parsing error",
			configData: `apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeLabelConf
metadata:
  name: "unclosed quote
spec:
  invalid yaml`,
			shouldError: true,
			errorText:   "invalid YAML",
		},
		{
			name:        "missing_required_fields_error",
			description: "Configuration missing required fields should fail validation",
			configData: `apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeLabelConf
metadata:
  name: ""
spec:
  nodeRoles: {}`,
			shouldError: true,
			errorText:   "metadata.name is required",
		},
		{
			name:        "invalid_api_version_error",
			description: "Invalid API version should fail validation",
			configData: `apiVersion: invalid-version
kind: NodeLabelConf
metadata:
  name: invalid-api`,
			shouldError: true,
			errorText:   "apiVersion must end with '/v1'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Temporary config file
			tempDir := t.TempDir()
			configFile := filepath.Join(tempDir, "config.yaml")
			err := os.WriteFile(configFile, []byte(tt.configData), 0644)
			assert.NoError(t, err, "Failed to create temp config file")

			// When: Load configuration
			config, err := LoadConfig(configFile)

			// Then: Verify loading result
			if tt.shouldError {
				assert.Error(t, err, "Expected loading error")
				assert.Nil(t, config, "Config should be nil on error")
				if tt.errorText != "" {
					assert.Contains(t, err.Error(), tt.errorText, "Error should contain expected text")
				}
			} else {
				assert.NoError(t, err, "Unexpected loading error")
				assert.NotNil(t, config, "Config should not be nil")
				assert.Equal(t, tt.expectKind, config.GetKind(), "Kind mismatch")
				assert.Equal(t, "openstack.kictl.icycloud.io/v1", config.GetAPIVersion(), "API version mismatch")
			}
		})
	}
}

// TestLoadConfig_EdgeCases tests edge cases for single config loading
// WHY: Production systems must handle edge cases gracefully without crashes
func TestLoadConfig_EdgeCases(t *testing.T) {
	t.Run("empty_config_path_error", func(t *testing.T) {
		// Given: Empty config path
		// When: Load configuration
		config, err := LoadConfig("")

		// Then: Should return appropriate error
		assert.Error(t, err, "Empty path should return error")
		assert.Nil(t, config, "Config should be nil")
		assert.Contains(t, err.Error(), "configuration file is required", "Should indicate missing file")
	})

	t.Run("nonexistent_file_error", func(t *testing.T) {
		// Given: Non-existent file path
		nonExistentPath := "/tmp/does-not-exist-" + t.Name() + ".yaml"

		// When: Load configuration
		config, err := LoadConfig(nonExistentPath)

		// Then: Should return file read error
		assert.Error(t, err, "Non-existent file should return error")
		assert.Nil(t, config, "Config should be nil")
		assert.Contains(t, err.Error(), "failed to read config file", "Should indicate read failure")
	})

	t.Run("empty_file_error", func(t *testing.T) {
		// Given: Empty config file
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "empty.yaml")
		err := os.WriteFile(configFile, []byte(""), 0644)
		assert.NoError(t, err)

		// When: Load configuration
		config, err := LoadConfig(configFile)

		// Then: Should return parsing error
		assert.Error(t, err, "Empty file should return error")
		assert.Nil(t, config, "Config should be nil")
	})
}

// TestLoadMultipleConfigs tests multi-document configuration loading
// WHY: Multi-CRD configurations enable unified infrastructure deployment and must work flawlessly
func TestLoadMultipleConfigs(t *testing.T) {
	tests := []struct {
		name             string
		description      string
		configData       string
		expectNodeLabels bool
		expectVLANs      bool
		expectTests      bool
		expectCount      int
		shouldError      bool
		errorText        string
	}{
		{
			name:        "single_document_node_labels",
			description: "Single NodeLabelConf document should load into bundle successfully",
			configData: `apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeLabelConf
metadata:
  name: single-labels
spec:
  nodeRoles:
    control:
      nodes: ["rsb2"]
      labels:
        role: "control"`,
			expectNodeLabels: true,
			expectVLANs:      false,
			expectTests:      false,
			expectCount:      1,
			shouldError:      false,
		},
		{
			name:        "complete_multi_crd_deployment",
			description: "Complete multi-CRD configuration should enable unified infrastructure deployment",
			configData: `apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeLabelConf
metadata:
  name: production-labels
spec:
  nodeRoles:
    control:
      nodes: ["rsb2", "rsb3"]
      labels:
        role: "control"
    storage:
      nodes: ["rsb5", "rsb6"]
      labels:
        role: "storage"
---
apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeVLANConf
metadata:
  name: production-vlans
spec:
  vlans:
    management:
      id: 100
      subnet: "192.168.100.0/24"
      nodeMapping:
        rsb2: "192.168.100.12"
        rsb3: "192.168.100.13"
    storage:
      id: 200
      subnet: "192.168.200.0/24"
      nodeMapping:
        rsb5: "192.168.200.15"
        rsb6: "192.168.200.16"
---
apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeTestConf
metadata:
  name: production-tests
spec:
  tests:
    - name: management-connectivity
      source: rsb2
      targets: ["rsb3"]
      timeout: 30
      expectSuccess: true
    - name: storage-isolation
      source: storage
      targets: ["management"]
      expectSuccess: false`,
			expectNodeLabels: true,
			expectVLANs:      true,
			expectTests:      true,
			expectCount:      3,
			shouldError:      false,
		},
		{
			name:        "partial_multi_crd_vlans_and_tests",
			description: "Partial multi-CRD with VLANs and Tests should load successfully",
			configData: `apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeVLANConf
metadata:
  name: network-config
spec:
  vlans:
    tenant:
      id: 300
      subnet: "10.0.0.0/24"
      nodeMapping:
        rsb4: "10.0.0.14"
---
apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeTestConf
metadata:
  name: network-tests
spec:
  tests:
    - name: tenant-connectivity
      source: rsb4
      targets: ["external"]`,
			expectNodeLabels: false,
			expectVLANs:      true,
			expectTests:      true,
			expectCount:      2,
			shouldError:      false,
		},
		{
			name:        "invalid_document_in_multi_yaml",
			description: "Invalid document in multi-YAML should fail with clear error",
			configData: `apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeLabelConf
metadata:
  name: valid-config
spec:
  nodeRoles:
    control:
      nodes: ["rsb2"]
      labels:
        role: "control"
---
apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeVLANConf
metadata:
  name: ""
spec:
  vlans: {}`,
			shouldError: true,
			errorText:   "metadata.name is required",
		},
		{
			name:        "unsupported_kind_in_multi_yaml",
			description: "Unsupported CRD kind in multi-YAML should fail with clear error",
			configData: `apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeLabelConf
metadata:
  name: valid-config
spec:
  nodeRoles:
    control:
      nodes: ["rsb2"]
      labels:
        role: "control"
---
apiVersion: openstack.kictl.icycloud.io/v1
kind: UnsupportedConf
metadata:
  name: unsupported`,
			shouldError: true,
			errorText:   "unsupported config kind 'UnsupportedConf'",
		},
		{
			name:        "duplicate_crd_types_error",
			description: "Duplicate CRD types in multi-YAML should override (last wins)",
			configData: `apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeLabelConf
metadata:
  name: first-labels
spec:
  nodeRoles:
    control:
      nodes: ["rsb2"]
      labels:
        role: "control"
---
apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeLabelConf
metadata:
  name: second-labels
spec:
  nodeRoles:
    storage:
      nodes: ["rsb5"]
      labels:
        role: "storage"`,
			expectNodeLabels: true,
			expectVLANs:      false,
			expectTests:      false,
			expectCount:      1,
			shouldError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Temporary multi-config file
			tempDir := t.TempDir()
			configFile := filepath.Join(tempDir, "multi-config.yaml")
			err := os.WriteFile(configFile, []byte(tt.configData), 0644)
			assert.NoError(t, err, "Failed to create temp config file")

			// When: Load multiple configurations
			bundle, err := LoadMultipleConfigs(configFile)

			// Then: Verify loading result
			if tt.shouldError {
				assert.Error(t, err, "Expected loading error")
				assert.Nil(t, bundle, "Bundle should be nil on error")
				if tt.errorText != "" {
					assert.Contains(t, err.Error(), tt.errorText, "Error should contain expected text")
				}
			} else {
				assert.NoError(t, err, "Unexpected loading error")
				assert.NotNil(t, bundle, "Bundle should not be nil")
				assert.Equal(t, tt.expectCount, bundle.GetConfigCount(), "Config count mismatch")
				assert.Equal(t, tt.expectNodeLabels, bundle.HasNodeLabels(), "NodeLabels presence mismatch")
				assert.Equal(t, tt.expectVLANs, bundle.HasVLANs(), "VLANs presence mismatch")
				assert.Equal(t, tt.expectTests, bundle.HasTests(), "Tests presence mismatch")

				// Verify bundle validation passes
				err = bundle.Validate()
				assert.NoError(t, err, "Loaded bundle should be valid")
			}
		})
	}
}

// TestLoadMultipleConfigs_EdgeCases tests edge cases for multi-config loading
// WHY: Edge case handling prevents production failures and ensures robust operation
func TestLoadMultipleConfigs_EdgeCases(t *testing.T) {
	t.Run("empty_path_error", func(t *testing.T) {
		// Given: Empty config path
		// When: Load multiple configurations
		bundle, err := LoadMultipleConfigs("")

		// Then: Should return appropriate error
		assert.Error(t, err, "Empty path should return error")
		assert.Nil(t, bundle, "Bundle should be nil")
		assert.Contains(t, err.Error(), "configuration file is required", "Should indicate missing file")
	})

	t.Run("malformed_multi_document_yaml", func(t *testing.T) {
		// Given: Malformed multi-document YAML
		malformedYAML := `apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeLabelConf
metadata:
  name: valid-start
---
invalid yaml content
  missing: proper structure
    bad: indentation
---
apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeVLANConf
metadata:
  name: valid-end`

		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "malformed.yaml")
		err := os.WriteFile(configFile, []byte(malformedYAML), 0644)
		assert.NoError(t, err)

		// When: Load multiple configurations
		bundle, err := LoadMultipleConfigs(configFile)

		// Then: Should return parsing error
		assert.Error(t, err, "Malformed YAML should return error")
		assert.Nil(t, bundle, "Bundle should be nil")
	})

	t.Run("empty_bundle_validation_error", func(t *testing.T) {
		// Given: Multi-document YAML that results in empty bundle
		emptyBundleYAML := `---
# Comment only document
---
   
   
---
# Another comment only`

		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "empty-bundle.yaml")
		err := os.WriteFile(configFile, []byte(emptyBundleYAML), 0644)
		assert.NoError(t, err)

		// When: Load multiple configurations
		bundle, err := LoadMultipleConfigs(configFile)

		// Then: Should return validation error
		assert.Error(t, err, "Empty bundle should return validation error")
		assert.Nil(t, bundle, "Bundle should be nil")
		assert.Contains(t, err.Error(), "invalid YAML document", "Should indicate empty content")
	})
}

// TestLoadNodeVLANConf tests VLAN configuration loading
// WHY: VLAN configurations are critical for OpenStack networking and must validate properly
func TestLoadNodeVLANConf(t *testing.T) {
	tests := []struct {
		name        string
		description string
		configData  string
		expectValid bool
		errorText   string
	}{
		{
			name:        "complete_vlan_configuration",
			description: "Complete VLAN configuration should load successfully for production networking",
			configData: `apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeVLANConf
metadata:
  name: production-vlans
  namespace: openstack
spec:
  vlans:
    management:
      id: 100
      subnet: "192.168.100.0/24"
      interface: "eth0"
      nodeMapping:
        rsb2: "192.168.100.12"
        rsb3: "192.168.100.13"
    storage:
      id: 200
      subnet: "192.168.200.0/24"
      interface: "eth1"
      nodeMapping:
        rsb5: "192.168.200.15"
        rsb6: "192.168.200.16"`,
			expectValid: true,
		},
		{
			name:        "minimal_vlan_configuration",
			description: "Minimal VLAN configuration should load with defaults applied",
			configData: `apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeVLANConf
metadata:
  name: minimal-vlans
spec:
  vlans:
    tenant:
      id: 300
      subnet: "10.0.0.0/24"
      nodeMapping:
        rsb4: "10.0.0.14"`,
			expectValid: true,
		},
		{
			name:        "invalid_vlan_kind",
			description: "Invalid VLAN kind should fail validation",
			configData: `apiVersion: openstack.kictl.icycloud.io/v1
kind: WrongKind
metadata:
  name: invalid-kind`,
			expectValid: false,
			errorText:   "config kind must be 'NodeVLANConf'",
		},
		{
			name:        "missing_vlans_error",
			description: "VLAN configuration without VLANs should fail validation",
			configData: `apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeVLANConf
metadata:
  name: no-vlans
spec:
  vlans: {}`,
			expectValid: false,
			errorText:   "config must contain at least one VLAN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Load VLAN configuration
			config, err := loadNodeVLANConf([]byte(tt.configData))

			// Then: Verify loading result
			if tt.expectValid {
				assert.NoError(t, err, "Unexpected error loading VLAN config")
				assert.NotNil(t, config, "Config should not be nil")
				assert.Equal(t, "NodeVLANConf", config.Kind, "Kind should be NodeVLANConf")
				assert.NotEmpty(t, config.Spec.VLANs, "Should have VLANs")
			} else {
				assert.Error(t, err, "Expected error loading VLAN config")
				assert.Nil(t, config, "Config should be nil on error")
				if tt.errorText != "" {
					assert.Contains(t, err.Error(), tt.errorText, "Error should contain expected text")
				}
			}
		})
	}
}

// TestLoadNodeTestConf tests connectivity test configuration loading
// WHY: Test configurations validate infrastructure connectivity and must load correctly
func TestLoadNodeTestConf(t *testing.T) {
	tests := []struct {
		name        string
		description string
		configData  string
		expectValid bool
		errorText   string
	}{
		{
			name:        "comprehensive_test_configuration",
			description: "Comprehensive test configuration should load for infrastructure validation",
			configData: `apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeTestConf
metadata:
  name: infrastructure-tests
  namespace: openstack
spec:
  tests:
    - name: management-connectivity
      description: "Test management network connectivity"
      source: management
      targets: ["storage", "compute"]
      timeout: 30
      expectSuccess: true
    - name: tenant-isolation
      description: "Verify tenant network isolation"
      source: tenant
      targets: ["management"]
      timeout: 10
      expectSuccess: false`,
			expectValid: true,
		},
		{
			name:        "minimal_test_configuration",
			description: "Minimal test configuration should load with defaults",
			configData: `apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeTestConf
metadata:
  name: basic-tests
spec:
  tests:
    - name: ping-test
      source: node1
      targets: ["node2"]`,
			expectValid: true,
		},
		{
			name:        "invalid_test_kind",
			description: "Invalid test kind should fail validation",
			configData: `apiVersion: openstack.kictl.icycloud.io/v1
kind: WrongKind
metadata:
  name: invalid-kind`,
			expectValid: false,
			errorText:   "config kind must be 'NodeTestConf'",
		},
		{
			name:        "missing_tests_error",
			description: "Test configuration without tests should fail validation",
			configData: `apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeTestConf
metadata:
  name: no-tests
spec:
  tests: []`,
			expectValid: false,
			errorText:   "config must contain at least one test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Load test configuration
			config, err := loadNodeTestConf([]byte(tt.configData))

			// Then: Verify loading result
			if tt.expectValid {
				assert.NoError(t, err, "Unexpected error loading test config")
				assert.NotNil(t, config, "Config should not be nil")
				assert.Equal(t, "NodeTestConf", config.Kind, "Kind should be NodeTestConf")
				assert.NotEmpty(t, config.Spec.Tests, "Should have tests")
			} else {
				assert.Error(t, err, "Expected error loading test config")
				assert.Nil(t, config, "Config should be nil on error")
				if tt.errorText != "" {
					assert.Contains(t, err.Error(), tt.errorText, "Error should contain expected text")
				}
			}
		})
	}
}

// TestApplyDefaults tests default value application
// WHY: Default values ensure configurations work with minimal specification and reasonable fallbacks
func TestApplyDefaults(t *testing.T) {
	t.Run("node_label_defaults_applied", func(t *testing.T) {
		// Given: Minimal NodeLabelConf
		config := NodeLabelConf{
			APIVersion: "openstack.kictl.icycloud.io/v1",
			Kind:       "NodeLabelConf",
			Metadata:   Metadata{Name: "minimal-config"},
			Spec: NodeLabelSpec{
				NodeRoles: map[string]NodeRole{
					"worker": {
						Nodes:  []string{"rsb7"},
						Labels: map[string]string{"role": "worker"},
					},
				},
			},
		}

		// When: Apply defaults
		result := applyNodeLabelDefaults(config)

		// Then: Verify defaults are applied
		assert.Equal(t, "default", result.Metadata.Namespace, "Should apply default namespace")
		assert.Equal(t, "info", result.Tools.Nlabel.LogLevel, "Should apply default log level")
		assert.True(t, result.Tools.Nlabel.ValidateNodes, "Should enable node validation by default")
		assert.False(t, result.Tools.Nlabel.DryRun, "Should disable dry-run by default")
	})

	t.Run("node_vlan_defaults_applied", func(t *testing.T) {
		// Given: Minimal NodeVLANConf
		config := NodeVLANConf{
			APIVersion: "openstack.kictl.icycloud.io/v1",
			Kind:       "NodeVLANConf",
			Metadata:   Metadata{Name: "minimal-vlans"},
			Spec: NodeVLANSpec{
				VLANs: map[string]VLANConfig{
					"tenant": {
						ID:     300,
						Subnet: "10.0.0.0/24",
						NodeMapping: map[string]string{
							"rsb4": "10.0.0.14",
						},
					},
				},
			},
		}

		// When: Apply defaults
		result := applyNodeVLANDefaults(config)

		// Then: Verify defaults are applied
		assert.Equal(t, "default", result.Metadata.Namespace, "Should apply default namespace")
	})

	t.Run("node_test_defaults_applied", func(t *testing.T) {
		// Given: Minimal NodeTestConf
		config := NodeTestConf{
			APIVersion: "openstack.kictl.icycloud.io/v1",
			Kind:       "NodeTestConf",
			Metadata:   Metadata{Name: "minimal-tests"},
			Spec: NodeTestSpec{
				Tests: []ConnectivityTest{
					{
						Name:    "basic-test",
						Source:  "node1",
						Targets: []string{"node2"},
					},
				},
			},
		}

		// When: Apply defaults
		result := applyNodeTestDefaults(config)

		// Then: Verify defaults are applied
		assert.Equal(t, "default", result.Metadata.Namespace, "Should apply default namespace")
	})
}

// TestGetDefaultConfigurations tests default configuration generation
// WHY: Default configurations provide starting templates and must be valid and useful
func TestGetDefaultConfigurations(t *testing.T) {
	t.Run("default_node_label_conf_valid", func(t *testing.T) {
		// When: Get default NodeLabelConf
		config := GetDefaultNodeLabelConf()

		// Then: Verify default configuration is valid
		assert.Equal(t, "openstack.kictl.icycloud.io/v1", config.APIVersion, "Should have correct API version")
		assert.Equal(t, "NodeLabelConf", config.Kind, "Should have correct kind")
		assert.NotEmpty(t, config.Metadata.Name, "Should have default name")
		assert.NotEmpty(t, config.Spec.NodeRoles, "Should have default node roles")

		// Verify it validates successfully
		err := validateNodeLabelConf(config)
		assert.NoError(t, err, "Default config should be valid")
	})

	t.Run("default_node_vlan_conf_valid", func(t *testing.T) {
		// When: Get default NodeVLANConf
		config := GetDefaultNodeVLANConf()

		// Then: Verify default configuration is valid
		assert.Equal(t, "openstack.kictl.icycloud.io/v1", config.APIVersion, "Should have correct API version")
		assert.Equal(t, "NodeVLANConf", config.Kind, "Should have correct kind")
		assert.NotEmpty(t, config.Metadata.Name, "Should have default name")
		assert.NotEmpty(t, config.Spec.VLANs, "Should have default VLANs")

		// Verify it validates successfully
		err := validateNodeVLANConf(config)
		assert.NoError(t, err, "Default config should be valid")
	})

	t.Run("default_node_test_conf_valid", func(t *testing.T) {
		// When: Get default NodeTestConf
		config := GetDefaultNodeTestConf()

		// Then: Verify default configuration is valid
		assert.Equal(t, "openstack.kictl.icycloud.io/v1", config.APIVersion, "Should have correct API version")
		assert.Equal(t, "NodeTestConf", config.Kind, "Should have correct kind")
		assert.NotEmpty(t, config.Metadata.Name, "Should have default name")
		assert.NotEmpty(t, config.Spec.Tests, "Should have default tests")

		// Verify it validates successfully
		err := validateNodeTestConf(config)
		assert.NoError(t, err, "Default config should be valid")
	})
}

// TestConfigurationValidation tests comprehensive validation logic
// WHY: Validation prevents invalid configurations from causing runtime failures in production
func TestConfigurationValidation(t *testing.T) {
	t.Run("comprehensive_node_label_validation", func(t *testing.T) {
		tests := []struct {
			name        string
			config      NodeLabelConf
			expectValid bool
			errorText   string
		}{
			{
				name: "valid_production_config",
				config: NodeLabelConf{
					APIVersion: "openstack.kictl.icycloud.io/v1",
					Kind:       "NodeLabelConf",
					Metadata:   Metadata{Name: "production"},
					Spec: NodeLabelSpec{
						NodeRoles: map[string]NodeRole{
							"control": {
								Nodes:  []string{"rsb2"},
								Labels: map[string]string{"role": "control"},
							},
						},
					},
				},
				expectValid: true,
			},
			{
				name: "invalid_kind",
				config: NodeLabelConf{
					APIVersion: "openstack.kictl.icycloud.io/v1",
					Kind:       "WrongKind",
					Metadata:   Metadata{Name: "invalid"},
				},
				expectValid: false,
				errorText:   "config kind must be 'NodeLabelConf'",
			},
			{
				name: "invalid_api_version",
				config: NodeLabelConf{
					APIVersion: "invalid-version",
					Kind:       "NodeLabelConf",
					Metadata:   Metadata{Name: "invalid-api"},
				},
				expectValid: false,
				errorText:   "apiVersion must end with '/v1'",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// When: Validate configuration
				err := validateNodeLabelConf(tt.config)

				// Then: Verify validation result
				if tt.expectValid {
					assert.NoError(t, err, "Expected valid configuration")
				} else {
					assert.Error(t, err, "Expected validation error")
					if tt.errorText != "" {
						assert.Contains(t, err.Error(), tt.errorText, "Error should contain expected text")
					}
				}
			})
		}
	})
}

// TestSampleConfigGeneration tests sample configuration generation
// WHY: Sample configs help users understand the format and provide working templates
func TestSampleConfigGeneration(t *testing.T) {
	t.Run("generate_single_sample_config", func(t *testing.T) {
		// Given: Temporary file for sample config
		tempDir := t.TempDir()
		sampleFile := filepath.Join(tempDir, "sample.yaml")

		// When: Generate sample configuration
		err := GenerateSampleConfig(sampleFile)

		// Then: Verify sample config generation
		assert.NoError(t, err, "Sample config generation should succeed")

		// Verify file was created and is readable
		data, err := os.ReadFile(sampleFile)
		assert.NoError(t, err, "Should be able to read generated sample")
		assert.NotEmpty(t, data, "Sample file should not be empty")

		// Verify generated sample can be loaded
		config, err := LoadConfig(sampleFile)
		assert.NoError(t, err, "Generated sample should be loadable")
		assert.NotNil(t, config, "Loaded sample should not be nil")
		assert.Equal(t, "NodeLabelConf", config.GetKind(), "Sample should be NodeLabelConf")
	})

	t.Run("generate_multi_crd_sample_config", func(t *testing.T) {
		// Given: Temporary file for multi-CRD sample
		tempDir := t.TempDir()
		sampleFile := filepath.Join(tempDir, "multi-sample.yaml")

		// When: Generate multi-CRD sample configuration
		err := GenerateMultiCRDSampleConfig(sampleFile)

		// Then: Verify multi-CRD sample generation
		assert.NoError(t, err, "Multi-CRD sample generation should succeed")

		// Verify file was created and is readable
		data, err := os.ReadFile(sampleFile)
		assert.NoError(t, err, "Should be able to read generated multi-sample")
		assert.NotEmpty(t, data, "Multi-sample file should not be empty")

		// Verify generated sample can be loaded as bundle
		bundle, err := LoadMultipleConfigs(sampleFile)
		assert.NoError(t, err, "Generated multi-sample should be loadable")
		assert.NotNil(t, bundle, "Loaded bundle should not be nil")
		assert.True(t, bundle.HasNodeLabels(), "Multi-sample should include NodeLabels")
		assert.True(t, bundle.HasVLANs(), "Multi-sample should include VLANs")
		assert.True(t, bundle.HasTests(), "Multi-sample should include Tests")
		assert.Equal(t, 3, bundle.GetConfigCount(), "Multi-sample should have all three CRDs")
	})
}
