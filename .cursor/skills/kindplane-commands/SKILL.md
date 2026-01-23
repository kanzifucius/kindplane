---
name: kindplane-commands
description: Create new CLI commands for the kindplane tool following established patterns and UI styling. Use when adding commands, subcommands, or modifying CLI behaviour.
---

# Creating kindplane CLI Commands

## Quick Reference

| Command Type | Location | Registration |
|--------------|----------|--------------|
| Top-level | `internal/cmd/<name>.go` | `root.go` → `RootCmd.AddCommand()` |
| Subcommand | `internal/cmd/<group>/<name>.go` | `<group>.go` → `ParentCmd.AddCommand()` |

## Command Structure

### Top-Level Command Template

```go
package cmd

import (
    "context"
    "fmt"
    "time"

    "github.com/spf13/cobra"

    "github.com/kanzi/kindplane/internal/config"
    "github.com/kanzi/kindplane/internal/kind"
    "github.com/kanzi/kindplane/internal/ui"
)

var (
    myTimeout time.Duration
    myFlag    string
)

var myCmd = &cobra.Command{
    Use:   "mycommand",
    Short: "One-line description (max 50 chars)",
    Long: `Extended description of what this command does.

Include context about when to use it and what it affects.`,
    Example: `  # Basic usage
  kindplane mycommand

  # With options
  kindplane mycommand --flag value`,
    RunE: runMyCommand,
}

func init() {
    myCmd.Flags().DurationVar(&myTimeout, "timeout", 5*time.Minute, "operation timeout")
    myCmd.Flags().StringVarP(&myFlag, "flag", "f", "", "flag description")
}

func runMyCommand(cmd *cobra.Command, args []string) error {
    requireConfig()

    ctx, cancel := context.WithTimeout(context.Background(), myTimeout)
    defer cancel()

    // Print header
    fmt.Println()
    fmt.Println(ui.Title(ui.IconRocket + " My Command"))
    fmt.Println(ui.Divider())
    fmt.Println()

    // Command logic here...

    return nil
}
```

### Subcommand Group Template

Create a parent command in `internal/cmd/<group>/<group>.go`:

```go
package mygroup

import "github.com/spf13/cobra"

var MyGroupCmd = &cobra.Command{
    Use:   "mygroup",
    Short: "Manage mygroup resources",
    Long: `Manage mygroup resources in the cluster.

Available subcommands:
  list   - List resources
  add    - Add a resource
  remove - Remove a resource`,
}

func init() {
    MyGroupCmd.AddCommand(listCmd)
    MyGroupCmd.AddCommand(addCmd)
    MyGroupCmd.AddCommand(removeCmd)
}
```

Register in `root.go`:

```go
import "github.com/kanzi/kindplane/internal/cmd/mygroup"

// In init():
RootCmd.AddCommand(mygroup.MyGroupCmd)
```

## UI Styling

### Required Import

```go
import "github.com/kanzi/kindplane/internal/ui"
```

### Message Functions

Use helper functions from `root.go` for commands in the `cmd` package:

```go
printSuccess("Operation completed: %s", name)   // ✓ green
printError("Failed: %v", err)                   // ✗ red
printWarn("Warning: %s", message)               // ! yellow
printInfo("Processing %s...", item)             // • blue
printStep("Step 1: %s", description)            // indented bullet
```

For subcommand packages, use `ui` functions directly:

```go
fmt.Println(ui.Success("Completed: %s", name))
fmt.Println(ui.Error("Failed: %v", err))
fmt.Println(ui.Warning("Warning: %s", message))
fmt.Println(ui.Info("Processing..."))
```

### Headers and Dividers

```go
fmt.Println(ui.Title(ui.IconCluster + " Command Title"))
fmt.Println(ui.Divider())
```

### Available Icons

| Icon | Constant | Use for |
|------|----------|---------|
| ✓ | `ui.IconSuccess` | Success states |
| ✗ | `ui.IconError` | Errors |
| ! | `ui.IconWarning` | Warnings |
| • | `ui.IconInfo` | Info messages |
| ⎈ | `ui.IconCluster` | Cluster operations |
| 📦 | `ui.IconPackage` | Packages, installations |
| ⚙ | `ui.IconGear` | Configuration, providers |
| 🔒 | `ui.IconLock` | Credentials, secrets |
| 📁 | `ui.IconFolder` | Directories |
| 📄 | `ui.IconFile` | Files, logs |
| 🚀 | `ui.IconRocket` | Apply, deploy |
| 🔧 | `ui.IconWrench` | Doctor, diagnostics |
| 🔍 | `ui.IconMagnifier` | Search, diff |
| → | `ui.IconArrow` | Suggestions, next steps |

### Tables

```go
headers := []string{"NAME", "STATUS", "VERSION"}
rows := [][]string{
    {"item1", ui.IconSuccess + " Ready", "1.0.0"},
    {"item2", ui.IconWarning + " Pending", "1.0.1"},
}
fmt.Println(ui.RenderTable(headers, rows))
```

### Boxes

```go
fmt.Println(ui.SuccessBox("Title", "All operations completed."))
fmt.Println(ui.ErrorBox("Error", "Operation failed."))
fmt.Println(ui.WarningBox("Warning", "Proceed with caution."))
fmt.Println(ui.InfoBox("Hint", "Run 'kindplane up' first."))
```

### Styles for Custom Output

```go
import "github.com/charmbracelet/lipgloss"

// Use ui package colours
style := lipgloss.NewStyle().
    Foreground(ui.ColorSuccess).  // Green
    Bold(true)

// Available colours:
// ui.ColorPrimary, ui.ColorSecondary
// ui.ColorSuccess, ui.ColorError, ui.ColorWarning, ui.ColorInfo
// ui.ColorMuted, ui.ColorText, ui.ColorBorder

// Pre-built styles:
ui.StyleSuccess, ui.StyleError, ui.StyleWarning
ui.StyleInfo, ui.StyleMuted, ui.StyleBold
```

## Common Patterns

### Cluster Check

```go
exists, err := kind.ClusterExists(cfg.Cluster.Name)
if err != nil {
    fmt.Println(ui.Error("Failed to check cluster: %v", err))
    return err
}
if !exists {
    fmt.Println(ui.Error("Cluster '%s' not found. Run 'kindplane up' first.", cfg.Cluster.Name))
    return fmt.Errorf("cluster not found")
}
```

### Get Kubernetes Client

```go
kubeClient, err := kind.GetKubeClient(cfg.Cluster.Name)
if err != nil {
    fmt.Println(ui.Error("Failed to connect to cluster: %v", err))
    return err
}
```

### Confirmation Prompt

```go
import "github.com/AlecAivazis/survey/v2"

if !forceFlag {
    confirm := false
    prompt := &survey.Confirm{
        Message: fmt.Sprintf("Delete '%s'? This cannot be undone.", name),
    }
    if err := survey.AskOne(prompt, &confirm); err != nil {
        fmt.Println(ui.Error("Prompt failed: %v", err))
        return err
    }
    if !confirm {
        fmt.Println(ui.Warning("Operation cancelled"))
        return nil
    }
}
```

### Spinners and Progress

```go
// Simple spinner
err := ui.RunSpinner("Installing component", func() error {
    return doInstall(ctx)
})

// With context cancellation support
err := ui.RunSpinnerWithContext(ctx, "Installing", func(ctx context.Context) error {
    return doInstall(ctx)
})

// Progress through items
err := ui.RunProgress("Processing items", items, func(item string) error {
    return processItem(item)
})
```

## Flag Conventions

| Type | Example | Notes |
|------|---------|-------|
| Timeout | `--timeout 5m` | Use `time.Duration` |
| Namespace | `-n, --namespace` | Short flag `-n` |
| Force | `-f, --force` | Skip confirmations |
| Format | `--format table\|json` | Output format |
| Values file | `-f, --values` | Helm-style values |
| Verbose | `-V, --verbose` | Global flag (note: capital V) |

**Avoid `-c` shorthand** - reserved for global `--config` flag.

## Regenerate Documentation

After adding commands, regenerate CLI docs:

```bash
go run ./cmd/gendocs
```

## Checklist

- [ ] Command file created in correct location
- [ ] Registered in `root.go` or parent command
- [ ] Uses `ui` package for all output
- [ ] Header with icon and `ui.Divider()`
- [ ] Error handling uses `ui.Error()`
- [ ] Success messages use `ui.Success()`
- [ ] Tables use `ui.RenderTable()`
- [ ] Boxes for hints/summaries
- [ ] Flags follow naming conventions
- [ ] `go build ./...` passes
- [ ] `go run ./cmd/gendocs` regenerated
