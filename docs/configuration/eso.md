# Installing External Secrets Operator

External Secrets Operator (ESO) can be installed via the `charts` section, which provides more flexibility than the previous dedicated installer.

## Installation via Charts

```yaml
charts:
  - name: external-secrets
    repo: https://charts.external-secrets.io
    chart: external-secrets
    version: "0.9.11"
    namespace: external-secrets
    phase: post-providers
    values:
      installCRDs: true
```

## Options

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Helm release name (typically `external-secrets`) |
| `repo` | string | Yes | Helm repository URL |
| `chart` | string | Yes | Chart name (`external-secrets`) |
| `version` | string | No | Chart version (uses latest if omitted) |
| `namespace` | string | Yes | Target namespace (`external-secrets`) |
| `phase` | string | No | Installation phase (default: `final`) |
| `values` | object | No | Inline Helm values |
| `valuesFiles` | array | No | Paths to external values files |

## Installation Phase

Set `phase: post-providers` to install ESO after Crossplane providers are ready, matching the previous behaviour.

```yaml
charts:
  - name: external-secrets
    repo: https://charts.external-secrets.io
    chart: external-secrets
    namespace: external-secrets
    phase: post-providers
    values:
      installCRDs: true
```

## Custom Values

Customise the ESO installation with Helm values:

```yaml
charts:
  - name: external-secrets
    repo: https://charts.external-secrets.io
    chart: external-secrets
    namespace: external-secrets
    phase: post-providers
    values:
      installCRDs: true
      replicaCount: 2
      resources:
        limits:
          memory: 512Mi
          cpu: 500m
```

## Using External Values Files

For complex configurations, use external values files:

```yaml
charts:
  - name: external-secrets
    repo: https://charts.external-secrets.io
    chart: external-secrets
    namespace: external-secrets
    phase: post-providers
    valuesFiles:
      - ./values/eso-base.yaml
      - ./values/eso-overrides.yaml
    values:
      installCRDs: true
```

## What ESO Provides

External Secrets Operator synchronises secrets from external stores into Kubernetes secrets:

- AWS Secrets Manager
- AWS Parameter Store
- Azure Key Vault
- Google Secret Manager
- HashiCorp Vault
- And many more...

## Using ESO

After installation, create a `SecretStore` or `ClusterSecretStore`:

### AWS Secrets Manager Example

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ClusterSecretStore
metadata:
  name: aws-secrets-manager
spec:
  provider:
    aws:
      service: SecretsManager
      region: eu-west-1
      auth:
        secretRef:
          accessKeyIDSecretRef:
            name: aws-credentials
            namespace: external-secrets
            key: access-key
          secretAccessKeySecretRef:
            name: aws-credentials
            namespace: external-secrets
            key: secret-access-key
```

### External Secret Example

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: database-credentials
spec:
  refreshInterval: 1h
  secretStoreRef:
    kind: ClusterSecretStore
    name: aws-secrets-manager
  target:
    name: database-credentials
  data:
    - secretKey: username
      remoteRef:
        key: prod/database
        property: username
    - secretKey: password
      remoteRef:
        key: prod/database
        property: password
```

## Verifying Installation

Check ESO is running:

```bash
kubectl get pods -n external-secrets
```

Expected output:

```
NAME                                               READY   STATUS    RESTARTS   AGE
external-secrets-7f9d8b6c5-xk2jl                   1/1     Running   0          5m
external-secrets-cert-controller-5f8d9b7c4-lm3np   1/1     Running   0          5m
external-secrets-webhook-6d7e8f9c2-qr4st           1/1     Running   0          5m
```

## Complete Example

```yaml
cluster:
  name: eso-demo

crossplane:
  version: "1.15.0"
  providers:
    - name: provider-aws
      package: xpkg.upbound.io/upbound/provider-aws:v1.1.0

charts:
  - name: external-secrets
    repo: https://charts.external-secrets.io
    chart: external-secrets
    version: "0.9.11"
    namespace: external-secrets
    phase: post-providers
    values:
      installCRDs: true
  
  - name: my-app
    repo: https://charts.example.com
    chart: my-app
    namespace: default
    phase: final  # Install after ESO is ready
```

## Migration from Previous Configuration

If you previously used the `eso` section:

**Before:**
```yaml
eso:
  enabled: true
  version: "0.9.11"
```

**After:**
```yaml
charts:
  - name: external-secrets
    repo: https://charts.external-secrets.io
    chart: external-secrets
    version: "0.9.11"
    namespace: external-secrets
    phase: post-providers
    values:
      installCRDs: true
```
