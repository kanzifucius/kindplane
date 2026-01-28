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
	"sigs.k8s.io/kind/pkg/log"

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
	return CreateClusterWithProgress(ctx, cfg, nil)
}

// CreateClusterWithProgress creates a new Kind cluster with progress logging.
// If logger is provided, it will be used to capture cluster creation progress.
func CreateClusterWithProgress(ctx context.Context, cfg *config.Config, logger log.Logger) error {
	// Build provider options
	providerOpts := []cluster.ProviderOption{}
	if logger != nil {
		providerOpts = append(providerOpts, cluster.ProviderWithLogger(logger))
	}
	provider := cluster.NewProvider(providerOpts...)

	// Build Kind config
	kindConfig, err := BuildKindConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to build kind config: %w", err)
	}

	// Create cluster with config (node image is embedded in the config)
	opts := []cluster.CreateOption{
		cluster.CreateWithRawConfig([]byte(kindConfig)),
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

// GetNodeContainers returns the Docker container names for all nodes in a Kind cluster
func GetNodeContainers(clusterName string) ([]string, error) {
	// Kind uses the naming convention: <cluster-name>-control-plane, <cluster-name>-worker, etc.
	cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("label=io.x-k8s.kind.cluster=%s", clusterName), "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster containers: %w", err)
	}

	nodes := []string{}
	for _, name := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if name != "" {
			nodes = append(nodes, name)
		}
	}

	return nodes, nil
}

// UpdateCACertificates runs update-ca-certificates on all nodes in the cluster
// to regenerate the system CA bundle with any mounted CA certificates.
// This should be called after cluster creation when workload CAs are configured.
func UpdateCACertificates(ctx context.Context, clusterName string) error {
	nodes, err := GetNodeContainers(clusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster nodes: %w", err)
	}

	for _, node := range nodes {
		cmd := exec.CommandContext(ctx, "docker", "exec", node, "update-ca-certificates")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to update CA certificates on node %s: %w\nOutput: %s", node, err, string(output))
		}
	}

	return nil
}
