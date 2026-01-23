package credentials

import (
	"github.com/spf13/cobra"
)

// CredentialsCmd is the parent command for credentials subcommands
var CredentialsCmd = &cobra.Command{
	Use:   "credentials",
	Short: "Manage cloud provider credentials",
	Long: `Manage cloud provider credentials for Crossplane.

Available subcommands:
  configure - Configure credentials for cloud providers`,
}

func init() {
	CredentialsCmd.AddCommand(configureCmd)
}
