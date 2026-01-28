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

## Description

The `status` command displays the current state of your cluster and all installed components.

## Examples

### Check Status

```bash
kindplane status
```

### Status from Specific Configuration

```bash
kindplane status --config production.yaml
```

## Output

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
│  Helm Charts                                                   │
│    ✓ cert-manager      cert-manager       1.14.0               │
│    ✓ ingress-nginx     ingress-nginx      4.9.0                │
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

### Quick Health Check

Combine with doctor for comprehensive check:

```bash
kindplane doctor
kindplane status
```
