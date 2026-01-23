# Compositions

kindplane can apply Crossplane Compositions and XRDs from local paths or Git repositories.

## Configuration

```yaml
compositions:
  sources:
    - path: ./compositions
    - git: https://github.com/org/crossplane-compositions.git
      ref: main
      path: compositions/
```

## Source Types

### Local Path

Load compositions from a local directory:

```yaml
compositions:
  sources:
    - path: ./compositions
```

The path is relative to the configuration file location.

### Git Repository

Clone and load compositions from a Git repository:

```yaml
compositions:
  sources:
    - git: https://github.com/org/crossplane-compositions.git
      ref: main
      path: compositions/
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `git` | string | Yes | Git repository URL |
| `ref` | string | No | Branch, tag, or commit (default: `main`) |
| `path` | string | No | Path within the repository (default: `/`) |

## Directory Structure

Compositions directories should contain YAML files with Crossplane resources:

```
compositions/
├── xrd/
│   ├── database.yaml
│   └── network.yaml
├── composition/
│   ├── database-aws.yaml
│   ├── database-azure.yaml
│   └── network-aws.yaml
└── claims/
    └── example-database.yaml
```

kindplane will apply all YAML files found in the configured paths.

## Resource Types

### Composite Resource Definition (XRD)

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: CompositeResourceDefinition
metadata:
  name: xdatabases.example.org
spec:
  group: example.org
  names:
    kind: XDatabase
    plural: xdatabases
  versions:
    - name: v1alpha1
      served: true
      referenceable: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                region:
                  type: string
                size:
                  type: string
                  enum: [small, medium, large]
```

### Composition

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: database-aws
  labels:
    provider: aws
spec:
  compositeTypeRef:
    apiVersion: example.org/v1alpha1
    kind: XDatabase
  resources:
    - name: rds-instance
      base:
        apiVersion: rds.aws.upbound.io/v1beta1
        kind: Instance
        spec:
          forProvider:
            engine: postgres
            instanceClass: db.t3.micro
```

## Multiple Sources

Combine local and Git sources:

```yaml
compositions:
  sources:
    # Local development compositions
    - path: ./compositions

    # Shared team compositions
    - git: https://github.com/team/shared-compositions.git
      ref: v1.0.0
      path: crossplane/

    # Organisation-wide standards
    - git: https://github.com/org/platform-compositions.git
      ref: main
      path: compositions/standard/
```

## Authentication for Private Repositories

For private Git repositories, ensure your environment has appropriate credentials:

### SSH Key

```bash
# Ensure SSH agent is running
eval "$(ssh-agent -s)"
ssh-add ~/.ssh/id_rsa
```

### HTTPS with Token

```bash
# Using environment variable
export GIT_ASKPASS=echo
export GIT_USERNAME=username
export GIT_PASSWORD=token
```

## Complete Example

```yaml
cluster:
  name: compositions-demo

crossplane:
  version: "1.15.0"
  providers:
    - name: provider-aws
      package: xpkg.upbound.io/upbound/provider-aws:v1.1.0

compositions:
  sources:
    # Local XRDs and Compositions
    - path: ./platform/crossplane

    # Team shared resources
    - git: https://github.com/myteam/crossplane-configs.git
      ref: main
      path: compositions/

    # Example claims for testing
    - path: ./examples/claims
```
