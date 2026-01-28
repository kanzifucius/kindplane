# kindplane up

![kindplane up demo](../assets/vhs/up.gif)

Create and bootstrap a Kind cluster with Crossplane.

## Usage

```bash
kindplane up [flags]
```

## Flags

| Flag | Description |
|------|-------------|
| `--config`, `-c` | Configuration file (default: `kindplane.yaml`) |
| `--skip-crossplane` | Skip Crossplane installation |
| `--skip-providers` | Skip provider installation |
| `--skip-charts` | Skip all Helm chart installations |
| `--skip-compositions` | Skip composition deployment |
| `--rollback-on-failure` | Delete cluster if bootstrap fails |
| `--timeout` | Timeout for bootstrap operations (default: `10m`) |
| `--show-values` | Display merged Helm values before installation |

## Description

The `up` command creates a Kind cluster and bootstraps it with Crossplane and your configured components.

## Bootstrap Process

The command performs these steps in order:

1. **Create local registry** - Creates container registry (if enabled)
2. **Configure trusted CAs** - Validates CA certificates (if configured)
3. **Create Kind cluster** - Creates the cluster with configured nodes
4. **Connect to cluster** - Establishes Kubernetes client connection
5. **Install pre-crossplane charts** - Deploys charts with `phase: pre-crossplane`
6. **Install Crossplane** - Deploys Crossplane using Helm
7. **Install post-crossplane charts** - Deploys charts with `phase: post-crossplane`
8. **Install providers** - Deploys configured Crossplane providers
9. **Wait for providers** - Waits for all providers to be healthy
10. **Install post-providers charts** - Deploys charts with `phase: post-providers`
11. **Install final charts** - Deploys charts with `phase: final`
12. **Apply compositions** - Deploys XRDs and Compositions

## Display Modes

kindplane automatically detects your terminal:

- **Dashboard mode** (TTY) - Interactive TUI with real-time progress
- **Print mode** (non-TTY/CI) - Traditional log output

## Examples

### Full Bootstrap

```bash
kindplane up
```

### Skip Crossplane

Useful for testing cluster creation only:

```bash
kindplane up --skip-crossplane
```

### Skip Providers

```bash
kindplane up --skip-providers
```

### Skip All Optional Components

```bash
kindplane up --skip-providers --skip-charts --skip-compositions
```

### Rollback on Failure

Automatically delete the cluster if bootstrap fails:

```bash
kindplane up --rollback-on-failure
```

### Custom Timeout

Increase timeout for slow networks:

```bash
kindplane up --timeout 20m
```

### Show Helm Values

Display merged values during installation:

```bash
kindplane up --show-values
```

### Use Different Configuration

```bash
kindplane up --config production.yaml
```

## Progress Output

The command shows real-time progress:

```
 Bootstrap Cluster
--------------------------------------------------
  Cluster: kindplane-dev
  Config:  kindplane.yaml

  → Create Kind cluster
  → Connect to cluster
  → Install Crossplane
  → Install providers

→ Create Kind cluster
  ✓ Preparing nodes
  ✓ Writing configuration
  ✓ Starting control-plane
  ✓ Installing CNI
  ✓ Installing StorageClass

✓ Cluster created
→ Connect to cluster...
✓ Connected
→ Installing Crossplane 1.15.0...
  ✓ Adding Helm repository
  ✓ Creating namespace
  ✓ Installing Helm chart
→ Waiting for Crossplane pods...
  crossplane-6d4f8b9c7-xk2jl    Running  1/1
  crossplane-rbac-manager-5f7d  Running  1/1

✓ Bootstrap complete!
```

## Failure Diagnostics

When bootstrap fails, kindplane shows detailed diagnostics:

```
✗ Providers failed to become healthy: context deadline exceeded

╭────────────────────────────────────────────────────────────────╮
│  Provider Diagnostics                                           │
│                                                                │
│  ✗ provider-aws                                                │
│    Package: xpkg.upbound.io/upbound/provider-aws:v1.1.0        │
│    Conditions:                                                 │
│      ✗ Healthy: False                                          │
│        Reason: UnhealthyPackageRevision                        │
│        Message: cannot resolve package dependencies...          │
│                                                                │
│  Pod Status                                                    │
│    provider-aws-7b8f9d6c5-xk2jl (crossplane-system)            │
│      Phase: Running                                            │
│      Ready: 0/1 containers                                     │
│      Recent Logs:                                              │
│        error: failed to initialize provider                    │
╰────────────────────────────────────────────────────────────────╯
```

## Existing Cluster

If a cluster with the same name already exists:

```
! Cluster 'kindplane-dev' already exists (skipped)
```

The cluster phase is skipped and bootstrap continues with the existing cluster.

## Tips

- Use `--rollback-on-failure` in CI/CD to ensure clean state
- Increase `--timeout` for slow container registries
- Use `--skip-*` flags to isolate problems during debugging
- Run `kindplane doctor` before first bootstrap to verify prerequisites
