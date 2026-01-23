package chart

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kanzi/kindplane/internal/config"
	"github.com/kanzi/kindplane/internal/helm"
	"github.com/kanzi/kindplane/internal/kind"
)

var (
	uninstallNamespace string
	uninstallTimeout   time.Duration
	uninstallForce     bool
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <release-name>",
	Short: "Uninstall a Helm release",
	Long:  `Uninstall a Helm release from the cluster.`,
	Example: `  # Uninstall a release
  kindplane chart uninstall nginx-ingress --namespace ingress-nginx

  # Force uninstall without confirmation
  kindplane chart uninstall prometheus --namespace monitoring --force`,
	Args: cobra.ExactArgs(1),
	RunE: runUninstall,
}

func init() {
	uninstallCmd.Flags().StringVarP(&uninstallNamespace, "namespace", "n", "", "Release namespace (required)")
	uninstallCmd.Flags().DurationVar(&uninstallTimeout, "timeout", 5*time.Minute, "Timeout for uninstallation")
	uninstallCmd.Flags().BoolVarP(&uninstallForce, "force", "f", false, "Skip confirmation prompt")

	_ = uninstallCmd.MarkFlagRequired("namespace")
}

func runUninstall(cmd *cobra.Command, args []string) error {
	releaseName := args[0]

	// Load config to get cluster name
	cfg, err := config.Load("")
	if err != nil {
		color.Red("✗ %v", err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), uninstallTimeout)
	defer cancel()

	// Check cluster exists
	exists, err := kind.ClusterExists(cfg.Cluster.Name)
	if err != nil {
		color.Red("✗ Failed to check cluster: %v", err)
		return err
	}
	if !exists {
		color.Red("✗ Cluster '%s' not found. Run 'kindplane up' first.", cfg.Cluster.Name)
		return fmt.Errorf("cluster not found")
	}

	// Get kube client
	kubeClient, err := kind.GetKubeClient(cfg.Cluster.Name)
	if err != nil {
		color.Red("✗ Failed to connect to cluster: %v", err)
		return err
	}

	helmInstaller := helm.NewInstaller(kubeClient)

	// Check if release exists
	installed, err := helmInstaller.IsInstalled(ctx, releaseName, uninstallNamespace)
	if err != nil {
		color.Red("✗ Failed to check release: %v", err)
		return err
	}
	if !installed {
		color.Yellow("! Release '%s' not found in namespace '%s'", releaseName, uninstallNamespace)
		return nil
	}

	// Confirm unless --force
	if !uninstallForce {
		color.Yellow("! This will uninstall release '%s' from namespace '%s'", releaseName, uninstallNamespace)
		fmt.Println("  Use --force to skip this confirmation")
		color.Red("✗ Uninstall cancelled. Use --force to confirm.")
		return nil
	}

	// Uninstall
	color.Cyan("• Uninstalling release %s from namespace %s...", releaseName, uninstallNamespace)
	if err := helmInstaller.UninstallRelease(ctx, releaseName, uninstallNamespace); err != nil {
		color.Red("✗ Failed to uninstall release: %v", err)
		return err
	}

	color.Green("✓ Release %s uninstalled", releaseName)
	return nil
}
