package credentials

import (
	"context"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// KubernetesProviderConfigName is the name of the Kubernetes ProviderConfig
	KubernetesProviderConfigName = "default"
)

// ConfigureKubernetesInCluster creates Kubernetes provider config for in-cluster usage
func (m *Manager) ConfigureKubernetesInCluster(ctx context.Context) error {
	return m.createKubernetesProviderConfig(ctx, "InjectedIdentity", nil)
}

// ConfigureKubernetesFromKubeconfig creates Kubernetes provider config using kubeconfig
func (m *Manager) ConfigureKubernetesFromKubeconfig(ctx context.Context) error {
	// Read kubeconfig file
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		kubeconfigPath = fmt.Sprintf("%s/.kube/config", home)
	}

	kubeconfigData, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to read kubeconfig: %w", err)
	}

	// Create secret with kubeconfig
	secretName := "kubernetes-provider-creds"
	if err := m.createSecret(ctx, secretName, CrossplaneSystemNamespace, map[string][]byte{
		"kubeconfig": kubeconfigData,
	}); err != nil {
		return fmt.Errorf("failed to create kubeconfig secret: %w", err)
	}

	// Create ProviderConfig
	secretRef := map[string]interface{}{
		"namespace": CrossplaneSystemNamespace,
		"name":      secretName,
		"key":       "kubeconfig",
	}
	return m.createKubernetesProviderConfig(ctx, "Secret", secretRef)
}

// createKubernetesProviderConfig creates the Kubernetes ProviderConfig resource
func (m *Manager) createKubernetesProviderConfig(ctx context.Context, source string, secretRef map[string]interface{}) error {
	credentials := map[string]interface{}{
		"source": source,
	}
	if secretRef != nil {
		credentials["secretRef"] = secretRef
	}

	providerConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kubernetes.crossplane.io/v1alpha1",
			"kind":       "ProviderConfig",
			"metadata": map[string]interface{}{
				"name": KubernetesProviderConfigName,
			},
			"spec": map[string]interface{}{
				"credentials": credentials,
			},
		},
	}

	dynamicClient, err := m.getDynamicClient()
	if err != nil {
		return err
	}

	gvr := schema.GroupVersionResource{
		Group:    "kubernetes.crossplane.io",
		Version:  "v1alpha1",
		Resource: "providerconfigs",
	}

	_, err = dynamicClient.Resource(gvr).Create(ctx, providerConfig, metav1.CreateOptions{})
	if err != nil {
		// Try update if create fails
		_, err = dynamicClient.Resource(gvr).Update(ctx, providerConfig, metav1.UpdateOptions{})
	}

	return err
}
