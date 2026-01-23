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
  show - Display the current configuration
  diff - Compare two configuration files`,
}

func init() {
	ConfigCmd.AddCommand(showCmd)
	ConfigCmd.AddCommand(diffCmd)
}
