package crossplane

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/kanzi/kindplane/internal/helm"
)

const (
	// CrossplaneNamespace is the namespace where Crossplane is installed
	CrossplaneNamespace = "crossplane-system"

	// CrossplaneRepoURL is the Crossplane Helm repository URL
	CrossplaneRepoURL   = "https://charts.crossplane.io/stable"
	CrossplaneRepoName  = "crossplane-stable"
	CrossplaneChartName = "crossplane"
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
func (i *Installer) Install(ctx context.Context, version string) error {
	// Add Crossplane Helm repo
	if err := i.helmInstaller.AddRepo(ctx, CrossplaneRepoName, CrossplaneRepoURL); err != nil {
		return fmt.Errorf("failed to add crossplane repo: %w", err)
	}

	// Install Crossplane chart
	spec := helm.ChartSpec{
		RepoURL:     CrossplaneRepoURL,
		RepoName:    CrossplaneRepoName,
		ChartName:   CrossplaneChartName,
		ReleaseName: "crossplane",
		Namespace:   CrossplaneNamespace,
		Version:     version,
		Wait:        true,
		Timeout:     5 * time.Minute,
		Values:      map[string]interface{}{},
	}

	if err := i.helmInstaller.Install(ctx, spec); err != nil {
		return fmt.Errorf("failed to install crossplane: %w", err)
	}

	return nil
}

// WaitForReady waits for Crossplane to be ready
func (i *Installer) WaitForReady(ctx context.Context) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			ready, err := i.isReady(ctx)
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
	pods, err := i.kubeClient.CoreV1().Pods(CrossplaneNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=crossplane",
	})
	if err != nil {
		return false, err
	}

	if len(pods.Items) == 0 {
		return false, nil
	}

	for _, pod := range pods.Items {
		for _, cond := range pod.Status.Conditions {
			if cond.Type == "Ready" && cond.Status != "True" {
				return false, nil
			}
		}
	}

	return true, nil
}

// InstallProvider installs a Crossplane provider
// name is the Kubernetes resource name for the provider
// pkg is the full OCI package path (e.g., xpkg.upbound.io/upbound/provider-aws:v1.1.0)
func (i *Installer) InstallProvider(ctx context.Context, name, pkg string) error {
	// Create provider manifest
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

	// Get dynamic client
	dynamicClient, err := i.getDynamicClient()
	if err != nil {
		return fmt.Errorf("failed to get dynamic client: %w", err)
	}

	// Create or update provider
	gvr := schema.GroupVersionResource{
		Group:    "pkg.crossplane.io",
		Version:  "v1",
		Resource: "providers",
	}

	_, err = dynamicClient.Resource(gvr).Create(ctx, provider, metav1.CreateOptions{})
	if err != nil {
		// Try update if create fails
		_, err = dynamicClient.Resource(gvr).Update(ctx, provider, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create/update provider: %w", err)
		}
	}

	return nil
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
