# Quick Start

This guide will help you get a local Crossplane development environment running in minutes.

## Prerequisites

Before starting, ensure you have:

- [Docker](https://docs.docker.com/get-docker/) installed and running
- [kubectl](https://kubernetes.io/docs/tasks/tools/) installed

!!! note "No Helm or Kind Required"
    Helm and Kind are embedded as Go libraries in kindplane—no separate installation required.

## Step 1: Install kindplane

```bash
curl -fsSL https://raw.githubusercontent.com/kanzifucius/kindplane/main/install.sh | bash
```

## Step 2: Check Prerequisites

Verify your system is ready:

```bash
kindplane doctor
```

## Step 3: Initialise Configuration

Create a configuration file:

```bash
kindplane init
```

This creates a `kindplane.yaml` file with sensible defaults. The file includes:

- A Kind cluster with 1 control plane and 1 worker node
- Crossplane installation
- AWS and Kubernetes providers

## Step 4: Bootstrap the Cluster

Create and configure your cluster:

```bash
kindplane up
```

This command will:

1. Create a Kind cluster
2. Install Crossplane
3. Install configured providers
4. Wait for providers to become healthy
5. Install any configured Helm charts
6. Apply compositions (if configured)

!!! tip "Watch Progress"
    kindplane shows real-time progress with a beautiful TUI dashboard in interactive mode.

## Step 5: Verify Status

Check that everything is running:

```bash
kindplane status
```

## Step 6: Use Your Cluster

Your cluster is now ready! You can:

```bash
# List Crossplane providers
kubectl get providers

# Check provider status
kubectl get providers -o wide

# Apply a Crossplane resource
kubectl apply -f my-resource.yaml
```

## Step 7: Clean Up

When you're done, delete the cluster:

```bash
kindplane down --force
```

## Next Steps

- [Configure your cluster](../configuration/overview.md) with custom settings
- [Add more providers](../configuration/providers.md) for different clouds
- [Install Helm charts](../configuration/charts.md) for additional tools
- [Export resources](../commands/dump.md) for GitOps workflows

## Example: Minimal Configuration

Here's a minimal configuration to get started:

```yaml
cluster:
  name: my-dev-cluster

crossplane:
  version: "1.15.0"
  providers:
    - name: provider-kubernetes
      package: xpkg.upbound.io/crossplane-contrib/provider-kubernetes:v0.12.0
```

## Example: Full Development Setup

Here's a more complete development configuration:

```yaml
cluster:
  name: dev-cluster
  kubernetesVersion: "1.29.0"
  nodes:
    controlPlane: 1
    workers: 2
  portMappings:
    - containerPort: 80
      hostPort: 8080
      protocol: TCP
  ingress:
    enabled: true

crossplane:
  version: "1.15.0"
  providers:
    - name: provider-aws
      package: xpkg.upbound.io/upbound/provider-aws-s3:v1.1.0
    - name: provider-kubernetes
      package: xpkg.upbound.io/crossplane-contrib/provider-kubernetes:v0.12.0

charts:
  # Install External Secrets Operator via charts
  - name: external-secrets
    repo: https://charts.external-secrets.io
    chart: external-secrets
    namespace: external-secrets
    phase: post-providers
    values:
      installCRDs: true

  # Install ingress controller
  - name: ingress-nginx
    repo: https://kubernetes.github.io/ingress-nginx
    chart: ingress-nginx
    namespace: ingress-nginx
    phase: final
    wait: true
```
