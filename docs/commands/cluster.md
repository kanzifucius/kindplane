# kindplane cluster

Manage Kind clusters created by kindplane.

## Usage

```bash
kindplane cluster <subcommand> [flags]
```

## Subcommands

| Subcommand | Description |
|------------|-------------|
| `list` | List all kindplane-managed Kind clusters |

---

## kindplane cluster list

![kindplane cluster list demo](../assets/vhs/cluster-list.gif)

List all Kind clusters on the system.

### Usage

```bash
kindplane cluster list [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `--all` | Show all Kind clusters, not just kindplane-managed |

### Description

Lists Kind clusters on your system. By default, shows clusters managed by kindplane based on naming conventions.

### Examples

#### List kindplane Clusters

```bash
kindplane cluster list
```

#### List All Kind Clusters

```bash
kindplane cluster list --all
```

### Output

```
╭────────────────────────────────────────────────────────────────╮
│  Kind Clusters                                                  │
├────────────────────────────────────────────────────────────────┤
│  NAME              STATUS    NODES   KUBERNETES                │
│  kindplane-dev     Running   3       v1.29.0                   │
│  kindplane-test    Running   2       v1.28.0                   │
╰────────────────────────────────────────────────────────────────╯
```

### Use Cases

#### Check Existing Clusters

Before creating a new cluster:

```bash
# See what's already running
kindplane cluster list

# Create new cluster
kindplane up
```

#### Clean Up Old Clusters

Identify clusters to remove:

```bash
# List all clusters
kindplane cluster list --all

# Delete specific cluster
kindplane down --config ./old-project/kindplane.yaml --force
```

#### CI/CD Cleanup

Ensure clean state in pipelines:

```bash
# Check for existing clusters
kindplane cluster list

# Delete if exists
kindplane down --force || true

# Create fresh cluster
kindplane up
```

## Related Commands

- [up](up.md) - Create and bootstrap a cluster
- [down](down.md) - Delete a cluster
- [status](status.md) - Show cluster status
