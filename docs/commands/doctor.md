# kindplane doctor

![kindplane doctor demo](../assets/vhs/doctor.gif)

Check system requirements and prerequisites.

## Usage

```bash
kindplane doctor [flags]
```

## Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--quiet` | `-q` | Only show failures |
| `--timeout` | | Timeout for checks (default: `30s`) |

## Description

The `doctor` command runs pre-flight checks to verify that your system meets all requirements for running kindplane.

This command checks:

- Docker daemon is running
- Required binaries are available (kind, kubectl)
- Sufficient disk space
- Optional tools (helm)
- Cluster connectivity (if a cluster exists)
- Crossplane installation status

## Examples

### Run All Checks

```bash
kindplane doctor
```

### Quiet Mode

Only show failures:

```bash
kindplane doctor --quiet
```

### Custom Timeout

```bash
kindplane doctor --timeout 60s
```

## Output

### All Checks Passing

```
 kindplane doctor
--------------------------------------------------

  ✓ Docker: Docker daemon is running
    Docker version 24.0.7
  ✓ kubectl: kubectl is installed
    Client Version: v1.29.0
  ✓ Disk Space: Sufficient disk space available
    20 GB free
  ✓ Helm: Helm is available (optional)
    v3.14.0

╭────────────────────────────────────────────────╮
│  All Checks Passed                              │
│  All 4 checks passed! Your system is ready.    │
╰────────────────────────────────────────────────╯
```

### With Failures

```
 kindplane doctor
--------------------------------------------------

  ✓ Docker: Docker daemon is running
  ✗ kubectl: kubectl is not installed
    → Install kubectl: https://kubernetes.io/docs/tasks/tools/
  ✓ Disk Space: Sufficient disk space available
  ! Helm: Helm is not installed (optional)
    → Install Helm for chart management: https://helm.sh/docs/intro/install/

╭────────────────────────────────────────────────╮
│  Checks Failed                                  │
│  3/4 checks passed (1 failure)                 │
╰────────────────────────────────────────────────╯

Please fix the required issues before running kindplane.
```

## Check Types

| Icon | Type | Description |
|------|------|-------------|
| ✓ | Passed | Check passed successfully |
| ✗ | Failed (Required) | Required check failed - must fix |
| ! | Warning (Optional) | Optional check failed - can continue |

## When to Use

Run `kindplane doctor` before:

- First time setup
- After system updates
- When troubleshooting issues
- Before CI/CD pipeline runs

## CI/CD Usage

```bash
# Fail pipeline if requirements not met
kindplane doctor || exit 1

# Continue with bootstrap
kindplane up
```

## Tips

### Before Initial Setup

Always run doctor before your first `kindplane up`:

```bash
kindplane doctor
kindplane init
kindplane up
```

### In Scripts

Use quiet mode with exit code:

```bash
if kindplane doctor --quiet; then
    echo "System ready"
else
    echo "Please fix issues before continuing"
    exit 1
fi
```
