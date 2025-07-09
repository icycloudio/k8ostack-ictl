# k8ostack-ictl Architecture

## Multi-CRD Bundle Processing

```
┌─────────────────────────────────────────────────────────────┐
│                    Configuration Bundle                      │
├─────────────────────────────────────────────────────────────┤
│ NodeLabelConf  │ NodeVLANConf  │ NodeTestConf               │
│ ✅ Active     │ ✅ Active     │ 🔄 Coming                  │
├─────────────────────────────────────────────────────────────┤
│                Global CLI Precedence                        │
│            (Affects ALL services uniformly)                 │
├─────────────────────────────────────────────────────────────┤
│ nlabel service │ nvlan service │ ntest service              │
│ ✅ Production │ ✅ Production │ 🔄 Ready                   │
└─────────────────────────────────────────────────────────────┘
```

## Service-Based Tools Configuration

```yaml
tools:
  nlabel:    # Node labeling service - ACTIVE
    dryRun: false
    validateNodes: true
    logLevel: "info"
  
  nvlan:     # VLAN service - ACTIVE
    dryRun: false
    validateConnectivity: true
    persistentConfig: false
    defaultInterface: "eth0"
    
  ntest:     # Testing service - READY FOR IMPLEMENTATION
    parallel: true
    retries: 3
    outputFormat: "detailed"
```

## Project Structure

```
k8ostack-ictl/
├── src/
│   ├── cmd/k8ostack-ictl/     # Main application
│   ├── internal/
│   │   ├── config/            # Configuration management
│   │   │   └── precedence/    # Global CLI precedence
│   │   ├── labeler/           # Node labeling service
│   │   ├── kubectl/           # Kubectl integration
│   │   ├── logging/           # Structured logging
│   │   └── vlan/              # VLAN service
│   ├── go.mod
│   └── go.sum
├── docs/                      # Architecture documentation
├── logs/                      # Generated log files
├── justfile                   # Build automation
└── sample-*.yaml             # Generated configurations
```

## Processing Flow

1. **Configuration Loading**: Multi-document YAML files are parsed and validated
2. **Global Precedence**: CLI flags override service-specific configurations
3. **Service Routing**: Each CRD type is routed to its corresponding service
4. **Parallel Execution**: Services can run independently with shared logging
5. **Result Aggregation**: Results from all services are combined and reported

## Extensibility

The architecture is designed for easy extension:
- New CRD types can be added by implementing the service interface
- Global precedence automatically applies to new services
- Logging and error handling are standardized across all services 