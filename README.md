# k8ostack-ictl

**Modern Kubernetes OpenStack Infrastructure Control Platform**

A unified, multi-CRD infrastructure automation tool for Kubernetes OpenStack deployments. Control node labeling, VLAN configuration, and network testing through a single, powerful CLI with global precedence and comprehensive validation.

## 🎯 Overview

`k8ostack-ictl` (command: `kictl`) is a next-generation infrastructure control platform that manages complex Kubernetes OpenStack deployments through a unified multi-CRD architecture. It supports single and multi-document YAML configurations with global CLI precedence across all services.

**🏆 Production-Ready with 80%+ Test Coverage** - Enterprise-grade reliability with comprehensive test coverage across all packages and robust error handling.

### **Supported CRD Types**
- 🏷️ **NodeLabelConf** - Kubernetes node labeling and role management ✅ **Active**
- 🌐 **NodeVLANConf** - VLAN configuration and network topology ✅  **Active**
- 🧪 **NodeTestConf** - Network connectivity testing and validation 🔄 **Ready for implementation**

## ✨ Features

### **Unified Multi-CRD Architecture**
- 📦 **Single binary** controls multiple infrastructure services (nlabel, nvlan, ntest)
- 🔄 **Multi-document YAML** support with automatic CRD detection
- 🎛️ **Global CLI precedence** - flags override all service configurations
- 📋 **Configuration bundles** with comprehensive validation

### **Enterprise-Grade Capabilities**
- 🧪 **Dry-run mode** - Test changes safely across all services
- 📝 **Structured logging** - Timestamped audit trails with configurable levels
- ⚡ **Zero dependencies** - Single binary with embedded kubectl integration
- 🔒 **Service isolation** - Clean separation between services
- 💅 **Modern CLI** - Professional help system and error handling
- 🔧 **Robust error handling** - Graceful handling of node failures and network issues
- 📊 **Real-time validation** - Node existence and connectivity checks
- 🎯 **Flexible targeting** - Role-based node grouping with any label patterns

## 🚀 Quick Start

### **1. Generate Sample Configurations**

```bash
# Generate single NodeLabelConf sample
kictl --generate-config

# Generate multi-CRD sample (all types)
kictl --generate-multi-config
```

### **2. Configuration Examples**

**Complete Multi-CRD Configuration:**
```yaml
# Node Labeling Configuration
apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeLabelConf
metadata:
  name: example-node-labels
  namespace: openstack
spec:
  nodeRoles:
    controlPlane:
      nodes: [node-ctrl-01, node-ctrl-02, node-ctrl-03]
      labels:
        "openstack-control-plane": "enabled"
        "cluster.openstack.io/role": "control-plane"
      description: "OpenStack control plane services"
    storage:
      nodes: [node-storage-01, node-storage-02] 
      labels:
        "openstack-storage-node": "enabled"
        "ceph-node": "enabled"
      description: "Dedicated storage nodes"
    compute:
      nodes: [node-compute-01, node-compute-02, node-compute-03]
      labels:
        "openstack-role": "compute"
        "nova-compute": "enabled"

tools:
  nlabel:
    dryRun: false
    validateNodes: true
    logLevel: "info"
---
# VLAN Configuration
apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeVLANConf
metadata:
  name: example-vlans
spec:
  vlans:
    management:
      id: 50
      subnet: "172.16.10.0/24"
      interface: "ens160"
      nodeMapping:
        node-ctrl-01: "172.16.10.21/24"
        node-ctrl-02: "172.16.10.22/24"
    storage:
      id: 60
      subnet: "172.16.20.0/24"
      interface: "ens192"
      nodeMapping:
        node-storage-01: "172.16.20.31/24"
        node-storage-02: "172.16.20.32/24"

tools:
  nvlan:
    dryRun: false
    validateConnectivity: true
    defaultInterface: "ens160"
---
# Network Testing Configuration  
apiVersion: openstack.kictl.icycloud.io/v1
kind: NodeTestConf  
metadata:
  name: example-tests
spec:
  tests:
    - name: "management-reachability"
      source: "management"
      targets: ["storage", "api"]
      expectSuccess: true
    - name: "storage-bandwidth"
      source: "compute"
      targets: ["storage"]
      testType: "bandwidth"
      minMbps: 1000

tools:
  ntest:
    parallel: true
    retries: 3
    outputFormat: "detailed"
```

### **3. Apply Infrastructure**

```bash
# Test with dry-run
kictl --config cluster-config.yaml --apply --dry-run --verbose

# Apply changes
kictl --config cluster-config.yaml --apply

# Remove configurations
kictl --config cluster-config.yaml --delete
```

## 📋 Command Reference

### **Core Operations**
```bash
# Apply infrastructure from configuration
kictl --config cluster-config.yaml --apply

# Remove applied configurations  
kictl --config cluster-config.yaml --delete

# Dry-run simulation (affects ALL services)
kictl --config multi-config.yaml --apply --dry-run

# Global verbose logging
kictl --config cluster-config.yaml --apply --verbose
```

### **Configuration Generation**
```bash
# Generate single NodeLabelConf sample
kictl --generate-config

# Generate multi-CRD sample (all types)
kictl --generate-multi-config
```

### **Global CLI Precedence**
CLI flags override ALL service configurations in the bundle:
```bash
# Override dryRun for all services
kictl --config multi-config.yaml --apply --dry-run

# Override log level globally
kictl --config cluster-config.yaml --apply --log-level=debug

# Override verbose mode
kictl --config cluster-config.yaml --apply --verbose
```

## 📦 Installation

```bash
# Clone and build
git clone <repository>
cd k8ostack-ictl
just build

# Production build
just build-prod && just install
```

## 🔧 Development

```bash
just build              # Development build
just test               # Run test suite  
just test-coverage      # Run tests with coverage report
just lint               # Code quality checks
just gen-multi-config   # Generate sample configs
just demo               # Full architecture demo
```

## 🔍 API Reference

**API Version:** `openstack.kictl.icycloud.io/v1`

**CRD Kinds:**
- `NodeLabelConf` - Node labeling and role management ✅ **Active**
- `NodeVLANConf` - VLAN configuration and network topology ✅ **Active**
- `NodeTestConf` - Network connectivity testing 🔄 **Ready**

**Tool Configurations:**
- `tools.nlabel` - Node labeling service ✅ **Active**
- `tools.nvlan` - VLAN service ✅ **Active**
- `tools.ntest` - Testing service 🔄 **Ready**

## 📊 Status & Roadmap

**✅ Completed (v1.0):**
Multi-CRD architecture • Node labeling • VLAN configuration • Global CLI precedence • Multi-document YAML • Comprehensive validation & logging • Robust error handling

**🔄 Next Phase:**
Network testing service • Enhanced templates • Advanced rollback • Performance optimizations

## 🛠️ Prerequisites

- `kubectl` configured with cluster access
- Appropriate RBAC permissions for node and network operations
- Go 1.19+ (for building from source)

## 📄 License

MIT License - see LICENSE file for details.

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Ensure all tests pass (`just test`)
4. Commit your changes (`git commit -m 'Add amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

---

**k8ostack-ictl** - Modern Kubernetes OpenStack infrastructure control made unified and powerful.

*Built with ❤️ for production Kubernetes OpenStack deployments* 
