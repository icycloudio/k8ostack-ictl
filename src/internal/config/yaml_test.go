// Package config provides unit tests for YAML document parsing utilities
// WHY: YAML parsing is critical for multi-CRD configuration loading and must handle various edge cases
package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSplitYAMLDocuments tests YAML document splitting functionality
// WHY: Multi-document YAML parsing enables unified configuration files for complex deployments
func TestSplitYAMLDocuments(t *testing.T) {
	tests := []struct {
		name          string
		description   string
		input         []byte
		expectedCount int
		shouldError   bool
		errorText     string
	}{
		{
			name:          "empty_yaml_error",
			description:   "Empty YAML data should return error as no configuration is provided",
			input:         []byte{},
			expectedCount: 0,
			shouldError:   true,
			errorText:     "empty YAML data",
		},
		{
			name:        "single_document_yaml",
			description: "Single YAML document should return one document for processing",
			input: []byte(`apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeLabelConf
metadata:
  name: test-config`),
			expectedCount: 1,
			shouldError:   false,
		},
		{
			name:        "multi_document_yaml_with_separators",
			description: "Multi-document YAML should split correctly for independent processing",
			input: []byte(`apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeLabelConf
metadata:
  name: labels-config
---
apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeVLANConf
metadata:
  name: vlans-config`),
			expectedCount: 2,
			shouldError:   false,
		},
		{
			name:        "three_document_yaml_complex",
			description: "Three-document YAML should handle complete multi-CRD configurations",
			input: []byte(`apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeLabelConf
metadata:
  name: production-labels
spec:
  nodeRoles:
    control:
      nodes: ["rsb2", "rsb3"]
      labels:
        role: "control"
---
apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeVLANConf
metadata:
  name: production-vlans
spec:
  vlans:
    management:
      id: 100
      subnet: "192.168.100.0/24"
---
apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeTestConf
metadata:
  name: production-tests
spec:
  tests:
    - name: ping-test
      source: node1
      targets: ["node2"]`),
			expectedCount: 3,
			shouldError:   false,
		},
		{
			name:        "yaml_with_empty_documents",
			description: "YAML with empty documents should skip empty sections and process valid ones",
			input: []byte(`apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeLabelConf
metadata:
  name: valid-config
---

---
apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeVLANConf
metadata:
  name: another-valid-config
---

`),
			expectedCount: 2,
			shouldError:   false,
		},
		{
			name:        "yaml_with_starting_separator",
			description: "YAML starting with separator should handle first document correctly",
			input: []byte(`---
apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeLabelConf
metadata:
  name: config-with-leading-separator
---
apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeVLANConf
metadata:
  name: second-config`),
			expectedCount: 2,
			shouldError:   false,
		},
		{
			name:        "only_separators_no_content",
			description: "YAML with only separators should return error as no valid content exists",
			input: []byte(`---
---
---`),
			expectedCount: 0,
			shouldError:   true,
			errorText:     "no valid YAML documents found",
		},
		{
			name:        "whitespace_only_documents",
			description: "YAML with whitespace-only documents should skip empty content",
			input: []byte(`   
   
---
apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeLabelConf
metadata:
  name: whitespace-test
---
   
   `),
			expectedCount: 1,
			shouldError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Split YAML documents
			documents, err := splitYAMLDocuments(tt.input)

			// Then: Verify splitting results
			if tt.shouldError {
				assert.Error(t, err, "Expected error for test case")
				if tt.errorText != "" {
					assert.Contains(t, err.Error(), tt.errorText, "Error should contain expected text")
				}
			} else {
				assert.NoError(t, err, "Unexpected error")
				assert.Equal(t, tt.expectedCount, len(documents), "Document count mismatch")

				// Verify each document is valid YAML content
				for i, doc := range documents {
					assert.NotEmpty(t, doc, "Document %d should not be empty", i)
					assert.NotContains(t, string(doc), "---", "Document %d should not contain separators", i)
				}
			}
		})
	}
}

// TestIsMultiDocumentYAML tests multi-document detection
// WHY: Accurate detection enables proper parsing strategy selection for performance and correctness
func TestIsMultiDocumentYAML(t *testing.T) {
	tests := []struct {
		name           string
		description    string
		input          []byte
		expectMultiDoc bool
	}{
		{
			name:        "single_document_no_separator",
			description: "Single document without separator should be detected as single-doc",
			input: []byte(`apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeLabelConf
metadata:
  name: single-config`),
			expectMultiDoc: false,
		},
		{
			name:        "multi_document_with_unix_separators",
			description: "Multi-document with Unix line endings should be detected correctly",
			input: []byte(`apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeLabelConf
metadata:
  name: first-config
---
apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeVLANConf
metadata:
  name: second-config`),
			expectMultiDoc: true,
		},
		{
			name:           "multi_document_with_windows_separators",
			description:    "Multi-document with Windows line endings should be detected correctly",
			input:          []byte("apiVersion: openstack.kictl.icycloud.io/v1\r\nkind: NodeLabelConf\r\n---\r\napiVersion: openstack.kictl.icycloud.io/v1\r\nkind: NodeVLANConf"),
			expectMultiDoc: true,
		},
		{
			name:        "single_document_with_separator_in_content",
			description: "Single document with separator-like content should not be confused as multi-doc",
			input: []byte(`apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeLabelConf
metadata:
  name: single-config
  annotations:
    description: "This content has --- but not as separator"`),
			expectMultiDoc: false,
		},
		{
			name:           "empty_input",
			description:    "Empty input should be treated as single document",
			input:          []byte{},
			expectMultiDoc: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Check if multi-document
			result := isMultiDocumentYAML(tt.input)

			// Then: Verify detection result
			assert.Equal(t, tt.expectMultiDoc, result, "Multi-document detection mismatch")
		})
	}
}

// TestValidateYAMLDocument tests document validation
// WHY: Document validation prevents processing of malformed configurations that could cause runtime failures
func TestValidateYAMLDocument(t *testing.T) {
	tests := []struct {
		name        string
		description string
		input       []byte
		shouldError bool
		errorText   string
	}{
		{
			name:        "valid_yaml_document",
			description: "Valid YAML document should pass validation for processing",
			input: []byte(`apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeLabelConf
metadata:
  name: valid-config
spec:
  nodeRoles:
    control:
      nodes: ["rsb2"]
      labels:
        role: "control"`),
			shouldError: false,
		},
		{
			name:        "valid_yaml_with_lists",
			description: "Valid YAML with list syntax should pass validation",
			input: []byte(`apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeTestConf
metadata:
  name: test-config
spec:
  tests:
    - name: ping-test
      source: node1
      targets:
        - node2
        - node3`),
			shouldError: false,
		},
		{
			name:        "empty_document_error",
			description: "Empty document should fail validation as it provides no configuration",
			input:       []byte{},
			shouldError: true,
			errorText:   "empty YAML document",
		},
		{
			name:        "whitespace_only_document_error",
			description: "Whitespace-only document should fail validation",
			input: []byte(`   
   
   `),
			shouldError: true,
			errorText:   "invalid YAML document: no key-value pairs or list items found",
		},
		{
			name:        "invalid_yaml_no_structure",
			description: "YAML without proper structure should fail validation",
			input:       []byte("just some random text without yaml structure"),
			shouldError: true,
			errorText:   "invalid YAML document: no key-value pairs or list items found",
		},
		{
			name:        "minimal_valid_yaml",
			description: "Minimal valid YAML with single key-value should pass",
			input:       []byte("key: value"),
			shouldError: false,
		},
		{
			name:        "minimal_valid_list",
			description: "Minimal valid YAML with list should pass",
			input: []byte(`- item1
- item2`),
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Validate document
			err := validateYAMLDocument(tt.input)

			// Then: Verify validation result
			if tt.shouldError {
				assert.Error(t, err, "Expected validation error")
				if tt.errorText != "" {
					assert.Contains(t, err.Error(), tt.errorText, "Error should contain expected text")
				}
			} else {
				assert.NoError(t, err, "Unexpected validation error")
			}
		})
	}
}

// TestYAMLUtilities_EdgeCases tests edge cases and complex scenarios
// WHY: Real-world YAML files contain various edge cases that must be handled robustly
func TestYAMLUtilities_EdgeCases(t *testing.T) {
	t.Run("deeply_nested_yaml_structure", func(t *testing.T) {
		// Given: Complex nested YAML structure
		complexYAML := []byte(`apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeLabelConf
metadata:
  name: complex-config
  labels:
    environment: production
    tier: control-plane
spec:
  nodeRoles:
    controlPlane:
      nodes:
        - rsb2
        - rsb3
        - rsb4
      labels:
        openstack-role: control-plane
        cluster.openstack.io/role: control-plane
        node.openstack.io/type: master
      description: "OpenStack control plane services"
    storage:
      nodes:
        - rsb5
        - rsb6
      labels:
        openstack-role: storage
        ceph-node: enabled
      description: "Dedicated storage nodes"`)

		// When: Validate complex document
		err := validateYAMLDocument(complexYAML)

		// Then: Should handle complexity properly
		assert.NoError(t, err, "Complex YAML should validate successfully")

		// When: Split as single document
		documents, err := splitYAMLDocuments(complexYAML)

		// Then: Should handle as single document
		assert.NoError(t, err, "Complex YAML should split successfully")
		assert.Len(t, documents, 1, "Should be treated as single document")
		assert.False(t, isMultiDocumentYAML(complexYAML), "Should not be detected as multi-document")
	})

	t.Run("yaml_with_special_characters", func(t *testing.T) {
		// Given: YAML with special characters and quotes
		specialYAML := []byte(`apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeLabelConf
metadata:
  name: "special-config"
  annotations:
    description: "Config with special chars: !@#$%^&*()"
    command: 'kubectl label node rsb2 key="value with spaces"'
spec:
  nodeRoles:
    "role-with-dashes":
      nodes: ["rsb2"]
      labels:
        "key.with.dots": "value-with-dashes"
        "key/with/slashes": "value:with:colons"`)

		// When: Process special character YAML
		err := validateYAMLDocument(specialYAML)
		documents, splitErr := splitYAMLDocuments(specialYAML)

		// Then: Should handle special characters properly
		assert.NoError(t, err, "Special character YAML should validate")
		assert.NoError(t, splitErr, "Special character YAML should split")
		assert.Len(t, documents, 1, "Should produce single document")
	})

	t.Run("mixed_line_ending_multi_document", func(t *testing.T) {
		// Given: Multi-document YAML with mixed line endings
		mixedLineEndings := []byte("apiVersion: openstack.kictl.icycloud.io/v1\nkind: NodeLabelConf\r\n---\r\napiVersion: openstack.kictl.icycloud.io/v1\nkind: NodeVLANConf")

		// When: Process mixed line endings
		isMulti := isMultiDocumentYAML(mixedLineEndings)
		documents, err := splitYAMLDocuments(mixedLineEndings)

		// Then: Should handle mixed line endings correctly
		assert.True(t, isMulti, "Should detect as multi-document despite mixed line endings")
		assert.NoError(t, err, "Should split mixed line endings successfully")
		assert.Len(t, documents, 2, "Should produce two documents")
	})

	t.Run("yaml_with_comments_and_separators", func(t *testing.T) {
		// Given: YAML with comments that might contain separator-like content
		yamlWithComments := []byte(`# This is a comment
# Another comment with --- in it
apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeLabelConf
metadata:
  name: commented-config
  # Inline comment
spec:
  nodeRoles:
    control:
      nodes: ["rsb2"]
      labels:
        role: "control"  # End-of-line comment`)

		// When: Process YAML with comments
		isMulti := isMultiDocumentYAML(yamlWithComments)
		err := validateYAMLDocument(yamlWithComments)

		// Then: Should handle comments correctly
		assert.False(t, isMulti, "Comments with --- should not trigger multi-document detection")
		assert.NoError(t, err, "YAML with comments should validate successfully")
	})
}
