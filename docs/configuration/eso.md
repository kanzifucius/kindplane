# External Secrets Operator

kindplane can install [External Secrets Operator](https://external-secrets.io/) (ESO) for managing secrets from external secret stores.

## Configuration

```yaml
eso:
  enabled: true
  version: "0.9.11"
```

## Options

### enabled

Enable or disable ESO installation.

```yaml
eso:
  enabled: true
```

- **Type:** boolean
- **Default:** `false`
- **Required:** No

### version

The ESO Helm chart version to install.

```yaml
eso:
  version: "0.9.11"
```

- **Type:** string
- **Required:** Yes (if enabled)

!!! tip "Version Selection"
    Check the [ESO releases](https://github.com/external-secrets/external-secrets/releases) for available versions.

## Skipping ESO During Bootstrap

If ESO is enabled in configuration but you want to skip it temporarily:

```bash
kindplane up --skip-eso
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

eso:
  enabled: true
  version: "0.9.11"

charts:
  - name: my-app
    repo: https://charts.example.com
    chart: my-app
    namespace: default
    phase: post-eso  # Install after ESO is ready
```
