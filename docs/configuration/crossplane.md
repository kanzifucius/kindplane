# Crossplane Configuration

The `crossplane` section configures the Crossplane installation.

## Basic Configuration

```yaml
crossplane:
  version: "1.15.0"
```

## Options

### version

The Crossplane version to install.

```yaml
crossplane:
  version: "1.15.0"
```

- **Type:** string
- **Required:** Yes

!!! tip "Version Selection"
    Check the [Crossplane releases](https://github.com/crossplane/crossplane/releases) for available versions.

### providers

List of Crossplane providers to install.

```yaml
crossplane:
  providers:
    - name: provider-aws
      package: xpkg.upbound.io/upbound/provider-aws:v1.1.0
    - name: provider-kubernetes
      package: xpkg.upbound.io/crossplane-contrib/provider-kubernetes:v0.12.0
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Unique name for the provider |
| `package` | string | Yes | Full OCI package path with version |

## Provider Packages

### Official Upbound Providers

Upbound maintains official providers with comprehensive cloud coverage:

```yaml
crossplane:
  providers:
    # AWS
    - name: provider-aws
      package: xpkg.upbound.io/upbound/provider-aws:v1.1.0
    
    # AWS S3 only (smaller footprint)
    - name: provider-aws-s3
      package: xpkg.upbound.io/upbound/provider-aws-s3:v1.1.0
    
    # Azure
    - name: provider-azure
      package: xpkg.upbound.io/upbound/provider-azure:v1.0.0
    
    # GCP
    - name: provider-gcp
      package: xpkg.upbound.io/upbound/provider-gcp:v1.0.0
```

### Community Providers

Community-maintained providers for various use cases:

```yaml
crossplane:
  providers:
    # Kubernetes provider (manage K8s resources)
    - name: provider-kubernetes
      package: xpkg.upbound.io/crossplane-contrib/provider-kubernetes:v0.12.0
    
    # Helm provider (manage Helm releases)
    - name: provider-helm
      package: xpkg.upbound.io/crossplane-contrib/provider-helm:v0.16.0
    
    # Terraform provider
    - name: provider-terraform
      package: xpkg.upbound.io/upbound/provider-terraform:v0.13.0
```

## Family Providers vs Monolithic Providers

Upbound offers two types of AWS/Azure/GCP providers:

### Monolithic Providers

Single package with all services:

```yaml
- name: provider-aws
  package: xpkg.upbound.io/upbound/provider-aws:v1.1.0
```

**Pros:** Simple, all-in-one
**Cons:** Larger memory footprint

### Family Providers

Individual packages per service:

```yaml
- name: provider-aws-s3
  package: xpkg.upbound.io/upbound/provider-aws-s3:v1.1.0
- name: provider-aws-ec2
  package: xpkg.upbound.io/upbound/provider-aws-ec2:v1.1.0
- name: provider-aws-rds
  package: xpkg.upbound.io/upbound/provider-aws-rds:v1.1.0
```

**Pros:** Smaller footprint, install only what you need
**Cons:** More configuration

## Complete Example

```yaml
crossplane:
  version: "1.15.0"
  providers:
    # AWS providers (family approach)
    - name: provider-aws-s3
      package: xpkg.upbound.io/upbound/provider-aws-s3:v1.1.0
    - name: provider-aws-iam
      package: xpkg.upbound.io/upbound/provider-aws-iam:v1.1.0
    
    # Kubernetes provider for in-cluster resources
    - name: provider-kubernetes
      package: xpkg.upbound.io/crossplane-contrib/provider-kubernetes:v0.12.0
    
    # Helm provider for managing Helm releases
    - name: provider-helm
      package: xpkg.upbound.io/crossplane-contrib/provider-helm:v0.16.0
```

## Provider Health

After bootstrap, verify providers are healthy:

```bash
kubectl get providers
```

Expected output:

```
NAME                  INSTALLED   HEALTHY   PACKAGE                                               AGE
provider-aws-s3       True        True      xpkg.upbound.io/upbound/provider-aws-s3:v1.1.0        5m
provider-kubernetes   True        True      xpkg.upbound.io/crossplane-contrib/provider-kubernetes:v0.12.0   5m
```

!!! info "Provider Startup Time"
    Providers may take a few minutes to become healthy as they download images and initialise.
