# kindplane validate

![kindplane validate demo](../assets/vhs/validate.gif)

Validate a configuration file without creating a cluster.

## Usage

```bash
kindplane validate [flags]
```

## Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-c` | Configuration file to validate (default: `kindplane.yaml`) |

## Description

The `validate` command checks your configuration file for errors without creating a cluster. This is useful for:

- Verifying configuration before bootstrap
- CI/CD pipelines
- Catching errors early

## Validation Checks

The command validates:

- YAML syntax
- Required fields are present
- Field types are correct
- Provider package URLs are valid
- Chart configurations are complete
- Composition sources exist (for local paths)

## Examples

### Validate Default Configuration

```bash
kindplane validate
```

Validates `kindplane.yaml` in the current directory.

### Validate Specific File

```bash
kindplane validate --config production.yaml
```

### CI/CD Usage

```bash
# Exit code 0 on success, non-zero on failure
kindplane validate --config kindplane.yaml || exit 1
```

## Output

### Success

```
✓ Configuration is valid
```

### Failure

```
✗ Configuration validation failed

Errors:
  - crossplane.version: required field missing
  - providers[0].package: invalid OCI package URL
  - charts[2].repo: required field missing
```

## Common Validation Errors

### Missing Required Field

```
✗ cluster.name: required field missing
```

Fix: Add the required field to your configuration.

### Invalid Provider Package

```
✗ providers[0].package: invalid OCI package URL
```

Fix: Ensure the package URL follows the format: `registry/namespace/name:version`

### Invalid Chart Phase

```
✗ charts[0].phase: must be one of: pre-crossplane, post-crossplane, post-providers, post-eso
```

Fix: Use a valid phase value.

### Local Path Not Found

```
✗ compositions.sources[0].path: directory does not exist: ./compositions
```

Fix: Create the directory or update the path.
