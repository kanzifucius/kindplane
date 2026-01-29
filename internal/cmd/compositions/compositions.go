package compositions

import (
	"github.com/spf13/cobra"
)

// CompositionsCmd is the parent command for composition subcommands
var CompositionsCmd = &cobra.Command{
	Use:   "compositions",
	Short: "Manage Crossplane compositions",
	Long: `Manage Crossplane compositions in your Kind cluster.

Available subcommands:
  reload - Reload compositions from configuration sources`,
}

func init() {
	CompositionsCmd.AddCommand(reloadCmd)
}
