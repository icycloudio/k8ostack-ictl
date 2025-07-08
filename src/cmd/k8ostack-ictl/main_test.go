// Package main provides unit tests for the k8ostack-ictl command-line interface functions
// WHY: Unit tests focus on testing individual functions in isolation without external dependencies
package main

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// TestCreateRootCommand_Unit tests the root command creation in isolation
// WHY: Validates command structure, flags, and metadata without executing the command
func TestCreateRootCommand_Unit(t *testing.T) {
	tests := []struct {
		name        string
		description string
		validator   func(*testing.T, *cobra.Command)
	}{
		{
			name:        "command_basic_properties",
			description: "Root command should have correct basic properties",
			validator: func(t *testing.T, cmd *cobra.Command) {
				assert.Equal(t, "kictl", cmd.Use, "Command name should be 'kictl'")
				assert.NotEmpty(t, cmd.Short, "Should have short description")
				assert.NotEmpty(t, cmd.Long, "Should have long description")
				assert.NotNil(t, cmd.RunE, "Should have run function")
			},
		},
		{
			name:        "command_flags_exist",
			description: "All required flags should be present and configured",
			validator: func(t *testing.T, cmd *cobra.Command) {
				flags := cmd.Flags()

				// Operation flags
				assert.NotNil(t, flags.Lookup("apply"), "Should have apply flag")
				assert.NotNil(t, flags.Lookup("delete"), "Should have delete flag")

				// Configuration flags
				assert.NotNil(t, flags.Lookup("config"), "Should have config flag")
				assert.NotNil(t, flags.Lookup("generate-config"), "Should have generate-config flag")
				assert.NotNil(t, flags.Lookup("generate-multi-config"), "Should have generate-multi-config flag")

				// Behavior flags
				assert.NotNil(t, flags.Lookup("dry-run"), "Should have dry-run flag")
				assert.NotNil(t, flags.Lookup("verbose"), "Should have verbose flag")

				// Future flags
				assert.NotNil(t, flags.Lookup("log-level"), "Should have log-level flag")
			},
		},
		{
			name:        "flag_default_values",
			description: "Flags should have correct default values",
			validator: func(t *testing.T, cmd *cobra.Command) {
				flags := cmd.Flags()

				// Boolean flags should default to false
				boolFlags := []string{"apply", "delete", "dry-run", "verbose", "generate-config", "generate-multi-config"}
				for _, flagName := range boolFlags {
					flag := flags.Lookup(flagName)
					assert.Equal(t, "false", flag.DefValue, "Flag %s should default to false", flagName)
				}

				// String flags should have appropriate defaults
				configFlag := flags.Lookup("config")
				assert.Equal(t, "", configFlag.DefValue, "Config flag should default to empty")

				logLevelFlag := flags.Lookup("log-level")
				assert.Equal(t, "info", logLevelFlag.DefValue, "Log level should default to info")
			},
		},
		{
			name:        "flag_shortcuts",
			description: "Shortcut flags should be properly configured",
			validator: func(t *testing.T, cmd *cobra.Command) {
				flags := cmd.Flags()

				// Check shortcut flags
				configFlag := flags.Lookup("config")
				assert.Equal(t, "c", configFlag.Shorthand, "Config flag should have 'c' shorthand")

				verboseFlag := flags.Lookup("verbose")
				assert.Equal(t, "v", verboseFlag.Shorthand, "Verbose flag should have 'v' shorthand")
			},
		},
		{
			name:        "help_content_validation",
			description: "Help content should mention all supported CRD types",
			validator: func(t *testing.T, cmd *cobra.Command) {
				longDesc := cmd.Long

				// Should mention all CRD types
				assert.Contains(t, longDesc, "NodeLabelConf", "Should mention NodeLabelConf")
				assert.Contains(t, longDesc, "NodeVLANConf", "Should mention NodeVLANConf")
				assert.Contains(t, longDesc, "NodeTestConf", "Should mention NodeTestConf")

				// Should include examples
				assert.Contains(t, longDesc, "Examples:", "Should include examples section")
				assert.Contains(t, longDesc, "--generate-config", "Should show generate-config example")
				assert.Contains(t, longDesc, "--dry-run", "Should show dry-run example")
				assert.Contains(t, longDesc, "--apply", "Should show apply example")
				assert.Contains(t, longDesc, "--delete", "Should show delete example")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Fresh root command
			cmd := createRootCommand()

			// When: Validate the command
			// Then: Run validation function
			tt.validator(t, cmd)
		})
	}
}

// TestFlagParsing_Unit tests flag parsing logic in isolation
// WHY: Validates that flag values are correctly parsed and accessible
func TestFlagParsing_Unit(t *testing.T) {
	tests := []struct {
		name        string
		description string
		args        []string
		validator   func(*testing.T, *cobra.Command)
	}{
		{
			name:        "boolean_flags_parsing",
			description: "Boolean flags should be parsed correctly",
			args:        []string{"--apply", "--dry-run", "--verbose"},
			validator: func(t *testing.T, cmd *cobra.Command) {
				apply, err := cmd.Flags().GetBool("apply")
				assert.NoError(t, err)
				assert.True(t, apply, "Apply flag should be true")

				dryRun, err := cmd.Flags().GetBool("dry-run")
				assert.NoError(t, err)
				assert.True(t, dryRun, "Dry-run flag should be true")

				verbose, err := cmd.Flags().GetBool("verbose")
				assert.NoError(t, err)
				assert.True(t, verbose, "Verbose flag should be true")
			},
		},
		{
			name:        "string_flags_parsing",
			description: "String flags should be parsed correctly",
			args:        []string{"--config", "test.yaml", "--log-level", "debug"},
			validator: func(t *testing.T, cmd *cobra.Command) {
				config, err := cmd.Flags().GetString("config")
				assert.NoError(t, err)
				assert.Equal(t, "test.yaml", config, "Config flag should be 'test.yaml'")

				logLevel, err := cmd.Flags().GetString("log-level")
				assert.NoError(t, err)
				assert.Equal(t, "debug", logLevel, "Log level should be 'debug'")
			},
		},
		{
			name:        "shorthand_flags_parsing",
			description: "Shorthand flags should work correctly",
			args:        []string{"-c", "config.yaml", "-v"},
			validator: func(t *testing.T, cmd *cobra.Command) {
				config, err := cmd.Flags().GetString("config")
				assert.NoError(t, err)
				assert.Equal(t, "config.yaml", config, "Config shorthand should work")

				verbose, err := cmd.Flags().GetBool("verbose")
				assert.NoError(t, err)
				assert.True(t, verbose, "Verbose shorthand should work")
			},
		},
		{
			name:        "mixed_flags_parsing",
			description: "Mixed long and short flags should work together",
			args:        []string{"--apply", "-c", "test.yaml", "--dry-run", "-v"},
			validator: func(t *testing.T, cmd *cobra.Command) {
				apply, _ := cmd.Flags().GetBool("apply")
				config, _ := cmd.Flags().GetString("config")
				dryRun, _ := cmd.Flags().GetBool("dry-run")
				verbose, _ := cmd.Flags().GetBool("verbose")

				assert.True(t, apply, "Apply flag should be true")
				assert.Equal(t, "test.yaml", config, "Config should be 'test.yaml'")
				assert.True(t, dryRun, "Dry-run should be true")
				assert.True(t, verbose, "Verbose should be true")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Fresh command with args
			cmd := createRootCommand()

			// When: Parse flags
			err := cmd.ParseFlags(tt.args)
			assert.NoError(t, err, "Flag parsing should succeed")

			// Then: Validate parsed values
			tt.validator(t, cmd)
		})
	}
}

// TestGlobalVariables_Unit tests global variable behavior in isolation
// WHY: Validates that global variables are properly managed and don't cause side effects
func TestGlobalVariables_Unit(t *testing.T) {
	t.Run("global_variable_initialization", func(t *testing.T) {
		// Given: Global variables (test their initial state)
		// Note: We can't test actual values as they may be modified by other tests
		// But we can test that they're declared and accessible

		// When: Access global variables
		// Then: Should not panic (validates they're properly declared)
		assert.NotPanics(t, func() {
			_ = configFile
			_ = dryRun
			_ = verbose
			_ = generateConfig
			_ = generateMultiConfig
		}, "Global variables should be accessible")
	})

	t.Run("global_variable_types", func(t *testing.T) {
		// Given: Global variables
		// When: Check types (using type assertions)
		// Then: Should have correct types
		assert.IsType(t, "", configFile, "configFile should be string")
		assert.IsType(t, false, dryRun, "dryRun should be bool")
		assert.IsType(t, false, verbose, "verbose should be bool")
		assert.IsType(t, false, generateConfig, "generateConfig should be bool")
		assert.IsType(t, false, generateMultiConfig, "generateMultiConfig should be bool")
	})

	t.Run("global_variable_modification", func(t *testing.T) {
		// Given: Original values
		originalConfigFile := configFile
		originalDryRun := dryRun
		originalVerbose := verbose
		originalGenerateConfig := generateConfig
		originalGenerateMultiConfig := generateMultiConfig

		// When: Modify globals
		configFile = "test-modification.yaml"
		dryRun = true
		verbose = true
		generateConfig = true
		generateMultiConfig = true

		// Then: Values should be modified
		assert.Equal(t, "test-modification.yaml", configFile)
		assert.True(t, dryRun)
		assert.True(t, verbose)
		assert.True(t, generateConfig)
		assert.True(t, generateMultiConfig)

		// Cleanup: Restore original values
		configFile = originalConfigFile
		dryRun = originalDryRun
		verbose = originalVerbose
		generateConfig = originalGenerateConfig
		generateMultiConfig = originalGenerateMultiConfig
	})
}

// TestCommandValidation_Unit tests command validation logic in isolation
// WHY: Validates business logic without executing external dependencies
func TestCommandValidation_Unit(t *testing.T) {
	t.Run("flag_combination_validation", func(t *testing.T) {
		// Given: Command with conflicting flags
		cmd := createRootCommand()

		// When: Set conflicting flags
		cmd.Flags().Set("apply", "true")
		cmd.Flags().Set("delete", "true")

		// Then: We can verify the flags are set (validation happens in runCommand)
		apply, _ := cmd.Flags().GetBool("apply")
		delete, _ := cmd.Flags().GetBool("delete")

		assert.True(t, apply, "Apply flag should be set")
		assert.True(t, delete, "Delete flag should be set")

		// Note: Actual conflict validation happens in runCommand function
		// This unit test validates that flags can be set and read correctly
	})

	t.Run("required_flag_detection", func(t *testing.T) {
		// Given: Command without required config
		cmd := createRootCommand()

		// When: Check config flag value
		config, err := cmd.Flags().GetString("config")
		assert.NoError(t, err)

		// Then: Should be empty (validation of requirement happens in runCommand)
		assert.Empty(t, config, "Config should be empty by default")
	})

	t.Run("flag_accessibility", func(t *testing.T) {
		// Given: Command with various flags set
		cmd := createRootCommand()
		cmd.Flags().Set("config", "test.yaml")
		cmd.Flags().Set("apply", "true")
		cmd.Flags().Set("dry-run", "true")
		cmd.Flags().Set("verbose", "true")
		cmd.Flags().Set("log-level", "debug")

		// When: Retrieve flag values
		config, err1 := cmd.Flags().GetString("config")
		apply, err2 := cmd.Flags().GetBool("apply")
		dryRun, err3 := cmd.Flags().GetBool("dry-run")
		verbose, err4 := cmd.Flags().GetBool("verbose")
		logLevel, err5 := cmd.Flags().GetString("log-level")

		// Then: All flags should be accessible without error
		assert.NoError(t, err1, "Config flag should be accessible")
		assert.NoError(t, err2, "Apply flag should be accessible")
		assert.NoError(t, err3, "Dry-run flag should be accessible")
		assert.NoError(t, err4, "Verbose flag should be accessible")
		assert.NoError(t, err5, "Log-level flag should be accessible")

		// And values should be correct
		assert.Equal(t, "test.yaml", config)
		assert.True(t, apply)
		assert.True(t, dryRun)
		assert.True(t, verbose)
		assert.Equal(t, "debug", logLevel)
	})
}

// TestCommandStructure_Unit tests command structure and hierarchy
// WHY: Validates that command structure is properly configured for CLI framework
func TestCommandStructure_Unit(t *testing.T) {
	t.Run("command_hierarchy", func(t *testing.T) {
		// Given: Root command
		cmd := createRootCommand()

		// When: Check command structure
		// Then: Should be root command with no parent
		assert.Nil(t, cmd.Parent(), "Root command should have no parent")
		assert.False(t, cmd.HasSubCommands(), "Root command should have no subcommands")
	})

	t.Run("command_execution_setup", func(t *testing.T) {
		// Given: Root command
		cmd := createRootCommand()

		// When: Check execution setup
		// Then: Should have RunE function set
		assert.NotNil(t, cmd.RunE, "Command should have RunE function")
		assert.NotNil(t, cmd.Flags(), "Command should have flags initialized")
	})

	t.Run("help_system_setup", func(t *testing.T) {
		// Given: Root command
		cmd := createRootCommand()

		// When: Check help system
		// Then: Help should be properly configured
		assert.NotEmpty(t, cmd.Use, "Command should have usage string")
		assert.NotEmpty(t, cmd.Short, "Command should have short description")
		assert.NotEmpty(t, cmd.Long, "Command should have long description")

		// Help functionality is built into cobra - verify the command can generate help
		helpOutput := cmd.UsageString()
		assert.NotEmpty(t, helpOutput, "Command should generate help output")
		assert.Contains(t, helpOutput, "kictl", "Help should contain command name")
	})
}

// TestCommandMetadata_Unit tests command metadata and documentation
// WHY: Validates that command provides proper user-facing information
func TestCommandMetadata_Unit(t *testing.T) {
	t.Run("usage_string_format", func(t *testing.T) {
		// Given: Root command
		cmd := createRootCommand()

		// When: Check usage string
		use := cmd.Use

		// Then: Should be properly formatted
		assert.Equal(t, "kictl", use, "Usage should be 'kictl'")
		assert.NotContains(t, use, " ", "Usage should not contain spaces")
	})

	t.Run("description_completeness", func(t *testing.T) {
		// Given: Root command
		cmd := createRootCommand()

		// When: Check descriptions
		short := cmd.Short
		long := cmd.Long

		// Then: Should be informative
		assert.NotEmpty(t, short, "Short description should not be empty")
		assert.NotEmpty(t, long, "Long description should not be empty")
		assert.Greater(t, len(long), len(short), "Long description should be longer than short")

		// Should contain key terms
		assert.Contains(t, short, "Kubernetes", "Short description should mention Kubernetes")
		assert.Contains(t, short, "OpenStack", "Short description should mention OpenStack")
	})

	t.Run("example_documentation", func(t *testing.T) {
		// Given: Root command
		cmd := createRootCommand()

		// When: Check long description for examples
		long := cmd.Long

		// Then: Should contain usage examples
		assert.Contains(t, long, "Examples:", "Should have examples section")
		assert.Contains(t, long, "kictl", "Examples should show command name")

		// Should show key operations
		assert.Contains(t, long, "--generate-config", "Should show config generation")
		assert.Contains(t, long, "--apply", "Should show apply operation")
		assert.Contains(t, long, "--delete", "Should show delete operation")
		assert.Contains(t, long, "--dry-run", "Should show dry-run option")
	})
}

// TestFlagConfiguration_Unit tests individual flag configuration
// WHY: Validates that each flag is properly configured with correct properties
func TestFlagConfiguration_Unit(t *testing.T) {
	flagTests := []struct {
		name         string
		flagName     string
		expectedType string
		hasShorthand bool
		shorthand    string
		defaultValue string
		description  string
	}{
		{
			name:         "apply_flag",
			flagName:     "apply",
			expectedType: "bool",
			hasShorthand: false,
			defaultValue: "false",
			description:  "should configure apply operation flag",
		},
		{
			name:         "delete_flag",
			flagName:     "delete",
			expectedType: "bool",
			hasShorthand: false,
			defaultValue: "false",
			description:  "should configure delete operation flag",
		},
		{
			name:         "config_flag",
			flagName:     "config",
			expectedType: "string",
			hasShorthand: true,
			shorthand:    "c",
			defaultValue: "",
			description:  "should configure config file flag with shorthand",
		},
		{
			name:         "dry_run_flag",
			flagName:     "dry-run",
			expectedType: "bool",
			hasShorthand: false,
			defaultValue: "false",
			description:  "should configure dry-run flag",
		},
		{
			name:         "verbose_flag",
			flagName:     "verbose",
			expectedType: "bool",
			hasShorthand: true,
			shorthand:    "v",
			defaultValue: "false",
			description:  "should configure verbose flag with shorthand",
		},
		{
			name:         "generate_config_flag",
			flagName:     "generate-config",
			expectedType: "bool",
			hasShorthand: false,
			defaultValue: "false",
			description:  "should configure generate-config flag",
		},
		{
			name:         "generate_multi_config_flag",
			flagName:     "generate-multi-config",
			expectedType: "bool",
			hasShorthand: false,
			defaultValue: "false",
			description:  "should configure generate-multi-config flag",
		},
		{
			name:         "log_level_flag",
			flagName:     "log-level",
			expectedType: "string",
			hasShorthand: false,
			defaultValue: "info",
			description:  "should configure log-level flag with default",
		},
	}

	for _, tt := range flagTests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Root command
			cmd := createRootCommand()
			flags := cmd.Flags()

			// When: Look up flag
			flag := flags.Lookup(tt.flagName)

			// Then: Flag should be properly configured
			assert.NotNil(t, flag, "Flag %s should exist", tt.flagName)
			assert.Equal(t, tt.defaultValue, flag.DefValue, "Flag %s should have correct default", tt.flagName)

			if tt.hasShorthand {
				assert.Equal(t, tt.shorthand, flag.Shorthand, "Flag %s should have correct shorthand", tt.flagName)
			} else {
				assert.Empty(t, flag.Shorthand, "Flag %s should not have shorthand", tt.flagName)
			}

			// Validate flag can be accessed with correct type
			switch tt.expectedType {
			case "bool":
				_, err := flags.GetBool(tt.flagName)
				assert.NoError(t, err, "Bool flag %s should be accessible", tt.flagName)
			case "string":
				_, err := flags.GetString(tt.flagName)
				assert.NoError(t, err, "String flag %s should be accessible", tt.flagName)
			}
		})
	}
}
