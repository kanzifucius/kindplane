package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/kanzi/kindplane/internal/config"
	"github.com/kanzi/kindplane/internal/crossplane"
	"github.com/kanzi/kindplane/internal/kind"
	"github.com/kanzi/kindplane/internal/ui"
)

var (
	addPackage string
	addTimeout time.Duration
	addWait    bool
)

var addCmd = &cobra.Command{
	Use:   "add <provider-name>",
	Short: "Add a Crossplane provider to the cluster",
	Long: `Add a new Crossplane provider to the running cluster.

The provider name is the Kubernetes resource name for the provider.
The package is the full OCI package path including version tag.`,
	Example: `  # Add AWS provider from Upbound
  kindplane provider add provider-aws --package xpkg.upbound.io/upbound/provider-aws:v1.1.0

  # Add Kubernetes provider from crossplane-contrib
  kindplane provider add provider-kubernetes --package xpkg.upbound.io/crossplane-contrib/provider-kubernetes:v0.12.0

  # Add provider and wait for it to be healthy
  kindplane provider add provider-gcp --package xpkg.upbound.io/upbound/provider-gcp:v0.22.0 --wait`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

func init() {
	addCmd.Flags().StringVarP(&addPackage, "package", "p", "", "full OCI package path with version (e.g., xpkg.upbound.io/upbound/provider-aws:v1.1.0) (required)")
	addCmd.Flags().DurationVar(&addTimeout, "timeout", 5*time.Minute, "timeout for installation")
	addCmd.Flags().BoolVar(&addWait, "wait", true, "wait for provider to be healthy")
	_ = addCmd.MarkFlagRequired("package")
}

func runAdd(cmd *cobra.Command, args []string) error {
	providerName := args[0]

	// Load config
	cfg, err := config.Load("")
	if err != nil {
		fmt.Println(ui.Error("%v", err))
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), addTimeout)
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

	// Install provider
	fmt.Println(ui.Info("Installing %s (%s)...", providerName, addPackage))
	installer := crossplane.NewInstaller(kubeClient)
	if err := installer.InstallProvider(ctx, providerName, addPackage); err != nil {
		fmt.Println(ui.Error("Failed to install provider: %v", err))
		return err
	}

	if addWait {
		fmt.Println(ui.Info("Waiting for provider to be healthy..."))
		if err := installer.WaitForProviders(ctx); err != nil {
			fmt.Println(ui.Error("Provider failed to become healthy: %v", err))
			return err
		}
	}

	fmt.Println(ui.Success("Provider %s installed (%s)", providerName, addPackage))
	return nil
}
