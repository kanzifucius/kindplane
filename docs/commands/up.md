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
| `--skip-providers` | Skip provider installation |
| `--skip-eso` | Skip External Secrets Operator installation |
| `--skip-charts` | Skip all Helm chart installations |
| `--skip-compositions` | Skip composition deployment |
| `--rollback-on-failure` | Delete cluster if bootstrap fails |
| `--timeout` | Timeout for bootstrap operations (default: `10m`) |

## Description

The `up` command creates a Kind cluster and bootstraps it with Crossplane and your configured components.

## Bootstrap Process

The command performs these steps in order:

1. **Create Kind cluster** - Creates the cluster with configured nodes
2. **Install Crossplane** - Deploys Crossplane using Helm
3. **Wait for Crossplane** - Waits for Crossplane pods to be ready
4. **Install pre-crossplane charts** - Deploys charts with `phase: pre-crossplane`
5. **Install providers** - Deploys configured Crossplane providers
6. **Wait for providers** - Waits for all providers to be healthy
7. **Install post-providers charts** - Deploys charts with `phase: post-providers`
8. **Install ESO** - Deploys External Secrets Operator (if enabled)
9. **Wait for ESO** - Waits for ESO pods to be ready
10. **Install post-eso charts** - Deploys charts with `phase: post-eso`
11. **Apply compositions** - Deploys XRDs and Compositions

## Examples

### Full Bootstrap

```bash
kindplane up
```

### Skip Providers

Useful for testing cluster creation:

```bash
kindplane up --skip-providers
```

### Skip ESO

```bash
kindplane up --skip-eso
```

### Skip All Optional Components

```bash
kindplane up --skip-providers --skip-eso --skip-charts --skip-compositions
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

### Use Different Configuration

```bash
kindplane up --config production.yaml
```

## Progress Output

The command shows real-time progress:

```
→ Creating Kind cluster 'kindplane-dev'...
  ✓ Cluster created successfully

→ Installing Crossplane 1.15.0...
  ✓ Crossplane installed
  ✓ Crossplane pods ready

→ Installing providers...
  → Installing provider-aws...
  → Installing provider-kubernetes...
  ✓ provider-aws installed
  ✓ provider-kubernetes installed
  ✓ All providers healthy

→ Installing External Secrets Operator 0.9.11...
  ✓ ESO installed
  ✓ ESO pods ready

→ Installing Helm charts...
  ✓ ingress-nginx installed

✓ Bootstrap complete!
```

## Failure Diagnostics

When bootstrap fails, kindplane shows detailed diagnostics:

```
✗ Providers failed to become healthy: context deadline exceeded

╭────────────────────────────────────────────────────────────────╮
│  📦 Provider Diagnostics                                       │
│                                                                │
│  ✗ provider-aws                                                │
│    Package: xpkg.upbound.io/upbound/provider-aws:v1.1.0        │
│    Conditions:                                                 │
│      ✗ Healthy: False                                          │
│        Reason: UnhealthyPackageRevision                        │
│        Message: cannot resolve package dependencies...          │
│                                                                │
│  ⎈ Pod Status                                                  │
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
✗ Cluster 'kindplane-dev' already exists

Use 'kindplane down --force' to delete it first.
```

## Tips

- Use `--rollback-on-failure` in CI/CD to ensure clean state
- Increase `--timeout` for slow container registries
- Use `--skip-*` flags to isolate problems during debugging
