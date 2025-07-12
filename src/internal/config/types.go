// Package config defines configuration structures for k8ostack-ictl
package config

// Kubernetes-style metadata
type Metadata struct {
	Name      string            `json:"name" yaml:"name"`
	Namespace string            `json:"namespace" yaml:"namespace"`
	Labels    map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

// NodeRole represents a role configuration with multiple labels
type NodeRole struct {
	Nodes       []string          `json:"nodes" yaml:"nodes"`
	Labels      map[string]string `json:"labels" yaml:"labels"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
}

// ToolConfig represents tool-specific configuration
type ToolConfig struct {
	// Common options
	DryRun        bool   `json:"dryRun,omitempty" yaml:"dryRun,omitempty"`
	ValidateNodes bool   `json:"validateNodes,omitempty" yaml:"validateNodes,omitempty"`
	LogLevel      string `json:"logLevel,omitempty" yaml:"logLevel,omitempty"`
	
	// VLAN-specific options
	ValidateConnectivity bool `json:"validateConnectivity,omitempty" yaml:"validateConnectivity,omitempty"`
	PersistentConfig     bool `json:"persistentConfig,omitempty" yaml:"persistentConfig,omitempty"`
	
	// NetHealthCheck-specific options
	Parallel     bool     `json:"parallel,omitempty" yaml:"parallel,omitempty"`
	Retries      int      `json:"retries,omitempty" yaml:"retries,omitempty"`
	OutputFormat string   `json:"outputFormat,omitempty" yaml:"outputFormat,omitempty"`
	ExcludeNodes []string `json:"excludeNodes,omitempty" yaml:"excludeNodes,omitempty"`
}

// NodeLabelSpec contains the specification for node labeling operations
type NodeLabelSpec struct {
	NodeRoles map[string]NodeRole `json:"nodeRoles" yaml:"nodeRoles"`
}

// Tools contains tool-specific configurations for the infrastructure control platform
type Tools struct {
	// Nlabel configuration for the node labeling service
	Nlabel ToolConfig `json:"nlabel,omitempty" yaml:"nlabel,omitempty"`

	// Future service configurations
	Nvlan ToolConfig `json:"nvlan,omitempty" yaml:"nvlan,omitempty"` // VLAN configuration service
	Ntest ToolConfig `json:"ntest,omitempty" yaml:"ntest,omitempty"` // Network testing service
}

// NodeLabelConf represents the CRD-based node labeling configuration
type NodeLabelConf struct {
	APIVersion string        `json:"apiVersion" yaml:"apiVersion"`
	Kind       string        `json:"kind" yaml:"kind"`
	Metadata   Metadata      `json:"metadata" yaml:"metadata"`
	Spec       NodeLabelSpec `json:"spec" yaml:"spec"`
	Tools      Tools         `json:"tools,omitempty" yaml:"tools,omitempty"`
}

// NodeVLANConf represents VLAN configuration for nodes
type NodeVLANConf struct {
	APIVersion string       `json:"apiVersion" yaml:"apiVersion"`
	Kind       string       `json:"kind" yaml:"kind"`
	Metadata   Metadata     `json:"metadata" yaml:"metadata"`
	Spec       NodeVLANSpec `json:"spec" yaml:"spec"`
	Tools      Tools        `json:"tools,omitempty" yaml:"tools,omitempty"`
}

// NodeVLANSpec contains the specification for VLAN operations
type NodeVLANSpec struct {
	VLANs map[string]VLANConfig `json:"vlans" yaml:"vlans"`
}

// VLANConfig represents a single VLAN configuration
type VLANConfig struct {
	ID          int               `json:"id" yaml:"id"`
	Subnet      string            `json:"subnet" yaml:"subnet"`
	Interface   string            `json:"interface,omitempty" yaml:"interface,omitempty"`
	NodeMapping map[string]string `json:"nodeMapping" yaml:"nodeMapping"`
}

// NodeTestConf represents connectivity testing configuration
type NodeTestConf struct {
	APIVersion string       `json:"apiVersion" yaml:"apiVersion"`
	Kind       string       `json:"kind" yaml:"kind"`
	Metadata   Metadata     `json:"metadata" yaml:"metadata"`
	Spec       NodeTestSpec `json:"spec" yaml:"spec"`
	Tools      Tools        `json:"tools,omitempty" yaml:"tools,omitempty"`
}

// NodeTestSpec contains the specification for connectivity tests
type NodeTestSpec struct {
	Tests []ConnectivityTest `json:"tests" yaml:"tests"`
}

// ConnectivityTest represents a single connectivity test
type ConnectivityTest struct {
	Name          string   `json:"name" yaml:"name"`
	Description   string   `json:"description,omitempty" yaml:"description,omitempty"`
	Source        string   `json:"source" yaml:"source"`
	Targets       []string `json:"targets" yaml:"targets"`
	Timeout       int      `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	ExpectSuccess bool     `json:"expectSuccess,omitempty" yaml:"expectSuccess,omitempty"`
}

// Common interface for all config types
type Config interface {
	GetAPIVersion() string
	GetKind() string
	GetMetadata() Metadata
	GetNodeRoles() map[string]NodeRole
	GetTools() Tools
}

// Implement Config interface for NodeLabelConf
func (c NodeLabelConf) GetAPIVersion() string {
	return c.APIVersion
}

func (c NodeLabelConf) GetKind() string {
	return c.Kind
}

func (c NodeLabelConf) GetMetadata() Metadata {
	return c.Metadata
}

func (c NodeLabelConf) GetNodeRoles() map[string]NodeRole {
	return c.Spec.NodeRoles
}

func (c NodeLabelConf) GetTools() Tools {
	return c.Tools
}

// Implement Config interface for NodeVLANConf
func (c NodeVLANConf) GetAPIVersion() string {
	return c.APIVersion
}

func (c NodeVLANConf) GetKind() string {
	return c.Kind
}

func (c NodeVLANConf) GetMetadata() Metadata {
	return c.Metadata
}

func (c NodeVLANConf) GetNodeRoles() map[string]NodeRole {
	// VLANs don't have node roles in the same way, return empty map
	// This maintains interface compatibility while each CRD type can have different specs
	return make(map[string]NodeRole)
}

func (c NodeVLANConf) GetTools() Tools {
	return c.Tools
}

// Implement Config interface for NodeTestConf
func (c NodeTestConf) GetAPIVersion() string {
	return c.APIVersion
}

func (c NodeTestConf) GetKind() string {
	return c.Kind
}

func (c NodeTestConf) GetMetadata() Metadata {
	return c.Metadata
}

func (c NodeTestConf) GetNodeRoles() map[string]NodeRole {
	// Tests don't have node roles in the same way, return empty map
	// This maintains interface compatibility while each CRD type can have different specs
	return make(map[string]NodeRole)
}

func (c NodeTestConf) GetTools() Tools {
	return c.Tools
}
