package kubectl

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// RealExecutor implements the Executor interface using actual kubectl commands
type RealExecutor struct {
	logger         Logger
	dryRun         bool
	pollingInterval time.Duration
}

// NewExecutor creates a new kubectl executor
func NewExecutor(logger Logger) DryRunExecutor {
	return &RealExecutor{
		logger:         logger,
		dryRun:         false,
		pollingInterval: 1 * time.Second, // Default polling interval
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

// SetPollingInterval sets the polling interval for waiting for pod completion
func (e *RealExecutor) SetPollingInterval(interval time.Duration) {
	e.pollingInterval = interval
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

// ExecNodeCommand executes a command on a specific node using kubectl debug
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

	// Execute kubectl debug command
	_, output, err := e.runCommand(ctx, args)
	if err != nil {
		return false, output, err
	}

	// kubectl debug is asynchronous and only returns pod creation message
	// We need to extract the pod name and get its logs
	podName := e.extractPodNameFromDebugOutput(output)
	if podName == "" {
		return false, output, fmt.Errorf("failed to extract pod name from debug output: %s", output)
	}

	// Wait for pod to complete and get logs
	logOutput, err := e.waitForPodLogsWithTimeout(ctx, podName, 30*time.Second)
	if err != nil {
		return false, logOutput, err
	}

	return true, logOutput, nil
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

// extractPodNameFromDebugOutput extracts the pod name from kubectl debug output
// Example input: "Creating debugging pod node-debugger-rsb4-q4cxv with container debugger on node rsb4."
// Example output: "node-debugger-rsb4-q4cxv"
func (e *RealExecutor) extractPodNameFromDebugOutput(output string) string {
	// Use regex to extract pod name from kubectl debug output
	regex := regexp.MustCompile(`Creating debugging pod ([a-zA-Z0-9-]+) with container`)
	matches := regex.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// waitForPodLogsWithTimeout waits for a pod to complete and returns its logs
func (e *RealExecutor) waitForPodLogsWithTimeout(ctx context.Context, podName string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Wait for pod to reach a terminal state (Succeeded or Failed)
	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("timeout waiting for pod %s to complete", podName)
		default:
			// Check pod status
			args := []string{"get", "pod", podName, "-o", "jsonpath={.status.phase}"}
			success, phase, err := e.runCommand(ctx, args)
			if err != nil {
				// Pod might not exist yet, wait a bit
				time.Sleep(e.pollingInterval)
				continue
			}

			if success && (phase == "Succeeded" || phase == "Failed") {
				// Pod completed, get logs
				logArgs := []string{"logs", podName}
				logSuccess, logs, logErr := e.runCommand(ctx, logArgs)
				if logErr != nil {
					return "", fmt.Errorf("failed to get logs from pod %s: %w", podName, logErr)
				}

				if !logSuccess {
					return "", fmt.Errorf("failed to retrieve logs from pod %s", podName)
				}

				if phase == "Failed" {
					return logs, fmt.Errorf("pod %s failed: %s", podName, logs)
				}

				return logs, nil
			}

			// Wait before checking again
			time.Sleep(e.pollingInterval)
		}
	}
}
