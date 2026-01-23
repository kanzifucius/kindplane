package chart

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/kanzi/kindplane/internal/config"
	"github.com/kanzi/kindplane/internal/helm"
	"github.com/kanzi/kindplane/internal/kind"
	"github.com/kanzi/kindplane/internal/ui"
)

var (
	upgradeRepo        string
	upgradeChart       string
	upgradeVersion     string
	upgradeNamespace   string
	upgradeValuesFiles []string
	upgradeSetValues   []string
	upgradeWait        bool
	upgradeTimeout     time.Duration
	upgradeReuseValues bool
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade <release-name>",
	Short: "Upgrade a Helm release",
	Long: `Upgrade an existing Helm release in the cluster.

This command upgrades an existing release with a new chart version
or updated configuration values.`,
	Example: `  # Upgrade to a new chart version
  kindplane chart upgrade nginx-ingress \
    --repo https://kubernetes.github.io/ingress-nginx \
    --chart ingress-nginx \
    --namespace ingress-nginx \
    --version 4.8.0

  # Upgrade with new values
  kindplane chart upgrade prometheus \
    --repo https://prometheus-community.github.io/helm-charts \
    --chart kube-prometheus-stack \
    --namespace monitoring \
    --values ./values/prometheus-updated.yaml

  # Upgrade with inline value changes
  kindplane chart upgrade nginx \
    --repo https://kubernetes.github.io/ingress-nginx \
    --chart ingress-nginx \
    --namespace ingress-nginx \
    --set controller.replicaCount=3`,
	Args: cobra.ExactArgs(1),
	RunE: runUpgrade,
}

func init() {
	upgradeCmd.Flags().StringVar(&upgradeRepo, "repo", "", "Helm repository URL (required)")
	upgradeCmd.Flags().StringVar(&upgradeChart, "chart", "", "Chart name in the repository (required)")
	upgradeCmd.Flags().StringVar(&upgradeVersion, "version", "", "Chart version (optional, latest if not specified)")
	upgradeCmd.Flags().StringVarP(&upgradeNamespace, "namespace", "n", "", "Release namespace (required)")
	upgradeCmd.Flags().StringArrayVarP(&upgradeValuesFiles, "values", "f", nil, "Path to values file (can be specified multiple times)")
	upgradeCmd.Flags().StringArrayVar(&upgradeSetValues, "set", nil, "Set values on command line (key=value, can be specified multiple times)")
	upgradeCmd.Flags().BoolVar(&upgradeWait, "wait", true, "Wait for resources to be ready")
	upgradeCmd.Flags().DurationVar(&upgradeTimeout, "timeout", 5*time.Minute, "Timeout for upgrade")
	upgradeCmd.Flags().BoolVar(&upgradeReuseValues, "reuse-values", false, "Reuse the last release's values and merge any overrides")

	_ = upgradeCmd.MarkFlagRequired("repo")
	_ = upgradeCmd.MarkFlagRequired("chart")
	_ = upgradeCmd.MarkFlagRequired("namespace")
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	releaseName := args[0]

	// Load config to get cluster name
	cfg, err := config.Load("")
	if err != nil {
		fmt.Println(ui.Error("%v", err))
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), upgradeTimeout)
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

	helmInstaller := helm.NewInstaller(kubeClient)

	// Check if release exists
	installed, err := helmInstaller.IsInstalled(ctx, releaseName, upgradeNamespace)
	if err != nil {
		fmt.Println(ui.Error("Failed to check release: %v", err))
		return err
	}
	if !installed {
		fmt.Println(ui.Error("Release '%s' not found in namespace '%s'. Use 'chart install' to install a new release.", releaseName, upgradeNamespace))
		return fmt.Errorf("release not found")
	}

	// Parse --set values
	setValues, err := helm.ParseSetValues(upgradeSetValues)
	if err != nil {
		fmt.Println(ui.Error("Failed to parse --set values: %v", err))
		return err
	}

	// Merge values: files first, then --set values (highest priority)
	mergedValues, err := helm.MergeValues(upgradeValuesFiles, setValues)
	if err != nil {
		fmt.Println(ui.Error("Failed to merge values: %v", err))
		return err
	}

	// Build chart spec for upgrade
	spec := helm.ChartSpec{
		RepoURL:     upgradeRepo,
		ChartName:   upgradeChart,
		ReleaseName: releaseName,
		Namespace:   upgradeNamespace,
		Version:     upgradeVersion,
		Values:      mergedValues,
		Wait:        upgradeWait,
		Timeout:     upgradeTimeout,
	}

	// Add repo
	repoName := helm.GenerateRepoName(upgradeRepo)
	spec.RepoName = repoName
	if err := helmInstaller.AddRepo(ctx, repoName, upgradeRepo); err != nil {
		fmt.Println(ui.Error("Failed to add helm repo: %v", err))
		return err
	}

	// Upgrade chart
	fmt.Println(ui.Info("Upgrading release %s (%s/%s)...", releaseName, upgradeRepo, upgradeChart))
	if err := helmInstaller.Upgrade(ctx, spec); err != nil {
		fmt.Println(ui.Error("Failed to upgrade release: %v", err))
		return err
	}

	fmt.Println(ui.Success("Release %s upgraded in namespace %s", releaseName, upgradeNamespace))
	return nil
}
