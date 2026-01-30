package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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

// bootstrapMode controls how the bootstrap UI is rendered
type bootstrapMode int

const (
	// bootstrapModeDashboard uses the TUI dashboard (for TTY)
	bootstrapModeDashboard bootstrapMode = iota
	// bootstrapModePrint uses traditional print-based output (for non-TTY/CI)
	bootstrapModePrint
)

var (
	upSkipCrossplane    bool
	upSkipProviders     bool
	upSkipCharts        bool
	upSkipCompositions  bool
	upTimeout           time.Duration
	upRollbackOnFailure bool
	upShowValues        bool
	upPullImages        bool
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
	upCmd.Flags().BoolVar(&upShowValues, "show-values", false, "display merged Helm values before installation")
	upCmd.Flags().BoolVar(&upPullImages, "pull-images", false, "automatically pull missing images without prompting")
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
	phaseImageCache     = "Pre-load images"
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
	pt.AddPhaseIf(shouldPreloadImages(cfg), phaseImageCache)
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

// shouldPreloadImages determines if image pre-loading should be attempted.
// We run the phase whenever we are installing Crossplane and image cache is not
// explicitly disabled, so that "Pre-load images" appears in the UI and we
// attempt to load/push images (PreloadImages no-ops when there are 0 images).
func shouldPreloadImages(cfg *config.Config) bool {
	// Skip if Crossplane installation is skipped
	if upSkipCrossplane {
		return false
	}

	// Skip only if image cache is explicitly disabled
	if cfg.Crossplane.ImageCache != nil && !cfg.Crossplane.ImageCache.IsEnabled() {
		return false
	}

	// Run preload phase whenever cache is enabled (or unset, default enabled):
	// we have something to preload if any of these are true
	hasProviders := len(cfg.Crossplane.Providers) > 0 && (cfg.Crossplane.ImageCache == nil || cfg.Crossplane.ImageCache.ShouldPreloadProviders())
	hasCrossplane := cfg.Crossplane.ImageCache == nil || cfg.Crossplane.ImageCache.ShouldPreloadCrossplane()
	hasAdditional := cfg.Crossplane.ImageCache != nil && len(cfg.Crossplane.ImageCache.AdditionalImages) > 0

	if hasProviders || hasCrossplane || hasAdditional {
		return true
	}

	// Even with nothing to preload, show the phase when user explicitly enabled
	// image cache so the UI reflects their config (phase will no-op quickly)
	if cfg.Crossplane.ImageCache != nil {
		return true
	}

	return false
}

func runUp(cmd *cobra.Command, args []string) error {
	// Require config
	requireConfig()

	// Determine bootstrap mode based on TTY
	mode := bootstrapModePrint
	if ui.IsTTY() {
		mode = bootstrapModeDashboard
	}

	// Build phase tracker
	pt := buildPhases()

	if mode == bootstrapModeDashboard {
		return runUpDashboard(pt)
	}
	return runUpPrint(pt)
}

// runUpDashboard runs the bootstrap with the TUI dashboard
func runUpDashboard(pt *ui.PhaseTracker) error {
	ctx := context.Background()

	// Build the next step hint
	nextStepHint := fmt.Sprintf("kubectl cluster-info --context %s", cfg.GetKubeContext())

	result, err := ui.RunBootstrapDashboard(
		ctx,
		pt,
		func(ctx context.Context, ctrl *ui.DashboardController) error {
			return executeBootstrap(ctx, pt, ctrl)
		},
		ui.WithTimeout(upTimeout),
		ui.WithExtendAmount(5*time.Minute),
		ui.WithNextStepHint(nextStepHint),
	)

	if err != nil {
		return err
	}

	if !result.Success {
		// Print final summary after dashboard exits
		pt.PrintSummary()
		if result.Error != nil {
			return result.Error
		}
		return fmt.Errorf("%s", result.Message)
	}

	// Success - the completion view already showed the success message
	return nil
}

// runUpPrint runs the bootstrap with traditional print-based output (non-TTY/CI)
func runUpPrint(pt *ui.PhaseTracker) error {
	ctx, cancel := context.WithTimeout(context.Background(), upTimeout)
	defer cancel()

	// Print header with phase list
	pt.PrintHeader()

	// Execute bootstrap with nil controller (print mode)
	if err := executeBootstrap(ctx, pt, nil); err != nil {
		pt.PrintSummary()
		return err
	}

	// Success!
	pt.PrintSuccessWithHint("Cluster ready!", fmt.Sprintf("kubectl cluster-info --context %s", cfg.GetKubeContext()))
	return nil
}

// createValuesLogger creates a ValuesLogger based on the current mode.
// If upShowValues is false, returns nil (no logging).
// If ctrl is non-nil (dashboard mode), logs to the dashboard log buffer.
// Otherwise (print mode), prints to stdout with styled output.
func createValuesLogger(ctrl *ui.DashboardController) helm.ValuesLogger {
	if !upShowValues {
		return nil
	}

	return func(releaseName string, values map[string]interface{}) {
		valuesYAML, err := yaml.Marshal(values)
		if err != nil {
			return
		}

		if ctrl != nil {
			// Dashboard mode: add to log buffer
			ctrl.Log(fmt.Sprintf("Merged values for %s:", releaseName))
			for _, line := range strings.Split(string(valuesYAML), "\n") {
				if line != "" {
					ctrl.Log("  " + line)
				}
			}
		} else {
			// Print mode: styled output
			fmt.Println(ui.Info("Merged values for %s:", releaseName))
			fmt.Println(ui.Code(string(valuesYAML)))
		}
	}
}

// executeBootstrap performs the actual bootstrap operations
// If ctrl is non-nil, sends updates to the dashboard; otherwise uses print-based output
func executeBootstrap(ctx context.Context, pt *ui.PhaseTracker, ctrl *ui.DashboardController) error {
	// Track if we created the cluster (for rollback)
	clusterCreated := false

	// Bootstrap context for shared resources
	var bc *bootstrapContext

	// Helper to handle phase failure
	handleFailure := func(phaseName string, err error) error {
		if ctrl != nil {
			ctrl.FailPhase(phaseName, err)
		} else {
			pt.FailPhase(err)
		}
		if upRollbackOnFailure && clusterCreated {
			if ctrl != nil {
				ctrl.Log("Rolling back: deleting cluster...")
			} else {
				fmt.Println(ui.Warning("Rolling back: deleting cluster..."))
			}
			if delErr := kind.DeleteCluster(ctx, cfg.Cluster.Name); delErr != nil {
				if ctrl != nil {
					ctrl.Log(fmt.Sprintf("Failed to delete cluster: %v", delErr))
				} else {
					fmt.Println(ui.Error("Failed to delete cluster during rollback: %v", delErr))
				}
			} else {
				if ctrl != nil {
					ctrl.Log("Cluster deleted")
				} else {
					fmt.Println(ui.Info("Cluster deleted"))
				}
			}
		}
		return err
	}

	// Helper to start a phase
	startPhase := func(name string) {
		if ctrl != nil {
			ctrl.StartPhase(name)
		} else {
			pt.StartPhase(name)
		}
	}

	// Helper to complete a phase
	completePhase := func(name string, message string) {
		if ctrl != nil {
			ctrl.CompletePhase(name, message)
		} else {
			if message != "" {
				pt.CompletePhaseWithMessage(message)
			} else {
				pt.CompletePhase()
			}
		}
	}

	// Helper to skip a phase
	skipPhase := func(name string, reason string) {
		if ctrl != nil {
			ctrl.SkipPhase(name, reason)
		} else {
			pt.SkipPhase(name, reason)
		}
	}

	// Helper to update operation status (dashboard only)
	updateOp := func(step string, progress float64) {
		if ctrl != nil {
			ctrl.UpdateOperation(step, progress)
		}
	}

	// Helper to log a message
	log := func(msg string) {
		if ctrl != nil {
			ctrl.Log(msg)
		}
	}

	// Phase: Create local registry
	var registryManager *registry.Manager
	if cfg.Cluster.Registry.Enabled {
		startPhase(phaseRegistry)
		updateOp("Creating registry container...", -1)
		registryManager = registry.NewManager(&cfg.Cluster.Registry)
		if err := registryManager.Create(ctx); err != nil {
			return handleFailure(phaseRegistry, fmt.Errorf("failed to create registry: %w", err))
		}
		completePhase(phaseRegistry, fmt.Sprintf("Registry available at localhost:%d", cfg.Cluster.Registry.GetPort()))
	}

	// Phase: Configure trusted CAs
	if kind.HasTrustedCAs(cfg) {
		startPhase(phaseTrustedCAs)
		updateOp("Validating CA certificates...", -1)
		var caSummary *kind.TrustedCAsSummary
		var validateErr error
		if ctrl != nil {
			// Dashboard mode: execute directly
			caSummary, validateErr = kind.ValidateTrustedCAs(cfg)
		} else {
			// Print mode: use spinner
			validateErr = ui.RunSpinnerWithContext(ctx, "Validating CA certificates", func(spinnerCtx context.Context) error {
				var err error
				caSummary, err = kind.ValidateTrustedCAs(cfg)
				return err
			})
		}
		if validateErr != nil {
			return handleFailure(phaseTrustedCAs, fmt.Errorf("failed to configure trusted CAs: %w", validateErr))
		}
		completePhase(phaseTrustedCAs, fmt.Sprintf("%d registry CA(s), %d workload CA(s)", caSummary.RegistryCount, caSummary.WorkloadCount))
	}

	// Phase: Create Kind cluster
	startPhase(phaseCluster)
	exists, err := kind.ClusterExists(cfg.Cluster.Name)
	if err != nil {
		return handleFailure(phaseCluster, fmt.Errorf("failed to check cluster status: %w", err))
	}

	if exists {
		skipPhase(phaseCluster, "already exists")
	} else {
		// Display cluster configuration details
		nodeImage, imageSource := kind.GetNodeImage(cfg)
		if ctrl != nil {
			if nodeImage != "" {
				log(fmt.Sprintf("Node Image: %s (%s)", nodeImage, imageSource))
			} else {
				log(fmt.Sprintf("Node Image: Kind default (%s)", imageSource))
			}
		} else {
			if nodeImage != "" {
				fmt.Println(ui.KeyValueIndented("Node Image", ui.Code(nodeImage), 2))
				fmt.Println(ui.KeyValueIndented("Source", imageSource, 2))
			} else {
				fmt.Println(ui.KeyValueIndented("Node Image", ui.Muted("Kind default"), 2))
				fmt.Println(ui.KeyValueIndented("Source", imageSource, 2))
			}
			fmt.Println()
		}

		// Create cluster with progress updates
		if ctrl != nil {
			// Dashboard mode: send updates via controller
			logger := ui.NewKindDashboardLogger(func(step string) {
				updateOp(step, -1)
			})
			if err := kind.CreateClusterWithProgress(ctx, cfg, logger); err != nil {
				return handleFailure(phaseCluster, fmt.Errorf("failed to create cluster: %w", err))
			}
		} else {
			// Print mode: use multi-step UI
			if err := ui.RunClusterCreate(ctx, cfg.Cluster.Name, func(ctx context.Context, updates chan<- ui.StepUpdate, done <-chan struct{}) error {
				logger := ui.NewKindLogger(updates, done)
				return kind.CreateClusterWithProgress(ctx, cfg, logger)
			}); err != nil {
				return handleFailure(phaseCluster, fmt.Errorf("failed to create cluster: %w", err))
			}
		}
		clusterCreated = true
		completePhase(phaseCluster, "")

		// Update system CA certificates if workload CAs are configured
		if len(cfg.Cluster.TrustedCAs.Workloads) > 0 {
			updateOp("Updating system CA certificates on nodes...", -1)
			var updateCAErr error
			if ctrl != nil {
				updateCAErr = kind.UpdateCACertificates(ctx, cfg.Cluster.Name)
			} else {
				updateCAErr = ui.RunSpinnerWithContext(ctx, "Updating system CA certificates on nodes", func(spinnerCtx context.Context) error {
					return kind.UpdateCACertificates(spinnerCtx, cfg.Cluster.Name)
				})
			}
			if updateCAErr != nil {
				return handleFailure(phaseCluster, fmt.Errorf("failed to update CA certificates: %w", updateCAErr))
			}
		}
	}

	// Configure registry for cluster nodes if enabled
	if cfg.Cluster.Registry.Enabled && registryManager != nil {
		updateOp("Configuring registry on cluster nodes...", -1)
		var configErr error
		if ctrl != nil {
			if err := registryManager.ConfigureNodes(ctx, cfg.Cluster.Name); err != nil {
				configErr = err
			} else {
				configErr = registryManager.ConnectToNetwork(ctx, "kind")
			}
		} else {
			configErr = ui.RunSpinnerWithContext(ctx, "Configuring registry on cluster nodes", func(spinnerCtx context.Context) error {
				if err := registryManager.ConfigureNodes(spinnerCtx, cfg.Cluster.Name); err != nil {
					return err
				}
				return registryManager.ConnectToNetwork(spinnerCtx, "kind")
			})
		}
		if configErr != nil {
			return handleFailure(phaseCluster, fmt.Errorf("failed to configure registry: %w", configErr))
		}
	}

	// Phase: Pre-load images from local Docker
	if pt.GetPhase(phaseImageCache) != nil {
		startPhase(phaseImageCache)
		updateOp("Checking for local images to pre-load...", -1)

		var result *kind.PreloadResult
		var imageCacheErr error

		// First attempt: Check for local images
		imageCacheFn := func(spinnerCtx context.Context) error {
			var err error
			result, err = kind.PreloadImages(spinnerCtx, cfg.Cluster.Name, cfg, func(msg string) {
				if ctrl != nil {
					updateOp(msg, -1)
				} else {
					log(msg)
				}
			})
			return err
		}

		if ctrl != nil {
			imageCacheErr = imageCacheFn(ctx)
		} else {
			imageCacheErr = ui.RunSpinnerWithContext(ctx, "Pre-loading images", imageCacheFn)
		}

		// If no images were loaded but some are missing, offer to pull them
		if imageCacheErr == nil && result != nil && result.LoadedCount == 0 && len(result.MissingImages) > 0 {
			shouldPull := upPullImages // Check flag first

			// If flag not set, prompt in TTY mode
			if !shouldPull && ui.IsTTY() {
				// Show the list of missing images
				log("\nNo local images found. The following images can be pulled for faster future bootstraps:\n")
				for _, img := range result.MissingImages {
					log(fmt.Sprintf("  - %s", img))
				}
				log("") // Empty line

				confirm, err := ui.ConfirmWithContext(ctx, fmt.Sprintf("Pull %d images now?", len(result.MissingImages)))
				if err == nil {
					shouldPull = confirm
				}
			}

			// Pull images if confirmed or flag set
			if shouldPull {
				updateOp("Pulling missing images...", -1)
				pulledCount, pullErr := kind.PullImages(ctx, result.MissingImages, func(msg string) {
					if ctrl != nil {
						updateOp(msg, -1)
					} else {
						log(msg)
					}
				})

				if pullErr != nil {
					log(fmt.Sprintf("Warning: Image pulling encountered issues: %v", pullErr))
				}

				// If we successfully pulled any images, try loading them again
				if pulledCount > 0 {
					updateOp("Loading pulled images into cluster...", -1)
					reloadFn := func(spinnerCtx context.Context) error {
						_, err := kind.PreloadImages(spinnerCtx, cfg.Cluster.Name, cfg, func(msg string) {
							if ctrl != nil {
								updateOp(msg, -1)
							} else {
								log(msg)
							}
						})
						return err
					}

					if ctrl != nil {
						_ = reloadFn(ctx)
					} else {
						_ = ui.RunSpinnerWithContext(ctx, "Loading images", reloadFn)
					}
				}
			}
		}

		// Don't fail bootstrap if image caching fails - just log warning
		if imageCacheErr != nil {
			log(fmt.Sprintf("Warning: Image pre-loading encountered issues: %v", imageCacheErr))
		}
		completePhase(phaseImageCache, "")
	}

	// Phase: Connect to cluster
	startPhase(phaseConnect)
	updateOp("Connecting to cluster...", -1)
	var kubeClient *kubernetes.Clientset
	var dynamicClient dynamic.Interface
	var connectErr error

	connectFn := func() error {
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
	}

	if ctrl != nil {
		connectErr = connectFn()
	} else {
		connectErr = ui.RunSpinnerWithContext(ctx, "Connecting to cluster", func(spinnerCtx context.Context) error {
			return connectFn()
		})
	}
	if connectErr != nil {
		return handleFailure(phaseConnect, connectErr)
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
	completePhase(phaseConnect, "")

	// Create local registry ConfigMap for discovery
	if cfg.Cluster.Registry.Enabled && registryManager != nil {
		if err := createRegistryConfigMap(ctx, kubeClient, &cfg.Cluster.Registry); err != nil {
			log(fmt.Sprintf("Warning: Failed to create registry ConfigMap: %v", err))
		}
	}

	// Create Helm installer for chart installations
	helmInstaller := helm.NewInstaller(kubeClient)

	// Phase: Install pre-crossplane charts
	if pt.GetPhase(phasePreCrossplane) != nil {
		if err := installChartsForPhaseWithTracker(ctx, helmInstaller, config.ChartPhasePrecrossplane, bc, pt, phasePreCrossplane, ctrl); err != nil {
			return handleFailure(phasePreCrossplane, err)
		}
	}

	// Phase: Install Crossplane
	if !upSkipCrossplane {
		startPhase(phaseCrossplane)
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

		// Create values logger for displaying merged values
		valuesLogger := createValuesLogger(ctrl)

		// Build installation steps
		steps := []string{
			"Adding Helm repository",
			"Creating namespace",
		}
		if crossplaneCfg.RegistryCaBundle != nil {
			steps = append(steps, "Creating registry CA bundle")
		}
		steps = append(steps, "Installing Helm chart")

		// Execute installation steps
		title := fmt.Sprintf("Installing Crossplane %s", crossplaneCfg.Version)
		var installErr error

		if ctrl != nil {
			// Dashboard mode: execute steps directly with updates
			for i, step := range steps {
				updateOp(step, float64(i)/float64(len(steps)))
				switch step {
				case "Adding Helm repository":
					installErr = installer.AddHelmRepo(ctx, repoName, repoURL)
				case "Creating namespace":
					installErr = installer.EnsureNamespace(ctx)
				case "Creating registry CA bundle":
					installErr = installer.CreateRegistryCaBundle(ctx, crossplaneCfg.RegistryCaBundle, cfg.Cluster.TrustedCAs.Workloads)
				case "Installing Helm chart":
					opts := helm.InstallOptions{
						ValuesLogger: valuesLogger,
					}
					installErr = installer.InstallHelmChartWithOptions(ctx, crossplaneCfg, repoURL, repoName, opts)
				}
				if installErr != nil {
					break
				}
			}
		} else {
			// Print mode: use progress bar
			installErr = ui.RunProgressWithContext(ctx, title, steps, func(stepCtx context.Context, step string) error {
				switch step {
				case "Adding Helm repository":
					return installer.AddHelmRepo(stepCtx, repoName, repoURL)
				case "Creating namespace":
					return installer.EnsureNamespace(stepCtx)
				case "Creating registry CA bundle":
					return installer.CreateRegistryCaBundle(stepCtx, crossplaneCfg.RegistryCaBundle, cfg.Cluster.TrustedCAs.Workloads)
				case "Installing Helm chart":
					opts := helm.InstallOptions{
						ValuesLogger: valuesLogger,
					}
					return installer.InstallHelmChartWithOptions(stepCtx, crossplaneCfg, repoURL, repoName, opts)
				default:
					return fmt.Errorf("unknown installation step: %s", step)
				}
			})
		}

		if installErr != nil {
			showHelmDiagnostics(bc, "crossplane", crossplane.CrossplaneNamespace)
			return handleFailure(phaseCrossplane, fmt.Errorf("failed to install Crossplane: %w", installErr))
		}

		// Wait for pods to be ready
		updateOp("Waiting for Crossplane pods...", -1)
		var podErr error
		if ctrl != nil {
			// Dashboard mode: poll directly
			podErr = waitForCrossplanePods(ctx, installer, ctrl)
		} else {
			// Print mode: use pod table
			podErr = ui.RunPodTable(ctx, "Waiting for Crossplane pods", func(pollCtx context.Context) ([]ui.PodInfo, bool, error) {
				podInfos, allReady, err := installer.GetPodStatus(pollCtx)
				if err != nil {
					return nil, false, err
				}

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
			})
		}
		if podErr != nil {
			showCrossplaneDiagnostics(bc)
			return handleFailure(phaseCrossplane, podErr)
		}
		completePhase(phaseCrossplane, "")
	}

	// Phase: Install post-crossplane charts
	if pt.GetPhase(phasePostCrossplane) != nil {
		if err := installChartsForPhaseWithTracker(ctx, helmInstaller, config.ChartPhasePostCrossplane, bc, pt, phasePostCrossplane, ctrl); err != nil {
			return handleFailure(phasePostCrossplane, err)
		}
	}

	// Phase: Install providers
	if pt.GetPhase(phaseProviders) != nil {
		startPhase(phaseProviders)
		providerNames := make([]string, len(cfg.Crossplane.Providers))
		providerMap := make(map[string]config.ProviderConfig)
		for i, p := range cfg.Crossplane.Providers {
			providerNames[i] = p.Name
			providerMap[p.Name] = p
		}

		// Install providers
		installer := crossplane.NewInstaller(kubeClient)
		var providerErr error

		if ctrl != nil {
			// Dashboard mode: install with updates
			for i, name := range providerNames {
				updateOp(fmt.Sprintf("Installing %s...", name), float64(i)/float64(len(providerNames)))
				provider := providerMap[name]
				if err := installer.InstallProvider(ctx, provider.Name, provider.Package); err != nil {
					providerErr = err
					break
				}
			}
		} else {
			// Print mode: use progress bar
			providerErr = ui.RunProgress("Installing providers", providerNames, func(name string) error {
				provider := providerMap[name]
				return installer.InstallProvider(ctx, provider.Name, provider.Package)
			})
		}

		if providerErr != nil {
			return handleFailure(phaseProviders, providerErr)
		}

		// Wait for providers to be healthy
		updateOp("Waiting for providers to be healthy...", -1)
		var healthErr error
		if ctrl != nil {
			healthErr = waitForProviders(ctx, installer, ctrl)
		} else {
			healthErr = ui.RunProviderTable(ctx, "Waiting for providers to be healthy", func(pollCtx context.Context) ([]ui.ProviderInfo, bool, error) {
				statuses, err := installer.GetProviderStatus(pollCtx)
				if err != nil {
					return nil, false, nil
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
			})
		}
		if healthErr != nil {
			showProviderDiagnostics(bc)
			return handleFailure(phaseProviders, healthErr)
		}
		completePhase(phaseProviders, fmt.Sprintf("%d provider(s) healthy", len(providerNames)))
	}

	// Phase: Install post-providers charts
	if pt.GetPhase(phasePostProviders) != nil {
		if err := installChartsForPhaseWithTracker(ctx, helmInstaller, config.ChartPhasePostProviders, bc, pt, phasePostProviders, ctrl); err != nil {
			return handleFailure(phasePostProviders, err)
		}
	}

	// Phase: Install final charts
	if pt.GetPhase(phaseFinal) != nil {
		if err := installChartsForPhaseWithTracker(ctx, helmInstaller, config.ChartPhaseFinal, bc, pt, phaseFinal, ctrl); err != nil {
			return handleFailure(phaseFinal, err)
		}
	}

	// Phase: Apply compositions
	if pt.GetPhase(phaseCompositions) != nil {
		startPhase(phaseCompositions)
		installer := crossplane.NewInstaller(kubeClient)
		for _, source := range cfg.Compositions.Sources {
			updateOp(fmt.Sprintf("Applying %s...", source.Path), -1)
			if err := installer.ApplyCompositions(ctx, source); err != nil {
				return handleFailure(phaseCompositions, fmt.Errorf("failed to apply compositions from %s: %w", source.Path, err))
			}
		}
		completePhase(phaseCompositions, fmt.Sprintf("%d source(s) applied", len(cfg.Compositions.Sources)))
	}

	return nil
}

// waitForCrossplanePods waits for Crossplane pods to be ready (dashboard mode)
func waitForCrossplanePods(ctx context.Context, installer *crossplane.Installer, ctrl *ui.DashboardController) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			podInfos, allReady, err := installer.GetPodStatus(ctx)
			if err != nil {
				ctrl.Log(fmt.Sprintf("Error checking pods: %v", err))
				continue
			}

			// Convert to ui.PodInfo and send update
			uiPods := make([]ui.PodInfo, len(podInfos))
			for i, p := range podInfos {
				uiPods[i] = ui.PodInfo{
					Name:    p.Name,
					Status:  p.Status,
					Ready:   p.Ready,
					Message: p.Message,
				}
			}
			ctrl.UpdatePods(uiPods)

			// Log current status
			for _, p := range podInfos {
				if !p.Ready {
					ctrl.UpdateOperation(fmt.Sprintf("Waiting for %s (%s)", p.Name, p.Status), -1)
					break
				}
			}

			if allReady && len(podInfos) > 0 {
				return nil
			}
		}
	}
}

// waitForProviders waits for providers to be healthy (dashboard mode)
func waitForProviders(ctx context.Context, installer *crossplane.Installer, ctrl *ui.DashboardController) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			statuses, err := installer.GetProviderStatus(ctx)
			if err != nil {
				continue
			}

			// Convert provider statuses to PodInfo for display
			uiPods := make([]ui.PodInfo, len(statuses))
			allHealthy := true
			for i, s := range statuses {
				status := "Pending"
				if s.Healthy {
					status = "Running"
				} else if s.Message != "" {
					status = "Installing"
				}
				uiPods[i] = ui.PodInfo{
					Name:    s.Name,
					Status:  status,
					Ready:   s.Healthy,
					Message: s.Message,
				}
				if !s.Healthy {
					allHealthy = false
				}
			}
			ctrl.UpdatePods(uiPods)

			// Update operation status
			for _, s := range statuses {
				if !s.Healthy {
					ctrl.UpdateOperation(fmt.Sprintf("Waiting for %s...", s.Name), -1)
					break
				}
			}

			if allHealthy && len(statuses) > 0 {
				return nil
			}
		}
	}
}

// installChartsForPhase installs all charts configured for a specific phase
// installChartsForPhaseWithTracker installs charts and tracks progress with the PhaseTracker
func installChartsForPhaseWithTracker(ctx context.Context, helmInstaller *helm.Installer, phase string, bc *bootstrapContext, pt *ui.PhaseTracker, phaseName string, ctrl *ui.DashboardController) error {
	charts := getChartsForPhase(phase)
	if len(charts) == 0 {
		return nil
	}

	// Start phase
	if ctrl != nil {
		ctrl.StartPhase(phaseName)
	} else {
		pt.StartPhase(phaseName)
	}

	// Create values logger for displaying merged values
	valuesLogger := createValuesLogger(ctrl)

	// Build chart names for progress display
	chartNames := make([]string, len(charts))
	chartMap := make(map[string]config.ChartConfig)
	for i, chart := range charts {
		chartNames[i] = chart.Name
		chartMap[chart.Name] = chart
	}

	// Track failed chart for diagnostics
	var failedChart config.ChartConfig
	var installErr error

	// Create install options with values logger
	opts := helm.InstallOptions{
		ValuesLogger: valuesLogger,
	}

	if ctrl != nil {
		// Dashboard mode: install with updates
		for i, name := range chartNames {
			ctrl.UpdateOperation(fmt.Sprintf("Installing %s...", name), float64(i)/float64(len(chartNames)))
			chart := chartMap[name]
			if err := helmInstaller.InstallChartFromConfigWithOptions(ctx, chart, opts); err != nil {
				failedChart = chart
				installErr = err
				break
			}
		}
	} else {
		// Print mode: use progress bar
		title := fmt.Sprintf("Installing %s charts", phase)
		installErr = ui.RunProgress(title, chartNames, func(name string) error {
			chart := chartMap[name]
			if err := helmInstaller.InstallChartFromConfigWithOptions(ctx, chart, opts); err != nil {
				failedChart = chart
				return err
			}
			return nil
		})
	}

	if installErr != nil {
		if failedChart.Name != "" {
			showChartDiagnostics(bc, failedChart)
		}
		return installErr
	}

	// Complete phase
	message := fmt.Sprintf("%d chart(s) installed", len(charts))
	if ctrl != nil {
		ctrl.CompletePhase(phaseName, message)
	} else {
		pt.CompletePhaseWithMessage(message)
	}
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
