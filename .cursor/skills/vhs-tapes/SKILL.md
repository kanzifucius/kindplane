---
name: vhs-tapes
description: Create VHS tape files for kindplane command screen recordings. Use when creating new CLI commands, subcommands, or when adding documentation GIFs. Works alongside the kindplane-commands skill.
---

# Creating VHS Tape Files

## Quick Reference

When creating a new command, also create a corresponding VHS tape file for documentation GIFs.

| Command Type | Tape File Location | Output GIF |
|--------------|-------------------|------------|
| Top-level | `vhs/<command-name>.tape` | `docs/assets/vhs/<command-name>.gif` |
| Subcommand | `vhs/<group>-<subcommand>.tape` | `docs/assets/vhs/<group>-<subcommand>.gif` |

## Tape File Structure

### Basic Template

```tape
Source config.tape
Output docs/assets/vhs/<command-name>.gif

Type "kindplane <command-name>"
Sleep 500ms
Enter
Sleep 2s
```

### With Prerequisites

**Commands requiring config file** (validate, config-show, etc.):
```tape
Source config.tape
Output docs/assets/vhs/<command-name>.gif

# Ensure config file exists
Hide
Type "kindplane init --force"
Enter
Sleep 1s
Show

# Run the command
Type "kindplane <command-name>"
Sleep 500ms
Enter
Sleep 2s
```

**Commands requiring cluster** (status, logs, chart commands, etc.):
```tape
Source config.tape
Output docs/assets/vhs/<command-name>.gif

# Note: Requires a running cluster (run 'kindplane up' first)
# For demo purposes, assuming cluster exists
Type "kindplane <command-name>"
Sleep 500ms
Enter
Sleep 3s
```

**Commands with flags** (showing help):
```tape
Source config.tape
Output docs/assets/vhs/<command-name>.gif

# Note: Requires a running cluster (run 'kindplane up' first)
# Showing help for demo purposes
Type "kindplane <command-name> --help"
Sleep 500ms
Enter
Sleep 2s
```

**Commands that stream output** (logs, follow mode):
```tape
Source config.tape
Output docs/assets/vhs/<command-name>.gif

# Note: Requires a running cluster (run 'kindplane up' first)
Type "kindplane <command-name> --component crossplane"
Sleep 500ms
Enter
Sleep 2s
Ctrl+C
```

## Naming Conventions

### Top-Level Commands
- Command: `kindplane doctor` → File: `vhs/doctor.tape`
- Command: `kindplane init` → File: `vhs/init.tape`
- Command: `kindplane status` → File: `vhs/status.tape`

### Subcommands
- Command: `kindplane chart list` → File: `vhs/chart-list.tape`
- Command: `kindplane provider add` → File: `vhs/provider-add.tape`
- Command: `kindplane config show` → File: `vhs/config-show.tape`

Use kebab-case with hyphens separating group and subcommand names.

## Prerequisites Detection

Determine prerequisites based on command behaviour:

| Prerequisite | Commands | Tape Pattern |
|--------------|----------|--------------|
| **None** | doctor, cluster-list | Basic template |
| **Config file** | validate, config-show, config-diff, up | Include `init --force` setup |
| **Running cluster** | status, logs, diagnostics, apply, dump, down, chart/*, provider/*, credentials/* | Add cluster requirement comment |

## Sleep Timing Guidelines

| Command Type | Sleep Duration | Reason |
|--------------|----------------|--------|
| Simple output | `Sleep 2s` | Quick commands (list, show, validate) |
| Cluster operations | `Sleep 3s` | Status checks, diagnostics |
| Long operations | `Sleep 10s` | Cluster creation (up command) |
| Streaming | `Sleep 2s` then `Ctrl+C` | Logs, follow modes |

## Integration with Command Creation

When creating a new command using the `kindplane-commands` skill:

1. **Create the command file** (via kindplane-commands skill)
2. **Create the VHS tape file** (this skill)
3. **Update Taskfile.yaml** - Add tape to `vhs:all` task in correct order
4. **Update documentation** - Add GIF reference to command docs

### Taskfile Integration

Add the new tape to `Taskfile.yaml` in the `vhs:all` task, maintaining dependency order:

```yaml
vhs:all:
  cmds:
    # ... existing tapes ...
    - vhs vhs/<new-command>.tape  # Add in appropriate section
```

### Documentation Integration

Add GIF reference to command documentation:

```markdown
# kindplane <command-name>

![kindplane <command-name> demo](../assets/vhs/<command-name>.gif)

Command description...
```

## Examples

### Example 1: Simple Command (doctor)

**File**: `vhs/doctor.tape`
```tape
Source config.tape
Output docs/assets/vhs/doctor.gif

Type "kindplane doctor"
Sleep 500ms
Enter
Sleep 3s
```

### Example 2: Command Requiring Config (validate)

**File**: `vhs/validate.tape`
```tape
Source config.tape
Output docs/assets/vhs/validate.gif

# Ensure config file exists
Hide
Type "kindplane init --force"
Enter
Sleep 1s
Show

# Validate the configuration
Type "kindplane validate"
Sleep 500ms
Enter
Sleep 2s
```

### Example 3: Subcommand (chart-list)

**File**: `vhs/chart-list.tape`
```tape
Source config.tape
Output docs/assets/vhs/chart-list.gif

# Note: Requires a running cluster (run 'kindplane up' first)
# For demo purposes, assuming cluster exists
Type "kindplane chart list"
Sleep 500ms
Enter
Sleep 2s
```

## Checklist

When creating a VHS tape for a new command:

- [ ] Tape file created in `vhs/` directory
- [ ] Filename follows naming convention (kebab-case)
- [ ] Uses `Source config.tape` at the top
- [ ] Output path is `docs/assets/vhs/<name>.gif`
- [ ] Includes appropriate prerequisites (init setup or cluster comment)
- [ ] Sleep timing matches command duration
- [ ] Added to `Taskfile.yaml` `vhs:all` task in correct order
- [ ] GIF reference added to command documentation

## Related Skills

- **kindplane-commands**: Create the actual CLI command implementation
- This skill handles the documentation recording

## Generating GIFs

After creating tape files:

```bash
# Generate single recording
task vhs:single TAPE=<command-name>

# Generate all recordings
task vhs:all
```

GIFs will be created in `docs/assets/vhs/` and automatically included in documentation builds.
