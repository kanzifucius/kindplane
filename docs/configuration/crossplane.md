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

### repo

Custom Helm repository URL for Crossplane charts. Useful for air-gapped environments or private registries.

```yaml
crossplane:
  version: "1.15.0"
  repo: "https://my-private-registry.com/charts/crossplane"
```

- **Type:** string
- **Required:** No
- **Default:** `https://charts.crossplane.io/stable`

!!! tip "Private Registries"
    When using a private Helm registry, ensure it's accessible from your cluster and any required authentication is configured.

### values

Inline Helm values to customise the Crossplane installation.

```yaml
crossplane:
  version: "1.15.0"
  values:
    resourcesCrossplane:
      limits:
        memory: 512Mi
        cpu: 500m
    args:
      - --enable-composition-functions
```

- **Type:** object
- **Required:** No

Values are merged with any values files specified in `valuesFiles` (values files are loaded first, then inline values override them).

!!! tip "Common Customisations"
    - Resource limits: `resourcesCrossplane.limits`
    - Command-line arguments: `args`
    - Replica count: `replicaCount`
    - Image pull policy: `imagePullPolicy`

### valuesFiles

Paths to external YAML files containing Helm values.

```yaml
crossplane:
  version: "1.15.0"
  valuesFiles:
    - ./values/crossplane-base.yaml
    - ./values/crossplane-overrides.yaml
```

- **Type:** array of strings
- **Required:** No

Files are loaded in order, with later files overriding earlier ones. Inline `values` are applied last and take highest priority.

!!! example "Example Values File"
    ```yaml
    # values/crossplane.yaml
    resourcesCrossplane:
      limits:
        memory: 1Gi
        cpu: 1000m
      requests:
        memory: 512Mi
        cpu: 500m
    args:
      - --enable-composition-functions
      - --enable-external-secret-stores
    ```

### registryCaBundle

Configure a CA bundle for Crossplane to trust when pulling Configuration and Provider packages from private registries with custom certificates. Multiple certificates can be specified and will be bundled together.

```yaml
crossplane:
  version: "1.15.0"
  registryCaBundle:
    # Direct paths to CA files (multiple allowed)
    caFiles:
      - "./certs/corporate-root-ca.crt"
      - "./certs/proxy-ca.crt"
    # Reference existing workload CAs by name
    workloadCARefs:
      - "internal-services-ca"
```

- **Type:** object
- **Required:** No

| Field | Type | Description |
|-------|------|-------------|
| `caFiles` | array of strings | Direct paths to CA certificate files on the host |
| `workloadCARefs` | array of strings | References to workload CAs defined in `cluster.trustedCAs.workloads` by name |

At least one of `caFiles` or `workloadCARefs` must be specified. Both can be used together - all certificates will be bundled into a single ConfigMap.

!!! example "Multiple CA Certificates"
    ```yaml
    cluster:
      trustedCAs:
        workloads:
          - name: "corporate-root-ca"
            caFile: "./certs/corporate-ca.crt"
          - name: "internal-services-ca"
            caFile: "./certs/internal-ca.crt"
    
    crossplane:
      version: "1.15.0"
      registryCaBundle:
        # Combine direct files and workload CA references
        caFiles:
          - "./certs/proxy-ca.crt"
        workloadCARefs:
          - "corporate-root-ca"
          - "internal-services-ca"
      
      providers:
        - name: provider-aws
          package: xpkg.upbound.io/upbound/provider-aws:v1.1.0
    ```

When configured, kindplane will:

1. Read all CA certificate files (from direct paths and resolved workload CA references)
2. Bundle all certificates together into a single PEM file
3. Create a ConfigMap named `crossplane-registry-ca-bundle` in the `crossplane-system` namespace
4. Automatically configure the Crossplane Helm chart with `registryCaBundleConfig` pointing to this ConfigMap

This enables Crossplane to trust custom CA certificates when pulling packages from private registries.

!!! tip "Corporate Environments"
    This is particularly useful in corporate environments where:
    
    - TLS-intercepting proxies re-sign HTTPS traffic with their own CA
    - Private registries use self-signed or internal CA certificates
    - Multiple CA certificates are required (e.g., corporate root CA + proxy CA)
    
    The CA bundle is automatically injected into Crossplane's configuration, allowing it to authenticate with registries through corporate proxies.

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
  # Optional: Custom repository for air-gapped environments
  # repo: "https://my-private-registry.com/charts"
  
  # Optional: Custom Helm values
  values:
    resourcesCrossplane:
      limits:
        memory: 512Mi
    args:
      - --enable-composition-functions
  
  # Optional: External values files
  # valuesFiles:
  #   - ./values/crossplane.yaml
  
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
