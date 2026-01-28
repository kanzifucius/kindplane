# Commands Overview

kindplane provides a comprehensive set of commands for managing your local Crossplane development environment.

## Command Structure

```
kindplane [command] [subcommand] [flags]
```

## Global Flags

These flags are available for all commands:

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-c` | Configuration file (default: `./kindplane.yaml`) |
| `--verbose` | `-V` | Enable verbose output |
| `--help` | `-h` | Show help for the command |

## Core Commands

| Command | Description |
|---------|-------------|
| [init](init.md) | Initialise a new configuration file |
| [validate](validate.md) | Validate configuration file |
| [up](up.md) | Create and bootstrap a cluster |
| [down](down.md) | Delete the cluster |
| [status](status.md) | Show cluster status |
| [dump](dump.md) | Export cluster resources |
| [doctor](doctor.md) | Check system requirements and prerequisites |
| [logs](logs.md) | Stream logs from cluster components |
| [diagnostics](diagnostics.md) | Run diagnostics on cluster components |
| [apply](apply.md) | Apply Crossplane resources to the cluster |

## Management Commands

| Command | Description |
|---------|-------------|
| [provider](provider.md) | Manage Crossplane providers |
| [chart](chart.md) | Manage Helm charts |
| [credentials](credentials.md) | Configure cloud credentials |
| [cluster](cluster.md) | Manage Kind clusters |
| [config](config.md) | View and compare configuration |

## Quick Reference

### Create a Development Environment

```bash
# Check system requirements
kindplane doctor

# Initialise configuration
kindplane init

# Bootstrap cluster
kindplane up

# Check status
kindplane status
```

### Manage the Cluster

```bash
# Add a provider
kindplane provider add provider-gcp xpkg.upbound.io/upbound/provider-gcp:v1.0.0

# Install a chart
kindplane chart install prometheus https://prometheus-community.github.io/helm-charts kube-prometheus-stack

# Apply compositions
kindplane apply --from-config

# View logs
kindplane logs --component crossplane

# Export resources
kindplane dump -o ./exported
```

### Troubleshoot Issues

```bash
# Run diagnostics
kindplane diagnostics

# Stream logs from providers
kindplane logs --component providers --follow

# View configuration
kindplane config show
```

### Clean Up

```bash
# Delete the cluster
kindplane down --force
```

## Getting Help

Get help for any command:

```bash
kindplane --help
kindplane up --help
kindplane provider --help
kindplane config show --help
```
