package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/kanzi/kindplane/internal/kind"
	"github.com/kanzi/kindplane/internal/registry"
	"github.com/kanzi/kindplane/internal/ui"
)

var (
	downForce   bool
	downTimeout time.Duration
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Delete the Kind cluster",
	Long: `Delete the Kind cluster and all its resources.

This command will permanently delete the cluster. All data and
resources within the cluster will be lost.`,
	Example: `  # Delete cluster (uses config file)
  kindplane down

  # Force delete without confirmation
  kindplane down --force`,
	RunE: runDown,
}

func init() {
	downCmd.Flags().BoolVarP(&downForce, "force", "f", false, "skip confirmation prompt")
	downCmd.Flags().DurationVar(&downTimeout, "timeout", 5*time.Minute, "timeout for the operation")
}

func runDown(cmd *cobra.Command, args []string) error {
	// Require config
	requireConfig()

	ctx, cancel := context.WithTimeout(context.Background(), downTimeout)
	defer cancel()

	clusterName := cfg.Cluster.Name

	// Check if cluster exists
	exists, err := kind.ClusterExists(clusterName)
	if err != nil {
		printError("Failed to check cluster status: %v", err)
		return err
	}

	if !exists {
		printWarn("Cluster '%s' does not exist", clusterName)
		return nil
	}

	// Confirm deletion unless --force
	if !downForce {
		confirm, err := ui.ConfirmWithContext(ctx, fmt.Sprintf("Delete cluster '%s'? This cannot be undone.", clusterName))
		if err != nil {
			if err == ui.ErrCancelled {
				printWarn("Deletion cancelled")
				return nil
			}
			printError("Prompt failed: %v", err)
			return err
		}
		if !confirm {
			printWarn("Deletion cancelled")
			return nil
		}
	}

	printInfo("Deleting cluster '%s'...", clusterName)

	if err := kind.DeleteCluster(ctx, clusterName); err != nil {
		printError("Failed to delete cluster: %v", err)
		return err
	}

	printSuccess("Cluster '%s' deleted", clusterName)

	// Clean up registry if enabled and not persistent
	if cfg.Cluster.Registry.Enabled && !cfg.Cluster.Registry.Persistent {
		printInfo("Removing local registry...")
		registryManager := registry.NewManager(&cfg.Cluster.Registry)
		if err := registryManager.Remove(ctx); err != nil {
			printWarn("Failed to remove registry: %v", err)
			// Non-fatal - continue
		} else {
			printSuccess("Local registry removed")
		}
	} else if cfg.Cluster.Registry.Enabled && cfg.Cluster.Registry.Persistent {
		printInfo("Local registry preserved (persistent mode)")
	}

	return nil
}
