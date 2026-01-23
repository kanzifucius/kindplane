package credentials

import (
	"context"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
