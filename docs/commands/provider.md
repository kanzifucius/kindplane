# kindplane provider

Manage Crossplane providers.

## Usage

```bash
kindplane provider <command> [flags]
```

## Subcommands

| Command | Description |
|---------|-------------|
| `list` | List installed providers |
| `add` | Add a new provider |

## kindplane provider list

![kindplane provider list demo](../assets/vhs/provider-list.gif)

List all installed Crossplane providers.

### Usage

```bash
kindplane provider list
```

### Output

```
╭────────────────────────────────────────────────────────────────╮
│  Installed Providers                                           │
├────────────────────────────────────────────────────────────────┤
│                                                                │
│  Name                   Package                        Status  │
│  ────                   ───────                        ──────  │
│  provider-aws           upbound/provider-aws:v1.1.0    Healthy │
│  provider-kubernetes    crossplane-contrib/...         Healthy │
│                                                                │
╰────────────────────────────────────────────────────────────────╯
```

## kindplane provider add

Add a new Crossplane provider to the cluster.

### Usage

```bash
kindplane provider add <name> <package>
```

### Arguments

| Argument | Description |
|----------|-------------|
| `name` | Unique name for the provider |
| `package` | Full OCI package URL with version |

### Examples

#### Add AWS Provider

```bash
kindplane provider add provider-aws xpkg.upbound.io/upbound/provider-aws:v1.1.0
```

#### Add GCP Provider

```bash
kindplane provider add provider-gcp xpkg.upbound.io/upbound/provider-gcp:v1.0.0
```

#### Add Kubernetes Provider

```bash
kindplane provider add provider-kubernetes xpkg.upbound.io/crossplane-contrib/provider-kubernetes:v0.12.0
```

### Output

```
→ Installing provider 'provider-gcp'...
  Package: xpkg.upbound.io/upbound/provider-gcp:v1.0.0
  ✓ Provider installed
  → Waiting for provider to become healthy...
  ✓ Provider is healthy
```

## Provider Package URLs

Provider packages follow OCI registry format:

```
xpkg.upbound.io/<namespace>/<name>:<version>
```

Examples:

- `xpkg.upbound.io/upbound/provider-aws:v1.1.0`
- `xpkg.upbound.io/upbound/provider-azure:v1.0.0`
- `xpkg.upbound.io/crossplane-contrib/provider-kubernetes:v0.12.0`

## Using kubectl

You can also manage providers with kubectl:

### List Providers

```bash
kubectl get providers
```

### Add Provider

```bash
cat <<EOF | kubectl apply -f -
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-gcp
spec:
  package: xpkg.upbound.io/upbound/provider-gcp:v1.0.0
EOF
```

### Delete Provider

```bash
kubectl delete provider provider-gcp
```

## Troubleshooting

### Provider Not Healthy

Check provider status:

```bash
kubectl describe provider <name>
```

Check provider pod logs:

```bash
kubectl logs -n crossplane-system -l pkg.crossplane.io/provider=<name>
```

### Package Not Found

Verify the package URL is correct and accessible:

```bash
# Check if package exists (requires crane CLI)
crane manifest xpkg.upbound.io/upbound/provider-aws:v1.1.0
```
