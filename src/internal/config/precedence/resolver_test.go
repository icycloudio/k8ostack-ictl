// Package precedence provides unit tests for the global precedence resolver
// WHY: This validates the critical CLI > Config > Defaults precedence chain
package precedence

import (
	"reflect"
	"testing"

	"k8ostack-ictl/internal/config"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// TestGlobalResolver_ApplyGlobalOverrides tests CLI precedence resolution
// WHY: Validates that CLI flags correctly override configuration values across all CRDs
func TestGlobalResolver_ApplyGlobalOverrides(t *testing.T) {
	tests := []struct {
		name            string
		description     string
		setupFlags      func(*cobra.Command)
		inputBundle     func() *config.ConfigBundle // Return pointer to ConfigBundle
		expectedChanges map[string]interface{}
		shouldError     bool
	}{
		{
			name:        "dry_run_flag_overrides_config",
			description: "CLI --dry-run flag overrides config file setting",
			setupFlags: func(cmd *cobra.Command) {
				// Simulate CLI flag being set
				cmd.Flags().Bool("dry-run", false, "Enable dry-run mode")
				cmd.Flags().Set("dry-run", "true")
			},
			inputBundle: func() *config.ConfigBundle {
				// Config has dry-run disabled, CLI should override to enabled
				return &config.ConfigBundle{
					NodeLabels: &config.NodeLabelConf{
						APIVersion: "openstack.kictl.icycloud.io/v1",
						Kind:       "NodeLabelConf",
						Tools: config.Tools{
							Nlabel: config.ToolConfig{
								DryRun:   false, // Config says false
								LogLevel: "info",
							},
						},
					},
				}
			},
			expectedChanges: map[string]interface{}{
				"dry-run": "true", // CLI should override to true
			},
			shouldError: false,
		},
		{
			name:        "log_level_flag_overrides_config",
			description: "CLI --log-level flag overrides config file setting",
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().String("log-level", "", "Set log level")
				cmd.Flags().Set("log-level", "debug")
			},
			inputBundle: func() *config.ConfigBundle {
				return &config.ConfigBundle{
					NodeLabels: &config.NodeLabelConf{
						APIVersion: "openstack.kictl.icycloud.io/v1",
						Kind:       "NodeLabelConf",
						Tools: config.Tools{
							Nlabel: config.ToolConfig{
								DryRun:   false,
								LogLevel: "warn", // Config says warn
							},
						},
					},
				}
			},
			expectedChanges: map[string]interface{}{
				"log-level": "debug", // CLI should override to debug
			},
			shouldError: false,
		},
		{
			name:        "multiple_flags_override_multiple_configs",
			description: "Multiple CLI flags override multiple config settings across multiple CRDs",
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("dry-run", false, "Enable dry-run mode")
				cmd.Flags().String("log-level", "", "Set log level")
				cmd.Flags().Set("dry-run", "true")
				cmd.Flags().Set("log-level", "debug")
			},
			inputBundle: func() *config.ConfigBundle {
				return &config.ConfigBundle{
					NodeLabels: &config.NodeLabelConf{
						APIVersion: "openstack.kictl.icycloud.io/v1",
						Kind:       "NodeLabelConf",
						Tools: config.Tools{
							Nlabel: config.ToolConfig{
								DryRun:   false,
								LogLevel: "info",
							},
						},
					},
					VLANs: &config.NodeVLANConf{
						APIVersion: "openstack.kictl.icycloud.io/v1",
						Kind:       "NodeVLANConf",
						Tools: config.Tools{
							Nlabel: config.ToolConfig{
								DryRun:   false,
								LogLevel: "warn",
							},
						},
					},
				}
			},
			expectedChanges: map[string]interface{}{
				"dry-run":   "true",
				"log-level": "debug",
			},
			shouldError: false,
		},
		{
			name:        "no_flags_set_no_overrides",
			description: "When no CLI flags are set, config values remain unchanged",
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("dry-run", false, "Enable dry-run mode")
				cmd.Flags().String("log-level", "", "Set log level")
				// No flags are explicitly set
			},
			inputBundle: func() *config.ConfigBundle {
				return &config.ConfigBundle{
					NodeLabels: &config.NodeLabelConf{
						APIVersion: "openstack.kictl.icycloud.io/v1",
						Kind:       "NodeLabelConf",
						Tools: config.Tools{
							Nlabel: config.ToolConfig{
								DryRun:   true,
								LogLevel: "warn",
							},
						},
					},
				}
			},
			expectedChanges: map[string]interface{}{},
			shouldError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Setup command with flags and resolver
			cmd := &cobra.Command{}
			tt.setupFlags(cmd)

			resolver := NewGlobalResolver(cmd)
			bundle := tt.inputBundle()

			// When: Apply global overrides
			err := resolver.ApplyGlobalOverrides(bundle)

			// Then: Verify results
			if tt.shouldError {
				assert.Error(t, err, "Test %s: expected error but got none", tt.name)
			} else {
				assert.NoError(t, err, "Test %s: unexpected error: %v", tt.name, err)

				// Verify that overrides were applied correctly
				appliedOverrides := resolver.GetAppliedOverrides()
				assert.Equal(t, tt.expectedChanges, appliedOverrides,
					"Test %s: applied overrides mismatch", tt.name)

				// For successful cases, verify the actual configuration was modified
				if !tt.shouldError && len(tt.expectedChanges) > 0 {
					configs := bundle.GetAllConfigs()
					for _, cfg := range configs {
						// Verify the configuration object itself was modified
						// This would require accessing the Tools.Nlabel fields
						verifyConfigOverrides(t, cfg, tt.expectedChanges)
					}
				}
			}
		})
	}
}

// Add test for invalid bundle type error
func TestGlobalResolver_InvalidBundleType(t *testing.T) {
	// Given: Setup command with flags and resolver
	cmd := &cobra.Command{}
	cmd.Flags().Bool("dry-run", false, "Enable dry-run mode")

	resolver := NewGlobalResolver(cmd)

	// When: Apply overrides to invalid type (string instead of ConfigBundle)
	err := resolver.ApplyGlobalOverrides("invalid-type")

	// Then: Should return error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bundle does not implement ConfigBundle interface")
}

// verifyConfigOverrides checks that the configuration object was actually modified
// WHY: Ensures that precedence resolution doesn't just track changes but applies them
func verifyConfigOverrides(t *testing.T, cfg interface{}, expectedChanges map[string]interface{}) {
	// Use type assertion to access the Tools field
	switch config := cfg.(type) {
	case *config.NodeLabelConf:
		if dryRunOverride, exists := expectedChanges["dry-run"]; exists && dryRunOverride == "true" {
			assert.True(t, config.Tools.Nlabel.DryRun, "DryRun should be overridden to true")
		}
		if logLevelOverride, exists := expectedChanges["log-level"]; exists {
			assert.Equal(t, logLevelOverride, config.Tools.Nlabel.LogLevel, "LogLevel should be overridden")
		}
	case *config.NodeVLANConf:
		if dryRunOverride, exists := expectedChanges["dry-run"]; exists && dryRunOverride == "true" {
			assert.True(t, config.Tools.Nlabel.DryRun, "DryRun should be overridden to true")
		}
		if logLevelOverride, exists := expectedChanges["log-level"]; exists {
			assert.Equal(t, logLevelOverride, config.Tools.Nlabel.LogLevel, "LogLevel should be overridden")
		}
	}
}

// TestGlobalResolver_GetAppliedOverrides tests override tracking
// WHY: Validates that we can accurately report which CLI flags were applied
func TestGlobalResolver_GetAppliedOverrides(t *testing.T) {
	tests := []struct {
		name              string
		description       string
		setupFlags        func(*cobra.Command)
		expectedOverrides map[string]interface{}
	}{
		{
			name:        "single_flag_tracked",
			description: "Single CLI flag override is correctly tracked",
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("dry-run", false, "Enable dry-run mode")
				cmd.Flags().Set("dry-run", "true")
			},
			expectedOverrides: map[string]interface{}{
				"dry-run": "true",
			},
		},
		{
			name:        "multiple_flags_tracked",
			description: "Multiple CLI flag overrides are correctly tracked",
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("dry-run", false, "Enable dry-run mode")
				cmd.Flags().String("log-level", "", "Set log level")
				cmd.Flags().Set("dry-run", "true")
				cmd.Flags().Set("log-level", "debug")
			},
			expectedOverrides: map[string]interface{}{
				"dry-run":   "true",
				"log-level": "debug",
			},
		},
		{
			name:        "no_flags_set_empty_tracking",
			description: "When no flags are set, no overrides are tracked",
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("dry-run", false, "Enable dry-run mode")
				cmd.Flags().String("log-level", "", "Set log level")
				// No flags explicitly set
			},
			expectedOverrides: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Setup command with flags
			cmd := &cobra.Command{}
			tt.setupFlags(cmd)

			resolver := NewGlobalResolver(cmd)

			// When: Get applied overrides
			overrides := resolver.GetAppliedOverrides()

			// Then: Verify tracking is correct
			assert.Equal(t, tt.expectedOverrides, overrides,
				"Test %s: override tracking mismatch", tt.name)
		})
	}
}

// TestGlobalResolver_EdgeCases tests edge cases and error conditions
// WHY: Ensures robust behavior under unusual or error conditions
func TestGlobalResolver_EdgeCases(t *testing.T) {
	t.Run("nil_command_doesnt_panic", func(t *testing.T) {
		// Given: Resolver with nil command (edge case)
		assert.NotPanics(t, func() {
			resolver := NewGlobalResolver(nil)
			assert.NotNil(t, resolver)
		})
	})

	t.Run("empty_config_bundle_succeeds", func(t *testing.T) {
		// Given: Empty config bundle
		cmd := &cobra.Command{}
		cmd.Flags().Bool("dry-run", false, "Enable dry-run mode")

		resolver := NewGlobalResolver(cmd)
		emptyBundle := &config.ConfigBundle{} // Use pointer

		// When: Apply overrides to empty bundle
		err := resolver.ApplyGlobalOverrides(emptyBundle)

		// Then: Should succeed without error
		assert.NoError(t, err)
	})

	t.Run("setFieldFromFlag_unsupported_type", func(t *testing.T) {
		// Given: Command with unsupported field type
		cmd := &cobra.Command{}
		cmd.Flags().Float64("float-flag", 0.0, "Float flag")
		cmd.Flags().Set("float-flag", "3.14")

		resolver := NewGlobalResolver(cmd)

		// When: Try to set unsupported field type
		floatField := reflect.ValueOf(new(float64)).Elem()
		err := resolver.setFieldFromFlag(floatField, "float-flag")

		// Then: Should return error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported field type")
	})

	t.Run("setFieldFromFlag_int_type", func(t *testing.T) {
		// Given: Command with int flag
		cmd := &cobra.Command{}
		cmd.Flags().Int("int-flag", 0, "Int flag")
		cmd.Flags().Set("int-flag", "42")

		resolver := NewGlobalResolver(cmd)

		// When: Set int field from flag
		intField := reflect.ValueOf(new(int)).Elem()
		err := resolver.setFieldFromFlag(intField, "int-flag")

		// Then: Should succeed
		assert.NoError(t, err)
		assert.Equal(t, 42, int(intField.Int()))
	})

	t.Run("setFieldFromFlag_flag_error", func(t *testing.T) {
		// Given: Command with flag
		cmd := &cobra.Command{}
		cmd.Flags().Bool("bool-flag", false, "Bool flag")

		resolver := NewGlobalResolver(cmd)

		// When: Try to get non-existent flag
		boolField := reflect.ValueOf(new(bool)).Elem()
		err := resolver.setFieldFromFlag(boolField, "non-existent")

		// Then: Should return error
		assert.Error(t, err)
	})

	t.Run("applyToToolConfig_unsettable_field", func(t *testing.T) {
		// Given: Command and resolver
		cmd := &cobra.Command{}
		cmd.Flags().Bool("dry-run", false, "Enable dry-run mode")
		cmd.Flags().Set("dry-run", "true")

		resolver := NewGlobalResolver(cmd)

		// When: Apply to unsettable field (create a field that cannot be set)
		testStruct := struct{ ReadOnly string }{ReadOnly: "test"}
		unsettableField := reflect.ValueOf(testStruct).Field(0)
		err := resolver.applyToToolConfig(unsettableField)

		// Then: Should return error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tool config is not settable")
	})

	t.Run("applyToConfig_no_tools_field", func(t *testing.T) {
		// Given: Command and resolver
		cmd := &cobra.Command{}
		cmd.Flags().Bool("dry-run", false, "Enable dry-run mode")
		cmd.Flags().Set("dry-run", "true")

		resolver := NewGlobalResolver(cmd)

		// Create a config without Tools field
		configWithoutTools := &struct {
			Name string
		}{Name: "test"}

		// When: Apply to config without Tools field
		err := resolver.applyToConfig(configWithoutTools)

		// Then: Should handle gracefully
		assert.NoError(t, err)
	})
}
