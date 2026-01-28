# kindplane diagnostics

![kindplane diagnostics demo](../assets/vhs/diagnostics.gif)

Run diagnostics on cluster components.

## Usage

```bash
kindplane diagnostics [flags]
```

## Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--component` | | Component to diagnose (crossplane, providers, eso, helm) |
| `--namespace` | `-n` | Namespace to inspect |
| `--release` | | Helm release name (for helm component) |
| `--max-logs` | | Maximum log lines per container (default: `30`) |
| `--timeout` | | Timeout for collection (default: `30s`) |

## Description

The `diagnostics` command collects and displays diagnostic information for cluster components to help troubleshoot issues.

It gathers detailed information about:

- Pod status and conditions
- Container states and restarts
- Recent container logs
- Provider health status
- Helm release status

## Components

| Component | Description |
|-----------|-------------|
| `crossplane` | Crossplane controller and RBAC manager |
| `providers` | Crossplane provider pods and status |
| `eso` | External Secrets Operator (if installed) |
| `helm` | Specific Helm release |

## Examples

### Diagnose All Components

```bash
kindplane diagnostics
```

### Diagnose Crossplane

```bash
kindplane diagnostics --component crossplane
```

### Diagnose Providers

```bash
kindplane diagnostics --component providers
```

### Diagnose Specific Helm Release

```bash
kindplane diagnostics --component helm --release nginx-ingress --namespace ingress-nginx
```

### Increase Log Output

```bash
kindplane diagnostics --max-logs 50
```

## Output

### Healthy Components

```
 Cluster Diagnostics
--------------------------------------------------

Running diagnostics for cluster 'kindplane-dev'...

✓ No issues found for component: crossplane
✓ No issues found for component: providers

╭────────────────────────────────────────────────╮
│  All Healthy                                    │
│  All components are functioning correctly.     │
╰────────────────────────────────────────────────╯
```

### Issues Found

```
 Cluster Diagnostics
--------------------------------------------------

Running diagnostics for cluster 'kindplane-dev'...

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
│      Restarts: 3                                               │
│                                                                │
│  Recent Logs:                                                  │
│    error: failed to initialize provider                        │
│    error: cannot connect to registry                           │
╰────────────────────────────────────────────────────────────────╯
```

## When to Use

### After Bootstrap Failures

When `kindplane up` fails, diagnostics are shown automatically. For additional investigation:

```bash
kindplane diagnostics --component providers
```

### Debugging Provider Issues

```bash
# Get detailed provider diagnostics
kindplane diagnostics --component providers --max-logs 100
```

### Helm Chart Issues

```bash
# Diagnose a specific Helm release
kindplane diagnostics --component helm --release my-app --namespace default
```

### Ongoing Monitoring

```bash
# Quick health check
kindplane diagnostics
```

## Tips

### Combine with Logs

For ongoing issues, combine diagnostics with log streaming:

```bash
# Get snapshot of issues
kindplane diagnostics --component providers

# Then stream live logs
kindplane logs --component providers --follow
```

### CI/CD Integration

Capture diagnostics on failure:

```bash
kindplane up || {
    kindplane diagnostics > diagnostics-report.txt
    exit 1
}
```

### Specific Namespace Investigation

```bash
kindplane diagnostics --component helm --namespace my-namespace
```
