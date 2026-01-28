# kindplane apply

![kindplane apply demo](../assets/vhs/apply.gif)

Apply Crossplane resources to the cluster.

## Usage

```bash
kindplane apply [flags]
```

## Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--file` | `-f` | Path to a YAML file to apply |
| `--directory` | `-d` | Path to a directory containing YAML files |
| `--recursive` | `-R` | Recursively apply files from subdirectories |
| `--dry-run` | | Show what would be applied without making changes |
| `--from-config` | | Apply compositions from the configuration file |
| `--timeout` | | Timeout for apply operation (default: `5m`) |

## Description

The `apply` command allows you to apply Crossplane compositions, XRDs, and other resources to the cluster without running the full `up` workflow.

This is useful for:

- Deploying new compositions to an existing cluster
- Testing composition changes
- Applying resources from Git repositories
- Iterative development workflows

## Examples

### Apply from Configuration

Apply all composition sources defined in `kindplane.yaml`:

```bash
kindplane apply --from-config
```

### Apply a Single File

```bash
kindplane apply --file ./compositions/database.yaml
```

### Apply Directory

Apply all YAML files in a directory:

```bash
kindplane apply --directory ./compositions
```

### Apply Recursively

Apply all YAML files including subdirectories:

```bash
kindplane apply --directory ./crossplane --recursive
```

### Dry Run

Preview what would be applied:

```bash
kindplane apply --file ./compositions/database.yaml --dry-run
```

## Output

### Applying from Config

```
 Apply Resources
--------------------------------------------------

Applying compositions from configuration...
→ Applying from local: ./compositions
✓ Applied compositions from: ./compositions
```

### Applying Files

```
 Apply Resources
--------------------------------------------------

Applying resources from: ./compositions
→ Found 5 YAML files
✓ Applied: database.yaml
✓ Applied: network.yaml
✓ Applied: storage.yaml
✓ Applied: xrd-database.yaml
✓ Applied: xrd-network.yaml
```

### Dry Run

```
 Apply Resources
--------------------------------------------------

Applying file: ./compositions/database.yaml
  [dry-run] Would apply: ./compositions/database.yaml
```

## Configuration Sources

In `kindplane.yaml`, you can define composition sources:

```yaml
compositions:
  sources:
    # Local directory
    - type: local
      path: ./compositions

    # Git repository
    - type: git
      repo: https://github.com/org/crossplane-compositions.git
      branch: main
      path: compositions
```

### Local Sources

Apply resources from local directories:

```yaml
compositions:
  sources:
    - type: local
      path: ./compositions
    - type: local
      path: ./xrds
```

### Git Sources

Apply resources from Git repositories:

```yaml
compositions:
  sources:
    - type: git
      repo: https://github.com/org/crossplane-compositions.git
      branch: main
      path: database
```

## Workflow Examples

### Iterative Development

Develop compositions with fast feedback:

```bash
# Make changes to composition
vim ./compositions/database.yaml

# Apply changes
kindplane apply --file ./compositions/database.yaml

# Test the composition
kubectl apply -f ./claims/test-database.yaml

# Check status
kubectl get composite
```

### Apply New Compositions

Add compositions to an existing cluster:

```bash
# After kindplane up has completed
kindplane apply --directory ./new-compositions
```

### Test Before Apply

```bash
# Preview changes
kindplane apply --directory ./compositions --dry-run

# Apply if satisfied
kindplane apply --directory ./compositions
```

## Tips

### Use with Git Hooks

Apply compositions on commit:

```bash
#!/bin/sh
# .git/hooks/post-commit
kindplane apply --from-config
```

### CI/CD Integration

Apply compositions in a pipeline:

```bash
# Validate cluster exists
kindplane status

# Apply compositions
kindplane apply --from-config

# Verify resources
kubectl get xrd
kubectl get compositions
```

### Combine Sources

Apply from multiple locations:

```bash
# Apply from config (includes Git repos)
kindplane apply --from-config

# Then apply local overrides
kindplane apply --directory ./local-overrides
```
