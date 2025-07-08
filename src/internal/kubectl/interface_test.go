// Package kubectl provides unit tests for Kubernetes operation interfaces
// WHY: Interface compliance testing ensures all implementations meet the contract for cluster operations
package kubectl

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestExecutor_Interface tests the Executor interface compliance
// WHY: Interface compliance ensures consistent behavior across different implementations
func TestExecutor_Interface(t *testing.T) {
	t.Run("real_executor_implements_executor", func(t *testing.T) {
		// Given: Real executor instance
		logger := newMockLogger()
		executor := NewExecutor(logger)

		// Then: Should implement Executor interface
		assert.Implements(t, (*Executor)(nil), executor, "RealExecutor should implement Executor interface")
	})

	t.Run("real_executor_implements_dry_run_executor", func(t *testing.T) {
		// Given: Real executor instance
		logger := newMockLogger()
		executor := NewExecutor(logger)

		// Then: Should implement DryRunExecutor interface
		assert.Implements(t, (*DryRunExecutor)(nil), executor, "RealExecutor should implement DryRunExecutor interface")
	})

	t.Run("executor_interface_methods_exist", func(t *testing.T) {
		// Given: Executor interface type
		var executor Executor
		logger := newMockLogger()
		executor = NewExecutor(logger)
		ctx := context.Background()

		// When/Then: Interface methods should be callable
		assert.NotPanics(t, func() {
			executor.GetNode(ctx, "test-node")
		}, "GetNode method should be callable")

		assert.NotPanics(t, func() {
			executor.LabelNode(ctx, "test-node", "test=value", false)
		}, "LabelNode method should be callable")

		assert.NotPanics(t, func() {
			executor.UnlabelNode(ctx, "test-node", "test")
		}, "UnlabelNode method should be callable")

		assert.NotPanics(t, func() {
			executor.GetNodeLabels(ctx, "test-node")
		}, "GetNodeLabels method should be callable")
	})

	t.Run("dry_run_interface_methods_exist", func(t *testing.T) {
		// Given: DryRunExecutor interface type
		var executor DryRunExecutor
		logger := newMockLogger()
		executor = NewExecutor(logger)

		// When/Then: DryRunExecutor methods should be callable
		assert.NotPanics(t, func() {
			executor.SetDryRun(true)
		}, "SetDryRun method should be callable")

		assert.NotPanics(t, func() {
			_ = executor.IsDryRun()
		}, "IsDryRun method should be callable")

		// Verify dry-run state management
		executor.SetDryRun(true)
		assert.True(t, executor.IsDryRun(), "Should return true when dry-run is enabled")

		executor.SetDryRun(false)
		assert.False(t, executor.IsDryRun(), "Should return false when dry-run is disabled")
	})

	t.Run("dry_run_executor_embeds_executor", func(t *testing.T) {
		// Given: DryRunExecutor instance
		logger := newMockLogger()
		dryRunExecutor := NewExecutor(logger)
		ctx := context.Background()

		// When: Cast to Executor interface
		var baseExecutor Executor = dryRunExecutor

		// Then: Should still have all Executor methods
		assert.NotPanics(t, func() {
			baseExecutor.GetNode(ctx, "test-node")
		}, "Should have GetNode method from embedded interface")

		assert.NotPanics(t, func() {
			baseExecutor.LabelNode(ctx, "test-node", "test=value", false)
		}, "Should have LabelNode method from embedded interface")
	})
}

// TestLogger_Interface tests the Logger interface compliance
// WHY: Logger interface ensures consistent logging behavior across different implementations
func TestLogger_Interface(t *testing.T) {
	t.Run("mock_logger_implements_logger", func(t *testing.T) {
		// Given: Mock logger instance
		logger := newMockLogger()

		// Then: Should implement Logger interface
		assert.Implements(t, (*Logger)(nil), logger, "mockLogger should implement Logger interface")
	})

	t.Run("logger_interface_methods_exist", func(t *testing.T) {
		// Given: Logger interface type
		var logger Logger = newMockLogger()

		// When/Then: Logger methods should be callable
		assert.NotPanics(t, func() {
			logger.Debug("test debug message")
		}, "Debug method should be callable")

		assert.NotPanics(t, func() {
			logger.Info("test info message")
		}, "Info method should be callable")

		assert.NotPanics(t, func() {
			logger.Warn("test warn message")
		}, "Warn method should be callable")

		assert.NotPanics(t, func() {
			logger.Error("test error message")
		}, "Error method should be callable")
	})

	t.Run("logger_message_storage", func(t *testing.T) {
		// Given: Mock logger
		mockLogger := newMockLogger()
		var logger Logger = mockLogger

		// When: Log messages at different levels
		logger.Debug("debug message")
		logger.Info("info message")
		logger.Warn("warn message")
		logger.Error("error message")

		// Then: Messages should be stored correctly
		assert.Len(t, mockLogger.debugMessages, 1, "Should store debug message")
		assert.Len(t, mockLogger.infoMessages, 1, "Should store info message")
		assert.Len(t, mockLogger.warnMessages, 1, "Should store warn message")
		assert.Len(t, mockLogger.errorMessages, 1, "Should store error message")

		assert.Equal(t, "debug message", mockLogger.debugMessages[0], "Debug message should match")
		assert.Equal(t, "info message", mockLogger.infoMessages[0], "Info message should match")
		assert.Equal(t, "warn message", mockLogger.warnMessages[0], "Warn message should match")
		assert.Equal(t, "error message", mockLogger.errorMessages[0], "Error message should match")
	})
}

// TestInterface_Composition tests interface composition and compatibility
// WHY: Interface composition ensures clean architecture and proper inheritance relationships
func TestInterface_Composition(t *testing.T) {
	t.Run("dry_run_executor_composition", func(t *testing.T) {
		// Given: DryRunExecutor instance
		logger := newMockLogger()
		dryRunExecutor := NewExecutor(logger)

		// When: Use as different interface types
		var executor Executor = dryRunExecutor
		var dryRunner DryRunExecutor = dryRunExecutor

		// Then: Should work as both interface types
		ctx := context.Background()

		// Test Executor interface methods
		assert.NotPanics(t, func() {
			executor.GetNode(ctx, "test")
		}, "Should work as Executor interface")

		// Test DryRunExecutor interface methods
		assert.NotPanics(t, func() {
			dryRunner.SetDryRun(true)
			_ = dryRunner.IsDryRun()
		}, "Should work as DryRunExecutor interface")

		// Test that dry-run state affects behavior
		dryRunner.SetDryRun(true)
		success, output, err := dryRunner.LabelNode(ctx, "test", "label=value", false)
		assert.True(t, success, "Dry-run should succeed")
		assert.NoError(t, err, "Dry-run should not error")
		assert.Contains(t, output, "labeled", "Dry-run should simulate success")
	})

	t.Run("interface_type_assertions", func(t *testing.T) {
		// Given: Various interface instances
		logger := newMockLogger()
		realExecutor := NewExecutor(logger)

		// When: Perform type assertions
		// Then: Should assert correctly
		if executor, ok := realExecutor.(Executor); ok {
			assert.NotNil(t, executor, "Should assert to Executor successfully")
		} else {
			t.Error("Should be able to assert to Executor interface")
		}

		if dryRunner, ok := realExecutor.(DryRunExecutor); ok {
			assert.NotNil(t, dryRunner, "Should assert to DryRunExecutor successfully")
		} else {
			t.Error("Should be able to assert to DryRunExecutor interface")
		}

		// Test Logger interface assertion with proper interface variable
		var loggerInterface Logger = logger
		if loggerAsInterface, ok := loggerInterface.(*mockLogger); ok {
			assert.NotNil(t, loggerAsInterface, "Should assert to mockLogger from Logger interface")
		} else {
			t.Error("Should be able to assert mockLogger from Logger interface")
		}
	})
}

// TestInterface_ErrorHandling tests error handling consistency across interfaces
// WHY: Consistent error handling ensures predictable behavior for all implementations
func TestInterface_ErrorHandling(t *testing.T) {
	t.Run("executor_error_consistency", func(t *testing.T) {
		// Given: Executor instances
		logger := newMockLogger()
		executor := NewExecutor(logger)
		ctx := context.Background()

		// When: Execute operations that will fail in test environment
		operations := []func() (bool, string, error){
			func() (bool, string, error) { return executor.GetNode(ctx, "nonexistent") },
			func() (bool, string, error) { return executor.LabelNode(ctx, "nonexistent", "test=value", false) },
			func() (bool, string, error) { return executor.UnlabelNode(ctx, "nonexistent", "test") },
			func() (bool, string, error) { return executor.GetNodeLabels(ctx, "nonexistent") },
		}

		// Then: All operations should handle errors consistently
		for i, op := range operations {
			success, output, err := op()
			assert.False(t, success, "Operation %d should fail in test environment", i)
			assert.Error(t, err, "Operation %d should return error", i)
			assert.NotEmpty(t, output, "Operation %d should have error output", i)
		}
	})

	t.Run("dry_run_error_consistency", func(t *testing.T) {
		// Given: Executor in dry-run mode
		logger := newMockLogger()
		executor := NewExecutor(logger)
		executor.SetDryRun(true)
		ctx := context.Background()

		// When: Execute dry-run operations
		dryRunOps := []func() (bool, string, error){
			func() (bool, string, error) { return executor.LabelNode(ctx, "test", "label=value", false) },
			func() (bool, string, error) { return executor.UnlabelNode(ctx, "test", "label") },
		}

		// Then: Dry-run operations should succeed consistently
		for i, op := range dryRunOps {
			success, output, err := op()
			assert.True(t, success, "Dry-run operation %d should succeed", i)
			assert.NoError(t, err, "Dry-run operation %d should not error", i)
			assert.NotEmpty(t, output, "Dry-run operation %d should have output", i)
		}
	})
}

// TestInterface_Documentation tests interface documentation compliance
// WHY: Proper interface documentation ensures correct usage and implementation
func TestInterface_Documentation(t *testing.T) {
	t.Run("interface_contract_compliance", func(t *testing.T) {
		// Given: Documented interface contracts
		logger := newMockLogger()
		executor := NewExecutor(logger)
		ctx := context.Background()

		// When: Test documented behaviors
		// 1. GetNode should retrieve node information
		success, output, err := executor.GetNode(ctx, "test-node")
		assert.False(t, success, "GetNode should return false in test environment")
		assert.NotEmpty(t, output, "GetNode should return output (error message)")
		assert.Error(t, err, "GetNode should return error in test environment")

		// 2. Dry-run mode should not execute actual commands
		executor.SetDryRun(true)
		success, output, err = executor.LabelNode(ctx, "test-node", "test=value", false)
		assert.True(t, success, "Dry-run LabelNode should succeed")
		assert.Contains(t, output, "labeled", "Dry-run should simulate success message")
		assert.NoError(t, err, "Dry-run should not return error")

		// 3. IsDryRun should reflect SetDryRun state
		executor.SetDryRun(false)
		assert.False(t, executor.IsDryRun(), "IsDryRun should return false")
		executor.SetDryRun(true)
		assert.True(t, executor.IsDryRun(), "IsDryRun should return true")
	})
}
