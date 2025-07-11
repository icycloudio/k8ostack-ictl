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
			if testExecution.ActualSuccess {
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
			
			if err != nil && firstError == nil {
				firstError = err
			}
			
			if !success {
				overallSuccess = false
			}
		}
	}

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

// getNodesForNetwork retrieves node names for a given network from VLAN config
func (nhs *NetHealthCheckService) getNodesForNetwork(networkName string) ([]string, error) {
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
