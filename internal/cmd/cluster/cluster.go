package cluster

import (
	"github.com/spf13/cobra"
)

// ClusterCmd is the parent command for cluster subcommands
var ClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Manage Kind clusters",
	Long: `Manage Kind clusters created by kindplane.

Available subcommands:
  list - List all Kind clusters`,
}

func init() {
	ClusterCmd.AddCommand(listCmd)
}
