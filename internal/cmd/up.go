package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"

	"github.com/kanzi/kindplane/internal/config"
	"github.com/kanzi/kindplane/internal/crossplane"
	"github.com/kanzi/kindplane/internal/diagnostics"
	"github.com/kanzi/kindplane/internal/helm"
	"github.com/kanzi/kindplane/internal/kind"
	"github.com/kanzi/kindplane/internal/registry"
	"github.com/kanzi/kindplane/internal/ui"
)

var (
	upSkipCrossplane    bool
	upSkipProviders     bool
	upSkipCharts        bool
	upSkipCompositions  bool
	upTimeout           time.Duration
	upRollbackOnFailure bool
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Create and bootstrap a Kind cluster with Crossplane",
	Long: `Create a Kind cluster and bootstrap it with Crossplane, providers,
Helm charts, and custom compositions.

This command requires a kindplane.yaml configuration file.
Run 'kindplane init' first if you don't have one.

The bootstrap process:
  1. Create Kind cluster
  2. Install charts with phase: pre-crossplane
  3. Install Crossplane
  4. Install charts with phase: post-crossplane
  5. Install Crossplane providers
  6. Install charts with phase: post-providers
  7. Install charts with phase: final (default)
  8. Apply custom compositions (if configured)`,
	Example: `  # Create cluster with full bootstrap
  kindplane up

  # Skip provider installation
  kindplane up --skip-providers

  # Skip all chart installations
  kindplane up --skip-charts

  # Rollback (delete cluster) on failure
  kindplane up --rollback-on-failure`,
	RunE: runUp,
}

func init() {
	upCmd.Flags().BoolVar(&upSkipCrossplane, "skip-crossplane", false, "skip Crossplane installation")
	upCmd.Flags().BoolVar(&upSkipProviders, "skip-providers", false, "skip Crossplane provider installation")
	upCmd.Flags().BoolVar(&upSkipCharts, "skip-charts", false, "skip all Helm chart installations")
	upCmd.Flags().BoolVar(&upSkipCompositions, "skip-compositions", false, "skip applying compositions")
	upCmd.Flags().DurationVar(&upTimeout, "timeout", 10*time.Minute, "timeout for the entire operation")
	upCmd.Flags().BoolVar(&upRollbackOnFailure, "rollback-on-failure", false, "delete cluster if bootstrap fails")
}

// bootstrapContext holds shared resources during bootstrap
type bootstrapContext struct {
	ctx           context.Context
	kubeClient    *kubernetes.Clientset
	dynamicClient dynamic.Interface
	diagCollector *diagnostics.Collector
}

// Phase names for the bootstrap process
const (
	phaseRegistry       = "Create local registry"
	phaseTrustedCAs     = "Configure trusted CAs"
	phaseCluster        = "Create Kind cluster"
	phaseConnect        = "Connect to cluster"
	phasePreCrossplane  = "Install pre-crossplane charts"
	phaseCrossplane     = "Install Crossplane"
	phasePostCrossplane = "Install post-crossplane charts"
	phaseProviders      = "Install providers"
	phasePostProviders  = "Install post-providers charts"
	phaseFinal          = "Install final charts"
	phaseCompositions   = "Apply compositions"
)

// buildPhases creates the phase tracker with phases based on config
func buildPhases() *ui.PhaseTracker {
	pt := ui.NewPhaseTracker("Bootstrap Cluster",
		ui.WithPhaseTrackerIcon(ui.IconRocket),
		ui.WithClusterInfo(cfg.Cluster.Name, cfgFile),
		ui.WithShowUpfrontList(true),
	)

	// Add phases conditionally based on configuration
	pt.AddPhaseIf(cfg.Cluster.Registry.Enabled, phaseRegistry)
	pt.AddPhaseIf(kind.HasTrustedCAs(cfg), phaseTrustedCAs)
	pt.AddPhase(phaseCluster)
	pt.AddPhase(phaseConnect)
	pt.AddPhaseIf(!upSkipCharts && len(getChartsForPhase(config.ChartPhasePrecrossplane)) > 0, phasePreCrossplane)
	pt.AddPhaseIf(!upSkipCrossplane, phaseCrossplane)
	pt.AddPhaseIf(!upSkipCharts && len(getChartsForPhase(config.ChartPhasePostCrossplane)) > 0, phasePostCrossplane)
	pt.AddPhaseIf(!upSkipProviders && !upSkipCrossplane && len(cfg.Crossplane.Providers) > 0, phaseProviders)
	pt.AddPhaseIf(!upSkipCharts && len(getChartsForPhase(config.ChartPhasePostProviders)) > 0, phasePostProviders)
	pt.AddPhaseIf(!upSkipCharts && len(getChartsForPhase(config.ChartPhaseFinal)) > 0, phaseFinal)
	pt.AddPhaseIf(!upSkipCompositions && len(cfg.Compositions.Sources) > 0, phaseCompositions)

	return pt
}

func runUp(cmd *cobra.Command, args []string) error {
	// Require config
	requireConfig()

	ctx, cancel := context.WithTimeout(context.Background(), upTimeout)
	defer cancel()

	// Build phase tracker
	pt := buildPhases()

	// Print header with phase list
	pt.PrintHeader()

	// Track if we created the cluster (for rollback)
	clusterCreated := false

	// Bootstrap context for shared resources
	var bc *bootstrapContext

	// Helper to handle phase failure with summary
	handleFailure := func(err error) error {
		pt.FailPhase(err)
		pt.PrintSummary()
		if upRollbackOnFailure && clusterCreated {
			fmt.Println(ui.Warning("Rolling back: deleting cluster..."))
			if delErr := kind.DeleteCluster(ctx, cfg.Cluster.Name); delErr != nil {
				fmt.Println(ui.Error("Failed to delete cluster during rollback: %v", delErr))
			} else {
				fmt.Println(ui.Info("Cluster deleted"))
			}
		}
		return err
	}

	// Phase: Create local registry
	var registryManager *registry.Manager
	if cfg.Cluster.Registry.Enabled {
		pt.StartPhase(phaseRegistry)
		registryManager = registry.NewManager(&cfg.Cluster.Registry)
		if err := registryManager.Create(ctx); err != nil {
			return handleFailure(fmt.Errorf("failed to create registry: %w", err))
		}
		pt.CompletePhaseWithMessage(fmt.Sprintf("Registry available at localhost:%d", cfg.Cluster.Registry.GetPort()))
	}

	// Phase: Configure trusted CAs
	if kind.HasTrustedCAs(cfg) {
		pt.StartPhase(phaseTrustedCAs)
		var caSummary *kind.TrustedCAsSummary
		if err := ui.RunSpinnerWithContext(ctx, "Validating CA certificates", func(spinnerCtx context.Context) error {
			var err error
			caSummary, err = kind.ValidateTrustedCAs(cfg)
			return err
		}); err != nil {
			return handleFailure(fmt.Errorf("failed to configure trusted CAs: %w", err))
		}
		pt.CompletePhaseWithMessage(fmt.Sprintf("%d registry CA(s), %d workload CA(s)", caSummary.RegistryCount, caSummary.WorkloadCount))
	}

	// Phase: Create Kind cluster
	pt.StartPhase(phaseCluster)
	exists, err := kind.ClusterExists(cfg.Cluster.Name)
	if err != nil {
		return handleFailure(fmt.Errorf("failed to check cluster status: %w", err))
	}

	if exists {
		pt.SkipPhase(phaseCluster, "already exists")
	} else {
		// Display cluster configuration details
		nodeImage, imageSource := kind.GetNodeImage(cfg)
		if nodeImage != "" {
			fmt.Println(ui.KeyValueIndented("Node Image", ui.Code(nodeImage), 2))
			fmt.Println(ui.KeyValueIndented("Source", imageSource, 2))
		} else {
			fmt.Println(ui.KeyValueIndented("Node Image", ui.Muted("Kind default"), 2))
			fmt.Println(ui.KeyValueIndented("Source", imageSource, 2))
		}
		fmt.Println()

		if err := ui.RunClusterCreate(ctx, cfg.Cluster.Name, func(ctx context.Context, updates chan<- ui.StepUpdate) error {
			logger := kind.NewLogger(updates)
			return kind.CreateClusterWithProgress(ctx, cfg, logger)
		}); err != nil {
			return handleFailure(fmt.Errorf("failed to create cluster: %w", err))
		}
		clusterCreated = true
		pt.CompletePhase()
	}

	// Configure registry for cluster nodes if enabled
	if cfg.Cluster.Registry.Enabled && registryManager != nil {
		if err := ui.RunSpinnerWithContext(ctx, "Configuring registry on cluster nodes", func(spinnerCtx context.Context) error {
			if err := registryManager.ConfigureNodes(spinnerCtx, cfg.Cluster.Name); err != nil {
				return err
			}
			return registryManager.ConnectToNetwork(spinnerCtx, "kind")
		}); err != nil {
			return handleFailure(fmt.Errorf("failed to configure registry: %w", err))
		}
	}

	// Phase: Connect to cluster
	pt.StartPhase(phaseConnect)
	var kubeClient *kubernetes.Clientset
	var dynamicClient dynamic.Interface
	if err := ui.RunSpinnerWithContext(ctx, "Connecting to cluster", func(spinnerCtx context.Context) error {
		var err error
		kubeClient, err = kind.GetKubeClient(cfg.Cluster.Name)
		if err != nil {
			return fmt.Errorf("failed to get kubernetes client: %w", err)
		}

		restConfig, err := kind.GetRESTConfig(cfg.Cluster.Name)
		if err != nil {
			return fmt.Errorf("failed to get REST config: %w", err)
		}

		dynamicClient, err = dynamic.NewForConfig(restConfig)
		if err != nil {
			return fmt.Errorf("failed to create dynamic client: %w", err)
		}

		return nil
	}); err != nil {
		return handleFailure(err)
	}

	// Create diagnostics collector
	diagCollector := diagnostics.NewCollector(kubeClient, dynamicClient)

	// Initialize bootstrap context
	bc = &bootstrapContext{
		ctx:           ctx,
		kubeClient:    kubeClient,
		dynamicClient: dynamicClient,
		diagCollector: diagCollector,
	}
	pt.CompletePhase()

	// Create local registry ConfigMap for discovery
	if cfg.Cluster.Registry.Enabled && registryManager != nil {
		if err := createRegistryConfigMap(ctx, kubeClient, &cfg.Cluster.Registry); err != nil {
			// Non-fatal - continue with bootstrap
			fmt.Println(ui.Warning("Failed to create registry ConfigMap: %v", err))
		}
	}

	// Create Helm installer for chart installations
	helmInstaller := helm.NewInstaller(kubeClient)

	// Phase: Install pre-crossplane charts
	if pt.GetPhase(phasePreCrossplane) != nil {
		if err := installChartsForPhaseWithTracker(ctx, helmInstaller, config.ChartPhasePrecrossplane, bc, pt, phasePreCrossplane); err != nil {
			return handleFailure(err)
		}
	}

	// Phase: Install Crossplane
	if !upSkipCrossplane {
		pt.StartPhase(phaseCrossplane)
		installer := crossplane.NewInstaller(kubeClient)
		crossplaneCfg := cfg.Crossplane

		// Determine repository URL and name
		repoURL := crossplaneCfg.Repo
		if repoURL == "" {
			repoURL = crossplane.CrossplaneRepoURL
		}
		repoName := crossplane.CrossplaneRepoName
		if crossplaneCfg.Repo != "" {
			repoName = helm.GenerateRepoName(repoURL)
		}

		// Build installation steps
		steps := []string{
			"Adding Helm repository",
			"Creating namespace",
		}
		if crossplaneCfg.RegistryCaBundle != nil {
			steps = append(steps, "Creating registry CA bundle")
		}
		steps = append(steps, "Installing Helm chart")

		// Show multi-step progress for installation
		title := fmt.Sprintf("Installing Crossplane %s", crossplaneCfg.Version)
		if err := ui.RunProgressWithContext(ctx, title, steps, func(stepCtx context.Context, step string) error {
			switch step {
			case "Adding Helm repository":
				return installer.AddHelmRepo(stepCtx, repoName, repoURL)
			case "Creating namespace":
				return installer.EnsureNamespace(stepCtx)
			case "Creating registry CA bundle":
				return installer.CreateRegistryCaBundle(stepCtx, crossplaneCfg.RegistryCaBundle, cfg.Cluster.TrustedCAs.Workloads)
			case "Installing Helm chart":
				return installer.InstallHelmChart(stepCtx, crossplaneCfg, repoURL, repoName)
			default:
				return fmt.Errorf("unknown installation step: %s", step)
			}
		}); err != nil {
			showHelmDiagnostics(bc, "crossplane", crossplane.CrossplaneNamespace)
			return handleFailure(fmt.Errorf("failed to install Crossplane: %w", err))
		}

		// Show live pod status table while waiting for pods to be ready
		if err := ui.RunPodTable(ctx, "Waiting for Crossplane pods", func(pollCtx context.Context) ([]ui.PodInfo, bool, error) {
			podInfos, allReady, err := installer.GetPodStatus(pollCtx)
			if err != nil {
				return nil, false, err
			}

			// Convert crossplane.PodInfo to ui.PodInfo
			uiPodInfos := make([]ui.PodInfo, len(podInfos))
			for i, p := range podInfos {
				uiPodInfos[i] = ui.PodInfo{
					Name:    p.Name,
					Status:  p.Status,
					Ready:   p.Ready,
					Message: p.Message,
				}
			}

			return uiPodInfos, allReady, nil
		}); err != nil {
			showCrossplaneDiagnostics(bc)
			return handleFailure(err)
		}
		pt.CompletePhase()
	}

	// Phase: Install post-crossplane charts
	if pt.GetPhase(phasePostCrossplane) != nil {
		if err := installChartsForPhaseWithTracker(ctx, helmInstaller, config.ChartPhasePostCrossplane, bc, pt, phasePostCrossplane); err != nil {
			return handleFailure(err)
		}
	}

	// Phase: Install providers
	if pt.GetPhase(phaseProviders) != nil {
		pt.StartPhase(phaseProviders)
		providerNames := make([]string, len(cfg.Crossplane.Providers))
		providerMap := make(map[string]config.ProviderConfig)
		for i, p := range cfg.Crossplane.Providers {
			providerNames[i] = p.Name
			providerMap[p.Name] = p
		}

		// Install providers with animated progress bar
		installer := crossplane.NewInstaller(kubeClient)
		if err := ui.RunProgress("Installing providers", providerNames, func(name string) error {
			provider := providerMap[name]
			return installer.InstallProvider(ctx, provider.Name, provider.Package)
		}); err != nil {
			return handleFailure(err)
		}

		// Wait for providers to be healthy with animated table
		if err := ui.RunProviderTable(ctx, "Waiting for providers to be healthy", func(pollCtx context.Context) ([]ui.ProviderInfo, bool, error) {
			statuses, err := installer.GetProviderStatus(pollCtx)
			if err != nil {
				return nil, false, nil // Keep trying on error
			}

			providers := make([]ui.ProviderInfo, len(statuses))
			allHealthy := true
			for i, s := range statuses {
				providers[i] = ui.ProviderInfo{
					Name:    s.Name,
					Package: s.Package,
					Healthy: s.Healthy,
					Message: s.Message,
				}
				if !s.Healthy {
					allHealthy = false
				}
			}

			return providers, allHealthy && len(providers) > 0, nil
		}); err != nil {
			showProviderDiagnostics(bc)
			return handleFailure(err)
		}
		pt.CompletePhase()
	}

	// Phase: Install post-providers charts
	if pt.GetPhase(phasePostProviders) != nil {
		if err := installChartsForPhaseWithTracker(ctx, helmInstaller, config.ChartPhasePostProviders, bc, pt, phasePostProviders); err != nil {
			return handleFailure(err)
		}
	}

	// Phase: Install final charts
	if pt.GetPhase(phaseFinal) != nil {
		if err := installChartsForPhaseWithTracker(ctx, helmInstaller, config.ChartPhaseFinal, bc, pt, phaseFinal); err != nil {
			return handleFailure(err)
		}
	}

	// Phase: Apply compositions
	if pt.GetPhase(phaseCompositions) != nil {
		pt.StartPhase(phaseCompositions)
		installer := crossplane.NewInstaller(kubeClient)
		for _, source := range cfg.Compositions.Sources {
			if err := installer.ApplyCompositions(ctx, source); err != nil {
				return handleFailure(fmt.Errorf("failed to apply compositions from %s: %w", source.Path, err))
			}
		}
		pt.CompletePhaseWithMessage(fmt.Sprintf("%d source(s) applied", len(cfg.Compositions.Sources)))
	}

	// Success!
	pt.PrintSuccessWithHint("Cluster ready!", fmt.Sprintf("kubectl cluster-info --context %s", cfg.GetKubeContext()))

	return nil
}

// installChartsForPhase installs all charts configured for a specific phase
// installChartsForPhaseWithTracker installs charts and tracks progress with the PhaseTracker
func installChartsForPhaseWithTracker(ctx context.Context, helmInstaller *helm.Installer, phase string, bc *bootstrapContext, pt *ui.PhaseTracker, phaseName string) error {
	charts := getChartsForPhase(phase)
	if len(charts) == 0 {
		return nil
	}

	pt.StartPhase(phaseName)

	// Build chart names for progress display
	chartNames := make([]string, len(charts))
	chartMap := make(map[string]config.ChartConfig)
	for i, chart := range charts {
		chartNames[i] = chart.Name
		chartMap[chart.Name] = chart
	}

	// Track failed chart for diagnostics
	var failedChart config.ChartConfig

	// Install charts with animated progress bar
	title := fmt.Sprintf("Installing %s charts", phase)
	err := ui.RunProgress(title, chartNames, func(name string) error {
		chart := chartMap[name]
		if installErr := helmInstaller.InstallChartFromConfig(ctx, chart); installErr != nil {
			failedChart = chart
			return installErr
		}
		return nil
	})

	if err != nil {
		if failedChart.Name != "" {
			showChartDiagnostics(bc, failedChart)
		}
		return err
	}

	pt.CompletePhaseWithMessage(fmt.Sprintf("%d chart(s) installed", len(charts)))
	return nil
}

// getChartsForPhase returns all charts configured for a specific phase
func getChartsForPhase(phase string) []config.ChartConfig {
	var charts []config.ChartConfig
	for _, chart := range cfg.Charts {
		if chart.GetPhase() == phase {
			charts = append(charts, chart)
		}
	}
	return charts
}

// showCrossplaneDiagnostics shows diagnostics for Crossplane failures
func showCrossplaneDiagnostics(bc *bootstrapContext) {
	if bc == nil || bc.diagCollector == nil {
		return
	}

	diagCtx := diagnostics.DefaultContext(diagnostics.ComponentCrossplane)
	report, err := bc.diagCollector.Collect(bc.ctx, diagCtx)
	if err != nil {
		printVerbose("Failed to collect diagnostics: %v", err)
		return
	}

	report.Print(os.Stdout)
}

// showProviderDiagnostics shows diagnostics for provider failures
func showProviderDiagnostics(bc *bootstrapContext) {
	if bc == nil || bc.diagCollector == nil {
		return
	}

	diagCtx := diagnostics.DefaultContext(diagnostics.ComponentProviders)
	// Don't filter by label - we want all pods in crossplane-system
	diagCtx.LabelSelector = ""
	report, err := bc.diagCollector.Collect(bc.ctx, diagCtx)
	if err != nil {
		printVerbose("Failed to collect diagnostics: %v", err)
		return
	}

	report.Print(os.Stdout)
}

// showHelmDiagnostics shows diagnostics for Helm failures
func showHelmDiagnostics(bc *bootstrapContext, releaseName, namespace string) {
	if bc == nil || bc.diagCollector == nil {
		return
	}

	diagCtx := diagnostics.Context{
		Component:   diagnostics.ComponentHelm,
		Namespace:   namespace,
		ReleaseName: releaseName,
		MaxLogLines: 30,
	}

	report, err := bc.diagCollector.Collect(bc.ctx, diagCtx)
	if err != nil {
		printVerbose("Failed to collect diagnostics: %v", err)
		return
	}

	report.Print(os.Stdout)
}

// showChartDiagnostics shows diagnostics for chart installation failures
func showChartDiagnostics(bc *bootstrapContext, chart config.ChartConfig) {
	if bc == nil || bc.diagCollector == nil {
		return
	}

	namespace := chart.Namespace
	if namespace == "" {
		namespace = "default"
	}

	diagCtx := diagnostics.Context{
		Component:   diagnostics.ComponentHelm,
		Namespace:   namespace,
		ReleaseName: chart.Name,
		MaxLogLines: 30,
	}

	report, err := bc.diagCollector.Collect(bc.ctx, diagCtx)
	if err != nil {
		printVerbose("Failed to collect diagnostics: %v", err)
		return
	}

	report.Print(os.Stdout)
}

// createRegistryConfigMap creates the local-registry-hosting ConfigMap
// This follows the KEP-1755 standard for documenting local registries
// https://github.com/kubernetes/enhancements/tree/master/keps/sig-cluster-lifecycle/generic/1755-communicating-a-local-registry
func createRegistryConfigMap(ctx context.Context, client *kubernetes.Clientset, regCfg *config.RegistryConfig) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "local-registry-hosting",
			Namespace: "kube-public",
		},
		Data: map[string]string{
			"localRegistryHosting.v1": fmt.Sprintf(`host: "localhost:%d"
help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
`, regCfg.GetPort()),
		},
	}

	cmClient := client.CoreV1().ConfigMaps("kube-public")
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_, err := cmClient.Create(ctx, configMap, metav1.CreateOptions{})
		if err == nil {
			return nil
		}
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create registry ConfigMap: %w", err)
		}

		existing, getErr := cmClient.Get(ctx, configMap.Name, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("failed to get existing registry ConfigMap: %w", getErr)
		}
		configMap.ResourceVersion = existing.ResourceVersion
		_, err = cmClient.Update(ctx, configMap, metav1.UpdateOptions{})
		return err
	})
	return err
}
