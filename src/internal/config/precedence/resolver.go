// Package precedence provides global precedence resolution for multi-CRD configurations
package precedence

import (
	"fmt"
	"reflect"

	"github.com/spf13/cobra"
)

// GlobalResolver handles precedence resolution across multiple configuration types
type GlobalResolver struct {
	cmd *cobra.Command
}

// NewGlobalResolver creates a new global precedence resolver
func NewGlobalResolver(cmd *cobra.Command) *GlobalResolver {
	return &GlobalResolver{
		cmd: cmd,
	}
}

// ApplyGlobalOverrides applies CLI flag overrides to all configurations in the bundle
// This maintains the existing precedence pattern: CLI > Config > Defaults
func (r *GlobalResolver) ApplyGlobalOverrides(bundle interface{}) error {
	// Use type assertion instead of reflection for better reliability
	type ConfigBundle interface {
		GetAllConfigs() []interface{}
	}

	configBundle, ok := bundle.(ConfigBundle)
	if !ok {
		return fmt.Errorf("bundle does not implement ConfigBundle interface")
	}

	configs := configBundle.GetAllConfigs()

	// Apply precedence to each configuration
	for i, cfg := range configs {
		if err := r.applyToConfig(cfg); err != nil {
			return fmt.Errorf("failed to apply precedence to config %d: %w", i, err)
		}
	}

	return nil
}

// applyToConfig applies CLI overrides to a single configuration
func (r *GlobalResolver) applyToConfig(cfg interface{}) error {
	cfgValue := reflect.ValueOf(cfg)

	// Handle both pointer and non-pointer configs
	if cfgValue.Kind() == reflect.Ptr {
		cfgValue = cfgValue.Elem()
	}

	// Look for the Tools field
	toolsField := cfgValue.FieldByName("Tools")
	if !toolsField.IsValid() {
		// Config doesn't have tools section, skip
		return nil
	}

	// Apply CLI overrides to ALL tool configurations
	toolNames := []string{"Nlabel", "Nvlan", "Ntest"}

	for _, toolName := range toolNames {
		toolField := toolsField.FieldByName(toolName)
		if toolField.IsValid() && toolField.CanSet() {
			if err := r.applyToToolConfig(toolField); err != nil {
				return fmt.Errorf("failed to apply overrides to %s: %w", toolName, err)
			}
		}
	}

	return nil
}

// applyToToolConfig applies CLI overrides to tool-specific configuration
func (r *GlobalResolver) applyToToolConfig(toolConfig reflect.Value) error {
	if !toolConfig.CanSet() {
		return fmt.Errorf("tool config is not settable")
	}

	toolType := toolConfig.Type()

	// Iterate through tool config fields and check for CLI overrides
	for i := 0; i < toolConfig.NumField(); i++ {
		field := toolConfig.Field(i)
		fieldType := toolType.Field(i)

		if !field.CanSet() {
			continue
		}

		// Map field names to CLI flags (following existing patterns)
		var flagName string
		switch fieldType.Name {
		case "DryRun":
			flagName = "dry-run"
		case "LogLevel":
			flagName = "log-level"
		default:
			continue // Skip unknown fields
		}

		// Check if CLI flag was explicitly set
		if r.cmd.Flags().Changed(flagName) {
			if err := r.setFieldFromFlag(field, flagName); err != nil {
				return fmt.Errorf("failed to set %s from flag: %w", fieldType.Name, err)
			}
		}
	}

	return nil
}

// setFieldFromFlag sets a field value from the corresponding CLI flag
func (r *GlobalResolver) setFieldFromFlag(field reflect.Value, flagName string) error {
	switch field.Kind() {
	case reflect.Bool:
		val, err := r.cmd.Flags().GetBool(flagName)
		if err != nil {
			return err
		}
		field.SetBool(val)

	case reflect.String:
		val, err := r.cmd.Flags().GetString(flagName)
		if err != nil {
			return err
		}
		field.SetString(val)

	case reflect.Int:
		val, err := r.cmd.Flags().GetInt(flagName)
		if err != nil {
			return err
		}
		field.SetInt(int64(val))

	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}

	return nil
}

// GetAppliedOverrides returns a summary of which CLI flags were applied
func (r *GlobalResolver) GetAppliedOverrides() map[string]interface{} {
	overrides := make(map[string]interface{})

	// Check which flags were explicitly set
	flagNames := []string{"dry-run", "log-level"}

	for _, flagName := range flagNames {
		if r.cmd.Flags().Changed(flagName) {
			// Get the value based on flag type
			if flag := r.cmd.Flags().Lookup(flagName); flag != nil {
				overrides[flagName] = flag.Value.String()
			}
		}
	}

	return overrides
}
