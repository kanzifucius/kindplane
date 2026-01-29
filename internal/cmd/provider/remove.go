package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/kanzi/kindplane/internal/config"
	"github.com/kanzi/kindplane/internal/crossplane"
	"github.com/kanzi/kindplane/internal/kind"
	"github.com/kanzi/kindplane/internal/ui"
)

var (
	removeTimeout time.Duration
	removeForce   bool
)

var removeCmd = &cobra.Command{
	Use:   "remove <provider-name>",
	Short: "Remove a Crossplane provider from the cluster",
	Long: `Remove a Crossplane provider from the running cluster.

This will delete the provider resource from the cluster. Any managed
resources created by this provider may become orphaned.`,
	Example: `  # Remove a provider (with confirmation)
  kindplane provider remove provider-aws

  # Force remove without confirmation
  kindplane provider remove provider-aws --force`,
	Args: cobra.ExactArgs(1),
	RunE: runRemove,
}

func init() {
	removeCmd.Flags().DurationVar(&removeTimeout, "timeout", 5*time.Minute, "timeout for removal")
	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "skip confirmation prompt")
}

func runRemove(cmd *cobra.Command, args []string) error {
	providerName := args[0]

	// Load config
	cfg, err := config.Load("")
	if err != nil {
		fmt.Println(ui.Error("%v", err))
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), removeTimeout)
	defer cancel()

	// Check cluster exists
	exists, err := kind.ClusterExists(cfg.Cluster.Name)
	if err != nil {
		fmt.Println(ui.Error("Failed to check cluster: %v", err))
		return err
	}
	if !exists {
		fmt.Println(ui.Error("Cluster '%s' not found. Run 'kindplane up' first.", cfg.Cluster.Name))
		return fmt.Errorf("cluster not found")
	}

	// Get kube client
	kubeClient, err := kind.GetKubeClient(cfg.Cluster.Name)
	if err != nil {
		fmt.Println(ui.Error("Failed to connect to cluster: %v", err))
		return err
	}

	installer := crossplane.NewInstaller(kubeClient)

	// Check if provider exists
	providerExists, err := installer.ProviderExists(ctx, providerName)
	if err != nil {
		fmt.Println(ui.Error("Failed to check provider: %v", err))
		return err
	}
	if !providerExists {
		fmt.Println(ui.Warning("Provider '%s' not found", providerName))
		return nil
	}

	// Confirm unless --force
	if !removeForce {
		confirm, err := ui.ConfirmWithContext(ctx, fmt.Sprintf("Remove provider '%s'? Managed resources may become orphaned.", providerName))
		if err != nil {
			if err == ui.ErrCancelled {
				fmt.Println(ui.Warning("Removal cancelled"))
				return nil
			}
			fmt.Println(ui.Error("Prompt failed: %v", err))
			return err
		}
		if !confirm {
			fmt.Println(ui.Warning("Removal cancelled"))
			return nil
		}
	}

	// Remove provider
	fmt.Println(ui.Info("Removing provider %s...", providerName))
	if err := installer.DeleteProvider(ctx, providerName); err != nil {
		fmt.Println(ui.Error("Failed to remove provider: %v", err))
		return err
	}

	fmt.Println(ui.Success("Provider %s removed", providerName))
	return nil
}
