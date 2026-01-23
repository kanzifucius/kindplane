package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kanzi/kindplane/internal/config"
	"github.com/kanzi/kindplane/internal/crossplane"
	"github.com/kanzi/kindplane/internal/kind"
)

var (
	listTimeout time.Duration
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed Crossplane providers",
	Long: `List all Crossplane providers installed in the cluster.

Shows provider name, version, and health status.`,
	Example: `  # List all providers
  kindplane provider list`,
	RunE: runList,
}

func init() {
	listCmd.Flags().DurationVar(&listTimeout, "timeout", 30*time.Second, "timeout for listing providers")
}

func runList(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load("")
	if err != nil {
		color.Red("✗ %v", err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), listTimeout)
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

	// Get provider status
	installer := crossplane.NewInstaller(kubeClient)
	providers, err := installer.GetProviderStatus(ctx)
	if err != nil {
		color.Red("✗ Failed to get providers: %v", err)
		return err
	}

	if len(providers) == 0 {
		color.Yellow("! No providers installed")
		return nil
	}

	fmt.Println()
	fmt.Println("Installed Providers:")
	fmt.Println("====================")
	fmt.Println()

	for _, p := range providers {
		statusIcon := color.GreenString("✓")
		statusText := "healthy"
		if !p.Healthy {
			statusIcon = color.RedString("✗")
			statusText = "unhealthy"
		}

		fmt.Printf("%s %s\n", statusIcon, p.Name)
		fmt.Printf("  Version: %s\n", p.Version)
		fmt.Printf("  Package: %s\n", p.Package)
		fmt.Printf("  Status:  %s\n", statusText)
		if p.Message != "" {
			fmt.Printf("  Message: %s\n", p.Message)
		}
		fmt.Println()
	}

	return nil
}
