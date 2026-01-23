# kindplane

<p align="center">
  <img src="assets/logo.png" alt="kindplane logo" width="200">
</p>

<p align="center">
  <a href="https://github.com/kanzifucius/kindplane/actions/workflows/ci.yaml"><img src="https://github.com/kanzifucius/kindplane/actions/workflows/ci.yaml/badge.svg" alt="Build"></a>
  <a href="https://github.com/kanzifucius/kindplane/releases/latest"><img src="https://img.shields.io/github/v/release/kanzifucius/kindplane?style=flat" alt="Release"></a>
  <img src="https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License">
  <img src="https://img.shields.io/badge/Crossplane-1.15+-7C3AED?style=flat" alt="Crossplane">
</p>

**kindplane** is a CLI tool for **local development** that helps developers quickly spin up [Kind](https://kind.sigs.k8s.io/) (Kubernetes in Docker) clusters pre-configured with [Crossplane](https://crossplane.io/), cloud providers, and other essential components.

It automates the tedious process of setting up a local Kubernetes development environment with Crossplane for infrastructure management.

!!! warning "Local Development Only"
    kindplane is designed for local development and testing only. Do not use it to manage production infrastructure.

## Features

- :rocket: **One-command bootstrap** - Create fully configured Kind clusters with a single command
- :gear: **Crossplane integration** - Automatic installation and configuration of Crossplane
- :package: **Provider management** - Install and manage Crossplane providers (AWS, Azure, GCP, etc.)
- :lock: **External Secrets Operator** - Optional ESO installation for secrets management
- :bar_chart: **Helm chart support** - Install any Helm chart with configurable phases
- :art: **Beautiful CLI** - Rich terminal output with colours, icons, and progress indicators
- :mag: **Smart diagnostics** - Detailed failure diagnostics with pod logs and conditions
- :floppy_disk: **GitOps export** - Dump cluster resources in GitOps-friendly format

## Quick Start

Get a local Crossplane development environment running in minutes:

```bash
# Install kindplane
curl -fsSL https://raw.githubusercontent.com/kanzifucius/kindplane/main/install.sh | bash

# Initialise configuration
kindplane init

# Create and bootstrap the cluster
kindplane up

# Check status
kindplane status

# Delete the cluster
kindplane down --force
```

## How It Works

```mermaid
graph LR
    A[kindplane init] --> B[kindplane.yaml]
    B --> C[kindplane up]
    C --> D[Kind Cluster]
    D --> E[Crossplane]
    E --> F[Providers]
    F --> G[ESO]
    G --> H[Helm Charts]
```

1. **Initialise** - Generate a configuration file with `kindplane init`
2. **Configure** - Customise your cluster, providers, and charts in `kindplane.yaml`
3. **Bootstrap** - Run `kindplane up` to create and configure everything
4. **Develop** - Use your local Crossplane environment for development
5. **Export** - Use `kindplane dump` to export resources for GitOps

## Next Steps

<div class="grid cards" markdown>

- :material-download: **[Installation](getting-started/installation.md)**

    ---

    Install kindplane on your system

- :material-rocket-launch: **[Quick Start](getting-started/quickstart.md)**

    ---

    Get up and running in minutes

- :material-cog: **[Configuration](configuration/overview.md)**

    ---

    Learn about all configuration options

- :material-console: **[Commands](commands/overview.md)**

    ---

    Explore all available commands

</div>
