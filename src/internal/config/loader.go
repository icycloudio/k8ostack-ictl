package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadConfig loads configuration from file with automatic format detection
// Supports NodeLabelConf only - our clean, preferred format
func LoadConfig(configPath string) (Config, error) {
	if configPath == "" {
		return nil, fmt.Errorf("configuration file is required")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	// Try to determine format by kind
	var kindDetector struct {
		Kind string `yaml:"kind"`
	}
	if err := yaml.Unmarshal(data, &kindDetector); err != nil {
		return nil, fmt.Errorf("failed to parse config: invalid YAML: %w", err)
	}

	switch kindDetector.Kind {
	case "NodeLabelConf":
		return loadNodeLabelConf(data)
	case "NodeVLANConf":
		cfg, err := loadNodeVLANConf(data)
		if err != nil {
			return nil, err
		}
		return cfg, nil
	case "NodeTestConf":
		cfg, err := loadNodeTestConf(data)
		if err != nil {
			return nil, err
		}
		return cfg, nil
	default:
		return nil, fmt.Errorf("unsupported config kind '%s'. Expected: NodeLabelConf, NodeVLANConf, or NodeTestConf", kindDetector.Kind)
	}
}

// LoadMultipleConfigs loads configuration from file supporting both single and multi-document YAML
// This is the primary entry point for our unified architecture
func LoadMultipleConfigs(configPath string) (*ConfigBundle, error) {
	if configPath == "" {
		return nil, fmt.Errorf("configuration file is required")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	bundle := NewEmptyBundle()
	bundle.Source = configPath

	// Check if this is a multi-document YAML
	if isMultiDocumentYAML(data) {
		return loadMultiDocumentBundle(data, bundle)
	}

	// Single document - use existing logic but wrap in bundle
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	return NewSingleConfigBundle(cfg), nil
}

// loadMultiDocumentBundle processes multiple YAML documents into a ConfigBundle
func loadMultiDocumentBundle(data []byte, bundle *ConfigBundle) (*ConfigBundle, error) {
	documents, err := splitYAMLDocuments(data)
	if err != nil {
		return nil, fmt.Errorf("failed to split YAML documents: %w", err)
	}

	for i, doc := range documents {
		if err := validateYAMLDocument(doc); err != nil {
			return nil, fmt.Errorf("invalid YAML document %d: %w", i+1, err)
		}

		var kindDetector struct {
			Kind string `yaml:"kind"`
		}

		if err := yaml.Unmarshal(doc, &kindDetector); err != nil {
			return nil, fmt.Errorf("failed to detect kind in document %d: %w", i+1, err)
		}

		switch kindDetector.Kind {
		case "NodeLabelConf":
			cfg, err := loadNodeLabelConf(doc)
			if err != nil {
				return nil, fmt.Errorf("failed to load NodeLabelConf in document %d: %w", i+1, err)
			}
			if nodeLabelConf, ok := cfg.(*NodeLabelConf); ok {
				bundle.NodeLabels = nodeLabelConf
			}

		case "NodeVLANConf":
			cfg, err := loadNodeVLANConf(doc)
			if err != nil {
				return nil, fmt.Errorf("failed to load NodeVLANConf in document %d: %w", i+1, err)
			}
			bundle.VLANs = cfg

		case "NodeTestConf":
			cfg, err := loadNodeTestConf(doc)
			if err != nil {
				return nil, fmt.Errorf("failed to load NodeTestConf in document %d: %w", i+1, err)
			}
			bundle.Tests = cfg

		default:
			return nil, fmt.Errorf("unsupported config kind '%s' in document %d. Expected: NodeLabelConf, NodeVLANConf, NodeTestConf", kindDetector.Kind, i+1)
		}
	}

	if err := bundle.Validate(); err != nil {
		return nil, fmt.Errorf("bundle validation failed: %w", err)
	}

	return bundle, nil
}

// loadNodeVLANConf loads VLAN configuration
func loadNodeVLANConf(data []byte) (*NodeVLANConf, error) {
	var config NodeVLANConf
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse NodeVLANConf: %w", err)
	}

	if err := validateNodeVLANConf(config); err != nil {
		return nil, err
	}

	config = applyNodeVLANDefaults(config)
	return &config, nil
}

// loadNodeTestConf loads test configuration
func loadNodeTestConf(data []byte) (*NodeTestConf, error) {
	var config NodeTestConf
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse NodeTestConf: %w", err)
	}

	if err := validateNodeTestConf(config); err != nil {
		return nil, err
	}

	config = applyNodeTestDefaults(config)
	return &config, nil
}

// validateNodeVLANConf validates VLAN configuration
func validateNodeVLANConf(config NodeVLANConf) error {
	if config.Kind != "NodeVLANConf" {
		return fmt.Errorf("config kind must be 'NodeVLANConf', got '%s'", config.Kind)
	}

	if !strings.HasSuffix(config.APIVersion, "/v1") {
		return fmt.Errorf("config apiVersion must end with '/v1', got '%s'", config.APIVersion)
	}

	if config.Metadata.Name == "" {
		return fmt.Errorf("config metadata.name is required")
	}

	if len(config.Spec.VLANs) == 0 {
		return fmt.Errorf("config must contain at least one VLAN")
	}

	return nil
}

// validateNodeTestConf validates test configuration
func validateNodeTestConf(config NodeTestConf) error {
	if config.Kind != "NodeTestConf" {
		return fmt.Errorf("config kind must be 'NodeTestConf', got '%s'", config.Kind)
	}

	if !strings.HasSuffix(config.APIVersion, "/v1") {
		return fmt.Errorf("config apiVersion must end with '/v1', got '%s'", config.APIVersion)
	}

	if config.Metadata.Name == "" {
		return fmt.Errorf("config metadata.name is required")
	}

	if len(config.Spec.Tests) == 0 {
		return fmt.Errorf("config must contain at least one test")
	}

	return nil
}

// applyNodeVLANDefaults applies default values to NodeVLANConf
func applyNodeVLANDefaults(config NodeVLANConf) NodeVLANConf {
	// Set default namespace if not specified
	if config.Metadata.Namespace == "" {
		config.Metadata.Namespace = "default"
	}

	// Apply VLAN-specific defaults
	for vlanName, vlanConfig := range config.Spec.VLANs {
		if vlanConfig.Interface == "" {
			vlanConfig.Interface = "eth0" // Default interface
			config.Spec.VLANs[vlanName] = vlanConfig
		}
	}

	return config
}

// applyNodeTestDefaults applies default values to NodeTestConf
func applyNodeTestDefaults(config NodeTestConf) NodeTestConf {
	// Set default namespace if not specified
	if config.Metadata.Namespace == "" {
		config.Metadata.Namespace = "default"
	}

	// Apply test-specific defaults
	for i, test := range config.Spec.Tests {
		if test.Timeout == 0 {
			config.Spec.Tests[i].Timeout = 30 // Default 30 seconds
		}
		if !test.ExpectSuccess {
			config.Spec.Tests[i].ExpectSuccess = true // Default expect success
		}
	}

	return config
}

// loadNodeLabelConf loads the CRD-based node label configuration
func loadNodeLabelConf(data []byte) (Config, error) {
	var config NodeLabelConf
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse NodeLabelConf: %w", err)
	}

	if err := validateNodeLabelConf(config); err != nil {
		return nil, err
	}

	config = applyNodeLabelDefaults(config)
	return &config, nil
}

// validateNodeLabelConf validates the NodeLabelConf configuration format
func validateNodeLabelConf(config NodeLabelConf) error {
	if config.Kind != "NodeLabelConf" {
		return fmt.Errorf("config kind must be 'NodeLabelConf', got '%s'", config.Kind)
	}

	if !strings.HasSuffix(config.APIVersion, "/v1") {
		return fmt.Errorf("config apiVersion must end with '/v1', got '%s'", config.APIVersion)
	}

	if config.Metadata.Name == "" {
		return fmt.Errorf("config metadata.name is required")
	}

	if len(config.Spec.NodeRoles) == 0 {
		return fmt.Errorf("config must contain at least one node role")
	}

	return nil
}

// applyNodeLabelDefaults applies default values to NodeLabelConf
func applyNodeLabelDefaults(config NodeLabelConf) NodeLabelConf {
	// Apply tool defaults if not specified
	if config.Tools.Nlabel.LogLevel == "" {
		config.Tools.Nlabel.LogLevel = "info"
	}
	if !config.Tools.Nlabel.ValidateNodes {
		config.Tools.Nlabel.ValidateNodes = true
	}

	// Set default namespace if not specified
	if config.Metadata.Namespace == "" {
		config.Metadata.Namespace = "default"
	}

	return config
}

// GetDefaultNodeLabelConf returns the default node label configuration
func GetDefaultNodeLabelConf() NodeLabelConf {
	return NodeLabelConf{
		APIVersion: "openstack.kictl.icycloud.io/v1",
		Kind:       "NodeLabelConf",
		Metadata: Metadata{
			Name:      "production-node-labels",
			Namespace: "openstack",
			Labels: map[string]string{
				"environment": "production",
				"region":      "datacenter",
			},
		},
		Spec: NodeLabelSpec{
			NodeRoles: map[string]NodeRole{
				"controlPlane": {
					Nodes: []string{"server-01", "server-02", "server-03"},
					Labels: map[string]string{
						"openstack-control-plane":   "enabled",
						"openstack-role":            "control-plane",
						"cluster.openstack.io/role": "control-plane",
					},
					Description: "OpenStack control plane services (Nova API, Keystone, etc.)",
				},
				"storage": {
					Nodes: []string{"server-04", "server-05"},
					Labels: map[string]string{
						"openstack-storage-node":    "enabled",
						"openstack-role":            "storage",
						"ceph-node":                 "enabled",
						"cluster.openstack.io/role": "storage",
					},
					Description: "Dedicated storage nodes for Ceph cluster",
				},
				"compute": {
					Nodes: []string{"server-06", "server-07"},
					Labels: map[string]string{
						"openstack-compute-node":    "enabled",
						"openstack-role":            "compute",
						"nova-compute":              "enabled",
						"cluster.openstack.io/role": "compute",
					},
					Description: "Compute nodes for VM workloads and nested Kubernetes",
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
}

// GetDefaultNodeVLANConf returns the default VLAN configuration
func GetDefaultNodeVLANConf() NodeVLANConf {
	return NodeVLANConf{
		APIVersion: "openstack.kictl.icycloud.io/v1",
		Kind:       "NodeVLANConf",
		Metadata: Metadata{
			Name:      "production-vlans",
			Namespace: "openstack",
			Labels: map[string]string{
				"environment": "production",
				"region":      "datacenter",
			},
		},
		Spec: NodeVLANSpec{
			VLANs: map[string]VLANConfig{
				"management": {
					ID:        100,
					Subnet:    "10.1.100.0/24",
					Interface: "eth0",
					NodeMapping: map[string]string{
						"server-01": "10.1.100.11",
						"server-02": "10.1.100.12",
						"server-03": "10.1.100.13",
					},
				},
				"storage": {
					ID:        200,
					Subnet:    "10.1.200.0/24",
					Interface: "eth1",
					NodeMapping: map[string]string{
						"server-04": "10.1.200.14",
						"server-05": "10.1.200.15",
					},
				},
			},
		},
		Tools: Tools{
			Nvlan: ToolConfig{
				DryRun:   false,
				LogLevel: "info",
			},
		},
	}
}

// GetDefaultNodeTestConf returns the default test configuration
func GetDefaultNodeTestConf() NodeTestConf {
	return NodeTestConf{
		APIVersion: "openstack.kictl.icycloud.io/v1",
		Kind:       "NodeTestConf",
		Metadata: Metadata{
			Name:      "production-tests",
			Namespace: "openstack",
			Labels: map[string]string{
				"environment": "production",
				"region":      "datacenter",
			},
		},
		Spec: NodeTestSpec{
			Tests: []ConnectivityTest{
				{
					Name:          "management-reachability",
					Description:   "Test management network connectivity",
					Source:        "management",
					Targets:       []string{"storage"},
					Timeout:       30,
					ExpectSuccess: true,
				},
				{
					Name:          "storage-isolation",
					Description:   "Verify storage network isolation",
					Source:        "storage",
					Targets:       []string{"external"},
					Timeout:       30,
					ExpectSuccess: false,
				},
			},
		},
		Tools: Tools{
			Ntest: ToolConfig{
				DryRun:   false,
				LogLevel: "info",
			},
		},
	}
}

// GenerateMultiCRDSampleConfig creates a sample multi-CRD configuration file
func GenerateMultiCRDSampleConfig(filename string) error {
	// Get all default configurations
	nodeLabels := GetDefaultNodeLabelConf()
	vlans := GetDefaultNodeVLANConf()
	tests := GetDefaultNodeTestConf()

	// Create YAML documents
	var documents [][]byte

	// Marshal each configuration
	labelData, err := yaml.Marshal(nodeLabels)
	if err != nil {
		return fmt.Errorf("failed to marshal node labels config: %w", err)
	}
	documents = append(documents, labelData)

	vlanData, err := yaml.Marshal(vlans)
	if err != nil {
		return fmt.Errorf("failed to marshal VLAN config: %w", err)
	}
	documents = append(documents, vlanData)

	testData, err := yaml.Marshal(tests)
	if err != nil {
		return fmt.Errorf("failed to marshal test config: %w", err)
	}
	documents = append(documents, testData)

	// Combine documents with YAML separator
	var combinedData []byte
	for i, doc := range documents {
		if i > 0 {
			combinedData = append(combinedData, []byte("---\n")...)
		}
		combinedData = append(combinedData, doc...)
	}

	if err := os.WriteFile(filename, combinedData, 0644); err != nil {
		return fmt.Errorf("failed to write multi-CRD config: %w", err)
	}

	return nil
}

// GenerateSampleConfig creates a sample configuration file
func GenerateSampleConfig(filename string) error {
	config := GetDefaultNodeLabelConf()

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal sample config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write sample config: %w", err)
	}

	return nil
}
