// Package main provides the command-line interface for k8ostack-ictl
package main

import (
	"context"
	"fmt"
	"os"

	"k8ostack-ictl/internal/config"
	"k8ostack-ictl/internal/config/precedence"
	"k8ostack-ictl/internal/kubectl"
	"k8ostack-ictl/internal/labeler"
	"k8ostack-ictl/internal/logging"
	"k8ostack-ictl/internal/vlan"

	"github.com/spf13/cobra"
)

// CLI flags
var (
	configFile          string
	dryRun              bool
	verbose             bool
	generateConfig      bool
	generateMultiConfig bool
)

func main() {
	rootCmd := createRootCommand()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}
}

func createRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "kictl",
		Short: "Modern Kubernetes OpenStack infrastructure control tool",
		Long: `kictl: A modern, configurable Kubernetes OpenStack infrastructure control tool

Multi-CRD unified infrastructure automation platform supporting:
- NodeLabelConf: Kubernetes node labeling
- NodeVLANConf: VLAN configuration and management  
- NodeTestConf: Network connectivity testing

The tool processes single or multi-document YAML configurations with
global CLI precedence and comprehensive validation.

Examples:
  # Generate sample configuration
  kictl --generate-config

  # Apply node labels from configuration
  kictl --config cluster-config.yaml --apply

  # Dry-run with verbose output
  kictl --config cluster-config.yaml --apply --dry-run --verbose

  # Remove applied labels
  kictl --config cluster-config.yaml --delete

  # Apply multi-CRD infrastructure
  kictl --config multi-infrastructure.yaml --apply`,
		RunE: runCommand,
	}

	// Operation flags
	rootCmd.Flags().Bool("apply", false, "Apply labels defined in the configuration file")
	rootCmd.Flags().Bool("delete", false, "Remove labels defined in the configuration file")

	// Configuration flags
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to YAML configuration file")
	rootCmd.Flags().BoolVar(&generateConfig, "generate-config", false, "Generate a sample configuration file and exit")
	rootCmd.Flags().BoolVar(&generateMultiConfig, "generate-multi-config", false, "Generate a sample multi-CRD configuration file and exit")

	// Behavior flags
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Simulate the operation without making actual changes")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose debug output")

	// Future extensibility flags (placeholders for other tools)
	rootCmd.Flags().String("log-level", "info", "Set log level (debug, info, warn, error)")

	return rootCmd
}

func runCommand(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Handle generate config flags
	if generateConfig {
		return config.GenerateSampleConfig("sample-config.yaml")
	}

	if generateMultiConfig {
		return config.GenerateMultiCRDSampleConfig("sample-multi-config.yaml")
	}

	// Get operation flags early for validation
	applyOp, _ := cmd.Flags().GetBool("apply")
	deleteOp, _ := cmd.Flags().GetBool("delete")

	// Validate operation flags BEFORE other checks
	if applyOp && deleteOp {
		return fmt.Errorf("cannot specify both --apply and --delete operations")
	}

	// Config-based mode - check after flag validation
	if configFile == "" {
		return fmt.Errorf("configuration file is required. Use --config to specify a YAML file, or --generate-config to create a sample")
	}

	// Initialize logger early for tests that expect logger errors
	logger, err := logging.NewFileLogger("logs", verbose)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer logger.Close()

	// Require explicit operation - no dangerous defaults!
	if !applyOp && !deleteOp {
		return fmt.Errorf("operation required: specify either --apply or --delete\n\nExamples:\n  kictl --config %s --apply    # Apply configuration\n  kictl --config %s --delete   # Remove configuration", configFile, configFile)
	}

	// Load configuration bundle (supports both single and multi-CRD configs)
	bundle, err := config.LoadMultipleConfigs(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create global precedence resolver
	resolver := precedence.NewGlobalResolver(cmd)

	// Apply global CLI precedence to ALL configurations in the bundle
	if err := resolver.ApplyGlobalOverrides(bundle); err != nil {
		return fmt.Errorf("failed to apply CLI precedence: %w", err)
	}

	// Log applied overrides for transparency
	overrides := resolver.GetAppliedOverrides()
	if len(overrides) > 0 {
		logger.Info("🔄 CLI flags overriding config settings:")
		for flag, value := range overrides {
			logger.Info(fmt.Sprintf("  --%s: %v", flag, value))
		}
	}

	// Display startup info with bundle summary
	fmt.Printf("📋 Using config file: %s\n", configFile)
	fmt.Printf("📦 Configuration bundle: %s\n", bundle.GetSummary())

	if len(overrides) > 0 {
		if _, isDryRun := overrides["dry-run"]; isDryRun {
			fmt.Printf("🧪 DRY RUN MODE: No changes will be made\n")
		}
	}

	logger.Info(fmt.Sprintf("Config file: %s", configFile))
	logger.Info(fmt.Sprintf("Bundle summary: %s", bundle.GetSummary()))

	// Execute operations based on what configurations are present
	// This is the beautiful extensible pattern you loved!
	var totalErrors []error

	// Process NodeLabels if present
	if bundle.HasNodeLabels() {
		logger.Info("🏷️  Processing node labeling configuration...")

		// Initialize kubectl executor
		kubectlExecutor := kubectl.NewExecutor(logger)
		// Speed up polling for tests
		if os.Getenv("KICTL_TEST_MODE") == "true" {
			kubectlExecutor.SetPollingInterval(0)
		}

		// Get final tool configuration from the resolved config
		tools := bundle.NodeLabels.GetTools()

		// Initialize labeling service with resolved configuration
		labelingService := labeler.NewService(kubectlExecutor, labeler.Options{
			DryRun:        tools.Nlabel.DryRun,
			Verbose:       verbose, // CLI verbose always applies
			ValidateNodes: tools.Nlabel.ValidateNodes,
			Logger:        logger,
		})

		// Execute labeling operation
		var results *labeler.OperationResults
		if deleteOp {
			results, err = labelingService.RemoveLabels(ctx, bundle.NodeLabels)
		} else {
			results, err = labelingService.ApplyLabels(ctx, bundle.NodeLabels)
		}

		if err != nil {
			totalErrors = append(totalErrors, fmt.Errorf("node labeling failed: %w", err))
		} else {
			// Verify labels if not in dry run mode and operation was apply
			if !tools.Nlabel.DryRun && applyOp {
				_, verifyErr := labelingService.VerifyLabels(ctx, bundle.NodeLabels)
				if verifyErr != nil {
					logger.Warn(fmt.Sprintf("Label verification failed: %v", verifyErr))
				}
			}

			// Handle any operation errors
			if len(results.Errors) > 0 {
				logger.Error("Some labeling operations failed:")
				for _, opErr := range results.Errors {
					logger.Error(fmt.Sprintf("  - %v", opErr))
				}
				totalErrors = append(totalErrors, fmt.Errorf("node labeling completed with %d errors", len(results.Errors)))
			}
		}
	}

	// Process VLANs if present
	if bundle.HasVLANs() {
		logger.Info("🌐 Processing VLAN configuration...")

		// Initialize kubectl executor (reuse from labeling or create new one)
		kubectlExecutor := kubectl.NewExecutor(logger)
		// Speed up polling for tests
		if os.Getenv("KICTL_TEST_MODE") == "true" {
			kubectlExecutor.SetPollingInterval(0)
		}

		// Get final tool configuration from the resolved config
		tools := bundle.VLANs.GetTools()

		// Initialize VLAN service with resolved configuration
		vlanService := vlan.NewService(kubectlExecutor, vlan.Options{
			DryRun:               tools.Nvlan.DryRun,
			Verbose:              verbose, // CLI verbose always applies
			ValidateConnectivity: true,    // Default to true for safety
			PersistentConfig:     false,   // Default to false for safety
			DefaultInterface:     "eth0",  // Default interface
			Logger:               logger,
		})

		// Execute VLAN operation
		var results *vlan.OperationResults
		if deleteOp {
			results, err = vlanService.RemoveVLANs(ctx, bundle.VLANs)
		} else {
			results, err = vlanService.ConfigureVLANs(ctx, bundle.VLANs)
		}

		if err != nil {
			totalErrors = append(totalErrors, fmt.Errorf("VLAN configuration failed: %w", err))
		} else {
			// Handle any operation errors
			if len(results.Errors) > 0 {
				logger.Error("Some VLAN operations failed:")
				for _, opErr := range results.Errors {
					logger.Error(fmt.Sprintf("  - %v", opErr))
				}
				totalErrors = append(totalErrors, fmt.Errorf("VLAN configuration completed with %d errors", len(results.Errors)))
			}
		}
	}

	// Process Tests if present (placeholder for future implementation)
	if bundle.HasTests() {
		logger.Info("🧪 Test configuration detected - feature coming soon!")
		logger.Info(fmt.Sprintf("  Found %d connectivity tests in %s",
			len(bundle.Tests.Spec.Tests), bundle.Tests.GetMetadata().Name))
		// TODO: Implement test service integration
		// testService := testing.NewService(...)
		// results, err := testService.RunTests(ctx, bundle.Tests)
	}

	// Summary
	if len(totalErrors) > 0 {
		logger.Error(fmt.Sprintf("❌ Operation completed with %d errors", len(totalErrors)))
		for _, err := range totalErrors {
			logger.Error(fmt.Sprintf("  - %v", err))
		}
		return fmt.Errorf("operation completed with %d errors", len(totalErrors))
	}

	logger.Info("✅ All operations completed successfully")
	return nil
}
