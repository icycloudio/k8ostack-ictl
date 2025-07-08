package labeler

import (
	"context"
	"fmt"
	"strings"

	"k8ostack-ictl/internal/config"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// ApplyLabels applies all labels defined in the configuration
func (ls *LabelingService) ApplyLabels(ctx context.Context, cfg config.Config) (*OperationResults, error) {
	return ls.processLabels(ctx, cfg, "apply")
}

// RemoveLabels removes all labels defined in the configuration
func (ls *LabelingService) RemoveLabels(ctx context.Context, cfg config.Config) (*OperationResults, error) {
	return ls.processLabels(ctx, cfg, "remove")
}

// VerifyLabels checks if labels are applied correctly
func (ls *LabelingService) VerifyLabels(ctx context.Context, cfg config.Config) (*OperationResults, error) {
	results := &OperationResults{
		AppliedLabels: make(map[string][]string),
	}

	ls.options.Logger.Info("🔍 Verifying applied labels...")

	for _, roleConfig := range cfg.GetNodeRoles() {
		for _, nodeName := range roleConfig.Nodes {
			results.TotalNodes++

			success, output, err := ls.kubectl.GetNodeLabels(ctx, nodeName)
			if err != nil {
				ls.options.Logger.Error(fmt.Sprintf("Failed to verify labels on node %s: %v", nodeName, err))
				results.FailedNodes = append(results.FailedNodes, nodeName)
				results.Errors = append(results.Errors, err)
				continue
			}

			if success {
				verified := []string{}
				for labelKey, labelValue := range roleConfig.Labels {
					expectedLabel := fmt.Sprintf("%s=%s", labelKey, labelValue)
					if strings.Contains(output, expectedLabel) {
						ls.options.Logger.Info(fmt.Sprintf("✅ Verified label %s on node %s", expectedLabel, nodeName))
						verified = append(verified, expectedLabel)
					} else {
						ls.options.Logger.Warn(fmt.Sprintf("⚠️  Label %s not found on node %s", expectedLabel, nodeName))
					}
				}
				results.AppliedLabels[nodeName] = verified
				if len(verified) == len(roleConfig.Labels) {
					results.SuccessfulNodes++
				} else {
					results.FailedNodes = append(results.FailedNodes, nodeName)
				}
			}
		}
	}

	return results, nil
}

// GetCurrentState discovers the current labeling state
func (ls *LabelingService) GetCurrentState(ctx context.Context, nodes []string) (map[string]map[string]string, error) {
	state := make(map[string]map[string]string)

	for _, nodeName := range nodes {
		success, _, err := ls.kubectl.GetNodeLabels(ctx, nodeName)
		if err != nil {
			return nil, fmt.Errorf("failed to get labels for node %s: %w", nodeName, err)
		}

		if success {
			// Parse labels from output - this is a simplified implementation
			nodeLabels := make(map[string]string)
			// TODO: Implement proper label parsing from kubectl output
			state[nodeName] = nodeLabels
		}
	}

	return state, nil
}

// processLabels handles both apply and remove operations
func (ls *LabelingService) processLabels(ctx context.Context, cfg config.Config, operation string) (*OperationResults, error) {
	ls.kubectl.SetDryRun(ls.options.DryRun)

	results := &OperationResults{
		AppliedLabels: make(map[string][]string),
	}

	caser := cases.Title(language.Und)
	operationName := operation
	if operation == "remove" {
		operationName = "remove"
	}

	// Get configuration name for logging
	configName := cfg.GetMetadata().Name
	if configName == "" {
		configName = "unknown"
	}

	ls.options.Logger.Info(strings.Repeat("=", 50))
	if ls.options.DryRun {
		ls.options.Logger.Info(fmt.Sprintf("🧪 DRY RUN: Simulating label %s for %s (%s %s)...",
			operationName, configName, cfg.GetKind(), cfg.GetAPIVersion()))
	} else {
		ls.options.Logger.Info(fmt.Sprintf("Starting label %s for %s (%s %s)...",
			operationName, configName, cfg.GetKind(), cfg.GetAPIVersion()))
	}

	for role, roleConfig := range cfg.GetNodeRoles() {
		roleName := caser.String(strings.ReplaceAll(role, "_", " "))

		ls.options.Logger.Info(fmt.Sprintf("Processing %s role with %d nodes...", roleName, len(roleConfig.Nodes)))
		if roleConfig.Description != "" {
			ls.options.Logger.Info(fmt.Sprintf("  Description: %s", roleConfig.Description))
		}

		// Log labels being processed
		labelList := []string{}
		for key, value := range roleConfig.Labels {
			labelList = append(labelList, fmt.Sprintf("%s=%s", key, value))
		}
		ls.options.Logger.Info(fmt.Sprintf("  Labels: %s", strings.Join(labelList, ", ")))

		for _, nodeName := range roleConfig.Nodes {
			results.TotalNodes++
			ls.options.Logger.Info(fmt.Sprintf("  Processing node: %s", nodeName))

			if ls.processNodeLabels(ctx, nodeName, roleConfig.Labels, operation, results) {
				results.SuccessfulNodes++
			}
		}

		ls.options.Logger.Info(fmt.Sprintf("Completed %s role processing", roleName))
	}

	// Print summary
	ls.options.Logger.Info(strings.Repeat("=", 50))
	ls.options.Logger.Info("📊 Operation Summary:")
	ls.options.Logger.Info(fmt.Sprintf("  Total node assignments processed: %d", results.TotalNodes))
	ls.options.Logger.Info(fmt.Sprintf("  Successful operations: %d", results.SuccessfulNodes))
	ls.options.Logger.Info(fmt.Sprintf("  Failed operations: %d", len(results.FailedNodes)))

	if len(results.FailedNodes) > 0 {
		ls.options.Logger.Warn(fmt.Sprintf("  Failed nodes: %s", strings.Join(results.FailedNodes, ", ")))
	}

	return results, nil
}

// processNodeLabels processes labels for a single node
func (ls *LabelingService) processNodeLabels(ctx context.Context, nodeName string, labels map[string]string, operation string, results *OperationResults) bool {
	// Check if node exists
	if ls.options.ValidateNodes {
		success, _, err := ls.kubectl.GetNode(ctx, nodeName)
		if err != nil || !success {
			ls.options.Logger.Error(fmt.Sprintf("Node %s does not exist in the cluster", nodeName))
			results.FailedNodes = append(results.FailedNodes, nodeName)
			if err != nil {
				results.Errors = append(results.Errors, err)
			}
			return false
		}
	}

	allSuccess := true
	appliedLabels := []string{}

	for labelKey, labelValue := range labels {
		var success bool
		var output string
		var err error

		if operation == "remove" {
			success, output, err = ls.kubectl.UnlabelNode(ctx, nodeName, labelKey)
			if success {
				ls.options.Logger.Info(fmt.Sprintf("✅ Removed label %s from node %s: %s", labelKey, nodeName, output))
				appliedLabels = append(appliedLabels, "-"+labelKey)
			}
		} else {
			labelStr := fmt.Sprintf("%s=%s", labelKey, labelValue)
			success, output, err = ls.kubectl.LabelNode(ctx, nodeName, labelStr, true)
			if success {
				ls.options.Logger.Info(fmt.Sprintf("✅ Applied label %s to node %s: %s", labelStr, nodeName, output))
				appliedLabels = append(appliedLabels, labelStr)
			}
		}

		if err != nil {
			ls.options.Logger.Error(fmt.Sprintf("Failed to process label %s on node %s: %v", labelKey, nodeName, err))
			allSuccess = false
			results.Errors = append(results.Errors, err)
		}
	}

	if allSuccess {
		results.AppliedLabels[nodeName] = appliedLabels
	} else {
		results.FailedNodes = append(results.FailedNodes, nodeName)
	}

	return allSuccess
}
