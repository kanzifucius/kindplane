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
		fmt.Println(ui.Error("%v", err))
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), listTimeout)
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

	// Get provider status
	installer := crossplane.NewInstaller(kubeClient)
	providers, err := installer.GetProviderStatus(ctx)
	if err != nil {
		fmt.Println(ui.Error("Failed to get providers: %v", err))
		return err
	}

	if len(providers) == 0 {
		fmt.Println(ui.Warning("No providers installed"))
		return nil
	}

	// Build table data
	headers := []string{"NAME", "VERSION", "PACKAGE", "STATUS"}
	var rows [][]string

	for _, p := range providers {
		status := ui.IconSuccess + " healthy"
		if !p.Healthy {
			status = ui.IconError + " unhealthy"
			if p.Message != "" {
				status = ui.IconError + " " + p.Message
			}
		}

		rows = append(rows, []string{
			p.Name,
			p.Version,
			p.Package,
			status,
		})
	}

	fmt.Println()
	fmt.Println(ui.Title(ui.IconPackage + " Installed Providers"))
	fmt.Println(ui.Divider())
	fmt.Println(ui.RenderTable(headers, rows))

	return nil
}
