// Package nethealthcheck provides the core business logic for network connectivity testing operations
package nethealthcheck

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8ostack-ictl/internal/config"
)


// RunTests executes all tests defined in the configuration
func (nhs *NetHealthCheckService) RunTests(ctx context.Context, cfg *config.NodeTestConf) (*TestResults, error) {
	return nhs.processTests(ctx, cfg, "run")
}

// processTests handles the execution of tests
func (nhs *NetHealthCheckService) processTests(ctx context.Context, cfg *config.NodeTestConf, operation string) (*TestResults, error) {
	nhs.kubectl.SetDryRun(nhs.options.DryRun)

	results := &TestResults{
		TestExecutions: []TestExecution{},
		NetworkValidation: make(map[string]NetworkHealth),
	}

	// Log configuration options being used
	if nhs.options.Parallel {
		nhs.options.Logger.Info("⚡ Parallel execution enabled")
	}
	if nhs.options.Retries > 0 {
		nhs.options.Logger.Info(fmt.Sprintf("🔄 Retry count: %d", nhs.options.Retries))
	}
	if nhs.options.OutputFormat != "" {
		nhs.options.Logger.Info(fmt.Sprintf("📄 Output format: %s", nhs.options.OutputFormat))
	}

	nhs.options.Logger.Info(strings.Repeat("=", 50))

	if nhs.options.DryRun {
		nhs.options.Logger.Info(fmt.Sprintf("🧪 DRY RUN: Simulating test %s for %s (%s)...",
			operation, cfg.GetMetadata().Name, time.Now().Format(time.RFC3339)))
	} else {
		nhs.options.Logger.Info(fmt.Sprintf("🧪 Starting test %s for %s (%s)...",
			operation, cfg.GetMetadata().Name, time.Now().Format(time.RFC3339)))
	}

	for _, testConfig := range cfg.Spec.Tests {
		nhs.options.Logger.Info(fmt.Sprintf("🔬 Executing test: %s", testConfig.Name))
		if testConfig.Description != "" {
			nhs.options.Logger.Info(fmt.Sprintf("  Description: %s", testConfig.Description))
		}

		// Execute the actual test
		testExecution, err := nhs.executeNetworkTest(ctx, testConfig)
		if err != nil {
			nhs.options.Logger.Error(fmt.Sprintf("Failed to execute test %s: %v", testConfig.Name, err))
			results.Errors = append(results.Errors, err)
			results.FailedTests++
		} else {
			// Check if test result matches expectation
			testPassed := testExecution.ActualSuccess == testExecution.ExpectSuccess
			
			// Debug logging for final comparison
			nhs.options.Logger.Debug(fmt.Sprintf("🔍 Test %s final: actualSuccess=%v expectSuccess=%v testPassed=%v", testConfig.Name, testExecution.ActualSuccess, testExecution.ExpectSuccess, testPassed))
			
			if testPassed {
				nhs.options.Logger.Info(fmt.Sprintf("✅ Test %s completed successfully in %v", testConfig.Name, testExecution.Duration))
				results.SuccessfulTests++
			} else {
				nhs.options.Logger.Warn(fmt.Sprintf("❌ Test %s failed: %s", testConfig.Name, testExecution.ErrorMessage))
				results.FailedTests++
			}
			results.TestExecutions = append(results.TestExecutions, *testExecution)
		}

		results.TotalTests++
	}

	// Print summary
	nhs.options.Logger.Info(strings.Repeat("=", 50))
	nhs.options.Logger.Info("📊 Network Test Summary:")
	nhs.options.Logger.Info(fmt.Sprintf("  Total tests executed: %d", results.TotalTests))
	nhs.options.Logger.Info(fmt.Sprintf("  Successful tests: %d", results.SuccessfulTests))
	nhs.options.Logger.Info(fmt.Sprintf("  Failed tests: %d", results.FailedTests))

	if len(results.Errors) > 0 {
		nhs.options.Logger.Warn(fmt.Sprintf("  Errors encountered: %d", len(results.Errors)))
	}

	// Cleanup test pods after operations
	if nhs.options.CleanupAfterTests {
		nhs.cleanupTestPods(ctx)
	}

	return results, nil
}

// StopTests stops any running tests
func (nhs *NetHealthCheckService) StopTests(ctx context.Context, config *config.NodeTestConf) (*TestResults, error) {
	fmt.Println("Stopping network health tests...")
	return &TestResults{}, nil
}

// VerifyTests checks if test infrastructure is configured correctly
func (nhs *NetHealthCheckService) VerifyTests(ctx context.Context, cfg *config.NodeTestConf) (*TestResults, error) {
	return nhs.processTests(ctx, cfg, "verify")
}

// GetCurrentState discovers the current network health state
func (nhs *NetHealthCheckService) GetCurrentState(ctx context.Context, networks []string) (map[string]NetworkHealth, error) {
	state := make(map[string]NetworkHealth)

	for _, networkName := range networks {
		health := NetworkHealth{
			NetworkName:     networkName,
			Subnet:          "10.100.0.0/24", // This would be discovered from actual network config
			HealthyNodes:    []string{"rsb2", "rsb3", "rsb4"},
			UnhealthyNodes:  []string{},
			ServiceStatus:   make(map[string]bool),
			IsolationStatus: make(map[string]bool),
			OverallHealth:   "healthy",
		}
		state[networkName] = health
	}

	return state, nil
}

// executeNetworkTest performs an actual network connectivity test
func (nhs *NetHealthCheckService) executeNetworkTest(ctx context.Context, testConfig config.ConnectivityTest) (*TestExecution, error) {
	startTime := time.Now()

	// Get source and target node mappings from network names
	sourceNodes, err := nhs.getNodesForNetwork(testConfig.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to get source nodes for network %s: %w", testConfig.Source, err)
	}

	if len(sourceNodes) == 0 {
		return nil, fmt.Errorf("no nodes found for source network %s", testConfig.Source)
	}

	// Use first source node for simplicity (could be enhanced to test from multiple)
	sourceNode := sourceNodes[0]

	// Test against each target network
	var allResults []string
	overallSuccess := true
	var firstError error

	for _, targetNetwork := range testConfig.Targets {
		targetNodes, err := nhs.getNodesForNetwork(targetNetwork)
		if err != nil {
			nhs.options.Logger.Warn(fmt.Sprintf("Failed to get target nodes for network %s: %v", targetNetwork, err))
			continue
		}

		for _, targetNode := range targetNodes {
			targetIP, err := nhs.getNodeIPForNetwork(targetNode, targetNetwork)
			if err != nil {
				nhs.options.Logger.Warn(fmt.Sprintf("Failed to get IP for node %s in network %s: %v", targetNode, targetNetwork, err))
				continue
			}

			success, output, err := nhs.executePingTest(ctx, sourceNode, targetIP)
			allResults = append(allResults, fmt.Sprintf("%s->%s(%s): %s", sourceNode, targetNode, targetIP, output))
			
			// Debug logging for ping test results
			nhs.options.Logger.Debug(fmt.Sprintf("🔍 Ping result: %s->%s success=%v err=%v", sourceNode, targetIP, success, err))
			
			if err != nil && firstError == nil {
				firstError = err
				nhs.options.Logger.Debug(fmt.Sprintf("🔍 First error captured: %v", err))
			}
			
			if !success {
				overallSuccess = false
				nhs.options.Logger.Debug(fmt.Sprintf("🔍 Overall success set to false due to ping failure"))
			}
		}
	}

	// Debug logging for test execution summary
	nhs.options.Logger.Debug(fmt.Sprintf("🔍 Test %s: overallSuccess=%v expectSuccess=%v firstError=%v", testConfig.Name, overallSuccess, testConfig.ExpectSuccess, firstError))

	// Create test execution result
	testExecution := &TestExecution{
		TestName:      testConfig.Name,
		TestType:      "ping",
		SourceNode:    sourceNode,
		TargetNode:    fmt.Sprintf("%v", testConfig.Targets), // Multiple targets
		SourceNetwork: testConfig.Source,
		TargetNetwork: strings.Join(testConfig.Targets, ","),
		Protocol:      "icmp",
		ExpectSuccess: testConfig.ExpectSuccess,
		ActualSuccess: overallSuccess,
		Duration:      time.Since(startTime),
		Output:        strings.Join(allResults, "; "),
	}

	if firstError != nil {
		testExecution.ErrorMessage = firstError.Error()
	}

	// Handle dry run mode
	if nhs.options.DryRun {
		nhs.options.Logger.Info(fmt.Sprintf("🧪 DRY RUN: Would execute ping test %s", testConfig.Name))
		testExecution.ActualSuccess = testConfig.ExpectSuccess // Assume expected result in dry run
		testExecution.Output = "DRY RUN: Test would execute as expected"
		testExecution.ErrorMessage = ""
	}

	return testExecution, nil
}

// getNodesForNetwork retrieves node names for a given network using role-based discovery
// This fixes the false positive issue where control plane nodes were being tested for isolation
func (nhs *NetHealthCheckService) getNodesForNetwork(networkName string) ([]string, error) {
	// Map network names to actual node roles for proper test selection
	roleMapping := map[string]string{
		"storage": "storage",      // Use dedicated storage nodes (rsb5, rsb6)
		"api": "control-plane",    // Use control plane nodes (rsb2, rsb3, rsb4)
		"tenant": "compute",       // Use compute nodes (rsb7, rsb8)
		"management": "all",       // Management network spans all nodes
	}

	// Get target role for this network
	targetRole, exists := roleMapping[networkName]
	if !exists {
		// Fallback to VLAN-based selection for unknown networks
		nhs.options.Logger.Warn(fmt.Sprintf("Unknown network %s, using VLAN-based selection", networkName))
		return nhs.getNodesForNetworkVLANBased(networkName)
	}

	// Handle special case for management network (all nodes)
	if targetRole == "all" {
		return nhs.getAllNodesFromCluster()
	}

	// Use role-based discovery to get nodes
	return nhs.getNodesByRole(targetRole)
}

// getNodesForNetworkVLANBased provides fallback VLAN-based node selection
func (nhs *NetHealthCheckService) getNodesForNetworkVLANBased(networkName string) ([]string, error) {
	if nhs.vlanConfig == nil {
		return nil, fmt.Errorf("no VLAN configuration available for network mapping")
	}

	// Get the VLAN configuration for the specified network
	vlanConfig, exists := nhs.vlanConfig.Spec.VLANs[networkName]
	if !exists {
		return nil, fmt.Errorf("network %s not found in VLAN configuration", networkName)
	}

	// Extract all node names from the nodeMapping
	var nodes []string
	for nodeName := range vlanConfig.NodeMapping {
		nodes = append(nodes, nodeName)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes found for network %s", networkName)
	}

	return nodes, nil
}

// getNodesByRole uses kubectl discovery to get nodes by their actual role labels
func (nhs *NetHealthCheckService) getNodesByRole(role string) ([]string, error) {
	// Get all nodes from cluster
	success, allNodesOutput, err := nhs.kubectl.GetAllNodes(context.Background())
	if err != nil || !success {
		return nil, fmt.Errorf("failed to get cluster nodes: %w", err)
	}

	// Parse node list
	nodeNames := strings.Split(strings.TrimSpace(allNodesOutput), "\n")
	var roleNodes []string

	for _, nodeName := range nodeNames {
		if nodeName == "" {
			continue
		}
		
		// Strip "node/" prefix if present
		cleanNodeName := strings.TrimPrefix(nodeName, "node/")
		
		// Get role for this node
		nodeRole, err := nhs.kubectl.GetNodeRole(context.Background(), cleanNodeName)
		if err != nil {
			nhs.options.Logger.Warn(fmt.Sprintf("Failed to get role for node %s: %v", cleanNodeName, err))
			continue
		}
		
		// Add node if role matches and not in exclusion list
		if nodeRole == role && !nhs.isNodeExcluded(cleanNodeName) {
			roleNodes = append(roleNodes, cleanNodeName)
		} else if nodeRole == role && nhs.isNodeExcluded(cleanNodeName) {
			nhs.options.Logger.Info(fmt.Sprintf("Excluding node %s from tests (in exclusion list)", cleanNodeName))
		}
	}

	if len(roleNodes) == 0 {
		return nil, fmt.Errorf("no nodes found with role %s (after applying exclusions)", role)
	}

	nhs.options.Logger.Info(fmt.Sprintf("Found %d nodes with role %s: %v", len(roleNodes), role, roleNodes))
	return roleNodes, nil
}

// getAllNodesFromCluster gets all nodes for management network tests
func (nhs *NetHealthCheckService) getAllNodesFromCluster() ([]string, error) {
	success, allNodesOutput, err := nhs.kubectl.GetAllNodes(context.Background())
	if err != nil || !success {
		return nil, fmt.Errorf("failed to get cluster nodes: %w", err)
	}

	// Parse node list
	nodeNames := strings.Split(strings.TrimSpace(allNodesOutput), "\n")
	var cleanNodes []string

	for _, nodeName := range nodeNames {
		if nodeName == "" {
			continue
		}
		// Strip "node/" prefix if present
		cleanNodeName := strings.TrimPrefix(nodeName, "node/")
		
		// Only include nodes that are not in the exclusion list
		if !nhs.isNodeExcluded(cleanNodeName) {
			cleanNodes = append(cleanNodes, cleanNodeName)
		} else {
			nhs.options.Logger.Info(fmt.Sprintf("Excluding node %s from management network tests (in exclusion list)", cleanNodeName))
		}
	}

	return cleanNodes, nil
}

// getNodeIPForNetwork retrieves a node's IP address for a given network from VLAN config
func (nhs *NetHealthCheckService) getNodeIPForNetwork(nodeName, networkName string) (string, error) {
	if nhs.vlanConfig == nil {
		return "", fmt.Errorf("no VLAN configuration available for IP mapping")
	}

	// Get the VLAN configuration for the specified network
	vlanConfig, exists := nhs.vlanConfig.Spec.VLANs[networkName]
	if !exists {
		return "", fmt.Errorf("network %s not found in VLAN configuration", networkName)
	}

	// Get the IP address for the specified node in this network
	ipAddress, exists := vlanConfig.NodeMapping[nodeName]
	if !exists {
		return "", fmt.Errorf("node %s not found in network %s", nodeName, networkName)
	}

	// Extract just the IP address (remove /24 CIDR notation)
	if strings.Contains(ipAddress, "/") {
		parts := strings.Split(ipAddress, "/")
		ipAddress = parts[0]
	}

	return ipAddress, nil
}

// executePingTest performs a ping test between two nodes
func (nhs *NetHealthCheckService) executePingTest(ctx context.Context, sourceNode, targetIP string) (bool, string, error) {
	command := fmt.Sprintf("ping -c 3 %s", targetIP)
	nhs.options.Logger.Info(fmt.Sprintf("📡 Executing ping test: %s -> %s", sourceNode, targetIP))

	success, output, err := nhs.kubectl.ExecNodeCommand(ctx, sourceNode, command)
	if err != nil {
		return false, output, fmt.Errorf("failed to execute ping test: %w", err)
	}

	return success, output, nil
}

// cleanupTestPods automatically cleans up test pods after operations
func (nhs *NetHealthCheckService) cleanupTestPods(ctx context.Context) {
	nhs.options.Logger.Info("🧹 Cleaning up test pods...")

	// Give pods a moment to transition to final status
	cleanupDelay := nhs.options.TestDelay
	if cleanupDelay == 0 {
		cleanupDelay = 3 * time.Second // Default for production
	}
	time.Sleep(cleanupDelay)

	// Get all pods
	success, output, err := nhs.kubectl.GetPods(ctx, "", "")
	if err != nil || !success {
		nhs.options.Logger.Warn(fmt.Sprintf("Failed to get pods: %v", err))
		return
	}

	// Filter for debug pods
	podNames := strings.Split(output, "\n")
	var debugPods []string
	for _, podName := range podNames {
		if strings.Contains(podName, "node-debugger") {
			debugPods = append(debugPods, strings.TrimPrefix(podName, "pod/"))
		}
	}

	// Delete each debug pod
	deletedCount := 0
	for _, podName := range debugPods {
		success, _, err := nhs.kubectl.DeletePod(ctx, podName)
		if err != nil {
			nhs.options.Logger.Warn(fmt.Sprintf("Failed to delete pod %s: %v", podName, err))
		} else if success {
			deletedCount++
		}
	}

	if deletedCount > 0 {
		nhs.options.Logger.Info(fmt.Sprintf("✅ Cleaned up %d test pods", deletedCount))
	} else {
		nhs.options.Logger.Info("✅ No test pods to clean up")
	}
}

// isNodeExcluded checks if a node is in the exclusion list
func (nhs *NetHealthCheckService) isNodeExcluded(nodeName string) bool {
	for _, excludedNode := range nhs.options.ExcludeNodes {
		if excludedNode == nodeName {
			return true
		}
	}
	return false
}
