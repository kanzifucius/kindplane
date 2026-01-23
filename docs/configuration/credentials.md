# Credentials Configuration

kindplane can automatically configure cloud provider credentials for Crossplane.

## Configuration

```yaml
credentials:
  aws:
    source: env
    profile: default
  azure:
    source: env
  kubernetes:
    source: incluster
```

## AWS Credentials

### Environment Variables

```yaml
credentials:
  aws:
    source: env
```

Uses `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` from environment.

### AWS Profile

```yaml
credentials:
  aws:
    source: profile
    profile: my-profile
```

Uses credentials from `~/.aws/credentials` for the specified profile.

### Credentials File

```yaml
credentials:
  aws:
    source: file
    path: /path/to/credentials
```

Uses a custom credentials file.

## Azure Credentials

### Environment Variables

```yaml
credentials:
  azure:
    source: env
```

Uses the following environment variables:

- `AZURE_SUBSCRIPTION_ID`
- `AZURE_TENANT_ID`
- `AZURE_CLIENT_ID`
- `AZURE_CLIENT_SECRET`

### Credentials File

```yaml
credentials:
  azure:
    source: file
    path: /path/to/azure-credentials.json
```

The file should contain:

```json
{
  "subscriptionId": "...",
  "tenantId": "...",
  "clientId": "...",
  "clientSecret": "..."
}
```

## Kubernetes Provider

### In-Cluster Credentials

```yaml
credentials:
  kubernetes:
    source: incluster
```

Uses the service account of the provider pod.

### Kubeconfig

```yaml
credentials:
  kubernetes:
    source: kubeconfig
    path: ~/.kube/config
    context: my-context
```

## Interactive Setup

Use the credentials command for interactive setup:

```bash
kindplane credentials configure
```

This guides you through configuring credentials for each provider.

## Provider Configurations Created

When credentials are configured, kindplane creates:

### AWS ProviderConfig

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

### Azure ProviderConfig

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

### Kubernetes ProviderConfig

```yaml
apiVersion: kubernetes.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: InjectedIdentity
```

## Security Considerations

!!! warning "Local Development Only"
    kindplane stores credentials as Kubernetes secrets for local development. Do not use this approach in production.

For production environments:

- Use IRSA (IAM Roles for Service Accounts) on AWS
- Use Workload Identity on Azure/GCP
- Use External Secrets Operator for secret management

## Complete Example

```yaml
cluster:
  name: creds-demo

crossplane:
  version: "1.15.0"
  providers:
    - name: provider-aws
      package: xpkg.upbound.io/upbound/provider-aws:v1.1.0
    - name: provider-azure
      package: xpkg.upbound.io/upbound/provider-azure:v1.0.0
    - name: provider-kubernetes
      package: xpkg.upbound.io/crossplane-contrib/provider-kubernetes:v0.12.0

credentials:
  aws:
    source: profile
    profile: development
  azure:
    source: env
  kubernetes:
    source: incluster
```

## Troubleshooting

### Missing Credentials

If providers show `Unhealthy` status due to credentials:

1. Check the secret exists:

    ```bash
    kubectl get secrets -n crossplane-system
    ```

2. Verify ProviderConfig:

    ```bash
    kubectl get providerconfig
    ```

3. Check provider logs:

    ```bash
    kubectl logs -n crossplane-system -l pkg.crossplane.io/provider=provider-aws
    ```
