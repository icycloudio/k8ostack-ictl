// Package config provides YAML document parsing utilities
package config

import (
	"fmt"
	"strings"
)

// splitYAMLDocuments splits a multi-document YAML file into individual documents
// It handles the standard YAML document separator "---" and various edge cases
func splitYAMLDocuments(data []byte) ([][]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty YAML data")
	}

	// Convert to string for easier processing
	content := string(data)

	// Split on document separator
	documents := strings.Split(content, "\n---")

	var result [][]byte

	for i, doc := range documents {
		// Clean up the document
		cleanDoc := strings.TrimSpace(doc)

		// Skip empty documents
		if cleanDoc == "" || cleanDoc == "---" {
			continue
		}

		// Handle documents that start with --- (first document case)
		if i == 0 && strings.HasPrefix(cleanDoc, "---") {
			cleanDoc = strings.TrimPrefix(cleanDoc, "---")
			cleanDoc = strings.TrimSpace(cleanDoc)
		}

		// Skip if still empty after cleaning
		if cleanDoc == "" {
			continue
		}

		// Add document to result
		result = append(result, []byte(cleanDoc))
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no valid YAML documents found")
	}

	return result, nil
}

// isMultiDocumentYAML checks if the data contains multiple YAML documents
func isMultiDocumentYAML(data []byte) bool {
	content := string(data)
	// Look for document separator patterns
	return strings.Contains(content, "\n---") || strings.Contains(content, "\r\n---")
}

// validateYAMLDocument performs basic validation on a YAML document
func validateYAMLDocument(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty YAML document")
	}

	// Check for basic YAML structure indicators
	content := string(data)
	content = strings.TrimSpace(content)

	// Must contain some key-value pairs or list items
	if !strings.Contains(content, ":") && !strings.Contains(content, "-") {
		return fmt.Errorf("invalid YAML document: no key-value pairs or list items found")
	}

	return nil
}
