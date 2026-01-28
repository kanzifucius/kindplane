package configcmd

import (
	"github.com/spf13/cobra"
)

// ConfigCmd is the parent command for config subcommands
var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long: `View and compare kindplane configuration files.

Available subcommands:
  show - Display the current kindplane configuration
  diff - Compare two configuration files
  kind - Output the Kind cluster configuration for use with kind CLI`,
}

func init() {
	ConfigCmd.AddCommand(showCmd)
	ConfigCmd.AddCommand(diffCmd)
	ConfigCmd.AddCommand(kindCmd)
}
