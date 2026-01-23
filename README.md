# kindplane

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License">
  <img src="https://img.shields.io/badge/Crossplane-1.15+-7C3AED?style=flat" alt="Crossplane">
</p>

**kindplane** is a CLI tool that helps developers quickly spin up [Kind](https://kind.sigs.k8s.io/) (Kubernetes in Docker) clusters pre-configured with [Crossplane](https://crossplane.io/), cloud providers, and other essential components.

It automates the tedious process of setting up a local Kubernetes development environment with Crossplane for infrastructure management.

## Features

- 🚀 **One-command bootstrap** - Create fully configured Kind clusters with a single command
- ⚙️ **Crossplane integration** - Automatic installation and configuration of Crossplane
- 📦 **Provider management** - Install and manage Crossplane providers (AWS, Azure, GCP, etc.)
- 🔐 **External Secrets Operator** - Optional ESO installation for secrets management
- 📊 **Helm chart support** - Install any Helm chart with configurable phases
- 🎨 **Beautiful CLI** - Rich terminal output with colors, icons, and progress indicators
- 🔍 **Smart diagnostics** - Detailed failure diagnostics with pod logs and conditions
- 💾 **GitOps export** - Dump cluster resources in GitOps-friendly format

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/kanzifucius/kindplane.git
cd kindplane

# Build
go build -o bin/kindplane ./cmd/kindplane

# Or use Task
task build

# Move to PATH (optional)
sudo mv bin/kindplane /usr/local/bin/
```

### Prerequisites

- [Go 1.21+](https://golang.org/dl/)
- [Docker](https://docs.docker.com/get-docker/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) (automatically managed)

## Quick Start

```bash
# Initialize configuration
kindplane init

# Create and bootstrap the cluster
kindplane up

# Check status
kindplane status

# Delete the cluster
kindplane down --force
```

## Configuration

kindplane uses a YAML configuration file (`kindplane.yaml`). Generate one with:

```bash
kindplane init
```

### Example Configuration

```yaml
# Cluster configuration
cluster:
  name: kindplane-dev
  kubernetesVersion: "1.29.0"
  nodes:
    controlPlane: 1
    workers: 1
  portMappings:
    - containerPort: 80
      hostPort: 8080
      protocol: TCP
  ingress:
    enabled: true

# Crossplane configuration
crossplane:
  version: "1.15.0"
  providers:
    - name: provider-aws
      package: xpkg.upbound.io/upbound/provider-aws-s3:v1.1.0
    - name: provider-kubernetes
      package: xpkg.upbound.io/crossplane-contrib/provider-kubernetes:v0.11.0

# External Secrets Operator
eso:
  enabled: true
  version: "0.9.11"

# Helm charts with installation phases
charts:
  - name: cert-manager
    repo: https://charts.jetstack.io
    chart: cert-manager
    namespace: cert-manager
    version: "1.14.0"
    phase: pre-crossplane
    values:
      installCRDs: true

  - name: ingress-nginx
    repo: https://kubernetes.github.io/ingress-nginx
    chart: ingress-nginx
    namespace: ingress-nginx
    phase: post-eso
    wait: true

# Compositions from local paths or Git
compositions:
  sources:
    - path: ./compositions
    - git: https://github.com/org/crossplane-compositions.git
      ref: main
      path: compositions/
```

### Chart Phases

Charts can be installed at different phases of the bootstrap process:

| Phase | Description |
|-------|-------------|
| `pre-crossplane` | Before Crossplane installation |
| `post-crossplane` | After Crossplane is ready |
| `post-providers` | After all providers are healthy |
| `post-eso` | After ESO is ready (default) |

## Commands

### `kindplane init`

Initialize a new configuration file.

```bash
kindplane init                    # Create kindplane.yaml
kindplane init --output my.yaml   # Custom filename
```

### `kindplane up`

Create and bootstrap a Kind cluster.

```bash
kindplane up                      # Full bootstrap
kindplane up --skip-providers     # Skip provider installation
kindplane up --skip-eso           # Skip ESO installation
kindplane up --skip-charts        # Skip all Helm charts
kindplane up --rollback-on-failure # Delete cluster if bootstrap fails
kindplane up --timeout 15m        # Custom timeout
```

### `kindplane down`

Delete the Kind cluster.

```bash
kindplane down --force            # Required flag to confirm deletion
```

### `kindplane status`

Show cluster and component status.

```bash
kindplane status                  # Basic status
kindplane status --detailed       # Include pod information
```

### `kindplane dump`

Export cluster resources for GitOps.

```bash
kindplane dump                    # Dump to ./dump directory
kindplane dump -o ./exported      # Custom output directory
kindplane dump --stdout           # Print to stdout
kindplane dump --dry-run          # Preview what would be dumped
kindplane dump --include=providers,compositions,xrds
kindplane dump --exclude=secrets,configmaps
kindplane dump --skip-secrets     # Skip secrets entirely
```

### `kindplane provider`

Manage Crossplane providers.

```bash
kindplane provider list           # List installed providers
kindplane provider add provider-aws xpkg.upbound.io/upbound/provider-aws:v1.1.0
```

### `kindplane chart`

Manage Helm charts.

```bash
kindplane chart list              # List installed charts
kindplane chart install my-chart https://charts.example.com my-chart
kindplane chart uninstall my-chart my-namespace
```

### `kindplane credentials`

Configure cloud provider credentials.

```bash
kindplane credentials configure   # Interactive setup
```

## Failure Diagnostics

When bootstrap fails, kindplane automatically shows detailed diagnostics:

```
✗ Providers failed to become healthy: context deadline exceeded

 ✗ DIAGNOSTICS 
╭──────────────────────────────────────────────────────────────────╮
│                                                                  │
│  📦 Providers                                                     │
│                                                                  │
│    ✗ provider-aws                                                │
│      Package:       xpkg.upbound.io/upbound/provider-aws:v1.1.0  │
│      Conditions:                                                 │
│        ✗ Healthy: False                                          │
│          Reason: UnhealthyPackageRevision                        │
│          Message: cannot resolve package dependencies...          │
│                                                                  │
│  ⎈ Pods                                                          │
│                                                                  │
│    ! provider-aws-7b8f9d6c5-xk2jl (crossplane-system)            │
│      Phase:         Running                                      │
│      Ready:         0/1 containers                               │
│      Container: provider                                         │
│        State:       CrashLoopBackOff                             │
│        Restarts:    5                                            │
│        Recent Logs:                                              │
│          error: failed to initialize provider                    │
│          error: missing credentials configuration                │
│                                                                  │
╰──────────────────────────────────────────────────────────────────╯
```

## Project Structure

```
kindplane/
├── cmd/kindplane/          # Main entry point
├── internal/
│   ├── cmd/                # CLI commands (Cobra)
│   │   ├── root.go         # Root command and global flags
│   │   ├── init.go         # init command
│   │   ├── up.go           # up command
│   │   ├── down.go         # down command
│   │   ├── status.go       # status command
│   │   ├── dump.go         # dump command
│   │   ├── provider/       # provider subcommands
│   │   ├── chart/          # chart subcommands
│   │   └── credentials/    # credentials subcommands
│   ├── config/             # Configuration loading and validation
│   ├── kind/               # Kind cluster management
│   ├── crossplane/         # Crossplane installation and management
│   ├── helm/               # Helm chart operations
│   ├── eso/                # External Secrets Operator
│   ├── credentials/        # Cloud credential management
│   ├── diagnostics/        # Failure diagnostics
│   ├── dump/               # Resource export functionality
│   ├── git/                # Git operations for compositions
│   └── ui/                 # Terminal UI components (lipgloss)
├── Taskfile.yaml           # Task runner configuration
├── go.mod
└── kindplane.yaml.example  # Example configuration
```

## Development

### Building

```bash
# Using Task
task build          # Build binary
task build-all      # Build for all platforms
task test           # Run tests
task lint           # Run linter
task dev            # Build and run

# Using Go directly
go build -o bin/kindplane ./cmd/kindplane
go test ./...
```

### Running Locally

```bash
# Build and run
task dev -- up

# Or directly
go run ./cmd/kindplane up
```

## Dependencies

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [Helm SDK](https://helm.sh/docs/topics/advanced/#go-sdk) - Helm operations
- [Kind](https://sigs.k8s.io/kind) - Kubernetes in Docker
- [client-go](https://github.com/kubernetes/client-go) - Kubernetes client

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Crossplane](https://crossplane.io/) - Cloud-native control planes
- [Kind](https://kind.sigs.k8s.io/) - Kubernetes in Docker
- [Charm](https://charm.sh/) - Beautiful terminal tools
