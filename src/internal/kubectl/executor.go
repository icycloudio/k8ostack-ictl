package kubectl

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// RealExecutor implements the Executor interface using actual kubectl commands
type RealExecutor struct {
	logger Logger
	dryRun bool
}

// NewExecutor creates a new kubectl executor
func NewExecutor(logger Logger) DryRunExecutor {
	return &RealExecutor{
		logger: logger,
		dryRun: false,
	}
}

// SetDryRun enables or disables dry-run mode
func (e *RealExecutor) SetDryRun(enabled bool) {
	e.dryRun = enabled
}

// IsDryRun returns whether dry-run mode is enabled
func (e *RealExecutor) IsDryRun() bool {
	return e.dryRun
}

// GetNode retrieves information about a specific node
func (e *RealExecutor) GetNode(ctx context.Context, nodeName string) (bool, string, error) {
	return e.runCommand(ctx, []string{"get", "node", nodeName})
}

// LabelNode applies a label to a node
func (e *RealExecutor) LabelNode(ctx context.Context, nodeName, label string, overwrite bool) (bool, string, error) {
	args := []string{"label", "node", nodeName, label}
	if overwrite {
		args = append(args, "--overwrite")
	}

	if e.dryRun {
		e.logger.Debug(fmt.Sprintf("DRY RUN: Would run: kubectl %s", strings.Join(args, " ")))
		return true, fmt.Sprintf("node/%s labeled", nodeName), nil
	}

	return e.runCommand(ctx, args)
}

// UnlabelNode removes a label from a node
func (e *RealExecutor) UnlabelNode(ctx context.Context, nodeName, labelKey string) (bool, string, error) {
	args := []string{"label", "node", nodeName, labelKey + "-"}

	if e.dryRun {
		e.logger.Debug(fmt.Sprintf("DRY RUN: Would run: kubectl %s", strings.Join(args, " ")))
		return true, fmt.Sprintf("node/%s unlabeled", nodeName), nil
	}

	return e.runCommand(ctx, args)
}

// GetNodeLabels retrieves all labels for a specific node
func (e *RealExecutor) GetNodeLabels(ctx context.Context, nodeName string) (bool, string, error) {
	return e.runCommand(ctx, []string{"get", "node", nodeName, "--show-labels"})
}

// ExecNodeCommand executes a command on a specific node
func (e *RealExecutor) ExecNodeCommand(ctx context.Context, nodeName, command string) (bool, string, error) {
	// Use kubectl debug to execute commands on the node
	args := []string{
		"debug", "node/" + nodeName,
		"--profile=sysadmin",
		"--image=busybox",
		"--", "chroot", "/host", "sh", "-c", command,
	}

	if e.dryRun {
		e.logger.Debug(fmt.Sprintf("DRY RUN: Would run: kubectl %s", strings.Join(args, " ")))
		return true, fmt.Sprintf("Command would be executed on node %s: %s", nodeName, command), nil
	}

	return e.runCommand(ctx, args)
}

// GetPods retrieves pods with optional filtering
func (e *RealExecutor) GetPods(ctx context.Context, fieldSelector, labelSelector string) (bool, string, error) {
	args := []string{"get", "pods", "-o", "name"}

	if fieldSelector != "" {
		args = append(args, "--field-selector", fieldSelector)
	}
	if labelSelector != "" {
		args = append(args, "--selector", labelSelector)
	}

	// No dry-run logic here - this is just a GET operation
	return e.runCommand(ctx, args)
}

// DeletePod deletes a specific pod
func (e *RealExecutor) DeletePod(ctx context.Context, podName string) (bool, string, error) {
	args := []string{"delete", "pod", podName}

	if e.dryRun {
		e.logger.Debug(fmt.Sprintf("DRY RUN: Would run: kubectl %s", strings.Join(args, " ")))
		return true, fmt.Sprintf("pod/%s deleted", podName), nil
	}

	return e.runCommand(ctx, args)
}

// runCommand executes a kubectl command
func (e *RealExecutor) runCommand(ctx context.Context, args []string) (bool, string, error) {
	e.logger.Debug(fmt.Sprintf("Running: kubectl %s", strings.Join(args, " ")))

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	if err != nil {
		e.logger.Error(fmt.Sprintf("Command failed: %s", outputStr))
		return false, outputStr, err
	}

	e.logger.Debug(fmt.Sprintf("Command output: %s", outputStr))
	return true, outputStr, nil
}
