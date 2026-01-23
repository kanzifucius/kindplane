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

## Step 2: Initialise Configuration

Create a configuration file:

```bash
kindplane init
```

This creates a `kindplane.yaml` file with sensible defaults. The file includes:

- A Kind cluster with 1 control plane and 1 worker node
- Crossplane installation
- AWS and Kubernetes providers
- External Secrets Operator

## Step 3: Bootstrap the Cluster

Create and configure your cluster:

```bash
kindplane up
```

This command will:

1. Create a Kind cluster
2. Install Crossplane
3. Install configured providers
4. Wait for providers to become healthy
5. Install External Secrets Operator (if enabled)
6. Install any configured Helm charts

!!! tip "Watch Progress"
    kindplane shows real-time progress with coloured output and status indicators.

## Step 4: Verify Status

Check that everything is running:

```bash
kindplane status
```

For more detailed information:

```bash
kindplane status --detailed
```

## Step 5: Use Your Cluster

Your cluster is now ready! You can:

```bash
# List Crossplane providers
kubectl get providers

# Check provider status
kubectl get providers -o wide

# Apply a Crossplane resource
kubectl apply -f my-resource.yaml
```

## Step 6: Clean Up

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

eso:
  enabled: true
  version: "0.9.11"

charts:
  - name: ingress-nginx
    repo: https://kubernetes.github.io/ingress-nginx
    chart: ingress-nginx
    namespace: ingress-nginx
    phase: post-eso
    wait: true
```
