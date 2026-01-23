# Command Examples

## Complete Top-Level Command

From `internal/cmd/doctor.go` - a pre-flight check command:

```go
package cmd

import (
    "context"
    "fmt"
    "time"

    "github.com/spf13/cobra"
    "k8s.io/client-go/kubernetes"

    "github.com/kanzi/kindplane/internal/doctor"
    "github.com/kanzi/kindplane/internal/kind"
    "github.com/kanzi/kindplane/internal/ui"
)

var doctorQuiet bool

var doctorCmd = &cobra.Command{
    Use:   "doctor",
    Short: "Check system requirements and cluster health",
    Long: `Run pre-flight checks to verify system requirements are met.

Checks include:
  - Docker daemon running
  - Required binaries (kind, kubectl, helm)
  - Disk space availability
  - Kubernetes API connectivity
  - Crossplane CRDs installed`,
    Example: `  # Run all checks
  kindplane doctor

  # Quiet mode (only show failures)
  kindplane doctor --quiet`,
    RunE: runDoctor,
}

func init() {
    doctorCmd.Flags().BoolVarP(&doctorQuiet, "quiet", "q", false, "only show failures")
}

func runDoctor(cmd *cobra.Command, args []string) error {
    // Header
    if !doctorQuiet {
        fmt.Println()
        fmt.Println(ui.Title(ui.IconWrench + " kindplane doctor"))
        fmt.Println(ui.Divider())
        fmt.Println()
    }

    // Run checks
    results := doctor.RunPreflightChecks()

    // Display results
    passedCount := 0
    failedCount := 0

    for _, result := range results {
        if result.Passed {
            passedCount++
            if !doctorQuiet {
                fmt.Printf("  %s %s: %s\n",
                    ui.StyleSuccess.Render(ui.IconSuccess),
                    result.Name,
                    result.Message,
                )
            }
        } else {
            failedCount++
            icon := ui.StyleError.Render(ui.IconError)
            if !result.Required {
                icon = ui.StyleWarning.Render(ui.IconWarning)
            }
            fmt.Printf("  %s %s: %s\n", icon, result.Name, result.Message)
            if result.Suggestion != "" {
                fmt.Printf("    %s %s\n", ui.StyleMuted.Render(ui.IconArrow), result.Suggestion)
            }
        }
    }

    // Summary
    fmt.Println()
    if failedCount == 0 {
        fmt.Println(ui.SuccessBox("All Checks Passed", 
            fmt.Sprintf("All %d checks passed!", passedCount)))
    } else {
        fmt.Println(ui.ErrorBox("Checks Failed", 
            fmt.Sprintf("%d/%d checks passed", passedCount, len(results))))
    }

    return nil
}
```

## Complete Subcommand

From `internal/cmd/credentials/list.go`:

```go
package credentials

import (
    "context"
    "fmt"
    "time"

    "github.com/spf13/cobra"

    "github.com/kanzi/kindplane/internal/config"
    "github.com/kanzi/kindplane/internal/credentials"
    "github.com/kanzi/kindplane/internal/kind"
    "github.com/kanzi/kindplane/internal/ui"
)

var (
    listProvider string
    listTimeout  time.Duration
)

var listCmd = &cobra.Command{
    Use:   "list",
    Short: "List configured credentials",
    Long:  `Display all configured cloud provider credentials and their status.`,
    Example: `  # List all credentials
  kindplane credentials list

  # List credentials for a specific provider
  kindplane credentials list --provider aws`,
    RunE: runList,
}

func init() {
    listCmd.Flags().StringVar(&listProvider, "provider", "", "filter by provider (aws, azure, kubernetes)")
    listCmd.Flags().DurationVar(&listTimeout, "timeout", 30*time.Second, "timeout")
}

func runList(cmd *cobra.Command, args []string) error {
    cfg, err := config.Load("")
    if err != nil {
        fmt.Println(ui.Error("%v", err))
        return err
    }

    ctx, cancel := context.WithTimeout(context.Background(), listTimeout)
    defer cancel()

    // Check cluster
    exists, err := kind.ClusterExists(cfg.Cluster.Name)
    if err != nil {
        fmt.Println(ui.Error("Failed to check cluster: %v", err))
        return err
    }
    if !exists {
        fmt.Println(ui.Error("Cluster '%s' not found.", cfg.Cluster.Name))
        return fmt.Errorf("cluster not found")
    }

    kubeClient, err := kind.GetKubeClient(cfg.Cluster.Name)
    if err != nil {
        fmt.Println(ui.Error("Failed to connect: %v", err))
        return err
    }

    credManager := credentials.NewManager(kubeClient)
    creds, err := credManager.ListCredentials(ctx, listProvider)
    if err != nil {
        fmt.Println(ui.Error("Failed to list: %v", err))
        return err
    }

    if len(creds) == 0 {
        fmt.Println(ui.Warning("No credentials configured"))
        fmt.Println(ui.InfoBox("Hint", "Run 'kindplane credentials configure'"))
        return nil
    }

    // Display table
    fmt.Println()
    fmt.Println(ui.Title(ui.IconLock + " Configured Credentials"))
    fmt.Println(ui.Divider())

    headers := []string{"PROVIDER", "SECRET", "CONFIG", "STATUS"}
    var rows [][]string

    for _, cred := range creds {
        status := ui.IconPending + " Not configured"
        if cred.Configured {
            status = ui.IconSuccess + " Configured"
        }
        rows = append(rows, []string{
            cred.Provider,
            cred.SecretName,
            cred.ConfigName,
            status,
        })
    }

    fmt.Println(ui.RenderTable(headers, rows))
    return nil
}
```

## Command with Confirmation

From `internal/cmd/provider/remove.go`:

```go
func runRemove(cmd *cobra.Command, args []string) error {
    providerName := args[0]

    // ... cluster checks ...

    // Confirm unless --force
    if !removeForce {
        confirm := false
        prompt := &survey.Confirm{
            Message: fmt.Sprintf("Remove provider '%s'?", providerName),
        }
        if err := survey.AskOne(prompt, &confirm); err != nil {
            fmt.Println(ui.Error("Prompt failed: %v", err))
            return err
        }
        if !confirm {
            fmt.Println(ui.Warning("Removal cancelled"))
            return nil
        }
    }

    fmt.Println(ui.Info("Removing provider %s...", providerName))
    if err := installer.DeleteProvider(ctx, providerName); err != nil {
        fmt.Println(ui.Error("Failed: %v", err))
        return err
    }

    fmt.Println(ui.Success("Provider %s removed", providerName))
    return nil
}
```

## Command with JSON Output Option

From `internal/cmd/cluster/list.go`:

```go
var listFormat string

func init() {
    listCmd.Flags().StringVar(&listFormat, "format", "table", "output format (table, json)")
}

func runList(cmd *cobra.Command, args []string) error {
    // ... gather data ...

    switch listFormat {
    case "json":
        output, err := json.MarshalIndent(clusterInfos, "", "  ")
        if err != nil {
            fmt.Println(ui.Error("Failed to marshal: %v", err))
            return err
        }
        fmt.Println(string(output))

    case "table":
        fmt.Println()
        fmt.Println(ui.Title(ui.IconCluster + " Kind Clusters"))
        fmt.Println(ui.Divider())

        headers := []string{"NAME", "STATUS", "VERSION", "NODES", "CONTEXT"}
        var rows [][]string
        for _, c := range clusters {
            rows = append(rows, []string{c.Name, c.Status, c.Version, ...})
        }
        fmt.Println(ui.RenderTable(headers, rows))

    default:
        fmt.Println(ui.Error("Unknown format: %s", listFormat))
        return fmt.Errorf("unknown format")
    }

    return nil
}
```

## Parent Command Registration

In `internal/cmd/root.go`:

```go
import (
    "github.com/kanzi/kindplane/internal/cmd/chart"
    "github.com/kanzi/kindplane/internal/cmd/cluster"
    "github.com/kanzi/kindplane/internal/cmd/configcmd"
    "github.com/kanzi/kindplane/internal/cmd/credentials"
    "github.com/kanzi/kindplane/internal/cmd/provider"
)

func init() {
    // Top-level commands
    RootCmd.AddCommand(statusCmd)
    RootCmd.AddCommand(doctorCmd)
    RootCmd.AddCommand(applyCmd)

    // Command groups
    RootCmd.AddCommand(cluster.ClusterCmd)
    RootCmd.AddCommand(provider.ProviderCmd)
    RootCmd.AddCommand(chart.ChartCmd)
    RootCmd.AddCommand(credentials.CredentialsCmd)
    RootCmd.AddCommand(configcmd.ConfigCmd)
}
```
