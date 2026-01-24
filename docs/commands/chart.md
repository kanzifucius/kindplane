# kindplane chart

Manage Helm charts.

## Usage

```bash
kindplane chart <command> [flags]
```

## Subcommands

| Command | Description |
|---------|-------------|
| `list` | List installed charts |
| `install` | Install a Helm chart |
| `uninstall` | Uninstall a Helm chart |

## kindplane chart list

![kindplane chart list demo](../assets/vhs/chart-list.gif)

List all Helm releases in the cluster.

### Usage

```bash
kindplane chart list
```

### Output

```
╭────────────────────────────────────────────────────────────────╮
│  Installed Helm Charts                                         │
├────────────────────────────────────────────────────────────────┤
│                                                                │
│  Name            Namespace        Version    Status            │
│  ────            ─────────        ───────    ──────            │
│  crossplane      crossplane-sys   1.15.0     deployed          │
│  cert-manager    cert-manager     1.14.0     deployed          │
│  ingress-nginx   ingress-nginx    4.9.0      deployed          │
│                                                                │
╰────────────────────────────────────────────────────────────────╯
```

## kindplane chart install

![kindplane chart install demo](../assets/vhs/chart-install.gif)

Install a Helm chart to the cluster.

### Usage

```bash
kindplane chart install <name> <repo> <chart> [flags]
```

### Arguments

| Argument | Description |
|----------|-------------|
| `name` | Release name for the installation |
| `repo` | Helm repository URL |
| `chart` | Chart name in the repository |

### Flags

| Flag | Description |
|------|-------------|
| `--namespace`, `-n` | Kubernetes namespace (default: `default`) |
| `--version` | Chart version to install |
| `--wait` | Wait for resources to be ready |
| `--timeout` | Timeout for installation (default: `5m`) |
| `--values`, `-f` | Values file path |
| `--set` | Set values on command line |

### Examples

#### Basic Installation

```bash
kindplane chart install nginx https://kubernetes.github.io/ingress-nginx ingress-nginx
```

#### With Namespace and Version

```bash
kindplane chart install nginx https://kubernetes.github.io/ingress-nginx ingress-nginx \
  --namespace ingress-nginx \
  --version 4.9.0
```

#### With Values File

```bash
kindplane chart install nginx https://kubernetes.github.io/ingress-nginx ingress-nginx \
  --namespace ingress-nginx \
  --values ./values/nginx.yaml
```

#### With Inline Values

```bash
kindplane chart install nginx https://kubernetes.github.io/ingress-nginx ingress-nginx \
  --set controller.replicaCount=2 \
  --set controller.service.type=ClusterIP
```

#### Wait for Ready

```bash
kindplane chart install nginx https://kubernetes.github.io/ingress-nginx ingress-nginx \
  --namespace ingress-nginx \
  --wait \
  --timeout 10m
```

## kindplane chart uninstall

![kindplane chart uninstall demo](../assets/vhs/chart-uninstall.gif)

Uninstall a Helm release from the cluster.

### Usage

```bash
kindplane chart uninstall <name> <namespace>
```

### Arguments

| Argument | Description |
|----------|-------------|
| `name` | Release name to uninstall |
| `namespace` | Namespace where the release is installed |

### Examples

```bash
kindplane chart uninstall nginx ingress-nginx
```

### Output

```
→ Uninstalling 'nginx' from namespace 'ingress-nginx'...
✓ Release 'nginx' uninstalled
```

## Using Helm Directly

You can also use the Helm CLI:

```bash
# List releases
helm list -A

# Install
helm install nginx ingress-nginx/ingress-nginx -n ingress-nginx

# Uninstall
helm uninstall nginx -n ingress-nginx
```

## Common Charts

### Ingress Controller

```bash
kindplane chart install ingress-nginx https://kubernetes.github.io/ingress-nginx ingress-nginx \
  --namespace ingress-nginx \
  --version 4.9.0
```

### Cert-Manager

```bash
kindplane chart install cert-manager https://charts.jetstack.io cert-manager \
  --namespace cert-manager \
  --version 1.14.0 \
  --set installCRDs=true
```

### Prometheus Stack

```bash
kindplane chart install prometheus https://prometheus-community.github.io/helm-charts kube-prometheus-stack \
  --namespace monitoring
```
