package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kind/pkg/cluster"

	"github.com/kanzi/kindplane/internal/kind"
	"github.com/kanzi/kindplane/internal/ui"
)

var (
	listAll     bool
	listFormat  string
	listTimeout time.Duration
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all Kind clusters",
	Long: `List all Kind clusters on the system.

By default, shows all Kind clusters. Use --format to change the output format.`,
	Example: `  # List all Kind clusters
  kindplane cluster list

  # List in JSON format
  kindplane cluster list --format json`,
	RunE: runList,
}

func init() {
	listCmd.Flags().BoolVarP(&listAll, "all", "a", true, "Show all Kind clusters (not just kindplane-managed)")
	listCmd.Flags().StringVar(&listFormat, "format", "table", "Output format (table, json)")
	listCmd.Flags().DurationVar(&listTimeout, "timeout", 30*time.Second, "Timeout for listing clusters")
}

// ClusterInfo contains information about a Kind cluster
type ClusterInfo struct {
	Name              string `json:"name"`
	Status            string `json:"status"`
	KubernetesVersion string `json:"kubernetesVersion,omitempty"`
	Nodes             int    `json:"nodes"`
	ControlPlanes     int    `json:"controlPlanes"`
	Workers           int    `json:"workers"`
	Context           string `json:"context"`
}

func runList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), listTimeout)
	defer cancel()

	// Get Kind provider
	provider := cluster.NewProvider()

	// List clusters
	clusters, err := provider.List()
	if err != nil {
		fmt.Println(ui.Error("Failed to list clusters: %v", err))
		return err
	}

	if len(clusters) == 0 {
		fmt.Println(ui.Warning("No Kind clusters found"))
		fmt.Println()
		fmt.Println(ui.InfoBox("Hint", "Run 'kindplane up' to create a cluster."))
		return nil
	}

	// Gather cluster information
	var clusterInfos []ClusterInfo
	for _, clusterName := range clusters {
		info := ClusterInfo{
			Name:    clusterName,
			Context: fmt.Sprintf("kind-%s", clusterName),
		}

		// Try to get more details
		kubeClient, err := kind.GetKubeClient(clusterName)
		if err != nil {
			info.Status = "Unknown"
		} else {
			// Check if cluster is accessible
			_, err := kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{Limit: 1})
			if err != nil {
				info.Status = "Unreachable"
			} else {
				info.Status = "Running"

				// Get node count
				nodes, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
				if err == nil {
					info.Nodes = len(nodes.Items)
					for _, node := range nodes.Items {
						// Check if control plane
						if _, ok := node.Labels["node-role.kubernetes.io/control-plane"]; ok {
							info.ControlPlanes++
						} else {
							info.Workers++
						}

						// Get Kubernetes version from first node
						if info.KubernetesVersion == "" {
							info.KubernetesVersion = node.Status.NodeInfo.KubeletVersion
						}
					}
				}
			}
		}

		clusterInfos = append(clusterInfos, info)
	}

	// Output based on format
	switch listFormat {
	case "json":
		output, err := json.MarshalIndent(clusterInfos, "", "  ")
		if err != nil {
			fmt.Println(ui.Error("Failed to marshal output: %v", err))
			return err
		}
		fmt.Println(string(output))

	case "table":
		printClusterTable(clusterInfos)

	default:
		fmt.Println(ui.Error("Unknown format: %s. Use 'table' or 'json'.", listFormat))
		return fmt.Errorf("unknown format: %s", listFormat)
	}

	return nil
}

func printClusterTable(clusters []ClusterInfo) {
	fmt.Println()
	fmt.Println(ui.Title(ui.IconCluster + " Kind Clusters"))
	fmt.Println(ui.Divider())

	// Build table data
	headers := []string{"NAME", "STATUS", "VERSION", "NODES", "CONTEXT"}
	var rows [][]string

	for _, c := range clusters {
		var statusDisplay string
		switch c.Status {
		case "Running":
			statusDisplay = ui.IconSuccess + " Running"
		case "Unreachable":
			statusDisplay = ui.IconError + " Unreachable"
		default:
			statusDisplay = ui.IconWarning + " Unknown"
		}

		nodeInfo := fmt.Sprintf("%d", c.Nodes)
		if c.ControlPlanes > 0 || c.Workers > 0 {
			nodeInfo = fmt.Sprintf("%d (%dc/%dw)", c.Nodes, c.ControlPlanes, c.Workers)
		}

		version := c.KubernetesVersion
		if version == "" {
			version = "-"
		}

		rows = append(rows, []string{c.Name, statusDisplay, version, nodeInfo, c.Context})
	}

	fmt.Println(ui.RenderTable(headers, rows))
}
