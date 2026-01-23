# Cluster Configuration

The `cluster` section configures the Kind cluster.

## Basic Configuration

```yaml
cluster:
  name: kindplane-dev
  kubernetesVersion: "1.29.0"
```

## Options

### name

The name of the Kind cluster.

```yaml
cluster:
  name: my-cluster
```

- **Type:** string
- **Default:** `kindplane-dev`
- **Required:** Yes

### kubernetesVersion

The Kubernetes version to use.

```yaml
cluster:
  kubernetesVersion: "1.29.0"
```

- **Type:** string
- **Default:** Kind's default version
- **Required:** No

!!! note "Available Versions"
    Check [Kind releases](https://github.com/kubernetes-sigs/kind/releases) for available Kubernetes versions.

### nodes

Configure the number of control plane and worker nodes.

```yaml
cluster:
  nodes:
    controlPlane: 1
    workers: 2
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `controlPlane` | int | 1 | Number of control plane nodes |
| `workers` | int | 1 | Number of worker nodes |

### portMappings

Expose container ports to the host.

```yaml
cluster:
  portMappings:
    - containerPort: 80
      hostPort: 8080
      protocol: TCP
    - containerPort: 443
      hostPort: 8443
      protocol: TCP
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `containerPort` | int | Yes | Port inside the container |
| `hostPort` | int | Yes | Port on the host machine |
| `protocol` | string | No | Protocol (TCP/UDP), defaults to TCP |

### ingress

Configure ingress controller support.

```yaml
cluster:
  ingress:
    enabled: true
```

When enabled, kindplane:

- Adds the `ingress-ready=true` label to nodes
- Configures appropriate port mappings for ingress controllers

### extraMounts

Mount host directories into Kind nodes.

```yaml
cluster:
  extraMounts:
    - hostPath: /tmp/kindplane-data
      containerPath: /data
      readOnly: false
    - hostPath: ~/.aws
      containerPath: /root/.aws
      readOnly: true
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `hostPath` | string | Yes | Path on the host machine |
| `containerPath` | string | Yes | Path inside the container |
| `readOnly` | bool | No | Mount as read-only (default: false) |

### rawConfigPath

Use a raw Kind configuration file as a base.

```yaml
cluster:
  rawConfigPath: ./kind-config.yaml
```

When specified, kindplane:

1. Loads the raw Kind config
2. Merges kindplane settings on top
3. kindplane settings take precedence

## Complete Example

```yaml
cluster:
  name: dev-cluster
  kubernetesVersion: "1.29.0"
  nodes:
    controlPlane: 1
    workers: 3
  portMappings:
    - containerPort: 80
      hostPort: 8080
      protocol: TCP
    - containerPort: 443
      hostPort: 8443
      protocol: TCP
    - containerPort: 30000
      hostPort: 30000
      protocol: TCP
  ingress:
    enabled: true
  extraMounts:
    - hostPath: /tmp/data
      containerPath: /data
      readOnly: false
```

## Multi-Node Clusters

For testing high availability or distributed workloads:

```yaml
cluster:
  name: ha-cluster
  nodes:
    controlPlane: 3
    workers: 5
```

!!! warning "Resource Requirements"
    Multi-node clusters require more resources. Ensure Docker has sufficient CPU and memory allocated.
