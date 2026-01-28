package crossplane

import (
	"context"
	"fmt"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/kanzi/kindplane/internal/config"
	"github.com/kanzi/kindplane/internal/helm"
)

const (
	// CrossplaneNamespace is the namespace where Crossplane is installed
	CrossplaneNamespace = "crossplane-system"

	// CrossplaneRepoURL is the Crossplane Helm repository URL
	CrossplaneRepoURL   = "https://charts.crossplane.io/stable"
	CrossplaneRepoName  = "crossplane-stable"
	CrossplaneChartName = "crossplane"

	// RegistryCaBundleConfigMapName is the name of the ConfigMap containing the registry CA bundle
	RegistryCaBundleConfigMapName = "crossplane-registry-ca-bundle"
	// RegistryCaBundleConfigMapKey is the key in the ConfigMap containing the CA bundle
	RegistryCaBundleConfigMapKey = "ca-bundle"
)

// Installer handles Crossplane installation and management
type Installer struct {
	kubeClient    *kubernetes.Clientset
	dynamicClient dynamic.Interface
	helmInstaller *helm.Installer
}

// Status represents Crossplane installation status
type Status struct {
	Installed bool
	Ready     bool
	Version   string
	Pods      []PodStatus
}

// PodStatus represents a pod's status
type PodStatus struct {
	Name  string
	Ready bool
	Phase string
}

// PodInfo represents pod information for UI display
type PodInfo struct {
	Name    string
	Status  string // Pod phase (Pending, Running, Succeeded, Failed)
	Ready   bool
	Message string // Optional status message
}

// ProviderStatus represents a Crossplane provider's status
type ProviderStatus struct {
	Name    string
	Package string
	Version string
	Healthy bool
	Message string
}

// NewInstaller creates a new Crossplane installer
func NewInstaller(kubeClient *kubernetes.Clientset) *Installer {
	return &Installer{
		kubeClient:    kubeClient,
		helmInstaller: helm.NewInstaller(kubeClient),
	}
}

// Install installs Crossplane using Helm
// This is a convenience method that calls all installation steps in sequence.
// For progress tracking, use the individual step methods instead.
func (i *Installer) Install(ctx context.Context, fullConfig *config.Config) error {
	cfg := fullConfig.Crossplane

	// Determine repository URL (use custom if provided, otherwise default)
	repoURL := cfg.Repo
	if repoURL == "" {
		repoURL = CrossplaneRepoURL
	}

	// Generate repo name from URL
	repoName := CrossplaneRepoName
	if cfg.Repo != "" {
		repoName = helm.GenerateRepoName(repoURL)
	}

	// Add Helm repository
	if err := i.AddHelmRepo(ctx, repoName, repoURL); err != nil {
		return err
	}

	// Ensure namespace exists
	if err := i.EnsureNamespace(ctx); err != nil {
		return err
	}

	// Create registry CA bundle if configured
	if cfg.RegistryCaBundle != nil {
		if err := i.CreateRegistryCaBundle(ctx, cfg.RegistryCaBundle, fullConfig.Cluster.TrustedCAs.Workloads); err != nil {
			return err
		}
	}

	// Install Helm chart
	if err := i.InstallHelmChart(ctx, cfg, repoURL, repoName); err != nil {
		return err
	}

	return nil
}

// AddHelmRepo adds the Crossplane Helm repository
func (i *Installer) AddHelmRepo(ctx context.Context, repoName, repoURL string) error {
	if err := i.helmInstaller.AddRepo(ctx, repoName, repoURL); err != nil {
		return fmt.Errorf("failed to add crossplane repo: %w", err)
	}
	return nil
}

// EnsureNamespace ensures the Crossplane namespace exists
func (i *Installer) EnsureNamespace(ctx context.Context) error {
	return helm.EnsureNamespace(ctx, i.kubeClient, CrossplaneNamespace)
}

// CreateRegistryCaBundle creates the registry CA bundle ConfigMap if configured
func (i *Installer) CreateRegistryCaBundle(ctx context.Context, rcb *config.RegistryCaBundleConfig, workloadCAs []config.WorkloadCA) error {
	if err := i.createRegistryCaBundleConfigMap(ctx, rcb, workloadCAs); err != nil {
		return fmt.Errorf("failed to create registry CA bundle ConfigMap: %w", err)
	}
	return nil
}

// InstallHelmChart installs the Crossplane Helm chart
// Deprecated: Use InstallHelmChartWithOptions for more control over the installation process.
func (i *Installer) InstallHelmChart(ctx context.Context, cfg config.CrossplaneConfig, repoURL, repoName string) error {
	return i.InstallHelmChartWithOptions(ctx, cfg, repoURL, repoName, helm.InstallOptions{})
}

// InstallHelmChartWithOptions installs the Crossplane Helm chart with optional callbacks.
// The opts parameter allows for value transformation, logging, and pre-install hooks.
// Note: This method automatically injects registryCaBundleConfig values if cfg.RegistryCaBundle is set.
func (i *Installer) InstallHelmChartWithOptions(ctx context.Context, cfg config.CrossplaneConfig, repoURL, repoName string, opts helm.InstallOptions) error {
	// Build ChartConfig from CrossplaneConfig
	chartCfg := config.ChartConfig{
		Name:        "crossplane",
		Repo:        repoURL,
		Chart:       CrossplaneChartName,
		Version:     cfg.Version,
		Namespace:   CrossplaneNamespace,
		Values:      cfg.Values,
		ValuesFiles: cfg.ValuesFiles,
	}

	// Wrap the provided transformer to also inject registryCaBundleConfig
	originalTransformer := opts.ValuesTransformer
	opts.ValuesTransformer = func(values map[string]interface{}) map[string]interface{} {
		// Apply original transformer first if provided
		if originalTransformer != nil {
			values = originalTransformer(values)
		}
		// Inject registryCaBundleConfig if configured
		if cfg.RegistryCaBundle != nil {
			if values == nil {
				values = make(map[string]interface{})
			}
			values["registryCaBundleConfig"] = map[string]interface{}{
				"name": RegistryCaBundleConfigMapName,
				"key":  RegistryCaBundleConfigMapKey,
			}
		}
		return values
	}

	// Use the helm installer with options
	if err := i.helmInstaller.InstallChartFromConfigWithOptions(ctx, chartCfg, opts); err != nil {
		return fmt.Errorf("failed to install crossplane: %w", err)
	}

	return nil
}

// createRegistryCaBundleConfigMap creates a ConfigMap containing the CA bundle for Crossplane registry access
// Multiple CA certificates are bundled together into a single PEM file
// Note: The namespace must already exist before calling this function.
func (i *Installer) createRegistryCaBundleConfigMap(ctx context.Context, rcb *config.RegistryCaBundleConfig, workloadCAs []config.WorkloadCA) error {
	// Resolve all CA file paths
	caFilePaths, err := rcb.ResolveCAFiles(workloadCAs)
	if err != nil {
		return fmt.Errorf("failed to resolve CA files: %w", err)
	}

	// Read and bundle all CA certificates
	var bundledCerts []byte
	for _, caFilePath := range caFilePaths {
		caContent, err := os.ReadFile(caFilePath)
		if err != nil {
			return fmt.Errorf("failed to read CA file %s: %w", caFilePath, err)
		}
		// Ensure each certificate ends with a newline for proper concatenation
		if len(caContent) > 0 && caContent[len(caContent)-1] != '\n' {
			caContent = append(caContent, '\n')
		}
		bundledCerts = append(bundledCerts, caContent...)
	}

	// Create or update the ConfigMap
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RegistryCaBundleConfigMapName,
			Namespace: CrossplaneNamespace,
		},
		Data: map[string]string{
			RegistryCaBundleConfigMapKey: string(bundledCerts),
		},
	}

	// Try to get existing ConfigMap
	_, err = i.kubeClient.CoreV1().ConfigMaps(CrossplaneNamespace).Get(ctx, RegistryCaBundleConfigMapName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// ConfigMap doesn't exist, create it
			_, err = i.kubeClient.CoreV1().ConfigMaps(CrossplaneNamespace).Create(ctx, configMap, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create ConfigMap %s: %w", RegistryCaBundleConfigMapName, err)
			}
		} else {
			return fmt.Errorf("failed to check ConfigMap %s: %w", RegistryCaBundleConfigMapName, err)
		}
	} else {
		// ConfigMap exists, update it
		_, err = i.kubeClient.CoreV1().ConfigMaps(CrossplaneNamespace).Update(ctx, configMap, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update ConfigMap %s: %w", RegistryCaBundleConfigMapName, err)
		}
	}

	return nil
}

// WaitForReady waits for Crossplane to be ready
func (i *Installer) WaitForReady(ctx context.Context) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Check immediately first
	ready, err := i.isReady(ctx)
	if err == nil && ready {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Create a timeout context for the isReady check to prevent hanging
			checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			ready, err := i.isReady(checkCtx)
			cancel()

			// If context was cancelled, return immediately
			if ctx.Err() != nil {
				return ctx.Err()
			}

			if err != nil {
				continue // Keep trying
			}
			if ready {
				return nil
			}
		}
	}
}

// isReady checks if Crossplane pods are ready
func (i *Installer) isReady(ctx context.Context) (bool, error) {
	// Check if context is already cancelled before making API call
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	pods, err := i.kubeClient.CoreV1().Pods(CrossplaneNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=crossplane",
	})
	if err != nil {
		// If context was cancelled during the call, return the cancellation error
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		return false, err
	}

	if len(pods.Items) == 0 {
		return false, nil
	}

	for _, pod := range pods.Items {
		// Check context cancellation during iteration
		if ctx.Err() != nil {
			return false, ctx.Err()
		}

		for _, cond := range pod.Status.Conditions {
			if cond.Type == "Ready" && cond.Status != "True" {
				return false, nil
			}
		}
	}

	return true, nil
}

// GetPodStatus returns pod status information for UI display
// Returns list of pod info, whether all pods are ready, and any error
func (i *Installer) GetPodStatus(ctx context.Context) ([]PodInfo, bool, error) {
	pods, err := i.kubeClient.CoreV1().Pods(CrossplaneNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=crossplane",
	})
	if err != nil {
		// If context was cancelled during the call, return the cancellation error
		if ctx.Err() != nil {
			return nil, false, ctx.Err()
		}
		return nil, false, err
	}

	if len(pods.Items) == 0 {
		return []PodInfo{}, false, nil
	}

	var podInfos []PodInfo
	allReady := true

	for _, pod := range pods.Items {
		// Check if pod is ready
		podReady := false
		var message string
		for _, cond := range pod.Status.Conditions {
			if cond.Type == "Ready" {
				if cond.Status == "True" {
					podReady = true
				} else {
					message = cond.Message
				}
				break
			}
		}

		// If no Ready condition found, check container statuses
		if !podReady && len(pod.Status.ContainerStatuses) > 0 {
			for _, cs := range pod.Status.ContainerStatuses {
				if cs.State.Waiting != nil {
					message = cs.State.Waiting.Reason
					if cs.State.Waiting.Message != "" {
						message = cs.State.Waiting.Message
					}
				} else if cs.State.Running != nil {
					// Container is running but not ready yet
					if !cs.Ready {
						message = "Starting..."
					}
				}
			}
		}

		podInfos = append(podInfos, PodInfo{
			Name:    pod.Name,
			Status:  string(pod.Status.Phase),
			Ready:   podReady,
			Message: message,
		})

		if !podReady {
			allReady = false
		}
	}

	return podInfos, allReady && len(podInfos) > 0, nil
}

// InstallProvider installs a Crossplane provider
// name is the Kubernetes resource name for the provider
// pkg is the full OCI package path (e.g., xpkg.upbound.io/upbound/provider-aws:v1.1.0)
func (i *Installer) InstallProvider(ctx context.Context, name, pkg string) error {
	// Get dynamic client
	dynamicClient, err := i.getDynamicClient()
	if err != nil {
		return fmt.Errorf("failed to get dynamic client: %w", err)
	}

	gvr := schema.GroupVersionResource{
		Group:    "pkg.crossplane.io",
		Version:  "v1",
		Resource: "providers",
	}

	// Check if provider already exists
	existing, err := dynamicClient.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		// Provider exists - update it with the current resourceVersion
		existing.Object["spec"] = map[string]interface{}{
			"package": pkg,
		}
		_, err = dynamicClient.Resource(gvr).Update(ctx, existing, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update provider: %w", err)
		}
		return nil
	}

	// Provider doesn't exist - create it
	provider := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "pkg.crossplane.io/v1",
			"kind":       "Provider",
			"metadata": map[string]interface{}{
				"name": name,
			},
			"spec": map[string]interface{}{
				"package": pkg,
			},
		},
	}

	_, err = dynamicClient.Resource(gvr).Create(ctx, provider, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	return nil
}

// DeleteProvider removes a Crossplane provider by name
func (i *Installer) DeleteProvider(ctx context.Context, name string) error {
	dynamicClient, err := i.getDynamicClient()
	if err != nil {
		return fmt.Errorf("failed to get dynamic client: %w", err)
	}

	gvr := schema.GroupVersionResource{
		Group:    "pkg.crossplane.io",
		Version:  "v1",
		Resource: "providers",
	}

	err = dynamicClient.Resource(gvr).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete provider: %w", err)
	}

	return nil
}

// ProviderExists checks if a provider with the given name exists
func (i *Installer) ProviderExists(ctx context.Context, name string) (bool, error) {
	dynamicClient, err := i.getDynamicClient()
	if err != nil {
		return false, fmt.Errorf("failed to get dynamic client: %w", err)
	}

	gvr := schema.GroupVersionResource{
		Group:    "pkg.crossplane.io",
		Version:  "v1",
		Resource: "providers",
	}

	_, err = dynamicClient.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return false, nil
	}

	return true, nil
}

// WaitForProviders waits for all providers to be healthy
func (i *Installer) WaitForProviders(ctx context.Context) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			providers, err := i.GetProviderStatus(ctx)
			if err != nil {
				continue // Keep trying
			}

			allHealthy := true
			for _, p := range providers {
				if !p.Healthy {
					allHealthy = false
					break
				}
			}

			if allHealthy && len(providers) > 0 {
				return nil
			}
		}
	}
}

// GetStatus returns the current Crossplane status
func (i *Installer) GetStatus(ctx context.Context) (*Status, error) {
	status := &Status{
		Installed: false,
		Ready:     false,
		Pods:      []PodStatus{},
	}

	// Check if namespace exists
	_, err := i.kubeClient.CoreV1().Namespaces().Get(ctx, CrossplaneNamespace, metav1.GetOptions{})
	if err != nil {
		return status, nil // Not installed
	}

	status.Installed = true

	// Get pods
	pods, err := i.kubeClient.CoreV1().Pods(CrossplaneNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return status, fmt.Errorf("failed to list pods: %w", err)
	}

	allReady := true
	for _, pod := range pods.Items {
		podReady := false
		for _, cond := range pod.Status.Conditions {
			if cond.Type == "Ready" && cond.Status == "True" {
				podReady = true
				break
			}
		}

		status.Pods = append(status.Pods, PodStatus{
			Name:  pod.Name,
			Ready: podReady,
			Phase: string(pod.Status.Phase),
		})

		if !podReady {
			allReady = false
		}
	}

	status.Ready = allReady

	// Try to get version from deployment
	deployments, err := i.kubeClient.AppsV1().Deployments(CrossplaneNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=crossplane",
	})
	if err == nil && len(deployments.Items) > 0 {
		for _, container := range deployments.Items[0].Spec.Template.Spec.Containers {
			if container.Name == "crossplane" {
				// Extract version from image tag
				status.Version = container.Image
				break
			}
		}
	}

	return status, nil
}

// GetProviderStatus returns the status of all installed providers
func (i *Installer) GetProviderStatus(ctx context.Context) ([]ProviderStatus, error) {
	dynamicClient, err := i.getDynamicClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get dynamic client: %w", err)
	}

	gvr := schema.GroupVersionResource{
		Group:    "pkg.crossplane.io",
		Version:  "v1",
		Resource: "providers",
	}

	providers, err := dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}

	var statuses []ProviderStatus
	for _, p := range providers.Items {
		status := ProviderStatus{
			Name: p.GetName(),
		}

		// Get package spec
		if spec, found, _ := unstructured.NestedMap(p.Object, "spec"); found {
			if pkg, ok := spec["package"].(string); ok {
				status.Package = pkg
			}
		}

		// Get status
		if statusMap, found, _ := unstructured.NestedMap(p.Object, "status"); found {
			// Check conditions
			if conditions, found, _ := unstructured.NestedSlice(statusMap, "conditions"); found {
				for _, c := range conditions {
					cond, ok := c.(map[string]interface{})
					if !ok {
						continue
					}
					condType, _ := cond["type"].(string)
					condStatus, _ := cond["status"].(string)
					message, _ := cond["message"].(string)

					if condType == "Healthy" {
						status.Healthy = condStatus == "True"
						status.Message = message
					}
				}
			}

			// Get version
			if currentRevision, ok := statusMap["currentRevision"].(string); ok {
				status.Version = currentRevision
			}
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}

// getDynamicClient creates a dynamic Kubernetes client
func (i *Installer) getDynamicClient() (dynamic.Interface, error) {
	if i.dynamicClient != nil {
		return i.dynamicClient, nil
	}

	// Get rest config from kubeClient
	// This is a simplified approach - in production you'd want to share the rest.Config
	config, err := getRestConfig()
	if err != nil {
		return nil, err
	}

	i.dynamicClient, err = dynamic.NewForConfig(config)
	return i.dynamicClient, err
}
