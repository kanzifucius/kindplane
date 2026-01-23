# Commands Overview

kindplane provides a set of commands for managing your local Crossplane development environment.

## Command Structure

```
kindplane [command] [flags]
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

## Management Commands

| Command | Description |
|---------|-------------|
| [provider](provider.md) | Manage Crossplane providers |
| [chart](chart.md) | Manage Helm charts |
| [credentials](credentials.md) | Configure cloud credentials |

## Quick Reference

### Create a Development Environment

```bash
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

# Export resources
kindplane dump -o ./exported
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
```
