package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"

	"github.com/kanzi/kindplane/internal/doctor"
	"github.com/kanzi/kindplane/internal/kind"
	"github.com/kanzi/kindplane/internal/ui"
)

var (
	doctorQuiet   bool
	doctorTimeout time.Duration
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system requirements and prerequisites",
	Long: `Run pre-flight checks to verify that your system meets all
requirements for running kindplane.

This command checks:
  - Docker daemon is running
  - Required binaries are available (kind, kubectl)
  - Sufficient disk space
  - Optional tools (helm)
  - Cluster connectivity (if a cluster exists)
  - Crossplane installation status`,
	Example: `  # Run all pre-flight checks
  kindplane doctor

  # Run checks with minimal output
  kindplane doctor --quiet`,
	RunE: runDoctor,
}

func init() {
	doctorCmd.Flags().BoolVarP(&doctorQuiet, "quiet", "q", false, "Only show failures")
	doctorCmd.Flags().DurationVar(&doctorTimeout, "timeout", 30*time.Second, "Timeout for checks")
}

func runDoctor(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), doctorTimeout)
	defer cancel()

	// Try to load config and get cluster name for cluster-specific checks
	var clusterName string
	if err := loadConfig(); err == nil && cfg != nil {
		clusterName = cfg.Cluster.Name
	}

	// Get kubeClient if cluster exists
	var kubeClient *kubernetes.Clientset
	if clusterName != "" {
		exists, _ := kind.ClusterExists(clusterName)
		if exists {
			client, err := kind.GetKubeClient(clusterName)
			if err == nil {
				kubeClient = client
			}
		}
	}

	// Run all checks
	results := doctor.RunAllChecks(ctx, kubeClient)

	// Print header
	if !doctorQuiet {
		fmt.Println()
		fmt.Println(ui.Title(ui.IconWrench + " kindplane doctor"))
		fmt.Println(ui.Divider())
		fmt.Println()
	}

	// Print results
	passedCount := 0
	failedCount := 0
	requiredFailures := 0

	for _, result := range results {
		if result.Passed {
			passedCount++
			if !doctorQuiet {
				fmt.Printf("  %s %s: %s\n",
					ui.StyleSuccess.Render(ui.IconSuccess),
					result.Name,
					result.Message,
				)
				if result.Details != "" {
					fmt.Printf("    %s\n", ui.StyleMuted.Render(result.Details))
				}
			}
		} else {
			failedCount++
			if result.Required {
				requiredFailures++
			}

			var icon string
			if result.Required {
				icon = ui.StyleError.Render(ui.IconError)
			} else {
				icon = ui.StyleWarning.Render(ui.IconWarning)
			}

			fmt.Printf("  %s %s: %s\n",
				icon,
				result.Name,
				result.Message,
			)
			if result.Details != "" {
				fmt.Printf("    %s\n", ui.StyleMuted.Render(result.Details))
			}
			if result.Suggestion != "" {
				fmt.Printf("    %s %s\n", ui.StyleMuted.Render(ui.IconArrow), result.Suggestion)
			}
		}
	}

	// Summary
	fmt.Println()
	if failedCount == 0 {
		fmt.Println(ui.SuccessBox("All Checks Passed", fmt.Sprintf("All %d checks passed! Your system is ready.", passedCount)))
	} else if requiredFailures == 0 {
		fmt.Println(ui.WarningBox("Checks Complete", fmt.Sprintf("%d/%d checks passed (%d warnings)", passedCount, len(results), failedCount)))
	} else {
		fmt.Println(ui.ErrorBox("Checks Failed", fmt.Sprintf("%d/%d checks passed (%d failures)", passedCount, len(results), failedCount)))
		fmt.Println()
		fmt.Println(ui.Muted("Please fix the required issues before running kindplane."))
		return fmt.Errorf("%d required checks failed", requiredFailures)
	}
	fmt.Println()

	return nil
}
