# kindplane credentials

Configure cloud provider credentials for Crossplane.

## Usage

```bash
kindplane credentials <command> [flags]
```

## Subcommands

| Command | Description |
|---------|-------------|
| `configure` | Interactive credential setup |

## kindplane credentials configure

![kindplane credentials configure demo](../assets/vhs/credentials-configure.gif)

Interactively configure credentials for cloud providers.

### Usage

```bash
kindplane credentials configure
```

### Description

The `configure` command guides you through setting up credentials for each Crossplane provider in your cluster.

### Interactive Flow

```
╭────────────────────────────────────────────────────────────────╮
│  Credential Configuration                                      │
╰────────────────────────────────────────────────────────────────╯

Select provider to configure:
  1. AWS
  2. Azure
  3. Kubernetes
  4. Exit

> 1

AWS Credential Configuration
─────────────────────────────
Select credential source:
  1. Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
  2. AWS CLI profile
  3. Credentials file
  4. Skip

> 2

Enter AWS profile name [default]: development

✓ AWS credentials configured
  Profile: development
  Secret: aws-credentials (crossplane-system)
  ProviderConfig: default
```

## What Gets Created

### Secrets

Credentials are stored as Kubernetes secrets:

```bash
kubectl get secrets -n crossplane-system
```

| Secret | Provider |
|--------|----------|
| `aws-credentials` | AWS |
| `azure-credentials` | Azure |

### ProviderConfigs

ProviderConfig resources are created to reference the secrets:

```bash
kubectl get providerconfig
```

## Manual Configuration

You can also configure credentials manually:

### AWS

1. Create the secret:

    ```bash
    kubectl create secret generic aws-credentials \
      -n crossplane-system \
      --from-literal=credentials="[default]
    aws_access_key_id = YOUR_ACCESS_KEY
    aws_secret_access_key = YOUR_SECRET_KEY"
    ```

2. Create ProviderConfig:

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

### Azure

1. Create the secret:

    ```bash
    kubectl create secret generic azure-credentials \
      -n crossplane-system \
      --from-literal=credentials='{
        "subscriptionId": "...",
        "tenantId": "...",
        "clientId": "...",
        "clientSecret": "..."
      }'
    ```

2. Create ProviderConfig:

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

### Kubernetes (In-Cluster)

```yaml
apiVersion: kubernetes.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: InjectedIdentity
```

## Verifying Credentials

After configuration, verify providers are healthy:

```bash
kindplane status
```

Or:

```bash
kubectl get providers
```

If a provider shows `Unhealthy`, check the provider logs:

```bash
kubectl logs -n crossplane-system -l pkg.crossplane.io/provider=provider-aws
```

## Security Notes

!!! warning "Local Development Only"
    The credential storage method used by kindplane is suitable for local development only. For production:

    - Use IRSA (AWS)
    - Use Workload Identity (Azure/GCP)
    - Use External Secrets Operator
    - Use Vault
