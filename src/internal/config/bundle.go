// Package config provides the ConfigBundle for handling multiple CRD configurations
package config

import (
	"fmt"
	"strings"
)

// ConfigBundle holds multiple related configurations that can be processed together
// This enables single-manifest deployment of complex infrastructure setups
type ConfigBundle struct {
	NodeLabels *NodeLabelConf // Node labeling configuration
	VLANs      *NodeVLANConf  // VLAN configuration
	Tests      *NodeTestConf  // Connectivity testing configuration

	// Metadata about the bundle
	Source string // Path to the source configuration file
}

// GetAllConfigs returns all non-nil configurations in the bundle
// This enables uniform processing across different CRD types
func (b *ConfigBundle) GetAllConfigs() []interface{} {
	var configs []interface{}

	if b.NodeLabels != nil {
		configs = append(configs, b.NodeLabels)
	}
	if b.VLANs != nil {
		configs = append(configs, b.VLANs)
	}
	if b.Tests != nil {
		configs = append(configs, b.Tests)
	}

	return configs
}

// GetAllConfigsTyped returns all non-nil configurations as Config interface
// This enables type-safe operations while maintaining compatibility
func (b *ConfigBundle) GetAllConfigsTyped() []Config {
	var configs []Config

	if b.NodeLabels != nil {
		configs = append(configs, b.NodeLabels)
	}
	if b.VLANs != nil {
		configs = append(configs, b.VLANs)
	}
	if b.Tests != nil {
		configs = append(configs, b.Tests)
	}

	return configs
}

// GetConfigCount returns the number of configurations in the bundle
func (b *ConfigBundle) GetConfigCount() int {
	return len(b.GetAllConfigs())
}

// HasNodeLabels returns true if the bundle contains node labeling configuration
func (b *ConfigBundle) HasNodeLabels() bool {
	return b.NodeLabels != nil
}

// HasVLANs returns true if the bundle contains VLAN configuration
func (b *ConfigBundle) HasVLANs() bool {
	return b.VLANs != nil
}

// HasTests returns true if the bundle contains test configuration
func (b *ConfigBundle) HasTests() bool {
	return b.Tests != nil
}

// GetSummary returns a human-readable summary of the bundle contents
func (b *ConfigBundle) GetSummary() string {
	var parts []string

	if b.HasNodeLabels() {
		nodeCount := 0
		for _, role := range b.NodeLabels.Spec.NodeRoles {
			nodeCount += len(role.Nodes)
		}
		parts = append(parts, fmt.Sprintf("NodeLabels(%d roles, %d nodes)",
			len(b.NodeLabels.Spec.NodeRoles), nodeCount))
	}

	if b.HasVLANs() {
		parts = append(parts, fmt.Sprintf("VLANs(%d vlans)", len(b.VLANs.Spec.VLANs)))
	}

	if b.HasTests() {
		parts = append(parts, fmt.Sprintf("Tests(%d tests)", len(b.Tests.Spec.Tests)))
	}

	if len(parts) == 0 {
		return "Empty bundle"
	}

	return strings.Join(parts, ", ")
}

// Validate performs validation across all configurations in the bundle
func (b *ConfigBundle) Validate() error {
	if b.GetConfigCount() == 0 {
		return fmt.Errorf("bundle contains no configurations")
	}

	// Validate each individual configuration using the typed version
	for _, cfg := range b.GetAllConfigsTyped() {
		if err := b.validateConfig(cfg); err != nil {
			return fmt.Errorf("validation failed for %s: %w", cfg.GetKind(), err)
		}
	}

	// Cross-configuration validation can be added here
	// For example, ensuring VLAN node mappings match node label assignments

	return nil
}

// validateConfig performs basic validation on a single configuration
func (b *ConfigBundle) validateConfig(cfg Config) error {
	if cfg.GetAPIVersion() == "" {
		return fmt.Errorf("apiVersion is required")
	}

	if cfg.GetKind() == "" {
		return fmt.Errorf("kind is required")
	}

	metadata := cfg.GetMetadata()
	if metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}

	return nil
}

// NewEmptyBundle creates a new empty ConfigBundle
func NewEmptyBundle() *ConfigBundle {
	return &ConfigBundle{}
}

// NewSingleConfigBundle creates a bundle from a single configuration
// This provides compatibility with single-config workflows
func NewSingleConfigBundle(cfg Config) *ConfigBundle {
	bundle := NewEmptyBundle()

	switch c := cfg.(type) {
	case *NodeLabelConf:
		bundle.NodeLabels = c
	case NodeLabelConf:
		bundle.NodeLabels = &c
	case *NodeVLANConf:
		bundle.VLANs = c
	case NodeVLANConf:
		bundle.VLANs = &c
	case *NodeTestConf:
		bundle.Tests = c
	case NodeTestConf:
		bundle.Tests = &c
	}

	return bundle
}
