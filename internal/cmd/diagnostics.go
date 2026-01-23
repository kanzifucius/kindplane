package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/client-go/dynamic"

	"github.com/kanzi/kindplane/internal/diagnostics"
	"github.com/kanzi/kindplane/internal/kind"
	"github.com/kanzi/kindplane/internal/ui"
)

var (
	diagComponent string
	diagNamespace string
	diagRelease   string
	diagMaxLogs   int64
	diagTimeout   time.Duration
)

var diagnosticsCmd = &cobra.Command{
	Use:   "diagnostics",
	Short: "Run diagnostics on cluster components",
	Long: `Collect and display diagnostic information for cluster components.

This command gathers detailed information about pods, providers, and
Helm releases to help troubleshoot issues.`,
	Example: `  # Run diagnostics for all Crossplane components
  kindplane diagnostics

  # Run diagnostics for a specific component
  kindplane diagnostics --component providers

  # Run diagnostics for a specific Helm release
  kindplane diagnostics --component helm --release nginx-ingress --namespace ingress-nginx

  # Customise log output
  kindplane diagnostics --max-logs 50`,
	RunE: runDiagnostics,
}

func init() {
	diagnosticsCmd.Flags().StringVar(&diagComponent, "component", "", "Component to diagnose (crossplane, providers, eso, helm)")
	diagnosticsCmd.Flags().StringVarP(&diagNamespace, "namespace", "n", "", "Namespace to inspect")
	diagnosticsCmd.Flags().StringVar(&diagRelease, "release", "", "Helm release name (for helm component)")
	diagnosticsCmd.Flags().Int64Var(&diagMaxLogs, "max-logs", 30, "Maximum number of log lines per container")
	diagnosticsCmd.Flags().DurationVar(&diagTimeout, "timeout", 30*time.Second, "Timeout for diagnostics collection")
}

func runDiagnostics(cmd *cobra.Command, args []string) error {
	// Require config
	requireConfig()

	ctx, cancel := context.WithTimeout(context.Background(), diagTimeout)
	defer cancel()

	clusterName := cfg.Cluster.Name

	// Check if cluster exists
	exists, err := kind.ClusterExists(clusterName)
	if err != nil {
		printError("Failed to check cluster status: %v", err)
		return err
	}

	if !exists {
		printError("Cluster '%s' does not exist. Run 'kindplane up' first.", clusterName)
		return fmt.Errorf("cluster not found")
	}

	// Get kubernetes client
	kubeClient, err := kind.GetKubeClient(clusterName)
	if err != nil {
		printError("Failed to connect to cluster: %v", err)
		return err
	}

	// Get dynamic client for provider diagnostics
	restConfig, err := kind.GetRESTConfig(clusterName)
	if err != nil {
		printError("Failed to get REST config: %v", err)
		return err
	}

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		printError("Failed to create dynamic client: %v", err)
		return err
	}

	// Create diagnostics collector
	collector := diagnostics.NewCollector(kubeClient, dynamicClient)

	// Determine components to diagnose
	components := []diagnostics.Component{}
	if diagComponent != "" {
		switch diagComponent {
		case "crossplane":
			components = append(components, diagnostics.ComponentCrossplane)
		case "providers":
			components = append(components, diagnostics.ComponentProviders)
		case "eso":
			components = append(components, diagnostics.ComponentESO)
		case "helm":
			components = append(components, diagnostics.ComponentHelm)
		default:
			printError("Unknown component: %s. Valid options: crossplane, providers, eso, helm", diagComponent)
			return fmt.Errorf("unknown component: %s", diagComponent)
		}
	} else {
		// Default: diagnose all components
		components = []diagnostics.Component{
			diagnostics.ComponentCrossplane,
			diagnostics.ComponentProviders,
		}
		if cfg.ESO.Enabled {
			components = append(components, diagnostics.ComponentESO)
		}
	}

	// Print header
	fmt.Println()
	fmt.Println(ui.Title(ui.IconMagnifier + " Cluster Diagnostics"))
	fmt.Println(ui.Divider())
	fmt.Println()

	// Collect and print diagnostics for each component
	printInfo("Running diagnostics for cluster '%s'...", clusterName)
	fmt.Println()

	hasIssues := false
	for _, component := range components {
		diagCtx := diagnostics.DefaultContext(component)
		diagCtx.MaxLogLines = diagMaxLogs

		// Override namespace if specified
		if diagNamespace != "" {
			diagCtx.Namespace = diagNamespace
		}

		// Set release name for helm diagnostics
		if component == diagnostics.ComponentHelm && diagRelease != "" {
			diagCtx.ReleaseName = diagRelease
		}

		report, err := collector.Collect(ctx, diagCtx)
		if err != nil {
			printWarn("Failed to collect diagnostics for %s: %v", component, err)
			continue
		}

		if report.HasIssues() {
			hasIssues = true
			report.Print(os.Stdout)
		} else {
			printSuccess("No issues found for component: %s", component)
		}
	}

	if !hasIssues {
		fmt.Println()
		fmt.Println(ui.SuccessBox("All Healthy", "All components are functioning correctly."))
	}

	return nil
}
