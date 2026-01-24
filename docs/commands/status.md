# kindplane status

![kindplane status demo](../assets/vhs/status.gif)

Show cluster and component status.

## Usage

```bash
kindplane status [flags]
```

## Flags

| Flag | Description |
|------|-------------|
| `--config`, `-c` | Configuration file (default: `kindplane.yaml`) |
| `--detailed` | Include detailed pod information |

## Description

The `status` command displays the current state of your cluster and all installed components.

## Examples

### Basic Status

```bash
kindplane status
```

### Detailed Status

```bash
kindplane status --detailed
```

### Status from Specific Configuration

```bash
kindplane status --config production.yaml
```

## Output

### Basic Status

```
╭────────────────────────────────────────────────────────────────╮
│  kindplane Status                                              │
├────────────────────────────────────────────────────────────────┤
│                                                                │
│  Cluster: kindplane-dev                                        │
│  Status:  Running                                              │
│  Nodes:   3 (1 control-plane, 2 workers)                       │
│                                                                │
│  Crossplane                                                    │
│    Version: 1.15.0                                             │
│    Status:  Healthy                                            │
│                                                                │
│  Providers                                                     │
│    ✓ provider-aws          Healthy                             │
│    ✓ provider-kubernetes   Healthy                             │
│                                                                │
│  External Secrets Operator                                     │
│    Version: 0.9.11                                             │
│    Status:  Healthy                                            │
│                                                                │
│  Helm Charts                                                   │
│    ✓ cert-manager      cert-manager       1.14.0               │
│    ✓ ingress-nginx     ingress-nginx      4.9.0                │
│                                                                │
╰────────────────────────────────────────────────────────────────╯
```

### Detailed Status

With `--detailed`, additional pod information is shown:

```
╭────────────────────────────────────────────────────────────────╮
│  kindplane Status (Detailed)                                   │
├────────────────────────────────────────────────────────────────┤
│                                                                │
│  Cluster: kindplane-dev                                        │
│  Status:  Running                                              │
│                                                                │
│  Nodes                                                         │
│    ✓ kindplane-dev-control-plane  Ready   v1.29.0              │
│    ✓ kindplane-dev-worker         Ready   v1.29.0              │
│    ✓ kindplane-dev-worker2        Ready   v1.29.0              │
│                                                                │
│  Crossplane Pods (crossplane-system)                           │
│    ✓ crossplane-6d4f8b9c7-xk2jl           Running  1/1         │
│    ✓ crossplane-rbac-manager-5f7d8b9c4    Running  1/1         │
│                                                                │
│  Provider Pods                                                 │
│    ✓ provider-aws-7b8f9d6c5-lm3np         Running  1/1         │
│    ✓ provider-kubernetes-6f9d8c7b4-qr4st  Running  1/1         │
│                                                                │
│  ESO Pods (external-secrets)                                   │
│    ✓ external-secrets-7f9d8b6c5-xk2jl     Running  1/1         │
│    ✓ external-secrets-webhook-5f8d9b7c4   Running  1/1         │
│    ✓ external-secrets-cert-ctrl-6d7e8f9   Running  1/1         │
│                                                                │
╰────────────────────────────────────────────────────────────────╯
```

## Status Indicators

| Icon | Meaning |
|------|---------|
| ✓ | Healthy/Running |
| ! | Warning/Degraded |
| ✗ | Unhealthy/Failed |

## Cluster Not Found

If the cluster doesn't exist:

```
✗ Cluster 'kindplane-dev' not found

Run 'kindplane up' to create the cluster.
```

## Tips

### Watch Status

Monitor status continuously:

```bash
watch kindplane status
```

### JSON Output (using kubectl)

For scripting, use kubectl directly:

```bash
kubectl get providers -o json
kubectl get pods -A -o json
```
