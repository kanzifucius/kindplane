package credentials

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/kanzi/kindplane/internal/config"
	"github.com/kanzi/kindplane/internal/credentials"
	"github.com/kanzi/kindplane/internal/kind"
	"github.com/kanzi/kindplane/internal/ui"
)

var (
	listProvider string
	listTimeout  time.Duration
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured credentials",
	Long: `List configured credentials for Crossplane cloud providers.

This command shows all credentials that have been configured for
Crossplane providers, including the associated secrets and ProviderConfigs.`,
	Example: `  # List all configured credentials
  kindplane credentials list

  # List credentials for a specific provider
  kindplane credentials list --provider aws`,
	RunE: runList,
}

func init() {
	listCmd.Flags().StringVarP(&listProvider, "provider", "p", "", "filter by provider (aws, azure, gcp, kubernetes)")
	listCmd.Flags().DurationVar(&listTimeout, "timeout", 30*time.Second, "timeout for listing")
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

	credManager := credentials.NewManager(kubeClient)

	// List credentials
	creds, err := credManager.ListCredentials(ctx, listProvider)
	if err != nil {
		fmt.Println(ui.Error("Failed to list credentials: %v", err))
		return err
	}

	if len(creds) == 0 {
		if listProvider != "" {
			fmt.Println(ui.Warning("No credentials found for provider: %s", listProvider))
		} else {
			fmt.Println(ui.Warning("No credentials configured"))
		}
		fmt.Println()
		fmt.Println(ui.InfoBox("Hint", "Run 'kindplane credentials configure' to set up credentials."))
		return nil
	}

	// Display results using ui.RenderTable
	fmt.Println()
	fmt.Println(ui.Title(ui.IconLock + " Configured Credentials"))
	fmt.Println(ui.Divider())

	// Build table data
	headers := []string{"PROVIDER", "SECRET", "CONFIG", "STATUS"}
	var rows [][]string

	for _, cred := range creds {
		secretName := cred.SecretName
		if secretName == "" {
			secretName = "-"
		}
		configName := cred.ConfigName
		if configName == "" {
			configName = "-"
		}

		var status string
		if cred.Configured {
			status = ui.IconSuccess + " Configured"
		} else {
			status = ui.IconPending + " Not configured"
		}

		rows = append(rows, []string{cred.Provider, secretName, configName, status})
	}

	fmt.Println(ui.RenderTable(headers, rows))
	return nil
}
