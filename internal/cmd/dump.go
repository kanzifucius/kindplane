package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kanzi/kindplane/internal/dump"
	"github.com/kanzi/kindplane/internal/kind"
)

var (
	// Dump command flags
	dumpOutputDir     string
	dumpStdout        bool
	dumpDryRun        bool
	dumpFormat        string
	dumpNamespace     string
	dumpAllNamespaces bool
	dumpInclude       string
	dumpExclude       string
	dumpSkipSecrets   bool
	dumpTimeout       time.Duration
	dumpIncludeSystem bool
	dumpDiscoverXRs   bool
	dumpNoReadme      bool
)

var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Export Crossplane and related resources from the cluster",
	Long: `Export Crossplane and related resources from a Kind cluster in a GitOps-friendly format.

This command exports:
  - Crossplane Providers, ProviderConfigs, Configurations, Functions
  - CompositeResourceDefinitions (XRDs) and Compositions
  - Composite Resources (XRs) and Claims (optional discovery)
  - External Secrets Operator SecretStores and ExternalSecrets
  - Custom namespaces, Secrets (redacted), and ConfigMaps

The exported resources are cleaned for GitOps:
  - Cluster-specific metadata removed (uid, resourceVersion, managedFields)
  - Unnecessary annotations stripped (kubectl.kubernetes.io/last-applied-configuration)
  - Secret data redacted with placeholders
  - Status fields removed

Examples:
  # Dump all resources to ./dump directory
  kindplane dump

  # Dump to a specific directory
  kindplane dump -o ./exported

  # Dump to stdout (single YAML)
  kindplane dump --stdout

  # Dry run - show what would be dumped
  kindplane dump --dry-run

  # Dump only specific resource types
  kindplane dump --include=providers,compositions,xrds

  # Exclude certain resource types
  kindplane dump --exclude=secrets,configmaps

  # Dump from a specific namespace
  kindplane dump -n my-namespace

  # Include system namespaces
  kindplane dump --include-system

  # Discover and dump composite resource instances
  kindplane dump --discover-composites`,
	RunE: runDump,
}

func init() {
	dumpCmd.Flags().StringVarP(&dumpOutputDir, "output-dir", "o", "./dump", "output directory for dumped resources")
	dumpCmd.Flags().BoolVar(&dumpStdout, "stdout", false, "print to stdout instead of files")
	dumpCmd.Flags().BoolVar(&dumpDryRun, "dry-run", false, "show what would be dumped without fetching")
	dumpCmd.Flags().StringVar(&dumpFormat, "format", "files", "output format: 'files' (separate files) or 'single' (one multi-doc YAML)")
	dumpCmd.Flags().StringVarP(&dumpNamespace, "namespace", "n", "", "dump from specific namespace only")
	dumpCmd.Flags().BoolVarP(&dumpAllNamespaces, "all-namespaces", "A", true, "dump from all namespaces")
	dumpCmd.Flags().StringVar(&dumpInclude, "include", "", "resource types to include (comma-separated)")
	dumpCmd.Flags().StringVar(&dumpExclude, "exclude", "", "resource types to exclude (comma-separated)")
	dumpCmd.Flags().BoolVar(&dumpSkipSecrets, "skip-secrets", false, "skip secrets entirely")
	dumpCmd.Flags().DurationVar(&dumpTimeout, "timeout", 2*time.Minute, "timeout for dump operation")
	dumpCmd.Flags().BoolVar(&dumpIncludeSystem, "include-system", false, "include system namespaces (kube-system, etc.)")
	dumpCmd.Flags().BoolVar(&dumpDiscoverXRs, "discover-composites", true, "discover and dump composite resource instances")
	dumpCmd.Flags().BoolVar(&dumpNoReadme, "no-readme", false, "skip generating README.md")
}

func runDump(cmd *cobra.Command, args []string) error {
	// Require config
	requireConfig()

	ctx, cancel := context.WithTimeout(context.Background(), dumpTimeout)
	defer cancel()

	// Check if cluster exists
	exists, err := kind.ClusterExists(cfg.Cluster.Name)
	if err != nil {
		printError("Failed to check cluster: %v", err)
		return err
	}

	if !exists {
		printError("Cluster '%s' not found", cfg.Cluster.Name)
		printStep("Run 'kindplane up' to create the cluster first")
		return fmt.Errorf("cluster not found")
	}

	printInfo("Dumping resources from cluster '%s'", cfg.Cluster.Name)

	// Build dump options
	opts := dump.DumpOptions{
		AllNamespaces:           dumpAllNamespaces,
		IncludeSystemNamespaces: dumpIncludeSystem,
		SkipSecrets:             dumpSkipSecrets,
		DryRun:                  dumpDryRun,
		DiscoverComposites:      dumpDiscoverXRs,
	}

	// Handle namespace flag
	if dumpNamespace != "" {
		opts.Namespaces = []string{dumpNamespace}
		opts.AllNamespaces = false
	}

	// Parse include/exclude
	if dumpInclude != "" {
		opts.IncludeTypes = dump.ParseResourceTypes(dumpInclude)
	}
	if dumpExclude != "" {
		opts.ExcludeTypes = dump.ParseResourceTypes(dumpExclude)
	}

	// Dry run mode
	if dumpDryRun {
		return runDumpDryRun(opts)
	}

	// Get REST config for the cluster
	restConfig, err := getRESTConfig(cfg.Cluster.Name)
	if err != nil {
		printError("Failed to get kubernetes config: %v", err)
		return err
	}

	// Create dumper
	dumper, err := dump.NewDumper(restConfig, opts)
	if err != nil {
		printError("Failed to create dumper: %v", err)
		return err
	}

	// Perform dump
	printStep("Fetching resources...")
	result, err := dumper.Dump(ctx)
	if err != nil {
		printError("Dump failed: %v", err)
		return err
	}

	// Report any non-fatal errors
	for _, e := range result.Errors {
		printWarn("Warning: %v", e)
	}

	// Write output
	if dumpStdout {
		// Write to stdout
		outputFormat := dump.OutputFormatSingle // stdout always uses single format
		writer := dump.NewWriter("", outputFormat)
		if err := writer.WriteToStdout(result); err != nil {
			printError("Failed to write output: %v", err)
			return err
		}
	} else {
		// Write to files
		outputFormat := dump.OutputFormatFiles
		if dumpFormat == "single" {
			outputFormat = dump.OutputFormatSingle
		}

		writer := dump.NewWriter(dumpOutputDir, outputFormat)
		writer.GenerateReadme = !dumpNoReadme

		printStep("Writing to %s...", dumpOutputDir)
		if err := writer.Write(result); err != nil {
			printError("Failed to write output: %v", err)
			return err
		}
	}

	// Print summary
	fmt.Println()
	printSuccess("Dump completed successfully")
	dump.WriteStats(os.Stdout, result.Stats)

	if !dumpStdout {
		fmt.Println()
		printStep("Output directory: %s", dumpOutputDir)
	}

	// Print discovered XRDs
	if len(result.DiscoveredXRDs) > 0 {
		fmt.Println()
		printInfo("Discovered %d Composite Resource Definition(s):", len(result.DiscoveredXRDs))
		for _, xrd := range result.DiscoveredXRDs {
			printStep("  %s (%s)", xrd.Name, xrd.Kind)
		}
	}

	return nil
}

// runDumpDryRun shows what would be dumped without actually fetching
func runDumpDryRun(opts dump.DumpOptions) error {
	printInfo("Dry run mode - showing what would be dumped")
	fmt.Println()

	// Get resources that would be fetched
	resources := dump.DefaultResources()
	resources = dump.FilterResources(resources, opts.IncludeTypes, opts.ExcludeTypes)

	// Filter out secrets if skip-secrets is set
	if opts.SkipSecrets {
		var filtered []dump.ResourceInfo
		for _, r := range resources {
			if r.Type != dump.ResourceTypeSecrets {
				filtered = append(filtered, r)
			}
		}
		resources = filtered
	}

	dump.WriteDryRunReport(os.Stdout, resources)

	// Show options
	fmt.Println("Options:")
	if len(opts.Namespaces) > 0 {
		printStep("  Namespaces: %v", opts.Namespaces)
	} else if opts.AllNamespaces {
		printStep("  Namespaces: all (non-system)")
	}
	if opts.IncludeSystemNamespaces {
		printStep("  Include system namespaces: yes")
	}
	if opts.DiscoverComposites {
		printStep("  Discover composite resources: yes")
	}
	fmt.Println()

	printInfo("Run without --dry-run to perform the dump")

	return nil
}

// getRESTConfig returns a REST config for the specified Kind cluster
func getRESTConfig(clusterName string) (*rest.Config, error) {
	kubeconfigPath := kind.GetKubeConfigPath(clusterName)
	contextName := kind.GetContextName(clusterName)

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{CurrentContext: contextName},
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	return config, nil
}
