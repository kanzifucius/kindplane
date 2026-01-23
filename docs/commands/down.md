# kindplane down

Delete the Kind cluster.

## Usage

```bash
kindplane down [flags]
```

## Flags

| Flag | Description |
|------|-------------|
| `--config`, `-c` | Configuration file (default: `kindplane.yaml`) |
| `--force` | Required flag to confirm deletion |

## Description

The `down` command deletes the Kind cluster defined in your configuration file.

!!! warning "Destructive Operation"
    This command permanently deletes the cluster and all resources within it. The `--force` flag is required to confirm the operation.

## Examples

### Delete Cluster

```bash
kindplane down --force
```

### Delete Cluster from Specific Configuration

```bash
kindplane down --config production.yaml --force
```

## Output

### Success

```
→ Deleting Kind cluster 'kindplane-dev'...
✓ Cluster 'kindplane-dev' deleted
```

### Cluster Not Found

```
! Cluster 'kindplane-dev' does not exist
```

### Missing Force Flag

```
✗ The --force flag is required to delete the cluster

This operation will permanently delete the cluster and all resources.
Use: kindplane down --force
```

## What Gets Deleted

When you run `kindplane down --force`:

1. **Kind cluster** - The Docker containers running the cluster nodes
2. **Kubernetes resources** - All deployments, pods, services, etc.
3. **Crossplane resources** - Providers, compositions, managed resources
4. **Secrets** - All secrets including credentials
5. **Persistent volumes** - Any PVs created in the cluster

## What Is Preserved

The following are NOT deleted:

- Configuration file (`kindplane.yaml`)
- Local composition files
- Docker images (cached for future use)
- kubeconfig (but the cluster context is removed)

## Tips

### Export Before Deleting

If you want to preserve resources for GitOps:

```bash
# Export resources
kindplane dump -o ./backup

# Then delete
kindplane down --force
```

### CI/CD Usage

```bash
# Always clean up, ignore errors if cluster doesn't exist
kindplane down --force || true
```
