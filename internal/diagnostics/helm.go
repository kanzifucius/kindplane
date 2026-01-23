package diagnostics

import (
	"context"
	"fmt"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
)

// HelmDiagnostic contains diagnostic information about a Helm release
type HelmDiagnostic struct {
	Name        string
	Namespace   string
	Status      string
	Version     int
	Chart       string
	AppVersion  string
	Description string
	Error       string
	Notes       string
}

// CollectHelmDiagnostics collects diagnostics for a Helm release
func (c *Collector) CollectHelmDiagnostics(ctx context.Context, diagCtx Context) (*HelmDiagnostic, error) {
	if diagCtx.ReleaseName == "" {
		return nil, fmt.Errorf("release name not specified")
	}

	namespace := diagCtx.Namespace
	if namespace == "" {
		namespace = "default"
	}

	// Create Helm settings and action configuration
	settings := cli.New()
	actionConfig := new(action.Configuration)

	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, "secret", nil); err != nil {
		return nil, fmt.Errorf("initializing helm config: %w", err)
	}

	// Get release status
	statusAction := action.NewStatus(actionConfig)
	release, err := statusAction.Run(diagCtx.ReleaseName)
	if err != nil {
		return &HelmDiagnostic{
			Name:      diagCtx.ReleaseName,
			Namespace: namespace,
			Status:    "not-found",
			Error:     err.Error(),
		}, nil
	}

	diag := &HelmDiagnostic{
		Name:      release.Name,
		Namespace: release.Namespace,
		Version:   release.Version,
	}

	if release.Chart != nil && release.Chart.Metadata != nil {
		diag.Chart = fmt.Sprintf("%s-%s", release.Chart.Metadata.Name, release.Chart.Metadata.Version)
		diag.AppVersion = release.Chart.Metadata.AppVersion
	}

	if release.Info != nil {
		diag.Status = string(release.Info.Status)
		diag.Description = release.Info.Description
		diag.Notes = release.Info.Notes
	}

	return diag, nil
}

// CollectHelmDiagnosticsSimple collects Helm diagnostics without requiring the Collector
// This is useful when you don't have a full Collector set up
func CollectHelmDiagnosticsSimple(releaseName, namespace string) (*HelmDiagnostic, error) {
	if namespace == "" {
		namespace = "default"
	}

	settings := cli.New()
	actionConfig := new(action.Configuration)

	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, "secret", nil); err != nil {
		return nil, fmt.Errorf("initializing helm config: %w", err)
	}

	statusAction := action.NewStatus(actionConfig)
	release, err := statusAction.Run(releaseName)
	if err != nil {
		return &HelmDiagnostic{
			Name:      releaseName,
			Namespace: namespace,
			Status:    "not-found",
			Error:     err.Error(),
		}, nil
	}

	diag := &HelmDiagnostic{
		Name:      release.Name,
		Namespace: release.Namespace,
		Version:   release.Version,
	}

	if release.Chart != nil && release.Chart.Metadata != nil {
		diag.Chart = fmt.Sprintf("%s-%s", release.Chart.Metadata.Name, release.Chart.Metadata.Version)
		diag.AppVersion = release.Chart.Metadata.AppVersion
	}

	if release.Info != nil {
		diag.Status = string(release.Info.Status)
		diag.Description = release.Info.Description
		diag.Notes = release.Info.Notes
	}

	return diag, nil
}

// IsFailed returns true if the Helm release is in a failed state
func (h *HelmDiagnostic) IsFailed() bool {
	return h.Status == "failed" ||
		h.Status == "pending-install" ||
		h.Status == "pending-upgrade" ||
		h.Status == "pending-rollback" ||
		h.Status == "not-found"
}

// GetHelmReleaseStatus gets the status of a Helm release
// Returns status string, description, and any error
func GetHelmReleaseStatus(releaseName, namespace string) (status string, description string, err error) {
	diag, err := CollectHelmDiagnosticsSimple(releaseName, namespace)
	if err != nil {
		return "", "", err
	}
	return diag.Status, diag.Description, nil
}
