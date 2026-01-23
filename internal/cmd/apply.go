package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/kanzi/kindplane/internal/config"
	"github.com/kanzi/kindplane/internal/crossplane"
	"github.com/kanzi/kindplane/internal/kind"
	"github.com/kanzi/kindplane/internal/ui"
)

var (
	applyFile      string
	applyDirectory string
	applyRecursive bool
	applyDryRun    bool
	applyFromConfig bool
	applyTimeout   time.Duration
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply Crossplane resources to the cluster",
	Long: `Apply Crossplane compositions, XRDs, and other resources to the cluster.

This command allows you to apply Crossplane resources without running the
full 'up' workflow. You can apply resources from files, directories, or
from the sources defined in your configuration file.`,
	Example: `  # Apply compositions from the config file
  kindplane apply --from-config

  # Apply a single file
  kindplane apply --file ./compositions/database.yaml

  # Apply all files in a directory
  kindplane apply --directory ./compositions

  # Apply recursively from a directory
  kindplane apply --directory ./crossplane --recursive

  # Dry run to see what would be applied
  kindplane apply --file ./compositions/database.yaml --dry-run`,
	RunE: runApply,
}

func init() {
	applyCmd.Flags().StringVarP(&applyFile, "file", "f", "", "Path to a YAML file to apply")
	applyCmd.Flags().StringVarP(&applyDirectory, "directory", "d", "", "Path to a directory containing YAML files")
	applyCmd.Flags().BoolVarP(&applyRecursive, "recursive", "R", false, "Recursively apply files from subdirectories")
	applyCmd.Flags().BoolVar(&applyDryRun, "dry-run", false, "Show what would be applied without making changes")
	applyCmd.Flags().BoolVar(&applyFromConfig, "from-config", false, "Apply compositions from the configuration file")
	applyCmd.Flags().DurationVar(&applyTimeout, "timeout", 5*time.Minute, "Timeout for apply operation")
}

func runApply(cmd *cobra.Command, args []string) error {
	// Require config
	requireConfig()

	ctx, cancel := context.WithTimeout(context.Background(), applyTimeout)
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

	// Validate flags
	if applyFile == "" && applyDirectory == "" && !applyFromConfig {
		printError("No source specified. Use --file, --directory, or --from-config")
		return fmt.Errorf("no source specified")
	}

	// Get kubernetes client
	kubeClient, err := kind.GetKubeClient(clusterName)
	if err != nil {
		printError("Failed to connect to cluster: %v", err)
		return err
	}

	installer := crossplane.NewInstaller(kubeClient)

	// Print header
	fmt.Println()
	fmt.Println(ui.Title(ui.IconRocket + " Apply Resources"))
	fmt.Println(ui.Divider())
	fmt.Println()

	// Apply from config file
	if applyFromConfig {
		if len(cfg.Compositions.Sources) == 0 {
			printWarn("No composition sources defined in configuration")
			return nil
		}

		printInfo("Applying compositions from configuration...")
		for _, source := range cfg.Compositions.Sources {
			printStep("Applying from %s: %s", source.Type, source.Path)
			if applyDryRun {
				printInfo("  [dry-run] Would apply compositions from: %s", source.Path)
				continue
			}
			if err := installer.ApplyCompositions(ctx, source); err != nil {
				printError("Failed to apply compositions: %v", err)
				return err
			}
			printSuccess("Applied compositions from: %s", source.Path)
		}
		return nil
	}

	// Apply from file
	if applyFile != "" {
		return applyFromFile(ctx, installer, applyFile, applyDryRun)
	}

	// Apply from directory
	if applyDirectory != "" {
		return applyFromDirectory(ctx, installer, applyDirectory, applyRecursive, applyDryRun)
	}

	return nil
}

func applyFromFile(ctx context.Context, installer *crossplane.Installer, path string, dryRun bool) error {
	// Check file exists
	info, err := os.Stat(path)
	if err != nil {
		printError("Failed to access file: %v", err)
		return err
	}

	if info.IsDir() {
		printError("'%s' is a directory. Use --directory instead.", path)
		return fmt.Errorf("expected file, got directory")
	}

	printStep("Applying file: %s", path)

	if dryRun {
		printInfo("  [dry-run] Would apply: %s", path)
		return nil
	}

	source := config.CompositionSource{
		Type: "local",
		Path: path,
	}

	if err := installer.ApplyCompositions(ctx, source); err != nil {
		printError("Failed to apply file: %v", err)
		return err
	}

	printSuccess("Applied: %s", path)
	return nil
}

func applyFromDirectory(ctx context.Context, installer *crossplane.Installer, dir string, recursive bool, dryRun bool) error {
	// Check directory exists
	info, err := os.Stat(dir)
	if err != nil {
		printError("Failed to access directory: %v", err)
		return err
	}

	if !info.IsDir() {
		printError("'%s' is not a directory. Use --file instead.", dir)
		return fmt.Errorf("expected directory, got file")
	}

	printInfo("Applying resources from: %s", dir)

	var files []string
	if recursive {
		err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".yaml" || ext == ".yml" {
				files = append(files, path)
			}
			return nil
		})
	} else {
		entries, err := os.ReadDir(dir)
		if err != nil {
			printError("Failed to read directory: %v", err)
			return err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext == ".yaml" || ext == ".yml" {
				files = append(files, filepath.Join(dir, entry.Name()))
			}
		}
	}

	if err != nil {
		printError("Failed to walk directory: %v", err)
		return err
	}

	if len(files) == 0 {
		printWarn("No YAML files found in: %s", dir)
		return nil
	}

	printStep("Found %d YAML files", len(files))

	for _, file := range files {
		relPath, _ := filepath.Rel(dir, file)
		if relPath == "" {
			relPath = filepath.Base(file)
		}

		if dryRun {
			printInfo("  [dry-run] Would apply: %s", relPath)
			continue
		}

		source := config.CompositionSource{
			Type: "local",
			Path: file,
		}

		if err := installer.ApplyCompositions(ctx, source); err != nil {
			printError("Failed to apply %s: %v", relPath, err)
			return err
		}
		printSuccess("Applied: %s", relPath)
	}

	return nil
}
