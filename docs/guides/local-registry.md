# Local Registry Guide

This guide explains how to use kindplane's built-in local container registry for faster development iteration.

## Overview

When developing with Kind, you often need to test locally built container images. By default, this requires pushing images to a remote registry, which can be slow. kindplane's local registry feature solves this by creating a local Docker registry that Kind nodes can pull from directly.

## Enabling the Local Registry

Add the `registry` configuration to your `kindplane.yaml`:

```yaml
cluster:
  name: kindplane-dev
  registry:
    enabled: true
    port: 5001
```

## Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | false | Enable local container registry |
| `port` | int | 5001 | Host port for the registry |
| `persistent` | bool | false | Keep registry container after `kindplane down` |
| `name` | string | kind-registry | Registry container name |

### Persistent Mode

By default, the registry container is removed when you run `kindplane down`. To preserve images across cluster recreations, enable persistent mode:

```yaml
cluster:
  registry:
    enabled: true
    persistent: true
```

This is useful when:

- You have large images that take time to build
- You want to preserve images between development sessions
- You're testing multiple cluster configurations

## Usage Workflow

### 1. Create the Cluster

```bash
kindplane up
```

This creates both the Kind cluster and the local registry.

### 2. Build and Tag Your Image

Tag your image to use the local registry:

```bash
# Build your image
docker build -t my-app:latest .

# Tag for the local registry
docker tag my-app:latest localhost:5001/my-app:latest
```

Or build directly with the registry tag:

```bash
docker build -t localhost:5001/my-app:latest .
```

### 3. Push to the Registry

```bash
docker push localhost:5001/my-app:latest
```

### 4. Use in Kubernetes

Reference the image in your Kubernetes manifests:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
        - name: my-app
          image: localhost:5001/my-app:latest
          imagePullPolicy: Always
```

Or use kubectl directly:

```bash
kubectl create deployment my-app --image=localhost:5001/my-app:latest
```

## How It Works

When you enable the local registry, kindplane:

1. **Creates a registry container** - Runs the `registry:2` Docker image bound to `localhost:{port}`

2. **Configures containerd** - Patches the Kind node's containerd configuration to recognise the registry

3. **Sets up hosts.toml** - Creates registry configuration on each Kind node to map `localhost:{port}` to the registry container

4. **Connects to Kind network** - Ensures the registry container is on the same Docker network as Kind nodes

5. **Creates a ConfigMap** - Adds `local-registry-hosting` ConfigMap to `kube-public` namespace for discovery

## Accessing from Pods

There's an important distinction between host access and in-cluster access:

| Context | Address | Protocol |
|---------|---------|----------|
| Host machine | `localhost:5001` | HTTP |
| Pod manifests | `localhost:5001` | HTTP (via containerd mapping) |
| Inside pods (code) | `kind-registry:5000` | HTTP |

!!! note "Why Different Addresses?"
    `localhost` inside a container refers to the container's own network namespace, not the host. The containerd configuration maps `localhost:5001` to the registry for image pulls, but code running inside pods must use the registry's Docker network name.

## Troubleshooting

### Check Registry Status

```bash
# Check if registry container is running
docker ps | grep kind-registry

# Check registry connectivity
curl http://localhost:5001/v2/_catalog
```

### Check Node Configuration

```bash
# List Kind nodes
docker exec kind-control-plane cat /etc/containerd/certs.d/localhost:5001/hosts.toml
```

### Check ConfigMap

```bash
kubectl get configmap local-registry-hosting -n kube-public -o yaml
```

### Image Pull Errors

If pods fail to pull images:

1. Verify the image was pushed successfully:
   ```bash
   curl http://localhost:5001/v2/my-app/tags/list
   ```

2. Check the image reference matches exactly (including tag)

3. Ensure `imagePullPolicy: Always` or `imagePullPolicy: IfNotPresent` is set appropriately

### Registry Container Missing

If the registry container was accidentally removed:

```bash
# Recreate by running up again
kindplane down --force
kindplane up
```

Or manually create it:

```bash
docker run -d --restart=always -p 127.0.0.1:5001:5000 --network bridge --name kind-registry registry:2
docker network connect kind kind-registry
```

## Best Practices

1. **Use consistent tagging** - Always include version tags, avoid using just `latest` in production
   
2. **Enable persistent mode** - If you're iterating on images frequently, enable `persistent: true` to avoid re-pushing after cluster recreation

3. **Clean up old images** - The registry can grow large over time. Consider periodically cleaning up:
   ```bash
   # List all images
   curl http://localhost:5001/v2/_catalog
   
   # To fully clean, remove and recreate the registry
   docker rm -f kind-registry
   ```

4. **Use in CI/CD** - The local registry is ideal for CI pipelines that need to test images in Kubernetes without pushing to external registries
