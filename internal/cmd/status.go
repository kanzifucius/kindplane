package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/kanzi/kindplane/internal/crossplane"
	"github.com/kanzi/kindplane/internal/kind"
	"github.com/kanzi/kindplane/internal/ui"
)

var (
	statusDetailed bool
	statusTimeout  time.Duration
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show cluster and component status",
	Long: `Display the status of the Kind cluster and installed components.

Shows:
  - Cluster status (exists/running)
  - Crossplane installation status
  - Provider health`,
	Example: `  # Show basic status
  kindplane status

  # Show detailed status with pod information
  kindplane status --detailed`,
	RunE: runStatus,
}

func init() {
	statusCmd.Flags().BoolVarP(&statusDetailed, "detailed", "d", false, "show detailed status with pod information")
	statusCmd.Flags().DurationVar(&statusTimeout, "timeout", 30*time.Second, "timeout for status checks")
}

// Styles for status display
var (
	statusTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ui.ColorPrimary).
				MarginBottom(1)

	statusSectionStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ui.ColorSecondary).
				MarginTop(1)

	statusBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ui.ColorBorder).
			Padding(1, 2).
			MarginTop(1)

	statusLabelStyle = lipgloss.NewStyle().
				Foreground(ui.ColorMuted).
				Width(16)

	statusHealthyStyle = lipgloss.NewStyle().
				Foreground(ui.ColorSuccess).
				Bold(true)

	statusUnhealthyStyle = lipgloss.NewStyle().
				Foreground(ui.ColorError).
				Bold(true)

	statusPendingStyle = lipgloss.NewStyle().
				Foreground(ui.ColorWarning)

	statusMutedStyle = lipgloss.NewStyle().
				Foreground(ui.ColorMuted)
)

func runStatus(cmd *cobra.Command, args []string) error {
	// Require config
	requireConfig()

	ctx, cancel := context.WithTimeout(context.Background(), statusTimeout)
	defer cancel()

	// Print header
	fmt.Println()
	fmt.Println(statusTitleStyle.Render(ui.IconCluster + " kindplane status"))
	fmt.Println(ui.Divider())

	// Check cluster status
	var statusContent strings.Builder

	statusContent.WriteString(statusSectionStyle.Render(ui.IconCluster + " Cluster"))
	statusContent.WriteString("\n\n")

	exists, err := kind.ClusterExists(cfg.Cluster.Name)
	if err != nil {
		printError("Failed to check cluster: %v", err)
		return err
	}

	if !exists {
		statusContent.WriteString("  ")
		statusContent.WriteString(statusLabelStyle.Render("Status:"))
		statusContent.WriteString(statusUnhealthyStyle.Render(ui.IconError + " not found"))
		statusContent.WriteString("\n")

		fmt.Println(statusBoxStyle.Render(statusContent.String()))
		fmt.Println()
		fmt.Println(ui.InfoBox("Hint", "Run 'kindplane up' to create the cluster"))
		return nil
	}

	statusContent.WriteString("  ")
	statusContent.WriteString(statusLabelStyle.Render("Name:"))
	statusContent.WriteString(cfg.Cluster.Name)
	statusContent.WriteString("\n")

	statusContent.WriteString("  ")
	statusContent.WriteString(statusLabelStyle.Render("Status:"))
	statusContent.WriteString(statusHealthyStyle.Render(ui.IconSuccess + " running"))
	statusContent.WriteString("\n")

	statusContent.WriteString("  ")
	statusContent.WriteString(statusLabelStyle.Render("Context:"))
	statusContent.WriteString(ui.Code(cfg.GetKubeContext()))
	statusContent.WriteString("\n")

	// Get kubernetes client
	kubeClient, err := kind.GetKubeClient(cfg.Cluster.Name)
	if err != nil {
		printError("Failed to connect to cluster: %v", err)
		return err
	}

	// Check Crossplane status
	statusContent.WriteString("\n")
	statusContent.WriteString(statusSectionStyle.Render(ui.IconPackage + " Crossplane"))
	statusContent.WriteString("\n\n")

	cpInstaller := crossplane.NewInstaller(kubeClient)
	cpStatus, err := cpInstaller.GetStatus(ctx)
	if err != nil {
		statusContent.WriteString("  ")
		statusContent.WriteString(statusUnhealthyStyle.Render(ui.IconError + " Failed to get status: " + err.Error()))
		statusContent.WriteString("\n")
	} else {
		if cpStatus.Installed {
			statusContent.WriteString("  ")
			statusContent.WriteString(statusLabelStyle.Render("Installed:"))
			statusContent.WriteString(statusHealthyStyle.Render(ui.IconSuccess + " yes"))
			statusContent.WriteString("\n")

			statusContent.WriteString("  ")
			statusContent.WriteString(statusLabelStyle.Render("Version:"))
			statusContent.WriteString(cpStatus.Version)
			statusContent.WriteString("\n")

			statusContent.WriteString("  ")
			statusContent.WriteString(statusLabelStyle.Render("Ready:"))
			if cpStatus.Ready {
				statusContent.WriteString(statusHealthyStyle.Render(ui.IconSuccess + " yes"))
			} else {
				statusContent.WriteString(statusPendingStyle.Render(ui.IconWarning + " no"))
			}
			statusContent.WriteString("\n")

			if statusDetailed && len(cpStatus.Pods) > 0 {
				statusContent.WriteString("\n  ")
				statusContent.WriteString(statusMutedStyle.Render("Pods:"))
				statusContent.WriteString("\n")
				for _, pod := range cpStatus.Pods {
					icon := ui.IconSuccess
					style := statusHealthyStyle
					if !pod.Ready {
						icon = ui.IconWarning
						style = statusPendingStyle
					}
					statusContent.WriteString("    ")
					statusContent.WriteString(style.Render(icon))
					statusContent.WriteString(" ")
					statusContent.WriteString(pod.Name)
					statusContent.WriteString(statusMutedStyle.Render(" (" + pod.Phase + ")"))
					statusContent.WriteString("\n")
				}
			}
		} else {
			statusContent.WriteString("  ")
			statusContent.WriteString(statusLabelStyle.Render("Installed:"))
			statusContent.WriteString(statusMutedStyle.Render("no"))
			statusContent.WriteString("\n")
		}
	}

	// Check provider status
	statusContent.WriteString("\n")
	statusContent.WriteString(statusSectionStyle.Render(ui.IconGear + " Providers"))
	statusContent.WriteString("\n\n")

	providers, err := cpInstaller.GetProviderStatus(ctx)
	if err != nil {
		statusContent.WriteString("  ")
		statusContent.WriteString(statusMutedStyle.Render("Unable to fetch provider status"))
		statusContent.WriteString("\n")
	} else if len(providers) == 0 {
		statusContent.WriteString("  ")
		statusContent.WriteString(statusMutedStyle.Render("No providers installed"))
		statusContent.WriteString("\n")
	} else {
		// Build provider table
		headers := []string{"NAME", "VERSION", "PACKAGE", "STATUS"}
		var rows [][]string

		for _, p := range providers {
			var status string
			if p.Healthy {
				status = ui.IconSuccess + " healthy"
			} else {
				status = ui.IconError + " unhealthy"
				if statusDetailed && p.Message != "" {
					status = ui.IconError + " " + p.Message
				}
			}

			rows = append(rows, []string{
				p.Name,
				p.Version,
				ui.TruncateWithEllipsis(p.Package, 35),
				status,
			})
		}

		// Render table and indent each line to fit within the box
		tableOutput := ui.RenderTable(headers, rows)
		for _, line := range strings.Split(tableOutput, "\n") {
			statusContent.WriteString("  ")
			statusContent.WriteString(line)
			statusContent.WriteString("\n")
		}
	}

	// Print the status box
	fmt.Println(statusBoxStyle.Render(statusContent.String()))
	fmt.Println()

	return nil
}
