# Helm Charts

kindplane can install Helm charts as part of the bootstrap process.

## Configuration

```yaml
charts:
  - name: cert-manager
    repo: https://charts.jetstack.io
    chart: cert-manager
    namespace: cert-manager
    version: "1.14.0"
    phase: pre-crossplane
    values:
      installCRDs: true
```

## Options

### name

A unique identifier for the chart installation.

- **Type:** string
- **Required:** Yes

### repo

The Helm repository URL.

- **Type:** string
- **Required:** Yes

### chart

The chart name within the repository.

- **Type:** string
- **Required:** Yes

### namespace

The Kubernetes namespace for installation.

- **Type:** string
- **Required:** Yes

### version

The chart version to install.

- **Type:** string
- **Required:** No (uses latest if not specified)

### phase

When to install the chart during bootstrap.

- **Type:** string
- **Default:** `final`
- **Required:** No

Available phases:

| Phase | Description |
|-------|-------------|
| `pre-crossplane` | Before Crossplane installation |
| `post-crossplane` | After Crossplane is ready |
| `post-providers` | After all providers are healthy |
| `final` | Final phase, after everything else (default) |

!!! note "Deprecated Phase"
    The `post-eso` phase is deprecated and will be mapped to `final` for backwards compatibility.

### wait

Wait for the chart to be fully deployed.

- **Type:** boolean
- **Default:** `true`
- **Required:** No

### timeout

Timeout for chart installation.

- **Type:** string (duration)
- **Default:** `5m`
- **Required:** No

### values

Inline values to pass to the chart.

```yaml
charts:
  - name: nginx
    # ...
    values:
      replicaCount: 3
      service:
        type: ClusterIP
```

### valuesFiles

External values files to use.

```yaml
charts:
  - name: prometheus
    # ...
    valuesFiles:
      - ./values/prometheus-base.yaml
      - ./values/prometheus-dev.yaml
```

## Phase Examples

### pre-crossplane

Install dependencies before Crossplane:

```yaml
charts:
  - name: cert-manager
    repo: https://charts.jetstack.io
    chart: cert-manager
    namespace: cert-manager
    version: "1.14.0"
    phase: pre-crossplane
    values:
      installCRDs: true
```

### post-crossplane

Install after Crossplane but before providers:

```yaml
charts:
  - name: crossplane-contrib
    repo: https://charts.crossplane.io/master
    chart: crossplane-contrib
    namespace: crossplane-system
    phase: post-crossplane
```

### post-providers

Install after providers are healthy:

```yaml
charts:
  - name: external-secrets
    repo: https://charts.external-secrets.io
    chart: external-secrets
    namespace: external-secrets
    phase: post-providers
    values:
      installCRDs: true

  - name: argo-cd
    repo: https://argoproj.github.io/argo-helm
    chart: argo-cd
    namespace: argocd
    phase: post-providers
    wait: true
```

### final (default)

Install in the final phase after everything else:

```yaml
charts:
  - name: my-app
    repo: https://charts.example.com
    chart: my-app
    namespace: default
    phase: final
```

## Managing Charts

### List Installed Charts

```bash
kindplane chart list
```

### Install a Chart Manually

```bash
kindplane chart install my-chart https://charts.example.com my-chart --namespace default
```

### Uninstall a Chart

```bash
kindplane chart uninstall my-chart my-namespace
```

## Complete Example

```yaml
charts:
  # Install cert-manager first (for TLS)
  - name: cert-manager
    repo: https://charts.jetstack.io
    chart: cert-manager
    namespace: cert-manager
    version: "1.14.0"
    phase: pre-crossplane
    wait: true
    values:
      installCRDs: true

  # Install External Secrets Operator after providers
  - name: external-secrets
    repo: https://charts.external-secrets.io
    chart: external-secrets
    namespace: external-secrets
    version: "0.9.11"
    phase: post-providers
    values:
      installCRDs: true

  # Install ingress controller in final phase
  - name: ingress-nginx
    repo: https://kubernetes.github.io/ingress-nginx
    chart: ingress-nginx
    namespace: ingress-nginx
    version: "4.9.0"
    phase: final
    wait: true
    timeout: 10m
    values:
      controller:
        replicaCount: 1
        nodeSelector:
          ingress-ready: "true"
        tolerations:
          - key: node-role.kubernetes.io/control-plane
            operator: Equal
            effect: NoSchedule
        service:
          type: NodePort

  # Install monitoring stack
  - name: prometheus
    repo: https://prometheus-community.github.io/helm-charts
    chart: kube-prometheus-stack
    namespace: monitoring
    phase: final
    valuesFiles:
      - ./values/prometheus.yaml
```
