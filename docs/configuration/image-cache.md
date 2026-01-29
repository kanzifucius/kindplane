# Image Cache Configuration

Speed up cluster bootstrapping by pre-loading images from your local Docker daemon.

## Overview

By default, kindplane automatically checks your local Docker daemon for Crossplane and provider images and pre-loads them into the Kind cluster. This eliminates slow registry pulls during bootstrap.

## How It Works

1. **Image Discovery** - kindplane analyzes your configuration to determine which images are needed
2. **Local Check** - Checks if images exist in your local Docker daemon
3. **Pre-loading** - Loads found images using one of two modes:
   - **Registry mode**: Pushes to local registry (if `cluster.registry.enabled: true`)
   - **Direct mode**: Loads directly into Kind nodes

## Interactive Pull Prompt

When running in a terminal (TTY mode), if kindplane detects that **none** of the expected images are available locally, it will display the missing images and prompt you to pull them:

```
No local images found. The following images can be pulled for faster future bootstraps:

  - crossplane/crossplane:v1.15.0
  - crossplane/crossplane-rbac-manager:v1.15.0
  - xpkg.upbound.io/upbound/provider-aws-controller:v1.1.0
  - xpkg.upbound.io/upbound/provider-aws:v1.1.0

Pull 4 images now? [y/N]
```

If you confirm, kindplane will:
1. Pull all missing images from their registries
2. Load them into the Kind cluster immediately
3. Cache them locally for future runs

### Non-Interactive Mode

In CI/CD environments or when output is piped (non-TTY), the prompt is skipped and kindplane continues normally, letting Kubernetes pull images as needed.

### Auto-Pull Flag

To automatically pull missing images without prompting, use the `--pull-images` flag:

```bash
kindplane up --pull-images
```

This is useful for:
- Automated scripts where user input isn't available
- Ensuring images are always cached locally
- First-time setups where you know images aren't present

**Example in CI:**
```yaml
# .github/workflows/test.yml
- name: Bootstrap cluster with auto-pull
  run: kindplane up --pull-images
```

**Behavior Matrix:**

| Environment | Flag | Behavior |
|-------------|------|----------|
| TTY (interactive) | Not set | Show images, prompt user |
| TTY (interactive) | `--pull-images` | Auto-pull, no prompt |
| Non-TTY (CI) | Not set | Skip silently, continue |
| Non-TTY (CI) | `--pull-images` | Auto-pull, no prompt |


## Configuration

### Basic Configuration

Image caching is **enabled by default**. To disable:

```yaml
crossplane:
  imageCache:
    enabled: false
```

### Full Configuration

```yaml
crossplane:
  version: "1.15.0"
  
  imageCache:
    # Enable/disable image caching (default: true)
    enabled: true
    
    # Pre-load provider images (default: true)
    preloadProviders: true
    
    # Pre-load Crossplane core images (default: true)
    preloadCrossplane: true
    
    # Additional images to pre-load
    additionalImages:
      - "xpkg.upbound.io/crossplane-contrib/function-patch-and-transform:v0.2.0"
      - "my-registry.io/custom-function:v1.0.0"
    
    # Override image names for non-standard providers
    imageOverrides:
      "my-registry.io/custom/provider:v1.0.0":
        - "my-registry.io/custom/provider-controller:v1.0.0"
        - "my-registry.io/custom/provider-webhook:v1.0.0"
```

## Image Derivation

kindplane uses naming conventions to automatically derive controller image names from provider packages.

### Convention Rules

**Provider Packages:**
```
Package: xpkg.upbound.io/upbound/provider-aws:v1.1.0

Derived Images:
  1. xpkg.upbound.io/upbound/provider-aws:v1.1.0 (the xpkg package)
  2. xpkg.upbound.io/upbound/provider-aws-controller:v1.1.0 (controller)
```

**Crossplane Core:**
```
Version: "1.15.0"

Derived Images:
  1. crossplane/crossplane:v1.15.0
  2. crossplane/crossplane-rbac-manager:v1.15.0
```

### Image Overrides

For providers that don't follow standard naming:

```yaml
crossplane:
  imageCache:
    imageOverrides:
      # Key: provider package
      # Value: list of actual container images
      "registry.example.com/team/custom-provider:v2.0.0":
        - "registry.example.com/team/custom-provider-main:v2.0.0"
        - "registry.example.com/team/custom-provider-webhook:v2.0.0"
```

## Pre-loading Modes

### Direct Mode (Default)

When `cluster.registry.enabled: false`, images are loaded directly into Kind nodes.

**Workflow:**
```
Local Docker → docker save → Load into Kind nodes
```

**Benefits:**
- Simple and fast
- No registry overhead
- Works immediately

**Example:**
```yaml
cluster:
  name: kindplane-dev

crossplane:
  imageCache:
    enabled: true  # Uses direct mode
```

### Registry Mode

When `cluster.registry.enabled: true`, images are pushed to the local registry.

**Workflow:**
```
Local Docker → Tag for registry → Push to localhost:5001 → Kind pulls from registry
```

**Benefits:**
- Registry acts as cache for all nodes
- Images persist (with `persistent: true`)
- Efficient for multi-node clusters

**Example:**
```yaml
cluster:
  name: kindplane-dev
  registry:
    enabled: true
    port: 5001
    persistent: true

crossplane:
  imageCache:
    enabled: true  # Uses registry mode
```

## Use Cases

### Development Workflow

Pre-pull images once, iterate quickly:

```bash
# Pull images once
docker pull crossplane/crossplane:v1.15.0
docker pull xpkg.upbound.io/crossplane-contrib/provider-kubernetes-controller:v0.12.0

# Fast iterations
kindplane down
kindplane up  # Fast! Uses cached images
```

### CI/CD Pipelines

Cache images in CI environment:

```yaml
# .github/workflows/test.yml
- name: Cache Crossplane images
  run: |
    docker pull crossplane/crossplane:v1.15.0
    docker pull xpkg.upbound.io/upbound/provider-aws-controller:v1.1.0

- name: Bootstrap cluster
  run: kindplane up  # Uses cached images
```

### Air-gapped Environments

Pre-load all images offline:

```bash
# On connected machine
docker pull crossplane/crossplane:v1.15.0
docker save crossplane/crossplane:v1.15.0 > crossplane.tar

# Transfer to air-gapped machine
docker load < crossplane.tar

# Bootstrap uses loaded image
kindplane up
```

## Troubleshooting

### Images Not Being Pre-loaded

Check if images exist locally:
```bash
docker images | grep crossplane
docker images | grep provider
```

Watch bootstrap output for pre-loading messages:
```bash
kindplane up
```

### Registry Push Failures

If using registry mode and pushes fail:

```bash
# Check registry is running
docker ps | grep kind-registry

# Test registry connectivity
curl http://localhost:5001/v2/_catalog

# Check registry logs
docker logs kind-registry
```

### Performance Not Improved

Verify images are actually being loaded:

```bash
# Before bootstrap
docker images --format "{{.Repository}}:{{.Tag}}" | grep -E "crossplane|provider"

# Watch for pre-loading during bootstrap
kindplane up 2>&1 | grep -i "pre-load"
```

## Best Practices

1. **Keep Images Updated** - Pull latest images before bootstrapping:
   ```bash
   docker pull crossplane/crossplane:v1.15.0
   ```

2. **Use Persistent Registry** - Preserve images across cluster recreations:
   ```yaml
   cluster:
     registry:
       persistent: true
   ```

3. **Pre-load Large Providers** - Focus on large provider images that take time to pull:
   ```bash
   docker pull xpkg.upbound.io/upbound/provider-aws-controller:v1.1.0
   ```

4. **Clean Up Periodically** - Remove unused images:
   ```bash
   docker image prune -a
   ```

## Options Reference

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | true | Enable/disable image caching |
| `preloadProviders` | bool | true | Pre-load provider images |
| `preloadCrossplane` | bool | true | Pre-load Crossplane core images |
| `additionalImages` | array | [] | Extra images to pre-load |
| `imageOverrides` | object | {} | Map packages to actual images |

## See Also

- [Local Registry Guide](../guides/local-registry.md) - Using the local registry feature
- [Crossplane Configuration](./crossplane.md) - Crossplane configuration options
- [Up Command](../commands/up.md) - Bootstrap command reference
