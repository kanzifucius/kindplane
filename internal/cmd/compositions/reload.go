package compositions

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
	reloadTimeout time.Duration
	reloadDryRun  bool
)

var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload compositions from configuration sources",
	Long: `Reload all Crossplane compositions defined in the configuration file.

This command re-applies all composition sources (local directories and git repos)
to the cluster, effectively refreshing any changes made to the compositions.`,
	Example: `  # Reload all compositions from config
  kindplane compositions reload

  # Dry run to see what would be reloaded
  kindplane compositions reload --dry-run`,
	RunE: runReload,
}

func init() {
	reloadCmd.Flags().DurationVar(&reloadTimeout, "timeout", 5*time.Minute, "Timeout for reload operation")
	reloadCmd.Flags().BoolVar(&reloadDryRun, "dry-run", false, "Show what would be applied without making changes")
}

func runReload(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load("")
	if err != nil {
		fmt.Println(ui.Error("Failed to load config: %v", err))
		return err
	}

	// Check if there are any composition sources
	if len(cfg.Compositions.Sources) == 0 {
		fmt.Println()
		fmt.Println(ui.Title(ui.IconPackage + " Reload Compositions"))
		fmt.Println(ui.Divider())
		fmt.Println()
		fmt.Println(ui.Warning("No composition sources defined in configuration"))
		fmt.Println(ui.InfoBox("Hint", "Add composition sources to your kindplane.yaml:\n\ncompositions:\n  sources:\n    - type: local\n      path: ./compositions"))
		return nil
	}

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

	// Create phase tracker for dashboard-style visuals
	pt := ui.NewPhaseTracker("Reload Compositions",
		ui.WithClusterInfo(cfg.Cluster.Name, ""),
	)

	// Add a phase for each composition source
	for _, source := range cfg.Compositions.Sources {
		phaseName := formatSourceName(source)
		pt.AddPhase(phaseName)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), reloadTimeout)
	defer cancel()

	// Work function that reloads compositions
	workFn := func(ctx context.Context, ctrl *ui.DashboardController) error {
		installer := crossplane.NewInstaller(kubeClient)

		for _, source := range cfg.Compositions.Sources {
			phaseName := formatSourceName(source)

			// Check for cancellation
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// Start phase
			if ctrl != nil {
				ctrl.StartPhase(phaseName)
				ctrl.UpdateOperation(fmt.Sprintf("Loading from %s...", source.Path), -1)
			} else {
				pt.StartPhase(phaseName)
				fmt.Println(ui.Info("Applying: %s", phaseName))
			}

			// Dry run mode
			if reloadDryRun {
				if ctrl != nil {
					ctrl.Log(fmt.Sprintf("[dry-run] Would apply: %s", source.Path))
					ctrl.CompletePhase(phaseName, "dry-run")
				} else {
					fmt.Println(ui.Step("[dry-run] Would apply: %s", source.Path))
					pt.CompletePhase()
				}
				continue
			}

			// Apply compositions
			if err := installer.ApplyCompositions(ctx, source); err != nil {
				if ctrl != nil {
					ctrl.FailPhase(phaseName, err)
				} else {
					pt.FailPhase(err)
				}
				return fmt.Errorf("failed to apply %s: %w", phaseName, err)
			}

			// Complete phase
			if ctrl != nil {
				ctrl.CompletePhase(phaseName, "applied")
			} else {
				pt.CompletePhase()
				fmt.Println(ui.Success("Applied: %s", phaseName))
			}
		}

		return nil
	}

	// Run with dashboard visuals
	result, err := ui.RunBootstrapDashboard(
		ctx,
		pt,
		workFn,
		ui.WithTimeout(reloadTimeout),
		ui.WithNextStepHint("kubectl get compositions"),
	)

	if err != nil {
		return err
	}

	if !result.Success {
		return result.Error
	}

	return nil
}

// formatSourceName creates a display name for a composition source
func formatSourceName(source config.CompositionSource) string {
	switch source.Type {
	case "git":
		if source.Branch != "" {
			return fmt.Sprintf("%s@%s:%s", source.Repo, source.Branch, source.Path)
		}
		return fmt.Sprintf("%s:%s", source.Repo, source.Path)
	case "local":
		return source.Path
	default:
		return fmt.Sprintf("%s:%s", source.Type, source.Path)
	}
}
