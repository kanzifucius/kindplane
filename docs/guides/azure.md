# Working with Azure

This guide covers setting up kindplane for Azure provider development.

## Prerequisites

- Azure subscription
- Azure CLI installed and configured

## Configuration

### Basic Azure Setup

```yaml
cluster:
  name: azure-dev

crossplane:
  version: "1.15.0"
  providers:
    - name: provider-azure
      package: xpkg.upbound.io/upbound/provider-azure:v1.0.0

credentials:
  azure:
    source: env
```

### Family Providers (Smaller Footprint)

For a smaller memory footprint:

```yaml
crossplane:
  version: "1.15.0"
  providers:
    - name: provider-azure-storage
      package: xpkg.upbound.io/upbound/provider-azure-storage:v1.0.0
    - name: provider-azure-network
      package: xpkg.upbound.io/upbound/provider-azure-network:v1.0.0
```

## Credential Configuration

### Create Service Principal

```bash
# Login to Azure
az login

# Create service principal
az ad sp create-for-rbac \
  --name kindplane-crossplane \
  --role Contributor \
  --scopes /subscriptions/<subscription-id>
```

This outputs:

```json
{
  "appId": "...",
  "displayName": "kindplane-crossplane",
  "password": "...",
  "tenant": "..."
}
```

### Using Environment Variables

Set the credentials:

```bash
export AZURE_SUBSCRIPTION_ID="your-subscription-id"
export AZURE_TENANT_ID="your-tenant-id"
export AZURE_CLIENT_ID="your-app-id"
export AZURE_CLIENT_SECRET="your-password"
```

Configure in kindplane:

```yaml
credentials:
  azure:
    source: env
```

### Using Credentials File

Create a credentials file:

```json
{
  "subscriptionId": "your-subscription-id",
  "tenantId": "your-tenant-id",
  "clientId": "your-app-id",
  "clientSecret": "your-password"
}
```

Configure in kindplane:

```yaml
credentials:
  azure:
    source: file
    path: ~/.azure/crossplane-credentials.json
```

## Bootstrap and Verify

```bash
# Bootstrap
kindplane up

# Verify provider is healthy
kubectl get providers
```

Expected output:

```
NAME             INSTALLED   HEALTHY   PACKAGE                                      AGE
provider-azure   True        True      xpkg.upbound.io/upbound/provider-azure:v1.0.0   5m
```

## Creating Azure Resources

### Resource Group Example

```yaml
apiVersion: azure.upbound.io/v1beta1
kind: ResourceGroup
metadata:
  name: my-resource-group
spec:
  forProvider:
    location: UK South
  providerConfigRef:
    name: default
```

### Storage Account Example

```yaml
apiVersion: storage.azure.upbound.io/v1beta1
kind: Account
metadata:
  name: mystorageaccount
spec:
  forProvider:
    resourceGroupNameRef:
      name: my-resource-group
    location: UK South
    accountTier: Standard
    accountReplicationType: LRS
  providerConfigRef:
    name: default
```

## Using Compositions

### Storage Composition

Create an XRD:

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: CompositeResourceDefinition
metadata:
  name: xstorages.azure.example.org
spec:
  group: azure.example.org
  names:
    kind: XStorage
    plural: xstorages
  claimNames:
    kind: Storage
    plural: storages
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
                location:
                  type: string
                tier:
                  type: string
                  enum: [Standard, Premium]
              required:
                - location
```

Create a Composition:

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: storage-azure
  labels:
    provider: azure
spec:
  compositeTypeRef:
    apiVersion: azure.example.org/v1alpha1
    kind: XStorage
  resources:
    - name: resource-group
      base:
        apiVersion: azure.upbound.io/v1beta1
        kind: ResourceGroup
        spec:
          forProvider: {}
      patches:
        - fromFieldPath: spec.location
          toFieldPath: spec.forProvider.location

    - name: storage-account
      base:
        apiVersion: storage.azure.upbound.io/v1beta1
        kind: Account
        spec:
          forProvider:
            accountReplicationType: LRS
      patches:
        - fromFieldPath: spec.location
          toFieldPath: spec.forProvider.location
        - fromFieldPath: spec.tier
          toFieldPath: spec.forProvider.accountTier
```

## Troubleshooting

### Provider Unhealthy

Check provider status:

```bash
kubectl describe provider provider-azure
```

Check provider logs:

```bash
kubectl logs -n crossplane-system -l pkg.crossplane.io/provider=provider-azure
```

### Authentication Errors

1. Verify service principal is valid:

    ```bash
    az login --service-principal \
      -u $AZURE_CLIENT_ID \
      -p $AZURE_CLIENT_SECRET \
      --tenant $AZURE_TENANT_ID
    ```

2. Check permissions:

    ```bash
    az role assignment list --assignee $AZURE_CLIENT_ID
    ```

3. Reconfigure credentials:

    ```bash
    kindplane credentials configure
    ```

### Resource Provisioning Failures

Check resource status:

```bash
kubectl describe resourcegroup my-resource-group
```

Look for error messages in `Status.Conditions`.

## Best Practices

1. **Use separate subscriptions** for development and production
2. **Apply least privilege** when creating service principals
3. **Use family providers** for smaller footprint
4. **Clean up resources** before deleting cluster
5. **Use resource groups** to organise related resources
