package diagnostics

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/kanzi/kindplane/internal/ui"
)

// Component represents the type of component that failed
type Component string

const (
	ComponentCrossplane Component = "crossplane"
	ComponentProviders  Component = "providers"
	ComponentESO        Component = "eso"
	ComponentHelm       Component = "helm"
)

// Context specifies what component failed and where to look for diagnostics
type Context struct {
	// Component that failed
	Component Component

	// Namespace to inspect for pods and resources
	Namespace string

	// ReleaseName for Helm failures
	ReleaseName string

	// ProviderNames for specific provider failures (empty = all providers)
	ProviderNames []string

	// LabelSelector for filtering pods
	LabelSelector string

	// MaxLogLines is the maximum number of log lines to fetch per container
	MaxLogLines int64
}

// DefaultContext returns a Context with sensible defaults
func DefaultContext(component Component) Context {
	ctx := Context{
		Component:   component,
		MaxLogLines: 30,
	}

	switch component {
	case ComponentCrossplane:
		ctx.Namespace = "crossplane-system"
		ctx.LabelSelector = "app=crossplane"
	case ComponentProviders:
		ctx.Namespace = "crossplane-system"
	case ComponentESO:
		ctx.Namespace = "external-secrets"
		ctx.LabelSelector = "app.kubernetes.io/name=external-secrets"
	}

	return ctx
}

// Report contains all collected diagnostics
type Report struct {
	// Component that was diagnosed
	Component Component

	// Pods contains pod diagnostics
	Pods []PodDiagnostic

	// Providers contains Crossplane provider diagnostics
	Providers []ProviderDiagnostic

	// HelmRelease contains Helm release diagnostics
	HelmRelease *HelmDiagnostic

	// Events contains relevant Kubernetes events
	Events []EventInfo
}

// EventInfo contains information about a Kubernetes event
type EventInfo struct {
	Type      string
	Reason    string
	Message   string
	Object    string
	Count     int32
	FirstSeen string
	LastSeen  string
}

// Collector gathers diagnostics from a Kubernetes cluster
type Collector struct {
	kubeClient    kubernetes.Interface
	dynamicClient dynamic.Interface
}

// NewCollector creates a new diagnostics Collector
func NewCollector(kubeClient kubernetes.Interface, dynamicClient dynamic.Interface) *Collector {
	return &Collector{
		kubeClient:    kubeClient,
		dynamicClient: dynamicClient,
	}
}

// NewCollectorFromClientset creates a Collector from just a Clientset
// It will create the dynamic client internally
func NewCollectorFromClientset(kubeClient *kubernetes.Clientset) (*Collector, error) {
	// We'll create the dynamic client when needed using the same rest config
	return &Collector{
		kubeClient: kubeClient,
	}, nil
}

// Collect gathers diagnostics based on the provided context
func (c *Collector) Collect(ctx context.Context, diagCtx Context) (*Report, error) {
	report := &Report{
		Component: diagCtx.Component,
	}

	// Always collect pod diagnostics if we have a namespace
	if diagCtx.Namespace != "" {
		pods, err := c.CollectPodDiagnostics(ctx, diagCtx)
		if err != nil {
			// Don't fail entirely, just note the error
			report.Pods = []PodDiagnostic{{
				Name:  "error",
				Error: fmt.Sprintf("failed to collect pod diagnostics: %v", err),
			}}
		} else {
			report.Pods = pods
		}
	}

	// Collect provider diagnostics for Crossplane-related failures
	if diagCtx.Component == ComponentCrossplane || diagCtx.Component == ComponentProviders {
		if c.dynamicClient != nil {
			providers, err := c.CollectProviderDiagnostics(ctx, diagCtx)
			if err == nil {
				report.Providers = providers
			}
		}
	}

	// Collect Helm diagnostics for Helm failures
	if diagCtx.Component == ComponentHelm && diagCtx.ReleaseName != "" {
		helm, err := c.CollectHelmDiagnostics(ctx, diagCtx)
		if err == nil {
			report.HelmRelease = helm
		}
	}

	return report, nil
}

// Diagnostic styles
var (
	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ui.ColorError).
			Background(lipgloss.AdaptiveColor{Light: "#FEE2E2", Dark: "#7F1D1D"}).
			Padding(0, 2)

	styleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ui.ColorError).
			Padding(1, 2)

	styleSectionTitle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ui.ColorSecondary).
				MarginTop(1).
				MarginBottom(1)

	styleHealthy = lipgloss.NewStyle().
			Foreground(ui.ColorSuccess).
			Bold(true)

	styleUnhealthy = lipgloss.NewStyle().
			Foreground(ui.ColorError).
			Bold(true)

	styleWarning = lipgloss.NewStyle().
			Foreground(ui.ColorWarning)

	styleMuted = lipgloss.NewStyle().
			Foreground(ui.ColorMuted)

	styleLabel = lipgloss.NewStyle().
			Foreground(ui.ColorMuted).
			Width(14)

	styleLogLine = lipgloss.NewStyle().
			Foreground(ui.ColorMuted).
			PaddingLeft(8)
)

// Print writes the diagnostic report to the given writer using lipgloss styling
func (r *Report) Print(w io.Writer) {
	var content strings.Builder

	// Print provider diagnostics first (most relevant for Crossplane failures)
	if len(r.Providers) > 0 {
		r.renderProviders(&content)
	}

	// Print pod diagnostics
	if len(r.Pods) > 0 {
		r.renderPods(&content)
	}

	// Print Helm diagnostics
	if r.HelmRelease != nil {
		r.renderHelm(&content)
	}

	// Wrap in diagnostic box
	header := styleHeader.Render(" " + ui.IconError + " DIAGNOSTICS ")
	box := styleBox.Render(content.String())

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, header)
	_, _ = fmt.Fprintln(w, box)
}

// renderProviders renders provider diagnostics
func (r *Report) renderProviders(sb *strings.Builder) {
	sb.WriteString(styleSectionTitle.Render(ui.IconPackage + " Providers"))
	sb.WriteString("\n")

	for _, p := range r.Providers {
		// Provider header with status
		var statusStyle lipgloss.Style
		var statusIcon string
		if p.Healthy {
			statusStyle = styleHealthy
			statusIcon = ui.IconSuccess
		} else {
			statusStyle = styleUnhealthy
			statusIcon = ui.IconError
		}

		sb.WriteString("  ")
		sb.WriteString(statusStyle.Render(statusIcon + " " + p.Name))
		sb.WriteString("\n")

		// Package
		sb.WriteString("    ")
		sb.WriteString(styleLabel.Render("Package:"))
		sb.WriteString(styleMuted.Render(p.Package))
		sb.WriteString("\n")

		// Conditions
		if len(p.Conditions) > 0 {
			sb.WriteString("    ")
			sb.WriteString(styleLabel.Render("Conditions:"))
			sb.WriteString("\n")

			for _, cond := range p.Conditions {
				var condIcon string
				var condStyle lipgloss.Style
				switch cond.Status {
				case "True":
					condIcon = ui.IconSuccess
					condStyle = styleHealthy
				case "False":
					condIcon = ui.IconError
					condStyle = styleUnhealthy
				default:
					condIcon = ui.IconPending
					condStyle = styleWarning
				}

				sb.WriteString("      ")
				sb.WriteString(condStyle.Render(condIcon))
				sb.WriteString(" ")
				sb.WriteString(lipgloss.NewStyle().Bold(true).Render(cond.Type))
				sb.WriteString(styleMuted.Render(": " + cond.Status))
				sb.WriteString("\n")

				if cond.Reason != "" {
					sb.WriteString("        ")
					sb.WriteString(styleMuted.Render("Reason: " + cond.Reason))
					sb.WriteString("\n")
				}
				if cond.Message != "" {
					msg := cond.Message
					if len(msg) > 80 {
						msg = msg[:80] + "..."
					}
					sb.WriteString("        ")
					sb.WriteString(styleMuted.Render("Message: " + msg))
					sb.WriteString("\n")
				}
			}
		}
		sb.WriteString("\n")
	}
}

// renderPods renders pod diagnostics
func (r *Report) renderPods(sb *strings.Builder) {
	sb.WriteString(styleSectionTitle.Render(ui.IconCluster + " Pods"))
	sb.WriteString("\n")

	for _, pod := range r.Pods {
		if pod.Error != "" {
			sb.WriteString("  ")
			sb.WriteString(styleUnhealthy.Render(ui.IconError + " Error: " + pod.Error))
			sb.WriteString("\n")
			continue
		}

		// Determine pod status
		var statusIcon string
		var statusStyle lipgloss.Style
		switch pod.Phase {
		case "Running":
			if pod.Ready {
				statusIcon = ui.IconSuccess
				statusStyle = styleHealthy
			} else {
				statusIcon = ui.IconWarning
				statusStyle = styleWarning
			}
		case "Pending":
			statusIcon = ui.IconPending
			statusStyle = styleWarning
		case "Failed", "Unknown":
			statusIcon = ui.IconError
			statusStyle = styleUnhealthy
		case "Succeeded":
			statusIcon = ui.IconSuccess
			statusStyle = styleHealthy
		default:
			statusIcon = ui.IconDot
			statusStyle = styleMuted
		}

		// Pod header
		sb.WriteString("  ")
		sb.WriteString(statusStyle.Render(statusIcon + " " + pod.Name))
		sb.WriteString(styleMuted.Render(" (" + pod.Namespace + ")"))
		sb.WriteString("\n")

		// Phase
		sb.WriteString("    ")
		sb.WriteString(styleLabel.Render("Phase:"))
		sb.WriteString(pod.Phase)
		sb.WriteString("\n")

		// Ready containers
		if pod.ReadyContainers != pod.TotalContainers {
			sb.WriteString("    ")
			sb.WriteString(styleLabel.Render("Ready:"))
			sb.WriteString(styleWarning.Render(fmt.Sprintf("%d/%d containers", pod.ReadyContainers, pod.TotalContainers)))
			sb.WriteString("\n")
		}

		// Conditions with issues
		for _, cond := range pod.Conditions {
			if cond.Status != "True" && cond.Message != "" {
				sb.WriteString("    ")
				sb.WriteString(styleWarning.Render(ui.IconWarning + " " + cond.Type + ": " + cond.Status))
				sb.WriteString("\n")
				sb.WriteString("      ")
				sb.WriteString(styleMuted.Render(cond.Message))
				sb.WriteString("\n")
			}
		}

		// Container issues
		for _, container := range pod.Containers {
			if container.HasIssue() {
				sb.WriteString("    ")
				sb.WriteString(styleSectionTitle.Render("Container: " + container.Name))
				sb.WriteString("\n")

				sb.WriteString("      ")
				sb.WriteString(styleLabel.Render("State:"))
				sb.WriteString(container.State)
				sb.WriteString("\n")

				if container.Restarts > 0 {
					sb.WriteString("      ")
					sb.WriteString(styleLabel.Render("Restarts:"))
					sb.WriteString(styleWarning.Render(fmt.Sprintf("%d", container.Restarts)))
					sb.WriteString("\n")
				}

				if container.WaitingReason != "" {
					sb.WriteString("      ")
					sb.WriteString(styleLabel.Render("Waiting:"))
					sb.WriteString(styleUnhealthy.Render(container.WaitingReason))
					sb.WriteString("\n")
					if container.WaitingMessage != "" {
						sb.WriteString("        ")
						sb.WriteString(styleMuted.Render(container.WaitingMessage))
						sb.WriteString("\n")
					}
				}

				if container.TerminatedReason != "" {
					sb.WriteString("      ")
					sb.WriteString(styleLabel.Render("Terminated:"))
					sb.WriteString(styleUnhealthy.Render(fmt.Sprintf("%s (exit code %d)", container.TerminatedReason, container.ExitCode)))
					sb.WriteString("\n")
					if container.TerminatedMessage != "" {
						sb.WriteString("        ")
						sb.WriteString(styleMuted.Render(container.TerminatedMessage))
						sb.WriteString("\n")
					}
				}

				// Recent logs
				if len(container.RecentLogs) > 0 {
					sb.WriteString("      ")
					sb.WriteString(styleSectionTitle.Render("Recent Logs:"))
					sb.WriteString("\n")
					for _, line := range container.RecentLogs {
						if len(line) > 80 {
							line = line[:80] + "..."
						}
						sb.WriteString(styleLogLine.Render(line))
						sb.WriteString("\n")
					}
				}
			}
		}
		sb.WriteString("\n")
	}
}

// renderHelm renders Helm diagnostics
func (r *Report) renderHelm(sb *strings.Builder) {
	h := r.HelmRelease
	sb.WriteString(styleSectionTitle.Render(ui.IconGear + " Helm Release"))
	sb.WriteString("\n")

	// Determine status style
	var statusStyle lipgloss.Style
	if h.Status == "deployed" {
		statusStyle = styleHealthy
	} else {
		statusStyle = styleUnhealthy
	}

	sb.WriteString("  ")
	sb.WriteString(styleLabel.Render("Name:"))
	sb.WriteString(h.Name)
	sb.WriteString("\n")

	sb.WriteString("  ")
	sb.WriteString(styleLabel.Render("Namespace:"))
	sb.WriteString(h.Namespace)
	sb.WriteString("\n")

	sb.WriteString("  ")
	sb.WriteString(styleLabel.Render("Status:"))
	sb.WriteString(statusStyle.Render(h.Status))
	sb.WriteString("\n")

	if h.Description != "" {
		sb.WriteString("  ")
		sb.WriteString(styleLabel.Render("Description:"))
		sb.WriteString(styleMuted.Render(h.Description))
		sb.WriteString("\n")
	}

	if h.Error != "" {
		sb.WriteString("  ")
		sb.WriteString(styleLabel.Render("Error:"))
		sb.WriteString(styleUnhealthy.Render(h.Error))
		sb.WriteString("\n")
	}
}

// HasIssues returns true if the report contains any issues
func (r *Report) HasIssues() bool {
	// Check pods for issues
	for _, pod := range r.Pods {
		if pod.Error != "" || !pod.Ready || pod.Phase == "Failed" {
			return true
		}
		for _, c := range pod.Containers {
			if c.HasIssue() {
				return true
			}
		}
	}

	// Check providers for issues
	for _, p := range r.Providers {
		if !p.Healthy {
			return true
		}
	}

	// Check Helm for issues
	if r.HelmRelease != nil && r.HelmRelease.Status != "deployed" {
		return true
	}

	return false
}
