package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/kanzi/kindplane/internal/config"
	"github.com/kanzi/kindplane/internal/crossplane"
	"github.com/kanzi/kindplane/internal/diagnostics"
	"github.com/kanzi/kindplane/internal/eso"
	"github.com/kanzi/kindplane/internal/helm"
	"github.com/kanzi/kindplane/internal/kind"
)

var (
	upSkipCrossplane    bool
	upSkipProviders     bool
	upSkipESO           bool
	upSkipCharts        bool
	upSkipCompositions  bool
	upTimeout           time.Duration
	upRollbackOnFailure bool
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Create and bootstrap a Kind cluster with Crossplane",
	Long: `Create a Kind cluster and bootstrap it with Crossplane, providers,
External Secrets Operator, Helm charts, and custom compositions.

This command requires a kindplane.yaml configuration file.
Run 'kindplane init' first if you don't have one.

The bootstrap process:
  1. Create Kind cluster
  2. Install charts with phase: pre-crossplane
  3. Install Crossplane
  4. Install charts with phase: post-crossplane
  5. Install Crossplane providers
  6. Install charts with phase: post-providers
  7. Install External Secrets Operator (if enabled)
  8. Install charts with phase: post-eso (default)
  9. Apply custom compositions (if configured)

Examples:
  # Create cluster with full bootstrap
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
	upCmd.Flags().BoolVar(&upSkipESO, "skip-eso", false, "skip External Secrets Operator installation")
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

func runUp(cmd *cobra.Command, args []string) error {
	// Require config
	requireConfig()

	ctx, cancel := context.WithTimeout(context.Background(), upTimeout)
	defer cancel()

	// Track if we created the cluster (for rollback)
	clusterCreated := false

	// Bootstrap context for shared resources
	var bc *bootstrapContext

	// Rollback function
	rollback := func() {
		if upRollbackOnFailure && clusterCreated {
			printWarn("Rolling back: deleting cluster...")
			if err := kind.DeleteCluster(ctx, cfg.Cluster.Name); err != nil {
				printError("Failed to delete cluster during rollback: %v", err)
			} else {
				printInfo("Cluster deleted")
			}
		}
	}

	// Step 1: Create Kind cluster
	printInfo("Creating Kind cluster '%s'...", cfg.Cluster.Name)

	exists, err := kind.ClusterExists(cfg.Cluster.Name)
	if err != nil {
		printError("Failed to check cluster status: %v", err)
		return err
	}

	if exists {
		printWarn("Cluster '%s' already exists, skipping creation", cfg.Cluster.Name)
	} else {
		if err := kind.CreateCluster(ctx, cfg); err != nil {
			printError("Failed to create cluster: %v", err)
			return err
		}
		clusterCreated = true
		printSuccess("Kind cluster created")
	}

	// Get kubernetes client
	kubeClient, err := kind.GetKubeClient(cfg.Cluster.Name)
	if err != nil {
		printError("Failed to get kubernetes client: %v", err)
		rollback()
		return err
	}

	// Get dynamic client for diagnostics
	restConfig, err := kind.GetRESTConfig(cfg.Cluster.Name)
	if err != nil {
		printError("Failed to get REST config: %v", err)
		rollback()
		return err
	}

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		printError("Failed to create dynamic client: %v", err)
		rollback()
		return err
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

	// Create Helm installer for chart installations
	helmInstaller := helm.NewInstaller(kubeClient)

	// Step 2: Install pre-crossplane charts
	if !upSkipCharts {
		if err := installChartsForPhase(ctx, helmInstaller, config.ChartPhasePrecrossplane, rollback, bc); err != nil {
			return err
		}
	}

	// Step 3: Install Crossplane
	if !upSkipCrossplane {
		printInfo("Installing Crossplane %s...", cfg.Crossplane.Version)
		installer := crossplane.NewInstaller(kubeClient)
		if err := installer.Install(ctx, cfg.Crossplane.Version); err != nil {
			printError("Failed to install Crossplane: %v", err)
			showHelmDiagnostics(bc, "crossplane", crossplane.CrossplaneNamespace)
			rollback()
			return err
		}
		printSuccess("Crossplane installed")

		// Wait for Crossplane to be ready
		printInfo("Waiting for Crossplane to be ready...")
		if err := installer.WaitForReady(ctx); err != nil {
			printError("Crossplane failed to become ready: %v", err)
			showCrossplaneDiagnostics(bc)
			rollback()
			return err
		}
		printSuccess("Crossplane is ready")
	}

	// Step 4: Install post-crossplane charts
	if !upSkipCharts {
		if err := installChartsForPhase(ctx, helmInstaller, config.ChartPhasePostCrossplane, rollback, bc); err != nil {
			return err
		}
	}

	// Step 5: Install providers
	if !upSkipProviders && !upSkipCrossplane {
		for _, provider := range cfg.Crossplane.Providers {
			printInfo("Installing %s (%s)...", provider.Name, provider.Package)
			installer := crossplane.NewInstaller(kubeClient)
			if err := installer.InstallProvider(ctx, provider.Name, provider.Package); err != nil {
				printError("Failed to install provider %s: %v", provider.Name, err)
				rollback()
				return err
			}
			printSuccess("Provider %s installed", provider.Name)
		}

		// Wait for providers to be healthy
		if len(cfg.Crossplane.Providers) > 0 {
			printInfo("Waiting for providers to be healthy...")
			installer := crossplane.NewInstaller(kubeClient)
			if err := installer.WaitForProviders(ctx); err != nil {
				printError("Providers failed to become healthy: %v", err)
				showProviderDiagnostics(bc)
				rollback()
				return err
			}
			printSuccess("All providers are healthy")
		}
	}

	// Step 6: Install post-providers charts
	if !upSkipCharts {
		if err := installChartsForPhase(ctx, helmInstaller, config.ChartPhasePostProviders, rollback, bc); err != nil {
			return err
		}
	}

	// Step 7: Install External Secrets Operator
	if !upSkipESO && cfg.ESO.Enabled {
		printInfo("Installing External Secrets Operator %s...", cfg.ESO.Version)
		esoInstaller := eso.NewInstaller(kubeClient)
		if err := esoInstaller.Install(ctx, cfg.ESO.Version); err != nil {
			printError("Failed to install ESO: %v", err)
			showHelmDiagnostics(bc, "external-secrets", eso.ESONamespace)
			rollback()
			return err
		}
		printSuccess("External Secrets Operator installed")

		// Wait for ESO to be ready
		printInfo("Waiting for ESO to be ready...")
		if err := esoInstaller.WaitForReady(ctx); err != nil {
			printError("ESO failed to become ready: %v", err)
			showESODiagnostics(bc)
			rollback()
			return err
		}
		printSuccess("ESO is ready")
	}

	// Step 8: Install post-eso charts (default phase)
	if !upSkipCharts {
		if err := installChartsForPhase(ctx, helmInstaller, config.ChartPhasePostESO, rollback, bc); err != nil {
			return err
		}
	}

	// Step 9: Apply compositions
	if !upSkipCompositions && len(cfg.Compositions.Sources) > 0 {
		printInfo("Applying compositions...")
		installer := crossplane.NewInstaller(kubeClient)
		for _, source := range cfg.Compositions.Sources {
			if err := installer.ApplyCompositions(ctx, source); err != nil {
				printError("Failed to apply compositions from %s: %v", source.Path, err)
				rollback()
				return err
			}
			printSuccess("Compositions applied from %s", source.Path)
		}
	}

	// Success!
	fmt.Println()
	printSuccess("Cluster ready!")
	fmt.Println()
	printStep("kubectl cluster-info --context %s", cfg.GetKubeContext())
	fmt.Println()

	return nil
}

// installChartsForPhase installs all charts configured for a specific phase
func installChartsForPhase(ctx context.Context, helmInstaller *helm.Installer, phase string, rollback func(), bc *bootstrapContext) error {
	charts := getChartsForPhase(phase)
	if len(charts) == 0 {
		return nil
	}

	printInfo("Installing %s charts...", phase)
	for _, chart := range charts {
		printInfo("  Installing chart %s (%s/%s)...", chart.Name, chart.Repo, chart.Chart)
		if err := helmInstaller.InstallChartFromConfig(ctx, chart); err != nil {
			printError("Failed to install chart %s: %v", chart.Name, err)
			showChartDiagnostics(bc, chart)
			rollback()
			return err
		}
		printSuccess("  Chart %s installed", chart.Name)
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

// showESODiagnostics shows diagnostics for ESO failures
func showESODiagnostics(bc *bootstrapContext) {
	if bc == nil || bc.diagCollector == nil {
		return
	}

	diagCtx := diagnostics.DefaultContext(diagnostics.ComponentESO)
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
