# GitOps Export

This guide covers exporting cluster resources for GitOps workflows.

## Overview

The `kindplane dump` command exports Crossplane resources in a format suitable for GitOps tools like ArgoCD and Flux.

## Basic Export

Export all resources to the default directory:

```bash
kindplane dump
```

This creates:

```
dump/
├── providers/
├── providerconfigs/
├── compositions/
├── xrds/
├── claims/
└── managed/
```

## Export Options

### Custom Output Directory

```bash
kindplane dump -o ./gitops/crossplane
```

### Print to Stdout

```bash
kindplane dump --stdout > resources.yaml
```

### Selective Export

Include only specific resources:

```bash
kindplane dump --include=compositions,xrds
```

Exclude specific resources:

```bash
kindplane dump --exclude=managed,secrets
```

### Skip Secrets

Avoid exporting sensitive data:

```bash
kindplane dump --skip-secrets
```

### Dry Run

Preview what would be exported:

```bash
kindplane dump --dry-run
```

## Resource Cleaning

Exported resources are automatically cleaned for GitOps:

### Removed Fields

- `metadata.uid`
- `metadata.resourceVersion`
- `metadata.generation`
- `metadata.creationTimestamp`
- `metadata.managedFields`
- `status` section (for most resources)

### Before/After Example

Original resource:

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: database-aws
  uid: 12345-abcde
  resourceVersion: "12345"
  generation: 3
  creationTimestamp: "2024-01-15T10:00:00Z"
  managedFields:
    - ...
spec:
  compositeTypeRef:
    apiVersion: example.org/v1alpha1
    kind: XDatabase
status:
  conditions:
    - type: Healthy
```

Cleaned resource:

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: database-aws
spec:
  compositeTypeRef:
    apiVersion: example.org/v1alpha1
    kind: XDatabase
```

## GitOps Integration

### ArgoCD

#### Application Manifest

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: crossplane-resources
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/org/gitops-repo.git
    targetRevision: main
    path: crossplane
  destination:
    server: https://kubernetes.default.svc
    namespace: crossplane-system
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

#### Directory Structure

```
gitops-repo/
├── crossplane/
│   ├── xrds/
│   │   ├── xdatabase.yaml
│   │   └── xnetwork.yaml
│   └── compositions/
│       ├── database-aws.yaml
│       └── network-aws.yaml
└── applications/
    └── crossplane-resources.yaml
```

### Flux

#### Kustomisation

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: crossplane-resources
  namespace: flux-system
spec:
  interval: 10m
  path: ./crossplane
  prune: true
  sourceRef:
    kind: GitRepository
    name: gitops-repo
  healthChecks:
    - apiVersion: apiextensions.crossplane.io/v1
      kind: Composition
      name: database-aws
```

#### GitRepository

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: gitops-repo
  namespace: flux-system
spec:
  interval: 1m
  url: https://github.com/org/gitops-repo.git
  ref:
    branch: main
```

## Workflow: Local to GitOps

### Development Workflow

```bash
# 1. Create local cluster
kindplane up

# 2. Develop compositions locally
kubectl apply -f my-composition.yaml

# 3. Test with claims
kubectl apply -f my-claim.yaml

# 4. Verify resources work
kubectl get managed

# 5. Export when ready
kindplane dump -o ./gitops/crossplane --skip-secrets

# 6. Commit to git
cd ./gitops
git add .
git commit -m "Add database composition"
git push
```

### CI/CD Integration

```yaml
# .github/workflows/export.yaml
name: Export Crossplane Resources

on:
  workflow_dispatch:

jobs:
  export:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Kind
        run: |
          kindplane up

      - name: Export Resources
        run: |
          kindplane dump -o ./crossplane --skip-secrets

      - name: Create PR
        uses: peter-evans/create-pull-request@v5
        with:
          title: "Update Crossplane resources"
          branch: crossplane-export
```

## Best Practices

### Separate Concerns

Organise exports by type:

```
crossplane/
├── base/
│   ├── xrds/           # XRDs are cluster-scoped
│   └── providers/      # Provider definitions
├── compositions/
│   ├── aws/
│   └── azure/
└── claims/
    ├── dev/
    └── staging/
```

### Version Control

Tag releases for rollback capability:

```bash
git tag -a v1.0.0 -m "Initial compositions"
git push --tags
```

### Environment Separation

Use different directories or branches:

```
crossplane/
├── dev/
│   └── claims/
├── staging/
│   └── claims/
└── production/
    └── claims/
```

### Secrets Management

Never export secrets to Git. Instead:

1. Use External Secrets Operator
2. Use sealed-secrets
3. Use Vault

```yaml
# Reference secrets, don't include them
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: db-credentials
spec:
  secretStoreRef:
    kind: ClusterSecretStore
    name: vault
  target:
    name: db-credentials
  data:
    - secretKey: password
      remoteRef:
        key: database/prod
        property: password
```
