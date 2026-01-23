package eso

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanzi/kindplane/internal/helm"
)

const (
	// ESONamespace is the namespace where ESO is installed
	ESONamespace = "external-secrets"

	// ESORepoURL is the ESO Helm repository URL
	ESORepoURL   = "https://charts.external-secrets.io"
	ESORepoName  = "external-secrets"
	ESOChartName = "external-secrets"
)

// Installer handles External Secrets Operator installation
type Installer struct {
	kubeClient    *kubernetes.Clientset
	helmInstaller *helm.Installer
}

// Status represents ESO installation status
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

// NewInstaller creates a new ESO installer
func NewInstaller(kubeClient *kubernetes.Clientset) *Installer {
	return &Installer{
		kubeClient:    kubeClient,
		helmInstaller: helm.NewInstaller(kubeClient),
	}
}

// Install installs External Secrets Operator using Helm
func (i *Installer) Install(ctx context.Context, version string) error {
	// Add ESO Helm repo
	if err := i.helmInstaller.AddRepo(ctx, ESORepoName, ESORepoURL); err != nil {
		return fmt.Errorf("failed to add external-secrets repo: %w", err)
	}

	// Install ESO chart
	spec := helm.ChartSpec{
		RepoURL:     ESORepoURL,
		RepoName:    ESORepoName,
		ChartName:   ESOChartName,
		ReleaseName: "external-secrets",
		Namespace:   ESONamespace,
		Version:     version,
		Wait:        true,
		Timeout:     5 * time.Minute,
		Values: map[string]interface{}{
			"installCRDs": true,
		},
	}

	if err := i.helmInstaller.Install(ctx, spec); err != nil {
		return fmt.Errorf("failed to install external-secrets: %w", err)
	}

	return nil
}

// WaitForReady waits for ESO to be ready
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

// isReady checks if ESO pods are ready
func (i *Installer) isReady(ctx context.Context) (bool, error) {
	pods, err := i.kubeClient.CoreV1().Pods(ESONamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=external-secrets",
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

// GetStatus returns the current ESO status
func (i *Installer) GetStatus(ctx context.Context) (*Status, error) {
	status := &Status{
		Installed: false,
		Ready:     false,
		Pods:      []PodStatus{},
	}

	// Check if namespace exists
	_, err := i.kubeClient.CoreV1().Namespaces().Get(ctx, ESONamespace, metav1.GetOptions{})
	if err != nil {
		return status, nil // Not installed
	}

	status.Installed = true

	// Get pods
	pods, err := i.kubeClient.CoreV1().Pods(ESONamespace).List(ctx, metav1.ListOptions{})
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
	deployments, err := i.kubeClient.AppsV1().Deployments(ESONamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=external-secrets",
	})
	if err == nil && len(deployments.Items) > 0 {
		for _, container := range deployments.Items[0].Spec.Template.Spec.Containers {
			if container.Name == "external-secrets" {
				status.Version = container.Image
				break
			}
		}
	}

	return status, nil
}

// Uninstall removes External Secrets Operator
func (i *Installer) Uninstall(ctx context.Context) error {
	return i.helmInstaller.Uninstall(ctx, "external-secrets", ESONamespace)
}
