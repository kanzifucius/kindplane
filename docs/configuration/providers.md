# Provider Management

This page covers managing Crossplane providers in kindplane.

## Configuration

Providers are configured in the `crossplane.providers` section:

```yaml
crossplane:
  providers:
    - name: provider-aws
      package: xpkg.upbound.io/upbound/provider-aws:v1.1.0
    - name: provider-kubernetes
      package: xpkg.upbound.io/crossplane-contrib/provider-kubernetes:v0.12.0
```

## Adding Providers After Bootstrap

Use the `provider add` command to add providers to a running cluster:

```bash
kindplane provider add provider-azure xpkg.upbound.io/upbound/provider-azure:v1.0.0
```

## Listing Providers

List all installed providers:

```bash
kindplane provider list
```

Or using kubectl:

```bash
kubectl get providers
```

## Provider Configuration

Most providers need a `ProviderConfig` to work with cloud resources. See the [Credentials](credentials.md) section for details.

### AWS Provider

```yaml
apiVersion: aws.upbound.io/v1beta1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: aws-credentials
      key: credentials
```

### Azure Provider

```yaml
apiVersion: azure.upbound.io/v1beta1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: azure-credentials
      key: credentials
```

### Kubernetes Provider

```yaml
apiVersion: kubernetes.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: InjectedIdentity
```

## Popular Providers

### AWS Providers

| Provider | Package | Description |
|----------|---------|-------------|
| provider-aws | `xpkg.upbound.io/upbound/provider-aws:v1.1.0` | All AWS services |
| provider-aws-s3 | `xpkg.upbound.io/upbound/provider-aws-s3:v1.1.0` | S3 only |
| provider-aws-ec2 | `xpkg.upbound.io/upbound/provider-aws-ec2:v1.1.0` | EC2 only |
| provider-aws-rds | `xpkg.upbound.io/upbound/provider-aws-rds:v1.1.0` | RDS only |
| provider-aws-iam | `xpkg.upbound.io/upbound/provider-aws-iam:v1.1.0` | IAM only |

### Azure Providers

| Provider | Package | Description |
|----------|---------|-------------|
| provider-azure | `xpkg.upbound.io/upbound/provider-azure:v1.0.0` | All Azure services |
| provider-azure-storage | `xpkg.upbound.io/upbound/provider-azure-storage:v1.0.0` | Storage only |

### GCP Providers

| Provider | Package | Description |
|----------|---------|-------------|
| provider-gcp | `xpkg.upbound.io/upbound/provider-gcp:v1.0.0` | All GCP services |
| provider-gcp-storage | `xpkg.upbound.io/upbound/provider-gcp-storage:v1.0.0` | Storage only |

### Utility Providers

| Provider | Package | Description |
|----------|---------|-------------|
| provider-kubernetes | `xpkg.upbound.io/crossplane-contrib/provider-kubernetes:v0.12.0` | Manage K8s resources |
| provider-helm | `xpkg.upbound.io/crossplane-contrib/provider-helm:v0.16.0` | Manage Helm releases |
| provider-terraform | `xpkg.upbound.io/upbound/provider-terraform:v0.13.0` | Run Terraform |

## Troubleshooting Providers

### Provider Not Healthy

Check provider status:

```bash
kubectl get providers -o wide
kubectl describe provider <provider-name>
```

### View Provider Logs

```bash
kubectl logs -n crossplane-system -l pkg.crossplane.io/provider=<provider-name>
```

### Common Issues

1. **Package not found** - Verify the package URL is correct
2. **Missing credentials** - Configure ProviderConfig with credentials
3. **Resource limits** - Increase Docker resource allocation
