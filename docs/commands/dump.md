# kindplane dump

![kindplane dump demo](../assets/vhs/dump.gif)

Export cluster resources for GitOps workflows.

## Usage

```bash
kindplane dump [flags]
```

## Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--output` | `-o` | Output directory (default: `./dump`) |
| `--stdout` | | Print to stdout instead of files |
| `--dry-run` | | Preview what would be dumped |
| `--include` | | Resource types to include (comma-separated) |
| `--exclude` | | Resource types to exclude (comma-separated) |
| `--skip-secrets` | | Skip all secrets |

## Description

The `dump` command exports Crossplane resources from your cluster in a GitOps-friendly format.

This is useful for:

- Migrating resources to a GitOps repository
- Backing up cluster state
- Creating reproducible configurations
- Sharing configurations between environments

## Examples

### Dump All Resources

```bash
kindplane dump
```

Creates files in `./dump` directory.

### Custom Output Directory

```bash
kindplane dump -o ./exported
```

### Print to Stdout

```bash
kindplane dump --stdout
```

### Dry Run

Preview what would be dumped:

```bash
kindplane dump --dry-run
```

### Include Specific Resources

```bash
kindplane dump --include=providers,compositions,xrds
```

### Exclude Resources

```bash
kindplane dump --exclude=secrets,configmaps
```

### Skip Secrets

```bash
kindplane dump --skip-secrets
```

## Output Structure

The dump command creates a structured directory:

```
dump/
├── providers/
│   ├── provider-aws.yaml
│   └── provider-kubernetes.yaml
├── providerconfigs/
│   ├── default-aws.yaml
│   └── default-kubernetes.yaml
├── compositions/
│   ├── database-aws.yaml
│   └── network-aws.yaml
├── xrds/
│   ├── xdatabases.example.org.yaml
│   └── xnetworks.example.org.yaml
├── claims/
│   └── my-database.yaml
└── managed/
    ├── s3-bucket-abc123.yaml
    └── rds-instance-xyz789.yaml
```

## Resource Types

Available resource types for `--include`/`--exclude`:

| Type | Description |
|------|-------------|
| `providers` | Crossplane Provider resources |
| `providerconfigs` | ProviderConfig resources |
| `compositions` | Composition resources |
| `xrds` | CompositeResourceDefinition resources |
| `claims` | Composite Resource Claims |
| `managed` | Managed Resources |
| `secrets` | Kubernetes Secrets |
| `configmaps` | Kubernetes ConfigMaps |

## Resource Cleaning

Dumped resources are cleaned for GitOps:

- Removed fields:
    - `metadata.uid`
    - `metadata.resourceVersion`
    - `metadata.generation`
    - `metadata.creationTimestamp`
    - `metadata.managedFields`
    - `status` (optional)

## GitOps Integration

### ArgoCD

```yaml
# application.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: crossplane-resources
spec:
  source:
    repoURL: https://github.com/org/gitops-repo.git
    path: crossplane/dump
  destination:
    server: https://kubernetes.default.svc
```

### Flux

```yaml
# kustomization.yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: crossplane-resources
spec:
  path: ./crossplane/dump
  sourceRef:
    kind: GitRepository
    name: gitops-repo
```

## Tips

### Regular Backups

```bash
# Timestamp-based backups
kindplane dump -o "./backups/$(date +%Y%m%d-%H%M%S)"
```

### Pipeline Integration

```bash
# CI/CD: Dump and commit
kindplane dump -o ./crossplane
git add ./crossplane
git commit -m "Update Crossplane resources"
git push
```
