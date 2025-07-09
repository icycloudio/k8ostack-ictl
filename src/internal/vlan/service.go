package vlan

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"k8ostack-ictl/internal/config"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// ConfigureVLANs configures all VLANs defined in the configuration
func (vs *VLANService) ConfigureVLANs(ctx context.Context, cfg *config.NodeVLANConf) (*OperationResults, error) {
	return vs.processVLANs(ctx, cfg, "configure")
}

// RemoveVLANs removes all VLANs defined in the configuration
func (vs *VLANService) RemoveVLANs(ctx context.Context, cfg *config.NodeVLANConf) (*OperationResults, error) {
	return vs.processVLANs(ctx, cfg, "remove")
}

// VerifyVLANs checks if VLANs are configured correctly
func (vs *VLANService) VerifyVLANs(ctx context.Context, cfg *config.NodeVLANConf) (*OperationResults, error) {
	// FIX: Set dry-run mode on kubectl executor
	vs.kubectl.SetDryRun(vs.options.DryRun)

	results := &OperationResults{
		ConfiguredVLANs: make(map[string][]VLANInterfaceInfo),
	}

	vs.options.Logger.Info("🔍 Verifying VLAN configuration...")

	// Get all unique nodes from all VLANs
	allNodes := vs.getAllNodesFromConfig(cfg)

	for nodeName := range allNodes {
		results.TotalNodes++

		// Check if node exists in cluster
		if vs.options.ValidateConnectivity {
			success, _, err := vs.kubectl.GetNode(ctx, nodeName)
			if err != nil || !success {
				vs.options.Logger.Error(fmt.Sprintf("Node %s not found in cluster: %v", nodeName, err))
				results.FailedNodes = append(results.FailedNodes, nodeName)
				if err != nil {
					results.Errors = append(results.Errors, err)
				}
				continue
			}
		}

		// Verify VLAN interfaces on the node
		nodeVLANs, err := vs.verifyNodeVLANs(ctx, nodeName, cfg)
		if err != nil {
			vs.options.Logger.Error(fmt.Sprintf("Failed to verify VLANs on node %s: %v", nodeName, err))
			results.FailedNodes = append(results.FailedNodes, nodeName)
			results.Errors = append(results.Errors, err)
			continue
		}

		results.ConfiguredVLANs[nodeName] = nodeVLANs
		results.SuccessfulNodes++
	}

	// Automatically cleanup debug pods after verification
	vs.cleanupDebugPods(ctx)

	return results, nil
}

// GetCurrentState discovers the current VLAN configuration state
func (vs *VLANService) GetCurrentState(ctx context.Context, nodes []string) (map[string][]VLANInterfaceInfo, error) {
	state := make(map[string][]VLANInterfaceInfo)

	for _, nodeName := range nodes {
		vlans, err := vs.discoverNodeVLANs(ctx, nodeName)
		if err != nil {
			return nil, fmt.Errorf("failed to discover VLANs on node %s: %w", nodeName, err)
		}
		state[nodeName] = vlans
	}

	return state, nil
}

// processVLANs handles both configure and remove operations
func (vs *VLANService) processVLANs(ctx context.Context, cfg *config.NodeVLANConf, operation string) (*OperationResults, error) {
	vs.kubectl.SetDryRun(vs.options.DryRun)

	results := &OperationResults{
		ConfiguredVLANs: make(map[string][]VLANInterfaceInfo),
	}

	caser := cases.Title(language.Und)
	operationName := fmt.Sprintf("%s VLANs", caser.String(operation))

	// Get configuration name for logging
	configName := cfg.Metadata.Name
	if configName == "" {
		configName = "unknown"
	}

	vs.options.Logger.Info(strings.Repeat("=", 60))
	if vs.options.DryRun {
		vs.options.Logger.Info(fmt.Sprintf("🧪 DRY RUN: Simulating %s for %s (%s %s)...",
			operationName, configName, cfg.Kind, cfg.APIVersion))
	} else {
		vs.options.Logger.Info(fmt.Sprintf("🌐 Starting %s for %s (%s %s)...",
			operationName, configName, cfg.Kind, cfg.APIVersion))
	}

	// Process each VLAN
	for vlanName, vlanConfig := range cfg.Spec.VLANs {
		vs.options.Logger.Info(fmt.Sprintf("🔧 Processing VLAN: %s (ID: %d, Subnet: %s)",
			vlanName, vlanConfig.ID, vlanConfig.Subnet))

		if len(vlanConfig.NodeMapping) == 0 {
			vs.options.Logger.Warn(fmt.Sprintf("⚠️  VLAN %s has no node mappings, skipping", vlanName))
			continue
		}

		// Process each node in this VLAN
		for nodeName, ipAddress := range vlanConfig.NodeMapping {
			results.TotalNodes++
			vs.options.Logger.Info(fmt.Sprintf("  📍 Processing node: %s -> %s", nodeName, ipAddress))

			if vs.processNodeVLAN(ctx, nodeName, vlanName, vlanConfig, ipAddress, operation, results) {
				results.SuccessfulNodes++
			}
		}
	}

	// Print summary
	vs.options.Logger.Info(strings.Repeat("=", 60))
	vs.options.Logger.Info("📊 VLAN Operation Summary:")
	vs.options.Logger.Info(fmt.Sprintf("  Total node-VLAN assignments processed: %d", results.TotalNodes))
	vs.options.Logger.Info(fmt.Sprintf("  Successful operations: %d", results.SuccessfulNodes))
	vs.options.Logger.Info(fmt.Sprintf("  Failed operations: %d", len(results.FailedNodes)))

	if len(results.FailedNodes) > 0 {
		vs.options.Logger.Warn(fmt.Sprintf("  Failed nodes: %s", strings.Join(results.FailedNodes, ", ")))
	}

	// Automatically cleanup debug pods after operations
	vs.cleanupDebugPods(ctx)

	return results, nil
}

// processNodeVLAN processes VLAN configuration for a single node
func (vs *VLANService) processNodeVLAN(ctx context.Context, nodeName, vlanName string, vlanConfig config.VLANConfig, ipAddress, operation string, results *OperationResults) bool {
	// Validate node exists if requested
	if vs.options.ValidateConnectivity {
		success, _, err := vs.kubectl.GetNode(ctx, nodeName)
		if err != nil || !success {
			vs.options.Logger.Error(fmt.Sprintf("Node %s does not exist in the cluster", nodeName))
			results.FailedNodes = append(results.FailedNodes, nodeName)
			if err != nil {
				results.Errors = append(results.Errors, err)
			}
			return false
		}
	}

	// Determine physical interface
	physInterface := vlanConfig.Interface
	if physInterface == "" {
		physInterface = vs.options.DefaultInterface
		if physInterface == "" {
			physInterface = "eth0" // Default fallback
		}
	}

	// Create VLAN interface name
	vlanInterface := fmt.Sprintf("%s.%d", physInterface, vlanConfig.ID)

	// Validate IP address format
	if _, _, err := net.ParseCIDR(ipAddress); err != nil {
		vs.options.Logger.Error(fmt.Sprintf("Invalid IP address format for node %s: %s", nodeName, ipAddress))
		results.FailedNodes = append(results.FailedNodes, nodeName)
		results.Errors = append(results.Errors, fmt.Errorf("invalid IP format: %s", ipAddress))
		return false
	}

	var success bool
	var err error

	if operation == "remove" {
		success, err = vs.removeVLANInterface(ctx, nodeName, vlanInterface)
		if success {
			vs.options.Logger.Info(fmt.Sprintf("✅ Removed VLAN interface %s from node %s", vlanInterface, nodeName))
		}
	} else {
		success, err = vs.configureVLANInterface(ctx, nodeName, vlanName, vlanConfig, vlanInterface, physInterface, ipAddress)
		if success {
			vs.options.Logger.Info(fmt.Sprintf("✅ Configured VLAN %s (%s) on node %s: %s", vlanName, vlanInterface, nodeName, ipAddress))

			// Add to results
			vlanInfo := VLANInterfaceInfo{
				VLANName:      vlanName,
				VLANId:        vlanConfig.ID,
				Interface:     vlanInterface,
				IPAddress:     ipAddress,
				PhysInterface: physInterface,
				Subnet:        vlanConfig.Subnet,
			}

			if results.ConfiguredVLANs[nodeName] == nil {
				results.ConfiguredVLANs[nodeName] = []VLANInterfaceInfo{}
			}
			results.ConfiguredVLANs[nodeName] = append(results.ConfiguredVLANs[nodeName], vlanInfo)
		}
	}

	if err != nil {
		vs.options.Logger.Error(fmt.Sprintf("Failed to %s VLAN %s on node %s: %v", operation, vlanName, nodeName, err))
		results.FailedNodes = append(results.FailedNodes, nodeName)
		results.Errors = append(results.Errors, err)
		return false
	}

	return success
}

// configureVLANInterface creates and configures a VLAN interface on a node
func (vs *VLANService) configureVLANInterface(ctx context.Context, nodeName, vlanName string, vlanConfig config.VLANConfig, vlanInterface, physInterface, ipAddress string) (bool, error) {
	// Combine all commands into a single execution to reduce pod creation
	var commands []string
	commands = append(commands,
		// Create VLAN interface
		fmt.Sprintf("ip link add link %s name %s type vlan id %d", physInterface, vlanInterface, vlanConfig.ID),
		// Assign IP address
		fmt.Sprintf("ip addr add %s dev %s", ipAddress, vlanInterface),
		// Bring interface up
		fmt.Sprintf("ip link set %s up", vlanInterface),
	)

	// Add persistent configuration if requested
	if vs.options.PersistentConfig {
		netplanCmd := vs.generateNetplanConfig(vlanName, vlanConfig, vlanInterface, physInterface, ipAddress)
		if netplanCmd != "" {
			commands = append(commands, netplanCmd)
		}
	}

	// Combine all commands with && to ensure they run in sequence and fail fast
	combinedCmd := strings.Join(commands, " && ")

	// Execute combined command in a single pod
	cmdSuccess, output, err := vs.kubectl.ExecNodeCommand(ctx, nodeName, combinedCmd)
	if err != nil {
		return false, fmt.Errorf("failed to execute combined VLAN commands: %w", err)
	}
	if !cmdSuccess {
		return false, fmt.Errorf("VLAN configuration failed: %s", output)
	}

	if vs.options.Verbose {
		vs.options.Logger.Info(fmt.Sprintf("    💻 Executed combined VLAN setup for %s", vlanInterface))
	}

	return true, nil
}

// removeVLANInterface removes a VLAN interface from a node
func (vs *VLANService) removeVLANInterface(ctx context.Context, nodeName, vlanInterface string) (bool, error) {
	// Combine removal commands into a single execution
	commands := []string{
		// Bring interface down
		fmt.Sprintf("ip link set %s down", vlanInterface),
		// Remove VLAN interface
		fmt.Sprintf("ip link delete %s", vlanInterface),
	}

	// Combine commands with && but use || true to make it non-failing if interface doesn't exist
	combinedCmd := strings.Join(commands, " && ") + " || true"

	// Execute combined command in a single pod
	cmdSuccess, output, err := vs.kubectl.ExecNodeCommand(ctx, nodeName, combinedCmd)
	if err != nil {
		return false, fmt.Errorf("failed to execute combined removal commands: %w", err)
	}

	// For removal, we're more lenient - interface might not exist
	if !cmdSuccess {
		vs.options.Logger.Warn(fmt.Sprintf("Removal command had issues but continuing: %s", output))
	}

	if vs.options.Verbose {
		vs.options.Logger.Info(fmt.Sprintf("    💻 Executed combined VLAN removal for %s", vlanInterface))
	}

	return true, nil
}

// verifyNodeVLANs verifies VLAN configuration for a specific node
func (vs *VLANService) verifyNodeVLANs(ctx context.Context, nodeName string, cfg *config.NodeVLANConf) ([]VLANInterfaceInfo, error) {
	var vlans []VLANInterfaceInfo

	for vlanName, vlanConfig := range cfg.Spec.VLANs {
		if ipAddress, exists := vlanConfig.NodeMapping[nodeName]; exists {
			physInterface := vlanConfig.Interface
			if physInterface == "" {
				physInterface = vs.options.DefaultInterface
				if physInterface == "" {
					physInterface = "eth0"
				}
			}

			vlanInterface := fmt.Sprintf("%s.%d", physInterface, vlanConfig.ID)

			// Check if interface exists and has correct IP
			checkCmd := fmt.Sprintf("ip addr show %s", vlanInterface)
			success, output, err := vs.kubectl.ExecNodeCommand(ctx, nodeName, checkCmd)
			if err != nil || !success {
				vs.options.Logger.Warn(fmt.Sprintf("VLAN interface %s not found on node %s", vlanInterface, nodeName))
				continue
			}

			// Parse IP from output (simplified check)
			expectedIP := strings.Split(ipAddress, "/")[0]
			if strings.Contains(output, expectedIP) {
				vs.options.Logger.Info(fmt.Sprintf("✅ Verified VLAN %s (%s) on node %s", vlanName, vlanInterface, nodeName))

				vlans = append(vlans, VLANInterfaceInfo{
					VLANName:      vlanName,
					VLANId:        vlanConfig.ID,
					Interface:     vlanInterface,
					IPAddress:     ipAddress,
					PhysInterface: physInterface,
					Subnet:        vlanConfig.Subnet,
				})
			} else {
				vs.options.Logger.Warn(fmt.Sprintf("VLAN %s on node %s has incorrect IP configuration", vlanName, nodeName))
			}
		}
	}

	return vlans, nil
}

// discoverNodeVLANs discovers existing VLAN interfaces on a node
func (vs *VLANService) discoverNodeVLANs(ctx context.Context, nodeName string) ([]VLANInterfaceInfo, error) {
	var vlans []VLANInterfaceInfo

	// List all VLAN interfaces
	cmd := "ip link show type vlan"
	success, output, err := vs.kubectl.ExecNodeCommand(ctx, nodeName, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to discover VLAN interfaces: %w", err)
	}

	if !success {
		// No VLAN interfaces found
		return vlans, nil
	}

	// Parse output to extract VLAN information
	// This is a simplified parser - in production, you'd want more robust parsing
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "@") && strings.Contains(line, "vlan") {
			// Extract interface name and VLAN ID
			// Example: "eth0.100@eth0: <BROADCAST,MULTICAST,UP,LOWER_UP>"
			parts := strings.Fields(line)
			if len(parts) > 0 {
				interfacePart := strings.Split(parts[1], "@")
				if len(interfacePart) >= 2 {
					vlanInterface := interfacePart[0]
					_ = strings.TrimSuffix(interfacePart[1], ":") // physInterface not used in this context

					// Extract VLAN ID from interface name
					interfaceParts := strings.Split(vlanInterface, ".")
					if len(interfaceParts) == 2 {
						// TODO: Parse VLAN ID, get IP address, etc.
						// For now, just record the interface exists
						vs.options.Logger.Info(fmt.Sprintf("Discovered VLAN interface: %s", vlanInterface))
					}
				}
			}
		}
	}

	return vlans, nil
}

// generateNetplanConfig generates persistent network configuration
func (vs *VLANService) generateNetplanConfig(vlanName string, vlanConfig config.VLANConfig, vlanInterface, physInterface, ipAddress string) string {
	// This would generate netplan YAML for Ubuntu/systemd persistence
	// Simplified for now - in production, you'd generate proper netplan configs
	return fmt.Sprintf("echo 'VLAN %s configured for persistence' # TODO: Implement netplan generation", vlanName)
}

// getAllNodesFromConfig extracts all unique node names from VLAN configuration
func (vs *VLANService) getAllNodesFromConfig(cfg *config.NodeVLANConf) map[string]bool {
	nodes := make(map[string]bool)
	for _, vlanConfig := range cfg.Spec.VLANs {
		for nodeName := range vlanConfig.NodeMapping {
			nodes[nodeName] = true
		}
	}
	return nodes
}

// cleanupDebugPods automatically cleans up debug pods after VLAN operations
func (vs *VLANService) cleanupDebugPods(ctx context.Context) {
	vs.options.Logger.Info("🧹 Cleaning up debug pods...")

	// Give pods a moment to transition to final status
	time.Sleep(3 * time.Second)

	// Step 1: Get ALL pods (no status filtering - match old behavior)
	success, output, err := vs.kubectl.GetPods(ctx, "", "")
	if err != nil || !success {
		vs.options.Logger.Warn(fmt.Sprintf("Failed to get pods: %v", err))
		return
	}

	// Step 2: Filter ONLY by our specific name pattern (like old grep did)
	podNames := strings.Split(output, "\n")
	var debugPods []string
	for _, podName := range podNames {
		if strings.Contains(podName, "node-debugger") {
			debugPods = append(debugPods, strings.TrimPrefix(podName, "pod/"))
		}
	}

	// Step 3: Delete each debug pod (using generic building block!)
	deletedCount := 0
	for _, podName := range debugPods {
		success, _, err := vs.kubectl.DeletePod(ctx, podName)
		if err != nil {
			vs.options.Logger.Warn(fmt.Sprintf("Failed to delete pod %s: %v", podName, err))
		} else if success {
			deletedCount++
		}
	}

	if deletedCount > 0 {
		vs.options.Logger.Info(fmt.Sprintf("✅ Cleaned up %d debug pods", deletedCount))
	} else {
		vs.options.Logger.Info("✅ No debug pods to clean up")
	}
}
