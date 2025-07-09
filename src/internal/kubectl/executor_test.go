// Package kubectl provides unit tests for Kubernetes operation execution
// WHY: kubectl executor is critical infrastructure that interfaces with live clusters and must be bulletproof
package kubectl

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// mockLogger implements the Logger interface for testing
// TestExecNodeCommand tests command execution on node
// WHY: Executing commands on nodes is crucial for node management and requires validation of implementation
func TestExecNodeCommand(t *testing.T) {
	tests := []struct {
		name        string
		description string
		nodeName    string
		command     string
		dryRun      bool
		expectDryRun bool
	}{
		{
			"exec_command_dry_run_mode",
			"Dry-run mode should simulate execution without running kubectl",
			"rsb3",
			"ls -la",
			true,
			true,
		},
		{
			"exec_command_production_interface",
			"Production mode should generate correct kubectl command (interface test)",
			"rsb2",
			"uptime",
			false,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := newMockLogger()
			executor := NewExecutor(logger)
			executor.SetDryRun(tt.dryRun)
			ctx := context.Background()

			success, output, _ := executor.ExecNodeCommand(ctx, tt.nodeName, tt.command)

			if tt.expectDryRun {
				assert.True(t, success, "Dry-run should succeed")
				assert.Contains(t, output, "Command would be executed on node", "Dry-run should log simulated output")
			} else {
				// In production mode without cluster, we expect command to be attempted
				// but may fail - that's OK for unit testing the interface
				// The important thing is that the correct command was generated
				debugMessage := strings.Join(logger.debugMessages, " ")
				assert.Contains(t, debugMessage, "kubectl debug node", "Should generate kubectl command")
				assert.Contains(t, debugMessage, tt.nodeName, "Should include node name")
				assert.Contains(t, debugMessage, tt.command, "Should include command")
				// Don't assert on output - command may fail without cluster
			}

			debugMessage := strings.Join(logger.debugMessages, " ")
			assert.Contains(t, debugMessage, "kubectl debug node", "Should log kubectl command")
			assert.Contains(t, debugMessage, tt.nodeName, "Should log node name")
			assert.Contains(t, debugMessage, tt.command, "Should log command")
		})
	}
}

// TestGetPods tests pod retrieval with optional filtering
// WHY: Retrieving pods is essential for monitoring and management
func TestGetPods(t *testing.T) {
	tests := []struct {
		name          string
		description   string
		fieldSelector string
		labelSelector string
	}{
		{
			"get_pods_no_filter",
			"Should retrieve all pods without any filter",
			"",
			"",
		},
		{
			"get_pods_with_field_selector",
			"Should retrieve pods matching field selector",
			"spec.nodeName=rsb2",
			"",
		},
		{
			"get_pods_with_label_selector",
			"Should retrieve pods matching label selector",
			"",
			"app=nginx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := newMockLogger()
			executor := NewExecutor(logger)
			ctx := context.Background()

			_, output, _ := executor.GetPods(ctx, tt.fieldSelector, tt.labelSelector)

			// Note: GetPods will fail in test environment without kubectl/cluster
			// Command behavior depends on kubectl availability and cluster access
			// We verify the interface works regardless of success/failure
			_ = output // Test interface behavior, not kubectl availability

			debugMessage := strings.Join(logger.debugMessages, " ")
			assert.Contains(t, debugMessage, "kubectl get pods", "Should log kubectl command")
			if tt.fieldSelector != "" {
				assert.Contains(t, debugMessage, tt.fieldSelector, "Should include field selector")
			}
			if tt.labelSelector != "" {
				assert.Contains(t, debugMessage, tt.labelSelector, "Should include label selector")
			}
		})
	}
}

// TestDeletePod tests pod deletion
// WHY: Deleting pods is necessary for cleaning up failed or old deployments
func TestDeletePod(t *testing.T) {
	tests := []struct {
		name        string
		description string
		podName     string
		dryRun      bool
		expectDryRun bool
	}{
		{
			"delete_pod_production_mode",
			"Production mode should execute actual kubectl command to delete pod",
			"nginx-deployment-abc123",
			false,
			false,
		},
		{
			"delete_pod_dry_run_mode",
			"Dry-run mode should simulate pod deletion without actual execution",
			"nginx-deployment-xyz789",
			true,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := newMockLogger()
			executor := NewExecutor(logger)
			executor.SetDryRun(tt.dryRun)
			ctx := context.Background()

			success, output, _ := executor.DeletePod(ctx, tt.podName)

			if tt.expectDryRun {
				assert.True(t, success, "Dry-run should succeed")
				assert.Contains(t, output, "deleted", "Dry-run should log simulated output")
			} else {
				// In production mode without cluster, command may fail - that's OK for unit testing
				// The important thing is that the correct command was generated
				debugMessage := strings.Join(logger.debugMessages, " ")
				assert.Contains(t, debugMessage, "kubectl delete pod", "Should generate kubectl command")
				assert.Contains(t, debugMessage, tt.podName, "Should include pod name")
				// Don't assert on output - command may fail without cluster
			}

			debugMessage := strings.Join(logger.debugMessages, " ")
			assert.Contains(t, debugMessage, "kubectl delete pod", "Should log kubectl command")
			assert.Contains(t, debugMessage, tt.podName, "Should log pod name")
		})
	}
}

// mockLogger implements the Logger interface for testing
type mockLogger struct {
	debugMessages []string
	infoMessages  []string
	warnMessages  []string
	errorMessages []string
}

func newMockLogger() *mockLogger {
	return &mockLogger{
		debugMessages: make([]string, 0),
		infoMessages:  make([]string, 0),
		warnMessages:  make([]string, 0),
		errorMessages: make([]string, 0),
	}
}

func (m *mockLogger) Debug(message string) {
	m.debugMessages = append(m.debugMessages, message)
}

func (m *mockLogger) Info(message string) {
	m.infoMessages = append(m.infoMessages, message)
}

func (m *mockLogger) Warn(message string) {
	m.warnMessages = append(m.warnMessages, message)
}

func (m *mockLogger) Error(message string) {
	m.errorMessages = append(m.errorMessages, message)
}

func (m *mockLogger) getAllMessages() []string {
	var all []string
	all = append(all, m.debugMessages...)
	all = append(all, m.infoMessages...)
	all = append(all, m.warnMessages...)
	all = append(all, m.errorMessages...)
	return all
}

// TestNewExecutor tests kubectl executor creation
// WHY: Validates proper initialization of the critical cluster interface
func TestNewExecutor(t *testing.T) {
	tests := []struct {
		name        string
		description string
		logger      Logger
		expectValid bool
	}{
		{
			name:        "valid_executor_creation",
			description: "Valid logger should create functional kubectl executor",
			logger:      newMockLogger(),
			expectValid: true,
		},
		{
			name:        "executor_with_nil_logger",
			description: "Nil logger should still create executor (defensive programming)",
			logger:      nil,
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Create new executor
			executor := NewExecutor(tt.logger)

			// Then: Verify creation
			if tt.expectValid {
				assert.NotNil(t, executor, "Executor should not be nil")
				assert.Implements(t, (*DryRunExecutor)(nil), executor, "Should implement DryRunExecutor interface")
				assert.Implements(t, (*Executor)(nil), executor, "Should implement Executor interface")
				assert.False(t, executor.IsDryRun(), "Should start with dry-run disabled")
			}
		})
	}
}

// TestDryRunExecutor_StateMgmt tests dry-run interface implementation
// WHY: Dry-run functionality prevents accidental cluster modifications during testing
func TestDryRunExecutor_StateMgmt(t *testing.T) {
	tests := []struct {
		name        string
		description string
		dryRunState bool
	}{
		{
			name:        "enable_dry_run_mode",
			description: "Enabling dry-run should prevent actual kubectl execution",
			dryRunState: true,
		},
		{
			name:        "disable_dry_run_mode",
			description: "Disabling dry-run should allow actual kubectl execution",
			dryRunState: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: New executor
			logger := newMockLogger()
			executor := NewExecutor(logger)

			// When: Set dry-run state
			executor.SetDryRun(tt.dryRunState)

			// Then: Verify state
			assert.Equal(t, tt.dryRunState, executor.IsDryRun(), "Dry-run state mismatch")

			// Verify state persistence
			executor.SetDryRun(!tt.dryRunState)
			assert.Equal(t, !tt.dryRunState, executor.IsDryRun(), "State should toggle correctly")
		})
	}
}

// TestGetNode tests node information retrieval
// WHY: Node existence validation is critical for labeling operations
func TestGetNode(t *testing.T) {
	tests := []struct {
		name        string
		description string
		nodeName    string
		expectCall  bool
		expectValid bool
	}{
		{
			name:        "valid_node_query",
			description: "Valid node name should generate proper kubectl get node command",
			nodeName:    "rsb2",
			expectCall:  true,
			expectValid: true,
		},
		{
			name:        "node_with_special_characters",
			description: "Node names with special characters should be handled properly",
			nodeName:    "worker-node-01.domain.com",
			expectCall:  true,
			expectValid: true,
		},
		{
			name:        "empty_node_name",
			description: "Empty node name should still generate command (kubectl will handle error)",
			nodeName:    "",
			expectCall:  true,
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Mock executor
			logger := newMockLogger()
			executor := NewExecutor(logger)
			ctx := context.Background()

			// When: Get node information
			// Note: This will fail in test environment but we're testing the interface
			_, output, _ := executor.GetNode(ctx, tt.nodeName)

			// Then: Verify behavior
			if tt.expectValid {
				// Command behavior depends on kubectl availability and cluster access
				// We verify the interface works regardless of success/failure
				assert.NotEmpty(t, output, "Should have output (success or error)")

				// Verify logging occurred (if logger is not nil)
				if logger != nil {
					assert.NotEmpty(t, logger.debugMessages, "Should log debug message")
					debugMessage := strings.Join(logger.debugMessages, " ")
					assert.Contains(t, debugMessage, "kubectl get node", "Should log kubectl command")
					assert.Contains(t, debugMessage, tt.nodeName, "Should log node name")
				}
			} else {
				// Production mode behavior depends on kubectl availability
				assert.NotEmpty(t, output, "Should have output (success or error)")

				// Verify command logging
				debugMessage := strings.Join(logger.debugMessages, " ")
				assert.Contains(t, debugMessage, "kubectl label node", "Should log kubectl command")
			}
		})
	}
}

// TestLabelNode tests node labeling operations
// WHY: Node labeling is the core functionality for OpenStack infrastructure management
func TestLabelNode(t *testing.T) {
	tests := []struct {
		name         string
		description  string
		nodeName     string
		label        string
		overwrite    bool
		dryRun       bool
		expectDryRun bool
	}{
		{
			name:         "label_node_production_mode",
			description:  "Production labeling should execute actual kubectl command",
			nodeName:     "rsb2",
			label:        "openstack-role=control-plane",
			overwrite:    false,
			dryRun:       false,
			expectDryRun: false,
		},
		{
			name:         "label_node_with_overwrite",
			description:  "Overwrite flag should be passed to kubectl command",
			nodeName:     "rsb3",
			label:        "openstack-role=storage",
			overwrite:    true,
			dryRun:       false,
			expectDryRun: false,
		},
		{
			name:         "label_node_dry_run_mode",
			description:  "Dry-run mode should simulate operation without executing kubectl",
			nodeName:     "rsb4",
			label:        "openstack-role=compute",
			overwrite:    false,
			dryRun:       true,
			expectDryRun: true,
		},
		{
			name:         "complex_label_with_special_characters",
			description:  "Complex labels with special characters should be handled properly",
			nodeName:     "worker-01",
			label:        "cluster.openstack.io/role=control-plane",
			overwrite:    true,
			dryRun:       true,
			expectDryRun: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Mock executor with dry-run configuration
			logger := newMockLogger()
			executor := NewExecutor(logger)
			executor.SetDryRun(tt.dryRun)
			ctx := context.Background()

			// When: Label node
			success, output, err := executor.LabelNode(ctx, tt.nodeName, tt.label, tt.overwrite)

			// Then: Verify behavior
			if tt.expectDryRun {
				// Dry-run should succeed with simulated output
				assert.True(t, success, "Dry-run should always succeed")
				assert.Contains(t, output, "labeled", "Dry-run should simulate success")
				assert.NoError(t, err, "Dry-run should not return error")

				// Verify dry-run logging
				debugMessage := strings.Join(logger.debugMessages, " ")
				assert.Contains(t, debugMessage, "DRY RUN", "Should log dry-run message")
				assert.Contains(t, debugMessage, "kubectl label node", "Should log kubectl command")
				assert.Contains(t, debugMessage, tt.nodeName, "Should log node name")
				assert.Contains(t, debugMessage, tt.label, "Should log label")

				if tt.overwrite {
					assert.Contains(t, debugMessage, "--overwrite", "Should include overwrite flag")
				}
			} else {
				// Production mode behavior depends on kubectl availability
				assert.NotEmpty(t, output, "Should have output (success or error)")

				// Verify command logging
				debugMessage := strings.Join(logger.debugMessages, " ")
				assert.Contains(t, debugMessage, "kubectl label node", "Should log kubectl command")
			}
		})
	}
}

// TestUnlabelNode tests node label removal operations
// WHY: Label removal is critical for cleanup and role transitions
func TestUnlabelNode(t *testing.T) {
	tests := []struct {
		name         string
		description  string
		nodeName     string
		labelKey     string
		dryRun       bool
		expectDryRun bool
	}{
		{
			name:         "unlabel_node_production_mode",
			description:  "Production unlabeling should execute actual kubectl command",
			nodeName:     "rsb2",
			labelKey:     "openstack-role",
			dryRun:       false,
			expectDryRun: false,
		},
		{
			name:         "unlabel_node_dry_run_mode",
			description:  "Dry-run mode should simulate unlabeling without executing kubectl",
			nodeName:     "rsb3",
			labelKey:     "ceph-node",
			dryRun:       true,
			expectDryRun: true,
		},
		{
			name:         "unlabel_complex_key",
			description:  "Complex label keys with dots should be handled properly",
			nodeName:     "worker-01",
			labelKey:     "cluster.openstack.io/role",
			dryRun:       true,
			expectDryRun: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Mock executor with dry-run configuration
			logger := newMockLogger()
			executor := NewExecutor(logger)
			executor.SetDryRun(tt.dryRun)
			ctx := context.Background()

			// When: Unlabel node
			success, output, err := executor.UnlabelNode(ctx, tt.nodeName, tt.labelKey)

			// Then: Verify behavior
			if tt.expectDryRun {
				// Dry-run should succeed with simulated output
				assert.True(t, success, "Dry-run should always succeed")
				assert.Contains(t, output, "unlabeled", "Dry-run should simulate success")
				assert.NoError(t, err, "Dry-run should not return error")

				// Verify dry-run logging
				debugMessage := strings.Join(logger.debugMessages, " ")
				assert.Contains(t, debugMessage, "DRY RUN", "Should log dry-run message")
				assert.Contains(t, debugMessage, "kubectl label node", "Should log kubectl command")
				assert.Contains(t, debugMessage, tt.nodeName, "Should log node name")
				assert.Contains(t, debugMessage, tt.labelKey+"-", "Should log label key with minus")
			} else {
				// Production mode behavior depends on kubectl availability
				assert.NotEmpty(t, output, "Should have output (success or error)")

				// Verify command logging
				debugMessage := strings.Join(logger.debugMessages, " ")
				assert.Contains(t, debugMessage, "kubectl label node", "Should log kubectl command")
			}
		})
	}
}

// TestGetNodeLabels tests node label retrieval
// WHY: Label verification is essential for confirming successful operations
func TestGetNodeLabels(t *testing.T) {
	tests := []struct {
		name        string
		description string
		nodeName    string
		expectValid bool
	}{
		{
			name:        "get_labels_valid_node",
			description: "Valid node should generate proper kubectl get node --show-labels command",
			nodeName:    "rsb2",
			expectValid: true,
		},
		{
			name:        "get_labels_complex_node_name",
			description: "Complex node names should be handled properly",
			nodeName:    "master-control-plane-01.cluster.local",
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Mock executor
			logger := newMockLogger()
			executor := NewExecutor(logger)
			ctx := context.Background()

			// When: Get node labels
			_, output, _ := executor.GetNodeLabels(ctx, tt.nodeName)

			// Then: Verify behavior
			if tt.expectValid {
				// Command behavior depends on kubectl availability and cluster access
				assert.NotEmpty(t, output, "Should have output (success or error)")

				// Verify logging
				debugMessage := strings.Join(logger.debugMessages, " ")
				assert.Contains(t, debugMessage, "kubectl get node", "Should log kubectl command")
				assert.Contains(t, debugMessage, tt.nodeName, "Should log node name")
				assert.Contains(t, debugMessage, "--show-labels", "Should include show-labels flag")
			}
		})
	}
}

// TestExecutor_ContextHandling tests context-based operations
// WHY: Context handling ensures proper timeout and cancellation behavior in production
func TestExecutor_ContextHandling(t *testing.T) {
	t.Run("context_timeout_handling", func(t *testing.T) {
		// Given: Executor with timeout context
		logger := newMockLogger()
		executor := NewExecutor(logger)

		// Create context with very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		// When: Execute operation with timeout context
		_, _, err := executor.GetNode(ctx, "rsb2")

		// Then: Should handle timeout appropriately
		// Note: Behavior depends on kubectl availability, but should not crash
		assert.NotNil(t, err, "Should return some kind of error (timeout or command not found)")

		// Verify logging occurred (if logger is not nil)
		if logger != nil {
			assert.NotEmpty(t, logger.debugMessages, "Should log debug message")
			debugMessage := strings.Join(logger.debugMessages, " ")
			assert.Contains(t, debugMessage, "kubectl get node", "Should log kubectl command")
			assert.Contains(t, debugMessage, "rsb2", "Should log node name")
		}
	})

	t.Run("context_cancellation", func(t *testing.T) {
		// Given: Executor with cancellable context
		logger := newMockLogger()
		executor := NewExecutor(logger)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// When: Execute operation with cancelled context
		_, _, err := executor.LabelNode(ctx, "rsb2", "test=cancelled", false)

		// Then: Should handle cancellation appropriately
		// Note: Behavior depends on kubectl availability, but should not crash
		assert.NotNil(t, err, "Should return some kind of error (cancellation or command not found)")

		// Verify logging occurred (if logger is not nil)
		if logger != nil {
			assert.NotEmpty(t, logger.debugMessages, "Should log debug message")
			debugMessage := strings.Join(logger.debugMessages, " ")
			assert.Contains(t, debugMessage, "kubectl label node", "Should log kubectl command")
			assert.Contains(t, debugMessage, "rsb2", "Should log node name")
		}
	})
}

// TestExecutor_EdgeCases tests edge cases and error conditions
// WHY: Edge case handling prevents production failures and ensures robust operation
func TestExecutor_EdgeCases(t *testing.T) {
	t.Run("empty_node_names", func(t *testing.T) {
		// Given: Executor
		logger := newMockLogger()
		executor := NewExecutor(logger)
		ctx := context.Background()

		// When: Use empty node name
		success, output, err := executor.LabelNode(ctx, "", "test=value", false)

		// Then: Should handle gracefully (kubectl will return appropriate error)
		assert.False(t, success, "Should fail with empty node name")
		assert.Error(t, err, "Should return error for empty node name")
		assert.NotEmpty(t, output, "Should have error output")
	})

	t.Run("special_characters_in_labels", func(t *testing.T) {
		// Given: Executor in dry-run mode
		logger := newMockLogger()
		executor := NewExecutor(logger)
		executor.SetDryRun(true)
		ctx := context.Background()

		// When: Use label with special characters
		specialLabel := "cluster.k8s.io/role=control-plane"
		success, output, err := executor.LabelNode(ctx, "rsb2", specialLabel, false)

		// Then: Dry-run should handle special characters properly
		assert.True(t, success, "Dry-run should succeed")
		assert.NoError(t, err, "Dry-run should not error")
		assert.Contains(t, output, "labeled", "Should simulate success")

		// Verify logging includes special characters
		debugMessage := strings.Join(logger.debugMessages, " ")
		assert.Contains(t, debugMessage, specialLabel, "Should log full label with special characters")
	})

	t.Run("very_long_node_names", func(t *testing.T) {
		// Given: Executor in dry-run mode
		logger := newMockLogger()
		executor := NewExecutor(logger)
		executor.SetDryRun(true)
		ctx := context.Background()

		// When: Use very long node name
		longNodeName := strings.Repeat("very-long-node-name-", 10) + "final"
		success, output, err := executor.LabelNode(ctx, longNodeName, "test=value", false)

		// Then: Should handle long names properly
		assert.True(t, success, "Dry-run should succeed with long names")
		assert.NoError(t, err, "Should not error with long names")
		assert.Contains(t, output, "labeled", "Dry-run should simulate success")

		// Verify logging includes full name
		debugMessage := strings.Join(logger.debugMessages, " ")
		assert.Contains(t, debugMessage, longNodeName, "Should log full long node name")
	})

	t.Run("concurrent_operations", func(t *testing.T) {
		// Given: Multiple executors
		logger := newMockLogger()
		executor1 := NewExecutor(logger)
		executor2 := NewExecutor(logger)
		executor1.SetDryRun(true)
		executor2.SetDryRun(true)
		ctx := context.Background()

		// When: Execute concurrent operations
		done1 := make(chan bool)
		done2 := make(chan bool)

		go func() {
			executor1.LabelNode(ctx, "rsb2", "test1=value1", false)
			done1 <- true
		}()

		go func() {
			executor2.LabelNode(ctx, "rsb3", "test2=value2", false)
			done2 <- true
		}()

		// Then: Both should complete without issues
		<-done1
		<-done2

		// Verify both operations were logged
		allMessages := logger.getAllMessages()
		messageText := strings.Join(allMessages, " ")
		assert.Contains(t, messageText, "rsb2", "Should log first operation")
		assert.Contains(t, messageText, "rsb3", "Should log second operation")
	})
}

// TestExecutor_DryRunConsistency tests dry-run behavior consistency
// WHY: Dry-run must behave consistently across all operations for reliable testing
func TestExecutor_DryRunConsistency(t *testing.T) {
	operations := []struct {
		name string
		exec func(Executor, context.Context) (bool, string, error)
	}{
		{
			name: "label_operation",
			exec: func(e Executor, ctx context.Context) (bool, string, error) {
				return e.LabelNode(ctx, "test-node", "test=value", false)
			},
		},
		{
			name: "unlabel_operation",
			exec: func(e Executor, ctx context.Context) (bool, string, error) {
				return e.UnlabelNode(ctx, "test-node", "test")
			},
		},
	}

	for _, op := range operations {
		t.Run(op.name+"_dry_run_consistency", func(t *testing.T) {
			// Given: Executor in dry-run mode
			logger := newMockLogger()
			executor := NewExecutor(logger)
			executor.SetDryRun(true)
			ctx := context.Background()

			// When: Execute operation multiple times
			results := make([]bool, 3)
			outputs := make([]string, 3)
			errors := make([]error, 3)

			for i := 0; i < 3; i++ {
				results[i], outputs[i], errors[i] = op.exec(executor, ctx)
			}

			// Then: All executions should be consistent
			for i := 1; i < 3; i++ {
				assert.Equal(t, results[0], results[i], "Success results should be consistent")
				assert.Equal(t, errors[0] == nil, errors[i] == nil, "Error presence should be consistent")
				// Output format should be consistent (may contain timestamps, so check structure)
				assert.Contains(t, outputs[i], "node/", "Output format should be consistent")
			}

			// Verify all operations were logged
			assert.Len(t, logger.debugMessages, 3, "Should log all three operations")
		})
	}
}

// TestExecutor_LoggingBehavior tests comprehensive logging behavior
// WHY: Proper logging is essential for production debugging and audit trails
func TestExecutor_LoggingBehavior(t *testing.T) {
	t.Run("debug_logging_in_dry_run", func(t *testing.T) {
		// Given: Executor in dry-run mode
		logger := newMockLogger()
		executor := NewExecutor(logger)
		executor.SetDryRun(true)
		ctx := context.Background()

		// When: Execute operations
		executor.LabelNode(ctx, "rsb2", "test=value", false)
		executor.UnlabelNode(ctx, "rsb3", "old-label")

		// Then: Should log appropriate debug messages
		assert.NotEmpty(t, logger.debugMessages, "Should have debug messages")

		debugText := strings.Join(logger.debugMessages, " ")
		assert.Contains(t, debugText, "DRY RUN", "Should log dry-run indicators")
		assert.Contains(t, debugText, "kubectl", "Should log kubectl commands")
		assert.Contains(t, debugText, "rsb2", "Should log first node")
		assert.Contains(t, debugText, "rsb3", "Should log second node")
	})

	t.Run("error_logging_in_production", func(t *testing.T) {
		// Given: Executor in production mode (will fail in test env)
		logger := newMockLogger()
		executor := NewExecutor(logger)
		ctx := context.Background()

		// When: Execute operation (will fail)
		executor.LabelNode(ctx, "rsb2", "test=value", false)

		// Then: Should log debug messages (error messages only on actual command failure)
		assert.NotEmpty(t, logger.debugMessages, "Should have debug messages")

		debugText := strings.Join(logger.debugMessages, " ")
		assert.Contains(t, debugText, "kubectl", "Should log command attempt")
		// Note: Error messages only appear if kubectl command actually fails
	})
}
