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
	installRepo        string
	installChart       string
	installVersion     string
	installNamespace   string
	installValuesFiles []string
	installSetValues   []string
	installWait        bool
	installTimeout     time.Duration
	installCreate      bool
)

var installCmd = &cobra.Command{
	Use:   "install <release-name>",
	Short: "Install a Helm chart",
	Long: `Install a Helm chart to the cluster.

The release name is used as the Kubernetes resource name.`,
	Example: `  # Install nginx ingress controller
  kindplane chart install nginx-ingress \
    --repo https://kubernetes.github.io/ingress-nginx \
    --chart ingress-nginx \
    --namespace ingress-nginx

  # Install with custom values file
  kindplane chart install prometheus \
    --repo https://prometheus-community.github.io/helm-charts \
    --chart kube-prometheus-stack \
    --namespace monitoring \
    --values ./values/prometheus.yaml

  # Install with inline values
  kindplane chart install nginx \
    --repo https://kubernetes.github.io/ingress-nginx \
    --chart ingress-nginx \
    --namespace ingress-nginx \
    --set controller.replicaCount=2`,
	Args: cobra.ExactArgs(1),
	RunE: runInstall,
}

func init() {
	installCmd.Flags().StringVar(&installRepo, "repo", "", "Helm repository URL (required)")
	installCmd.Flags().StringVar(&installChart, "chart", "", "Chart name in the repository (required)")
	installCmd.Flags().StringVar(&installVersion, "version", "", "Chart version (optional, latest if not specified)")
	installCmd.Flags().StringVarP(&installNamespace, "namespace", "n", "", "Target namespace (required)")
	installCmd.Flags().StringArrayVarP(&installValuesFiles, "values", "f", nil, "Path to values file (can be specified multiple times)")
	installCmd.Flags().StringArrayVar(&installSetValues, "set", nil, "Set values on command line (key=value, can be specified multiple times)")
	installCmd.Flags().BoolVar(&installWait, "wait", true, "Wait for resources to be ready")
	installCmd.Flags().DurationVar(&installTimeout, "timeout", 5*time.Minute, "Timeout for installation")
	installCmd.Flags().BoolVar(&installCreate, "create-namespace", true, "Create namespace if it doesn't exist")

	_ = installCmd.MarkFlagRequired("repo")
	_ = installCmd.MarkFlagRequired("chart")
	_ = installCmd.MarkFlagRequired("namespace")
}

func runInstall(cmd *cobra.Command, args []string) error {
	releaseName := args[0]

	// Load config to get cluster name
	cfg, err := config.Load("")
	if err != nil {
		fmt.Println(ui.Error("%v", err))
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), installTimeout)
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

	// Parse --set values
	setValues, err := helm.ParseSetValues(installSetValues)
	if err != nil {
		fmt.Println(ui.Error("Failed to parse --set values: %v", err))
		return err
	}

	// Merge values: files first, then --set values (highest priority)
	mergedValues, err := helm.MergeValues(installValuesFiles, setValues)
	if err != nil {
		fmt.Println(ui.Error("Failed to merge values: %v", err))
		return err
	}

	// Build chart config
	wait := installWait
	create := installCreate
	chartCfg := config.ChartConfig{
		Name:            releaseName,
		Repo:            installRepo,
		Chart:           installChart,
		Version:         installVersion,
		Namespace:       installNamespace,
		CreateNamespace: &create,
		Wait:            &wait,
		Timeout:         installTimeout.String(),
		Values:          mergedValues,
	}

	// Install chart
	fmt.Println(ui.Info("Installing chart %s (%s/%s)...", releaseName, installRepo, installChart))
	helmInstaller := helm.NewInstaller(kubeClient)
	if err := helmInstaller.InstallChartFromConfig(ctx, chartCfg); err != nil {
		fmt.Println(ui.Error("Failed to install chart: %v", err))
		return err
	}

	fmt.Println(ui.Success("Chart %s installed in namespace %s", releaseName, installNamespace))
	return nil
}
