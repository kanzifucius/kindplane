# Troubleshooting

This guide helps you diagnose and fix common issues with kindplane.

## Diagnostic Commands

### Pre-flight Checks

Before bootstrapping, verify your system is ready:

```bash
kindplane doctor
```

### Check Cluster Status

```bash
kindplane status
```

### Run Diagnostics

```bash
kindplane diagnostics
kindplane diagnostics --component providers
kindplane diagnostics --component crossplane
```

### Stream Logs

```bash
kindplane logs
kindplane logs --component providers --follow
```

### Check Providers

```bash
kubectl get providers
kubectl describe provider <provider-name>
```

### Check Provider Logs

```bash
kindplane logs --component providers
# Or via kubectl:
kubectl logs -n crossplane-system -l pkg.crossplane.io/provider=<provider-name>
```

### Check Crossplane Logs

```bash
kindplane logs --component crossplane
# Or via kubectl:
kubectl logs -n crossplane-system -l app=crossplane
```

### Check All Pods

```bash
kubectl get pods -A
```

## Common Issues

### Docker Not Running

**Symptom:**

```
✗ Docker is not running or not installed
```

**Solution:**

1. Start Docker Desktop
2. Or start Docker daemon: `sudo systemctl start docker`
3. Verify: `docker version`

### Cluster Already Exists

**Symptom:**

```
✗ Cluster 'kindplane-dev' already exists
```

**Solution:**

```bash
kindplane down --force
kindplane up
```

### Provider Not Healthy

**Symptom:**

```
NAME            INSTALLED   HEALTHY   AGE
provider-aws    True        False     5m
```

**Diagnosis:**

```bash
kubectl describe provider provider-aws
kubectl logs -n crossplane-system -l pkg.crossplane.io/provider=provider-aws
```

**Common Causes:**

1. **Invalid package URL** - Check the package exists
2. **Missing credentials** - Configure ProviderConfig
3. **Network issues** - Check Docker network settings
4. **Resource limits** - Increase Docker memory

### Credentials Not Working

**Symptom:**

Provider logs show authentication errors.

**Solution:**

1. Verify credentials are valid:

    ```bash
    # AWS
    aws sts get-caller-identity

    # Azure
    az account show
    ```

2. Check secret exists:

    ```bash
    kubectl get secrets -n crossplane-system
    ```

3. Reconfigure:

    ```bash
    kindplane credentials configure
    ```

### Resources Stuck in Creating

**Symptom:**

```bash
kubectl get managed
NAME           READY   SYNCED   AGE
my-bucket      False   True     10m
```

**Diagnosis:**

```bash
kubectl describe <resource-type> <resource-name>
```

Look at `Status.Conditions` for error messages.

**Common Causes:**

1. Missing permissions in cloud account
2. Invalid resource configuration
3. Quota limits reached

### Helm Chart Installation Fails

**Symptom:**

```
✗ Failed to install chart 'my-chart': ...
```

**Diagnosis:**

```bash
helm list -A
kubectl get pods -n <chart-namespace>
```

**Common Causes:**

1. Invalid chart repository URL
2. Chart version doesn't exist
3. Missing dependencies
4. Namespace doesn't exist

### Timeout During Bootstrap

**Symptom:**

```
✗ Providers failed to become healthy: context deadline exceeded
```

**Solution:**

1. Increase timeout:

    ```bash
    kindplane up --timeout 20m
    ```

2. Check network connectivity
3. Increase Docker resources

### Composition Not Taking Effect

**Symptom:**

Claims don't create managed resources.

**Diagnosis:**

```bash
kubectl describe composition <composition-name>
kubectl describe xrd <xrd-name>
kubectl get events
```

**Common Causes:**

1. Composition doesn't match XRD
2. Missing patches
3. Invalid field paths

## Network Issues

### Cannot Pull Images

**Symptom:**

Pods stuck in `ImagePullBackOff`.

**Solution:**

1. Check Docker network:

    ```bash
    docker network inspect kind
    ```

2. Check DNS resolution:

    ```bash
    docker run --rm alpine nslookup xpkg.upbound.io
    ```

3. Check corporate proxy settings

### Kind Network Issues

**Symptom:**

Services not accessible from host.

**Solution:**

Verify port mappings in configuration:

```yaml
cluster:
  portMappings:
    - containerPort: 80
      hostPort: 8080
      protocol: TCP
```

## Resource Issues

### Insufficient Memory

**Symptom:**

Pods in `OOMKilled` or pending state.

**Solution:**

1. Increase Docker Desktop memory (8GB+ recommended)
2. Reduce number of workers:

    ```yaml
    cluster:
      nodes:
        workers: 1
    ```

3. Use family providers instead of monolithic

### Disk Space

**Symptom:**

```
no space left on device
```

**Solution:**

```bash
# Clean Docker resources
docker system prune -a

# Remove unused Kind clusters
kind get clusters
kind delete cluster --name <unused-cluster>
```

## Debugging Tips

### Run Doctor First

Before troubleshooting, check system requirements:

```bash
kindplane doctor
```

### Enable Verbose Output

```bash
kindplane up --verbose
```

### Run Diagnostics

```bash
kindplane diagnostics
```

### Watch Resources

```bash
watch kubectl get providers,managed
```

### Get Events

```bash
kubectl get events --sort-by='.lastTimestamp'
```

### Export Diagnostics

```bash
# Use kindplane diagnostics
kindplane diagnostics > diagnostics-report.txt

# Or create a manual diagnostics bundle
mkdir -p diagnostics
kubectl get providers -o yaml > diagnostics/providers.yaml
kubectl get pods -A -o yaml > diagnostics/pods.yaml
kindplane logs --component crossplane --tail 500 > diagnostics/crossplane.log
kindplane logs --component providers --tail 500 > diagnostics/providers.log
```

## Getting Help

If you're still stuck:

1. Check the [GitHub Issues](https://github.com/kanzifucius/kindplane/issues)
2. Search for similar problems
3. Open a new issue with:
    - kindplane version
    - Configuration file (redact secrets)
    - Full error output
    - Steps to reproduce
