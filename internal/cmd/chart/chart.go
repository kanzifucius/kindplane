package chart

import (
	"github.com/spf13/cobra"
)

// ChartCmd is the parent command for chart subcommands
var ChartCmd = &cobra.Command{
	Use:   "chart",
	Short: "Manage Helm charts",
	Long: `Manage Helm charts in your Kind cluster.

Available subcommands:
  install   - Install a Helm chart
  upgrade   - Upgrade a Helm release
  list      - List installed Helm releases
  uninstall - Uninstall a Helm release`,
}

func init() {
	ChartCmd.AddCommand(installCmd)
	ChartCmd.AddCommand(upgradeCmd)
	ChartCmd.AddCommand(listCmd)
	ChartCmd.AddCommand(uninstallCmd)
}
