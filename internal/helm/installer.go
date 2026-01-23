package helm

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/client-go/kubernetes"

	"github.com/kanzi/kindplane/internal/config"
)

// Installer handles Helm chart installations
type Installer struct {
	kubeClient *kubernetes.Clientset
	settings   *cli.EnvSettings
}

// ChartSpec defines a Helm chart to install
type ChartSpec struct {
	RepoURL     string
	RepoName    string
	ChartName   string
	ReleaseName string
	Namespace   string
	Version     string
	Values      map[string]interface{}
	Wait        bool
	Timeout     time.Duration
}

// NewInstaller creates a new Helm installer
func NewInstaller(kubeClient *kubernetes.Clientset) *Installer {
	settings := cli.New()
	return &Installer{
		kubeClient: kubeClient,
		settings:   settings,
	}
}

// AddRepo adds a Helm repository
func (i *Installer) AddRepo(ctx context.Context, name, url string) error {
	repoFile := i.settings.RepositoryConfig

	// Load existing repo file
	r, err := repo.LoadFile(repoFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load repo file: %w", err)
	}
	if r == nil {
		r = repo.NewFile()
	}

	// Check if repo already exists
	for _, entry := range r.Repositories {
		if entry.Name == name {
			// Update URL if different
			if entry.URL != url {
				entry.URL = url
			}
			return r.WriteFile(repoFile, 0644)
		}
	}

	// Add new repo entry
	entry := &repo.Entry{
		Name: name,
		URL:  url,
	}

	r.Add(entry)

	if err := r.WriteFile(repoFile, 0644); err != nil {
		return fmt.Errorf("failed to write repo file: %w", err)
	}

	// Download index
	chartRepo, err := repo.NewChartRepository(entry, getter.All(i.settings))
	if err != nil {
		return fmt.Errorf("failed to create chart repository: %w", err)
	}

	if _, err := chartRepo.DownloadIndexFile(); err != nil {
		return fmt.Errorf("failed to download index: %w", err)
	}

	return nil
}

// Install installs a Helm chart
func (i *Installer) Install(ctx context.Context, spec ChartSpec) error {
	// Create action configuration
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(i.settings.RESTClientGetter(), spec.Namespace, "secret", debugLog); err != nil {
		return fmt.Errorf("failed to init action config: %w", err)
	}

	// Check if release already exists
	listAction := action.NewList(actionConfig)
	listAction.Filter = spec.ReleaseName
	releases, err := listAction.Run()
	if err != nil {
		return fmt.Errorf("failed to list releases: %w", err)
	}

	// If release exists, upgrade instead
	if len(releases) > 0 {
		return i.Upgrade(ctx, spec)
	}

	// Create install action
	installAction := action.NewInstall(actionConfig)
	installAction.ReleaseName = spec.ReleaseName
	installAction.Namespace = spec.Namespace
	installAction.CreateNamespace = true
	installAction.Wait = spec.Wait
	if spec.Timeout > 0 {
		installAction.Timeout = spec.Timeout
	} else {
		installAction.Timeout = 5 * time.Minute
	}
	if spec.Version != "" {
		installAction.Version = spec.Version
	}

	// Locate chart
	chartPath, err := installAction.ChartPathOptions.LocateChart(
		fmt.Sprintf("%s/%s", spec.RepoName, spec.ChartName),
		i.settings,
	)
	if err != nil {
		return fmt.Errorf("failed to locate chart: %w", err)
	}

	// Load chart
	chart, err := loader.Load(chartPath)
	if err != nil {
		return fmt.Errorf("failed to load chart: %w", err)
	}

	// Install chart
	_, err = installAction.RunWithContext(ctx, chart, spec.Values)
	if err != nil {
		return fmt.Errorf("failed to install chart: %w", err)
	}

	return nil
}

// Upgrade upgrades a Helm release
func (i *Installer) Upgrade(ctx context.Context, spec ChartSpec) error {
	// Create action configuration
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(i.settings.RESTClientGetter(), spec.Namespace, "secret", debugLog); err != nil {
		return fmt.Errorf("failed to init action config: %w", err)
	}

	// Create upgrade action
	upgradeAction := action.NewUpgrade(actionConfig)
	upgradeAction.Namespace = spec.Namespace
	upgradeAction.Wait = spec.Wait
	if spec.Timeout > 0 {
		upgradeAction.Timeout = spec.Timeout
	} else {
		upgradeAction.Timeout = 5 * time.Minute
	}
	if spec.Version != "" {
		upgradeAction.Version = spec.Version
	}

	// Locate chart
	chartPath, err := upgradeAction.ChartPathOptions.LocateChart(
		fmt.Sprintf("%s/%s", spec.RepoName, spec.ChartName),
		i.settings,
	)
	if err != nil {
		return fmt.Errorf("failed to locate chart: %w", err)
	}

	// Load chart
	chart, err := loader.Load(chartPath)
	if err != nil {
		return fmt.Errorf("failed to load chart: %w", err)
	}

	// Upgrade release
	_, err = upgradeAction.RunWithContext(ctx, spec.ReleaseName, chart, spec.Values)
	if err != nil {
		return fmt.Errorf("failed to upgrade release: %w", err)
	}

	return nil
}

// Uninstall removes a Helm release
func (i *Installer) Uninstall(ctx context.Context, releaseName, namespace string) error {
	// Create action configuration
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(i.settings.RESTClientGetter(), namespace, "secret", debugLog); err != nil {
		return fmt.Errorf("failed to init action config: %w", err)
	}

	// Create uninstall action
	uninstallAction := action.NewUninstall(actionConfig)

	// Uninstall release
	_, err := uninstallAction.Run(releaseName)
	if err != nil {
		return fmt.Errorf("failed to uninstall release: %w", err)
	}

	return nil
}

// IsInstalled checks if a release is installed
func (i *Installer) IsInstalled(ctx context.Context, releaseName, namespace string) (bool, error) {
	// Create action configuration
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(i.settings.RESTClientGetter(), namespace, "secret", debugLog); err != nil {
		return false, fmt.Errorf("failed to init action config: %w", err)
	}

	// Check if release exists
	listAction := action.NewList(actionConfig)
	listAction.Filter = releaseName
	releases, err := listAction.Run()
	if err != nil {
		return false, fmt.Errorf("failed to list releases: %w", err)
	}

	return len(releases) > 0, nil
}

// debugLog is a simple logger for Helm actions
func debugLog(format string, v ...interface{}) {
	// Suppress debug logs by default
	// Uncomment the following line for debugging:
	// fmt.Printf(format+"\n", v...)
}

// InstallChartFromConfig installs a Helm chart from a ChartConfig
func (i *Installer) InstallChartFromConfig(ctx context.Context, chartCfg config.ChartConfig) error {
	// Generate a unique repo name from the URL
	repoName := GenerateRepoName(chartCfg.Repo)

	// Add the repo
	if err := i.AddRepo(ctx, repoName, chartCfg.Repo); err != nil {
		return fmt.Errorf("failed to add repo: %w", err)
	}

	// Merge values from files and inline values
	values, err := MergeValues(chartCfg.ValuesFiles, chartCfg.Values)
	if err != nil {
		return fmt.Errorf("failed to merge values: %w", err)
	}

	// Parse timeout
	timeout := 5 * time.Minute
	if chartCfg.Timeout != "" {
		parsed, err := time.ParseDuration(chartCfg.Timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout: %w", err)
		}
		timeout = parsed
	}

	// Create chart spec
	spec := ChartSpec{
		RepoURL:     chartCfg.Repo,
		RepoName:    repoName,
		ChartName:   chartCfg.Chart,
		ReleaseName: chartCfg.Name,
		Namespace:   chartCfg.Namespace,
		Version:     chartCfg.Version,
		Values:      values,
		Wait:        chartCfg.ShouldWait(),
		Timeout:     timeout,
	}

	// Install with CreateNamespace option
	return i.InstallWithOptions(ctx, spec, chartCfg.ShouldCreateNamespace())
}

// InstallWithOptions installs a Helm chart with additional options
func (i *Installer) InstallWithOptions(ctx context.Context, spec ChartSpec, createNamespace bool) error {
	// Create action configuration
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(i.settings.RESTClientGetter(), spec.Namespace, "secret", debugLog); err != nil {
		return fmt.Errorf("failed to init action config: %w", err)
	}

	// Check if release already exists
	listAction := action.NewList(actionConfig)
	listAction.Filter = fmt.Sprintf("^%s$", spec.ReleaseName)
	releases, err := listAction.Run()
	if err != nil {
		return fmt.Errorf("failed to list releases: %w", err)
	}

	// If release exists, upgrade instead
	if len(releases) > 0 {
		return i.Upgrade(ctx, spec)
	}

	// Create install action
	installAction := action.NewInstall(actionConfig)
	installAction.ReleaseName = spec.ReleaseName
	installAction.Namespace = spec.Namespace
	installAction.CreateNamespace = createNamespace
	installAction.Wait = spec.Wait
	if spec.Timeout > 0 {
		installAction.Timeout = spec.Timeout
	} else {
		installAction.Timeout = 5 * time.Minute
	}
	if spec.Version != "" {
		installAction.Version = spec.Version
	}

	// Locate chart
	chartPath, err := installAction.ChartPathOptions.LocateChart(
		fmt.Sprintf("%s/%s", spec.RepoName, spec.ChartName),
		i.settings,
	)
	if err != nil {
		return fmt.Errorf("failed to locate chart: %w", err)
	}

	// Load chart
	chart, err := loader.Load(chartPath)
	if err != nil {
		return fmt.Errorf("failed to load chart: %w", err)
	}

	// Install chart
	_, err = installAction.RunWithContext(ctx, chart, spec.Values)
	if err != nil {
		return fmt.Errorf("failed to install chart: %w", err)
	}

	return nil
}

// ReleaseInfo contains information about a Helm release
type ReleaseInfo struct {
	Name       string
	Namespace  string
	Revision   int
	Status     string
	Chart      string
	AppVersion string
	Updated    time.Time
}

// ListReleases lists all Helm releases, optionally filtered by namespace
func (i *Installer) ListReleases(ctx context.Context, namespace string) ([]ReleaseInfo, error) {
	// Create action configuration
	// Use empty namespace to list all namespaces
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(i.settings.RESTClientGetter(), namespace, "secret", debugLog); err != nil {
		return nil, fmt.Errorf("failed to init action config: %w", err)
	}

	// Create list action
	listAction := action.NewList(actionConfig)
	if namespace == "" {
		listAction.AllNamespaces = true
	}
	listAction.SetStateMask()

	releases, err := listAction.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to list releases: %w", err)
	}

	var result []ReleaseInfo
	for _, rel := range releases {
		info := ReleaseInfo{
			Name:       rel.Name,
			Namespace:  rel.Namespace,
			Revision:   rel.Version,
			Status:     string(rel.Info.Status),
			Chart:      fmt.Sprintf("%s-%s", rel.Chart.Metadata.Name, rel.Chart.Metadata.Version),
			AppVersion: rel.Chart.Metadata.AppVersion,
		}
		if rel.Info.LastDeployed.IsZero() == false {
			info.Updated = rel.Info.LastDeployed.Time
		}
		result = append(result, info)
	}

	return result, nil
}

// UninstallRelease removes a Helm release by name and namespace
func (i *Installer) UninstallRelease(ctx context.Context, releaseName, namespace string) error {
	return i.Uninstall(ctx, releaseName, namespace)
}

// GenerateRepoName creates a unique repo name from a URL
func GenerateRepoName(url string) string {
	// Create a short hash of the URL for uniqueness
	hash := sha256.Sum256([]byte(url))
	shortHash := fmt.Sprintf("%x", hash[:4])

	// Extract a readable name from the URL
	name := url
	// Remove protocol
	name = strings.TrimPrefix(name, "https://")
	name = strings.TrimPrefix(name, "http://")
	// Take first part before /
	if idx := strings.Index(name, "/"); idx > 0 {
		name = name[:idx]
	}
	// Replace dots with dashes
	name = strings.ReplaceAll(name, ".", "-")
	// Truncate if too long
	if len(name) > 20 {
		name = name[:20]
	}

	return fmt.Sprintf("%s-%s", name, shortHash)
}
