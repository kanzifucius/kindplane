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

### registry

Configure a local container registry for development.

```yaml
cluster:
  registry:
    enabled: true
    port: 5001
    persistent: false
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | false | Enable local container registry |
| `port` | int | 5001 | Host port for the registry |
| `persistent` | bool | false | Keep registry container after `kindplane down` |
| `name` | string | kind-registry | Registry container name |

When enabled, kindplane:

1. Creates a local Docker registry container
2. Configures Kind nodes to pull images from the registry
3. Connects the registry to the Kind network
4. Creates a `local-registry-hosting` ConfigMap for discovery

!!! tip "Learn More"
    See [Local Registry Guide](../guides/local-registry.md) for usage examples and workflow.

### trustedCAs

Configure trusted CA certificates for private registries and workloads.

```yaml
cluster:
  trustedCAs:
    registries:
      - host: "harbor.mycompany.com"
        caFile: "./certs/harbor-ca.crt"
    workloads:
      - name: "corporate-root-ca"
        caFile: "./certs/corporate-ca.crt"
```

| Section | Field | Type | Required | Description |
|---------|-------|------|----------|-------------|
| `registries` | `host` | string | Yes | Registry host (e.g., "registry.example.com:5000") |
| `registries` | `caFile` | string | Yes | Path to CA certificate file |
| `workloads` | `name` | string | Yes | Identifier for the CA |
| `workloads` | `caFile` | string | Yes | Path to CA certificate file |

!!! tip "Learn More"
    See [Trusted CAs](trusted-cas.md) for detailed documentation on certificate configuration.

### nodeImage

Specify the full Kind node image path. Use this when your environment requires pulling images through a proxy registry like Artifactory due to network restrictions.

```yaml
cluster:
  nodeImage: "artifactory.example.com/kindest/node:v1.29.0"
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `nodeImage` | string | No | Full Kind node image path. If not specified, defaults to `kindest/node:v<kubernetesVersion>` |

**Default Behaviour:**
- If `nodeImage` is not specified and `kubernetesVersion` is set, Kind will use `kindest/node:v<version>`
- If `nodeImage` is specified, it takes precedence over the default

**Examples:**

Simple proxy registry:
```yaml
cluster:
  nodeImage: "artifactory.example.com/kindest/node:v1.29.0"
```

Docker Hub proxy with path:
```yaml
cluster:
  nodeImage: "artifactory.example.com/docker.io/kindest/node:v1.29.0"
```

Custom registry with port:
```yaml
cluster:
  nodeImage: "registry.example.com:5000/kindest/node:v1.29.0"
```

**Important Notes:**

1. **Certificate Configuration**: If your proxy registry uses custom certificates, ensure it's configured in `trustedCAs.registries`:

```yaml
cluster:
  nodeImage: "artifactory.example.com/kindest/node:v1.29.0"
  trustedCAs:
    registries:
      - host: "artifactory.example.com"
        caFile: "./certs/artifactory-ca.crt"
```

2. **Image Availability**: Ensure the proxy registry has the required Kind node images available, or that it can proxy requests to Docker Hub.

3. **Version Mismatch**: If you specify `nodeImage` manually, ensure the image version matches your `kubernetesVersion` (if specified) or your intended Kubernetes version.

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
