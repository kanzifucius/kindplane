package provider

import (
	"github.com/spf13/cobra"
)

// ProviderCmd is the parent command for provider subcommands
var ProviderCmd = &cobra.Command{
	Use:   "provider",
	Short: "Manage Crossplane providers",
	Long: `Manage Crossplane providers in your Kind cluster.

Available subcommands:
  add    - Add a new Crossplane provider
  list   - List installed providers
  remove - Remove a Crossplane provider`,
}

func init() {
	ProviderCmd.AddCommand(addCmd)
	ProviderCmd.AddCommand(listCmd)
	ProviderCmd.AddCommand(removeCmd)
}
