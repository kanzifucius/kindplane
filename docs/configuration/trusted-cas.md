# Trusted CA Certificates

Configure trusted CA certificates for private container registries and workloads.

## Overview

The `trustedCAs` section allows you to:

1. **Trust private container registries** - Configure containerd to trust custom CA certificates when pulling images from private registries
2. **Provide CA certificates to workloads** - Mount CA certificates into Kind nodes so applications can trust custom CAs

## Configuration

```yaml
cluster:
  trustedCAs:
    registries:
      - host: "registry.example.com:5000"
        caFile: "./certs/registry-ca.crt"
    workloads:
      - name: "corporate-root-ca"
        caFile: "./certs/corporate-ca.crt"
```

## Registry CAs

Configure CA certificates for private container registries. These are trusted by containerd, enabling you to pull images from registries using custom certificates.

### Options

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `host` | string | Yes | Registry host (e.g., "registry.example.com:5000") |
| `caFile` | string | Yes | Path to CA certificate file on the host |

### Example

```yaml
cluster:
  trustedCAs:
    registries:
      # Single registry
      - host: "harbor.mycompany.com"
        caFile: "./certs/harbor-ca.crt"
      
      # Registry with custom port
      - host: "registry.internal.com:5000"
        caFile: "/etc/ssl/certs/internal-registry.crt"
      
      # Wildcard for multiple subdomains
      - host: "*.container-registry.internal"
        caFile: "./certs/wildcard-registry-ca.crt"
```

### How It Works

When you configure registry CAs, kindplane:

1. Mounts each CA certificate file into the Kind nodes at `/etc/containerd/certs.d/<host>/ca.crt`
2. Adds containerd configuration patches to trust the CA for the specified registry host

This allows the cluster to pull images from private registries without disabling TLS verification.

## Workload CAs

Configure CA certificates that are mounted into Kind nodes for applications to use. These are useful when your workloads need to communicate with services using custom CA certificates.

### Options

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Identifier for the CA (used in the mount path) |
| `caFile` | string | Yes | Path to CA certificate file on the host |

### Example

```yaml
cluster:
  trustedCAs:
    workloads:
      # Corporate root CA
      - name: "corporate-root-ca"
        caFile: "./certs/corporate-ca.crt"
      
      # Internal services CA
      - name: "internal-services-ca"
        caFile: "/path/to/internal-ca.crt"
```

### How It Works

When you configure workload CAs, kindplane:

1. Mounts each CA certificate file into the Kind nodes at `/etc/ssl/certs/extra/<name>.crt`
2. Applications running in the cluster can then reference these certificates

!!! note "Application Configuration Required"
    Mounting the CA certificates makes them available on the nodes, but applications may need additional configuration to use them. Common approaches include:
    
    - Setting the `SSL_CERT_DIR` or `SSL_CERT_FILE` environment variables
    - Configuring application-specific CA bundle paths
    - Using init containers to update the system CA bundle

## Complete Example

```yaml
cluster:
  name: dev-cluster
  kubernetesVersion: "1.29.0"
  
  trustedCAs:
    # Private container registries
    registries:
      - host: "harbor.mycompany.com"
        caFile: "./certs/harbor-ca.crt"
      - host: "gcr.internal.mycompany.com"
        caFile: "./certs/gcr-mirror-ca.crt"
    
    # CAs for workload communication
    workloads:
      - name: "corporate-root-ca"
        caFile: "./certs/corporate-root-ca.crt"
      - name: "vault-ca"
        caFile: "./certs/vault-ca.crt"
  
  nodes:
    controlPlane: 1
    workers: 1
```

## Certificate File Requirements

- Certificate files must be in PEM format
- Paths can be absolute or relative to the working directory
- Files must exist when running `kindplane up` (validated during configuration load)
- Files are mounted as read-only into the Kind nodes

## Troubleshooting

### Image Pull Errors

If you see errors like `x509: certificate signed by unknown authority` when pulling images:

1. Verify the CA certificate file exists and is readable
2. Check the registry host matches exactly (including port if specified)
3. Ensure the CA certificate is the correct one for the registry

### Viewing Mounted Certificates

To verify certificates are mounted correctly:

```bash
# Check registry CAs
docker exec -it <node-container> ls -la /etc/containerd/certs.d/

# Check workload CAs
docker exec -it <node-container> ls -la /etc/ssl/certs/extra/
```

### Containerd Configuration

To view the containerd configuration patches applied:

```bash
docker exec -it <node-container> cat /etc/containerd/config.toml
```
