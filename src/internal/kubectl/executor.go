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
	logOutput, err := e.waitForPodLogsWithTimeout(ctx, podName, 60*time.Second)
	if err != nil {
		return false, logOutput, err
	}

	// Determine success based on ping results
	// For ping commands, success is determined by whether packets were received
	if strings.Contains(command, "ping") {
		// Check if ping was successful (packets received)
		pingSuccess := !strings.Contains(logOutput, "0 received, 100% packet loss")
		return pingSuccess, logOutput, nil
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

// GetAllNodes retrieves all nodes in the cluster
func (e *RealExecutor) GetAllNodes(ctx context.Context) (bool, string, error) {
	// No dry-run logic here - this is just a GET operation
	return e.runCommand(ctx, []string{"get", "nodes", "-o", "name"})
}

// GetNodesByLabel retrieves nodes using a specific label selector
func (e *RealExecutor) GetNodesByLabel(ctx context.Context, labelSelector string) (bool, string, error) {
	args := []string{"get", "nodes", "-o", "name"}
	
	if labelSelector != "" {
		args = append(args, "-l", labelSelector)
	}
	
	// No dry-run logic here - this is just a GET operation
	return e.runCommand(ctx, args)
}

// GetNodeRole retrieves node role based on labels and analysis
func (e *RealExecutor) GetNodeRole(ctx context.Context, nodeName string) (string, error) {
	// Get node labels first
	success, output, err := e.GetNodeLabels(ctx, nodeName)
	if err != nil || !success {
		return "", fmt.Errorf("failed to get node labels for %s: %w", nodeName, err)
	}
	
	// Analyze labels to determine role
	role := e.analyzeNodeRole(output)
	return role, nil
}

// analyzeNodeRole determines node role from label output
func (e *RealExecutor) analyzeNodeRole(labelOutput string) string {
	// Check for standard Kubernetes node roles
	if strings.Contains(labelOutput, "node-role.kubernetes.io/control-plane") {
		return "control-plane"
	}
	if strings.Contains(labelOutput, "node-role.kubernetes.io/master") {
		return "control-plane"
	}
	
	// Check for OpenStack-specific roles
	if strings.Contains(labelOutput, "openstack-role=storage") {
		return "storage"
	}
	if strings.Contains(labelOutput, "openstack-role=compute") {
		return "compute"
	}
	if strings.Contains(labelOutput, "openstack-role=control-plane") {
		return "control-plane"
	}
	
	// Default to worker if no specific role found
	return "worker"
}

// DiscoverClusterState returns comprehensive cluster overview
func (e *RealExecutor) DiscoverClusterState(ctx context.Context) (map[string]interface{}, error) {
	state := make(map[string]interface{})
	
	// Get all nodes
	success, nodesOutput, err := e.GetAllNodes(ctx)
	if err != nil || !success {
		return nil, fmt.Errorf("failed to get cluster nodes: %w", err)
	}
	
	// Parse node list
	nodeNames := strings.Split(strings.TrimSpace(nodesOutput), "\n")
	nodeCount := len(nodeNames)
	if nodeNames[0] == "" {
		nodeCount = 0
	}
	
	// Count roles
	roleCounts := make(map[string]int)
	for _, nodeName := range nodeNames {
		if nodeName == "" {
			continue
		}
		// Strip "node/" prefix if present
		cleanNodeName := strings.TrimPrefix(nodeName, "node/")
		role, _ := e.GetNodeRole(ctx, cleanNodeName)
		roleCounts[role]++
	}
	
	state["total_nodes"] = nodeCount
	state["node_roles"] = roleCounts
	state["nodes"] = nodeNames
	
	return state, nil
}

// DiscoverNodeVLANs detects VLAN configuration on a specific node
func (e *RealExecutor) DiscoverNodeVLANs(ctx context.Context, nodeName string) (bool, string, error) {
	// Use ExecNodeCommand to run VLAN discovery on the node
	command := "ip link show type vlan"
	
	if e.dryRun {
		e.logger.Debug(fmt.Sprintf("DRY RUN: Would discover VLANs on node %s: %s", nodeName, command))
		return true, fmt.Sprintf("DRY RUN: VLAN discovery on node %s", nodeName), nil
	}
	
	return e.ExecNodeCommand(ctx, nodeName, command)
}

// DiscoverAllVLANs maps VLAN configurations across all nodes
func (e *RealExecutor) DiscoverAllVLANs(ctx context.Context) (map[string]string, error) {
	vlanMap := make(map[string]string)
	
	// Get all nodes first
	success, nodesOutput, err := e.GetAllNodes(ctx)
	if err != nil || !success {
		return nil, fmt.Errorf("failed to get cluster nodes: %w", err)
	}
	
	// Parse node list
	nodeNames := strings.Split(strings.TrimSpace(nodesOutput), "\n")
	
	// Discover VLANs on each node
	for _, nodeName := range nodeNames {
		if nodeName == "" {
			continue
		}
		
		// Strip "node/" prefix if present
		cleanNodeName := strings.TrimPrefix(nodeName, "node/")
		
		success, vlanOutput, err := e.DiscoverNodeVLANs(ctx, cleanNodeName)
		if err != nil {
			e.logger.Warn(fmt.Sprintf("Failed to discover VLANs on node %s: %v", cleanNodeName, err))
			vlanMap[cleanNodeName] = "ERROR"
			continue
		}
		
		if success {
			vlanMap[cleanNodeName] = vlanOutput
		} else {
			vlanMap[cleanNodeName] = "NO_VLANS"
		}
	}
	
	return vlanMap, nil
}

// GetNodeNetworkInfo retrieves network interface information from a node
func (e *RealExecutor) GetNodeNetworkInfo(ctx context.Context, nodeName string) (bool, string, error) {
	// Get comprehensive network information
	command := "ip addr show && echo '---ROUTES---' && ip route show"
	
	if e.dryRun {
		e.logger.Debug(fmt.Sprintf("DRY RUN: Would get network info on node %s: %s", nodeName, command))
		return true, fmt.Sprintf("DRY RUN: Network info for node %s", nodeName), nil
	}
	
	return e.ExecNodeCommand(ctx, nodeName, command)
}

// GetNodeHardwareInfo gets basic hardware specifications for node categorization
func (e *RealExecutor) GetNodeHardwareInfo(ctx context.Context, nodeName string) (bool, string, error) {
	// Get CPU, memory, and storage information
	command := "echo 'CPU:' && lscpu | grep -E '^CPU\\(s\\)|^Model name' && echo 'MEMORY:' && free -h && echo 'STORAGE:' && lsblk"
	
	if e.dryRun {
		e.logger.Debug(fmt.Sprintf("DRY RUN: Would get hardware info on node %s: %s", nodeName, command))
		return true, fmt.Sprintf("DRY RUN: Hardware info for node %s", nodeName), nil
	}
	
	return e.ExecNodeCommand(ctx, nodeName, command)
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
					// Check if it's an expected failure (e.g., no ping response for isolation tests)
					if strings.Contains(logs, "0 received, 100% packet loss") {
						return logs, nil // Considered a success for isolation
					}
					return logs, fmt.Errorf("unexpected pod %s failure: %s", podName, logs)
				}

				return logs, nil
			}

			// Wait before checking again
			time.Sleep(e.pollingInterval)
		}
	}
}
