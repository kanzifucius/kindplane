package credentials

import (
	"context"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// CrossplaneSystemNamespace is the namespace for Crossplane system resources
	CrossplaneSystemNamespace = "crossplane-system"
)

// Manager handles credential configuration for Crossplane providers
type Manager struct {
	kubeClient    *kubernetes.Clientset
	dynamicClient dynamic.Interface
}

// NewManager creates a new credentials manager
func NewManager(kubeClient *kubernetes.Clientset) *Manager {
	return &Manager{
		kubeClient: kubeClient,
	}
}

// getDynamicClient creates a dynamic Kubernetes client
func (m *Manager) getDynamicClient() (dynamic.Interface, error) {
	if m.dynamicClient != nil {
		return m.dynamicClient, nil
	}

	config, err := getRestConfig()
	if err != nil {
		return nil, err
	}

	m.dynamicClient, err = dynamic.NewForConfig(config)
	return m.dynamicClient, err
}

// getRestConfig returns the Kubernetes REST config
func getRestConfig() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		kubeconfigPath = fmt.Sprintf("%s/.kube/config", home)
	}

	return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
}

// createSecret creates a Kubernetes secret
func (m *Manager) createSecret(ctx context.Context, name, namespace string, data map[string][]byte) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}

	_, err := m.kubeClient.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		// Try update if create fails
		_, err = m.kubeClient.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	}
	return err
}

// CredentialInfo contains information about a configured credential
type CredentialInfo struct {
	Provider   string
	SecretName string
	Namespace  string
	Configured bool
	LastUpdate string
	ConfigName string
}

// ListCredentials returns information about configured credentials
func (m *Manager) ListCredentials(ctx context.Context, providerFilter string) ([]CredentialInfo, error) {
	var results []CredentialInfo

	// Known credential secret patterns
	secretPatterns := map[string][]string{
		"aws":        {"aws-credentials", "aws-creds"},
		"azure":      {"azure-credentials", "azure-creds"},
		"kubernetes": {"kubernetes-credentials", "kubernetes-creds", "cluster-kubeconfig"},
		"gcp":        {"gcp-credentials", "gcp-creds"},
	}

	// Filter patterns if provider specified
	if providerFilter != "" {
		if patterns, ok := secretPatterns[providerFilter]; ok {
			secretPatterns = map[string][]string{providerFilter: patterns}
		} else {
			return results, nil
		}
	}

	// Check for secrets
	secrets, err := m.kubeClient.CoreV1().Secrets(CrossplaneSystemNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	for provider, patterns := range secretPatterns {
		for _, pattern := range patterns {
			for _, secret := range secrets.Items {
				if secret.Name == pattern {
					info := CredentialInfo{
						Provider:   provider,
						SecretName: secret.Name,
						Namespace:  secret.Namespace,
						Configured: len(secret.Data) > 0,
						LastUpdate: secret.CreationTimestamp.Format("2006-01-02 15:04:05"),
					}
					results = append(results, info)
				}
			}
		}
	}

	// Also check for ProviderConfigs using dynamic client
	dynamicClient, err := m.getDynamicClient()
	if err == nil {
		providerConfigGVRs := map[string]struct {
			Group    string
			Resource string
		}{
			"aws":        {"aws.upbound.io", "providerconfigs"},
			"azure":      {"azure.upbound.io", "providerconfigs"},
			"gcp":        {"gcp.upbound.io", "providerconfigs"},
			"kubernetes": {"kubernetes.crossplane.io", "providerconfigs"},
		}

		for provider, gvr := range providerConfigGVRs {
			if providerFilter != "" && provider != providerFilter {
				continue
			}

			configs, err := dynamicClient.Resource(
				schema.GroupVersionResource{
					Group:    gvr.Group,
					Version:  "v1beta1",
					Resource: gvr.Resource,
				},
			).List(ctx, metav1.ListOptions{})

			if err == nil && len(configs.Items) > 0 {
				for _, config := range configs.Items {
					// Check if we already have an entry for this provider
					found := false
					for i, existing := range results {
						if existing.Provider == provider {
							results[i].ConfigName = config.GetName()
							found = true
							break
						}
					}
					if !found {
						results = append(results, CredentialInfo{
							Provider:   provider,
							ConfigName: config.GetName(),
							Configured: true,
						})
					}
				}
			}
		}
	}

	return results, nil
}
