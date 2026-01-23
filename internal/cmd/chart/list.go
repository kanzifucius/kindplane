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
	listNamespace string
	listTimeout   time.Duration
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed Helm releases",
	Long: `List all Helm releases installed in the cluster.

By default, lists releases in all namespaces. Use --namespace to filter.

Examples:
  # List all releases
  kindplane chart list

  # List releases in a specific namespace
  kindplane chart list --namespace monitoring`,
	RunE: runList,
}

func init() {
	listCmd.Flags().StringVarP(&listNamespace, "namespace", "n", "", "Filter by namespace (default: all namespaces)")
	listCmd.Flags().DurationVar(&listTimeout, "timeout", 30*time.Second, "Timeout for listing releases")
}

func runList(cmd *cobra.Command, args []string) error {
	// Load config to get cluster name
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

	// List releases
	helmInstaller := helm.NewInstaller(kubeClient)
	releases, err := helmInstaller.ListReleases(ctx, listNamespace)
	if err != nil {
		color.Red("✗ Failed to list releases: %v", err)
		return err
	}

	if len(releases) == 0 {
		if listNamespace != "" {
			color.Yellow("! No releases found in namespace %s", listNamespace)
		} else {
			color.Yellow("! No releases found")
		}
		return nil
	}

	// Print header
	fmt.Println()
	fmt.Printf("%-20s %-20s %-10s %-12s %-30s %s\n",
		"NAME", "NAMESPACE", "REVISION", "STATUS", "CHART", "UPDATED")
	fmt.Println("----------------------------------------------------------------------------------------------------")

	// Print releases
	for _, rel := range releases {
		updated := ""
		if !rel.Updated.IsZero() {
			updated = rel.Updated.Format("2006-01-02 15:04:05")
		}

		statusColor := color.New(color.FgGreen)
		if rel.Status != "deployed" {
			statusColor = color.New(color.FgYellow)
		}

		fmt.Printf("%-20s %-20s %-10d ",
			truncate(rel.Name, 20),
			truncate(rel.Namespace, 20),
			rel.Revision)
		statusColor.Printf("%-12s ", rel.Status)
		fmt.Printf("%-30s %s\n",
			truncate(rel.Chart, 30),
			updated)
	}

	fmt.Println()
	return nil
}

// truncate shortens a string to maxLen characters, adding "..." if truncated
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
