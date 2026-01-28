# kindplane logs

![kindplane logs demo](../assets/vhs/logs.gif)

Stream logs from cluster components.

## Usage

```bash
kindplane logs [flags]
```

## Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--component` | | Component to view logs for (default: `crossplane`) |
| `--follow` | `-f` | Follow log output in real-time |
| `--tail` | | Number of lines from the end (default: `100`) |
| `--since` | | Only logs newer than duration (e.g., `5m`, `1h`) |
| `--container` | | Container name (for multi-container pods) |
| `--previous` | | Show previous terminated container logs |
| `--timestamps` | | Include timestamps in output |
| `--timeout` | | Timeout for log streaming (default: `5m`) |

## Description

The `logs` command streams logs from Crossplane and related components in the cluster.

By default, it streams logs from the Crossplane controller. You can specify different components or specific pod names.

## Components

| Component | Description |
|-----------|-------------|
| `crossplane` | Crossplane controller (default) |
| `providers` | Crossplane provider pods |
| `eso` | External Secrets Operator (if installed) |
| `<pod-name>` | Specific pod by name |

## Examples

### View Crossplane Logs

```bash
kindplane logs
```

### Follow Logs in Real-Time

```bash
kindplane logs --follow
```

### View Provider Logs

```bash
kindplane logs --component providers
```

### View Last 50 Lines

```bash
kindplane logs --tail 50
```

### View Logs from Last 10 Minutes

```bash
kindplane logs --since 10m
```

### View Specific Pod Logs

```bash
kindplane logs --component crossplane-rbac-manager-7b8f9d6c5-xk2jl
```

### View Previous Container Logs

Useful after pod restarts:

```bash
kindplane logs --previous
```

### Include Timestamps

```bash
kindplane logs --timestamps
```

### Combine Flags

```bash
kindplane logs --component providers --follow --tail 200 --timestamps
```

## Output

```
 Logs
--------------------------------------------------
Streaming logs from pod: crossplane-system/crossplane-6d4f8b9c7-xk2jl

2024-01-15T10:30:01Z controller-runtime/manager "msg"="Starting manager"
2024-01-15T10:30:02Z crossplane "msg"="Starting Crossplane"
2024-01-15T10:30:03Z provider "msg"="Watching provider packages"
...
```

## Troubleshooting

### No Logs Found

If no logs appear:

1. Check the cluster exists: `kindplane status`
2. Verify pods are running: `kubectl get pods -n crossplane-system`
3. Try a different component: `kindplane logs --component providers`

### Pod Not Found

If you get "Pod not found":

```bash
# List available pods
kubectl get pods -n crossplane-system

# Use the exact pod name
kindplane logs --component <pod-name>
```

### Connection Issues

If log streaming fails:

```bash
# Check cluster connectivity
kubectl cluster-info

# Verify kubeconfig
kubectl config current-context
```

## Tips

### Debugging Provider Issues

```bash
# Watch provider logs during installation
kindplane logs --component providers --follow

# Check previous logs if provider restarted
kindplane logs --component providers --previous
```

### Real-Time Monitoring

Keep logs running while making changes:

```bash
# Terminal 1: Watch logs
kindplane logs --component crossplane --follow

# Terminal 2: Apply changes
kubectl apply -f my-composition.yaml
```

### CI/CD Debugging

Capture logs on failure:

```bash
kindplane up || {
    echo "Bootstrap failed, collecting logs..."
    kindplane logs --tail 500 > crossplane-logs.txt
    kindplane logs --component providers --tail 500 > provider-logs.txt
    exit 1
}
```
