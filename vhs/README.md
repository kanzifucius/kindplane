# VHS Screen Recordings

This directory contains VHS tape files for generating GIF screen recordings of kindplane commands.

## Prerequisites

1. Install VHS: https://github.com/charmbracelet/vhs
2. Ensure `kindplane` binary is built and available in PATH (or use `task build` first)
3. For cluster-dependent commands, ensure a cluster is running (run `kindplane up` first)

## Generating Recordings

### Generate All Recordings

```bash
task vhs:all
```

This will generate all 20 GIF files in `docs/assets/vhs/`.

### Generate Single Recording

```bash
task vhs:single TAPE=init
```

Replace `init` with any tape filename (without `.tape` extension).

### Clean Generated GIFs

```bash
task vhs:clean
```

## Recording Order and Prerequisites

The tape files include prerequisite setup where needed. Commands are ordered as follows:

### Standalone Commands (No Prerequisites)
- `doctor` - Checks system prerequisites
- `init` - Creates config file (prerequisite for others)
- `cluster-list` - Lists Kind clusters (doesn't require kindplane cluster)

### Commands Requiring Config File
These commands run `init` first (hidden) to ensure config exists:
- `validate` - Validates config file
- `up` - Creates cluster (also creates config if missing)
- `config-show` - Shows current config
- `config-diff` - Compares config files

### Commands Requiring Running Cluster
These commands assume a cluster exists (run `kindplane up` first):
- `status` - Shows cluster status
- `logs` - Streams component logs
- `diagnostics` - Runs diagnostics
- `apply` - Applies resources
- `dump` - Exports resources
- `down` - Deletes cluster
- `chart-install`, `chart-upgrade`, `chart-list`, `chart-uninstall` - Helm chart management
- `provider-add`, `provider-list`, `provider-remove` - Provider management
- `credentials-configure`, `credentials-list` - Credential management

### Recommended Recording Order

1. **First** (no prerequisites):
   ```bash
   task vhs:single TAPE=doctor
   task vhs:single TAPE=init
   task vhs:single TAPE=cluster-list
   ```

2. **After init** (config file exists):
   ```bash
   task vhs:single TAPE=validate
   task vhs:single TAPE=config-show
   task vhs:single TAPE=config-diff
   ```

3. **Create cluster** (this takes longest):
   ```bash
   task vhs:single TAPE=up
   ```

4. **After cluster exists** (all remaining commands):
   ```bash
   task vhs:single TAPE=status
   task vhs:single TAPE=logs
   task vhs:single TAPE=diagnostics
   task vhs:single TAPE=apply
   task vhs:single TAPE=dump
   task vhs:single TAPE=chart-list
   task vhs:single TAPE=chart-install
   task vhs:single TAPE=chart-upgrade
   task vhs:single TAPE=chart-uninstall
   task vhs:single TAPE=provider-list
   task vhs:single TAPE=provider-add
   task vhs:single TAPE=provider-remove
   task vhs:single TAPE=credentials-list
   task vhs:single TAPE=credentials-configure
   task vhs:single TAPE=down
   ```

## Tape Files

Each `.tape` file uses the shared configuration from `config.tape` and demonstrates real command execution. You can edit individual tape files to customize timing, add pauses, or modify the demonstration.

## Customization

To customize recordings:

1. Edit `config.tape` to change global settings (theme, size, font, etc.)
2. Edit individual `.tape` files to adjust timing or add additional commands
3. Regenerate using `task vhs:single TAPE=<name>`
