# k8ostack-ictl Project Management
# Modern Kubernetes OpenStack infrastructure control tool
# Multi-CRD unified infrastructure automation platform with code-generated configs

# Default recipe to display available commands
default:
    @just --list

# Variables
binary_name := "kictl"
src_dir := "src"
build_dir := "dist"
config_file := "prod-infra.yaml"
sample_multi_config := "sample-multi-config.yaml"
verbose_flag := "--verbose"  # Set to empty space to disable verbose output, use --verbose to enable
head_n_count := "20"

# Build the application with multi-CRD support
build:
    @echo "🔨 Building {{binary_name}} with multi-CRD support..."
    cd {{src_dir}} && go build -o ../{{build_dir}}/{{binary_name}} ./cmd/k8ostack-ictl
    @echo "✅ Built {{build_dir}}/{{binary_name}} with unified architecture"

# Build for production with optimizations
build-prod:
    @echo "🔨 Building {{binary_name}} for production with multi-CRD support..."
    cd {{src_dir}} && CGO_ENABLED=0 go build -ldflags="-w -s" -o ../{{build_dir}}/{{binary_name}} ./cmd/k8ostack-ictl
    @echo "✅ Built optimized {{build_dir}}/{{binary_name}} with unified architecture"

# Clean build artifacts
clean:
    @echo "🧹 Cleaning build artifacts..."
    -rm -rf {{build_dir}}
    -rm -rf {{src_dir}}/logs
    -rm -f logs/node_labeling_*.log
    -rm -f {{sample_multi_config}}
    @echo "✅ Cleaned build artifacts"

# Clean up kubectl debug pods created during VLAN operations
cleanup-pods:
    @echo "🧹 Cleaning up kubectl debug pods..."
    -kubectl get pods | grep node-debugger | awk '{print $1}' | xargs kubectl delete pod
    @echo "✅ Debug pods cleaned up"

# Install dependencies
deps:
    @echo "📦 Installing Go dependencies..."
    cd {{src_dir}} && go mod tidy
    cd {{src_dir}} && go mod download
    @echo "✅ Dependencies installed"

# Run tests
test:
    @echo "🧪 Running tests..."
    cd {{src_dir}} && go test ./...
    @echo "✅ Tests completed"

# Run tests with coverage
test-coverage:
    @echo "🧪 Running tests with coverage..."
    cd {{src_dir}} && go test -coverprofile=coverage.out ./...
    cd {{src_dir}} && go tool cover -html=coverage.out -o coverage.html
    @echo "✅ Coverage report generated: {{src_dir}}/coverage.html"

# Run linter
lint:
    @echo "🔍 Running linter..."
    cd {{src_dir}} && golangci-lint run
    @echo "✅ Linting completed"

# Format code
fmt:
    @echo "🎨 Formatting code..."
    cd {{src_dir}} && go fmt ./...
    @echo "✅ Code formatted"

# Generate sample configuration (single NodeLabelConf)
gen-config: build
    @echo "📋 Generating sample single-CRD configuration..."
    {{build_dir}}/{{binary_name}} --generate-config
    @echo "✅ Generated sample-config.yaml"

# Generate sample multi-CRD configuration (NodeLabelConf + NodeVLANConf + NodeTestConf)
gen-multi-config: build
    @echo "📋 Generating sample multi-CRD configuration..."
    {{build_dir}}/{{binary_name}} --generate-multi-config
    @echo "✅ Generated sample-multi-config.yaml with multi-CRD example"

# Apply labels with current config
apply: build
    @echo "🚀 Applying labels from {{config_file}}..."
    {{build_dir}}/{{binary_name}} --config {{config_file}} --apply {{verbose_flag}}

# Apply labels in dry-run mode
apply-dry: build
    @echo "🧪 Dry-run: Applying labels from {{config_file}}..."
    {{build_dir}}/{{binary_name}} --config {{config_file}} --apply --dry-run {{verbose_flag}}

# Remove labels with current config
delete: build
    @echo "🗑️ Removing labels from {{config_file}}..."
    {{build_dir}}/{{binary_name}} --config {{config_file}} --delete {{verbose_flag}}

# Remove labels in dry-run mode
delete-dry: build
    @echo "🧪 Dry-run: Removing labels from {{config_file}}..."
    {{build_dir}}/{{binary_name}} --config {{config_file}} --delete --dry-run {{verbose_flag}}

# Show current node labels for quick verification
show-labels:
    @echo "🏷️ Current node labels:"
    @kubectl get nodes --show-labels

# Development setup
dev-setup: deps
    @echo "🛠️ Setting up development environment..."
    cd {{src_dir}} && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    mkdir -p {{build_dir}}
    mkdir -p logs
    @echo "✅ Development environment ready"

# Watch for changes and rebuild (requires entr)
watch:
    @echo "👀 Watching for changes..."
    find {{src_dir}} -name "*.go" | entr -r just build

# Show project status
status:
    @echo "📊 Project Status:"
    @echo "  Binary: {{build_dir}}/{{binary_name}}"
    @if [ -f "{{build_dir}}/{{binary_name}}" ]; then echo "  ✅ Binary exists"; else echo "  ❌ Binary not built"; fi
    @if [ -f "{{config_file}}" ]; then echo "  ✅ Config exists"; else echo "  ❌ Config missing"; fi
    @if [ -f "{{sample_multi_config}}" ]; then echo "  ✅ Multi-config exists"; else echo "  ❌ Multi-config not generated"; fi
    @echo "  Go version: $(cd {{src_dir}} && go version)"
    @echo "  Module: $(cd {{src_dir}} && go list -m)"

# Install binary to system PATH
install: build-prod
    @echo "📦 Installing {{binary_name}} to system..."
    sudo cp {{build_dir}}/{{binary_name}} /usr/local/bin/
    @echo "✅ Installed to /usr/local/bin/{{binary_name}}"

# Uninstall binary from system PATH
uninstall:
    @echo "🗑️ Uninstalling {{binary_name}} from system..."
    sudo rm -f /usr/local/bin/{{binary_name}}
    @echo "✅ Uninstalled from system"

# Create release package
package: clean build-prod
    @echo "📦 Creating release package..."
    mkdir -p {{build_dir}}/release
    cp {{build_dir}}/{{binary_name}} {{build_dir}}/release/
    cp README.md {{build_dir}}/release/ 2>/dev/null || echo "README.md not found, skipping"
    cp sample-config.yaml {{build_dir}}/release/ 2>/dev/null || echo "sample-config.yaml not found, skipping"
    cp {{sample_multi_config}} {{build_dir}}/release/ 2>/dev/null || echo "Multi-config not found, skipping"
    tar -czf {{build_dir}}/{{binary_name}}-$(date +%Y%m%d).tar.gz -C {{build_dir}}/release .
    @echo "✅ Release package created in {{build_dir}}/"

# Debug build with race detection
debug:
    @echo "🐛 Building debug version..."
    cd {{src_dir}} && go build -race -o ../{{build_dir}}/{{binary_name}}-debug ./cmd/k8ostack-ictl
    @echo "✅ Debug build completed"

# Run benchmarks
bench:
    @echo "⚡ Running benchmarks..."
    cd {{src_dir}} && go test -bench=. -benchmem ./...
    @echo "✅ Benchmarks completed"

# Demo the multi-CRD architecture
demo: build gen-config gen-multi-config
    @echo "🎬 Multi-CRD Architecture Demo"
    @echo "================================"
    @echo ""
    @echo "1. 📋 Generated configurations from code:"
    @echo "   - sample-config.yaml: Single NodeLabelConf"
    @echo "   - sample-multi-config.yaml: NodeLabelConf + NodeVLANConf + NodeTestConf"
    @echo ""
    @echo "2. 🔍 API Version: openstack.kictl.icycloud.io/v1"
    @echo "   All configs use the updated kictl API domain"
    @echo ""
    @echo "3. 🎛️ Configuration bundle sample (first {{head_n_count}} lines):"
    @echo
    @head -n {{head_n_count}} {{sample_multi_config}}
    @echo ""
    @echo "4. 🔄 Main production configuration:"
    {{build_dir}}/{{binary_name}} --config {{config_file}} --apply --dry-run {{verbose_flag}} | grep "Configuration bundle"
    @echo ""
    @echo "✨ Demo completed! Clean code-generated configs with updated API version."

# Check for security vulnerabilities (requires 'nancy' tool)
# security:
#     @echo "🔒 Checking for security vulnerabilities..."
#     cd {{src_dir}} && go list -json -m all | nancy sleuth
#     @echo "✅ Security check completed" 
