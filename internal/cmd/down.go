package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/kanzi/kindplane/internal/kind"
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
		printWarn("This will permanently delete cluster '%s'", clusterName)
		printStep("Use --force to skip this confirmation")
		// In a real implementation, you'd prompt for confirmation here
		// For now, we'll just require --force
		printError("Deletion cancelled. Use --force to confirm.")
		return nil
	}

	printInfo("Deleting cluster '%s'...", clusterName)

	if err := kind.DeleteCluster(ctx, clusterName); err != nil {
		printError("Failed to delete cluster: %v", err)
		return err
	}

	printSuccess("Cluster '%s' deleted", clusterName)

	return nil
}
