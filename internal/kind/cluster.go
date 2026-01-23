package kind

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/kind/pkg/cluster"

	"github.com/kanzi/kindplane/internal/config"
)

// ClusterExists checks if a Kind cluster with the given name exists
func ClusterExists(name string) (bool, error) {
	provider := cluster.NewProvider()
	clusters, err := provider.List()
	if err != nil {
		return false, fmt.Errorf("failed to list clusters: %w", err)
	}

	for _, c := range clusters {
		if c == name {
			return true, nil
		}
	}
	return false, nil
}

// CreateCluster creates a new Kind cluster based on the configuration
func CreateCluster(ctx context.Context, cfg *config.Config) error {
	provider := cluster.NewProvider()

	// Build Kind config
	kindConfig, err := BuildKindConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to build kind config: %w", err)
	}

	// Create cluster with config
	opts := []cluster.CreateOption{
		cluster.CreateWithRawConfig([]byte(kindConfig)),
	}

	// Add node image if kubernetes version is specified
	if cfg.Cluster.KubernetesVersion != "" {
		nodeImage := fmt.Sprintf("kindest/node:v%s", strings.TrimPrefix(cfg.Cluster.KubernetesVersion, "v"))
		opts = append(opts, cluster.CreateWithNodeImage(nodeImage))
	}

	if err := provider.Create(cfg.Cluster.Name, opts...); err != nil {
		return fmt.Errorf("failed to create cluster: %w", err)
	}

	return nil
}

// DeleteCluster deletes a Kind cluster
func DeleteCluster(ctx context.Context, name string) error {
	provider := cluster.NewProvider()
	if err := provider.Delete(name, ""); err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}
	return nil
}

// GetKubeClient returns a Kubernetes client for the specified Kind cluster
func GetKubeClient(clusterName string) (*kubernetes.Clientset, error) {
	// Get kubeconfig path
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		kubeconfigPath = fmt.Sprintf("%s/.kube/config", home)
	}

	// Build config with context
	contextName := fmt.Sprintf("kind-%s", clusterName)
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{CurrentContext: contextName},
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return clientset, nil
}

// GetKubeConfigPath returns the path to the kubeconfig file for the cluster
func GetKubeConfigPath(clusterName string) string {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		home, _ := os.UserHomeDir()
		kubeconfigPath = fmt.Sprintf("%s/.kube/config", home)
	}
	return kubeconfigPath
}

// GetContextName returns the kubectl context name for a Kind cluster
func GetContextName(clusterName string) string {
	return fmt.Sprintf("kind-%s", clusterName)
}

// ExportKubeConfig exports the kubeconfig for a Kind cluster
func ExportKubeConfig(clusterName string) error {
	cmd := exec.Command("kind", "export", "kubeconfig", "--name", clusterName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// GetRESTConfig returns a REST config for the specified Kind cluster
func GetRESTConfig(clusterName string) (*rest.Config, error) {
	kubeconfigPath := GetKubeConfigPath(clusterName)
	contextName := GetContextName(clusterName)

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{CurrentContext: contextName},
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	return config, nil
}
