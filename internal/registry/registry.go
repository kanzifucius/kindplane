package registry

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kanzi/kindplane/internal/config"
	"github.com/kanzi/kindplane/internal/kind"
)

const (
	// DefaultRegistryImage is the Docker registry image to use
	DefaultRegistryImage = "registry:2"
	// DefaultRegistryPort is the default host port for the registry
	DefaultRegistryPort = 5001
	// DefaultRegistryName is the default container name for the registry
	DefaultRegistryName = "kind-registry"
	// RegistryInternalPort is the port the registry listens on inside the container
	RegistryInternalPort = 5000
)

// Manager handles local container registry operations
type Manager struct {
	cfg *config.RegistryConfig
}

// NewManager creates a new registry manager
func NewManager(cfg *config.RegistryConfig) *Manager {
	return &Manager{cfg: cfg}
}

// IsRunning checks if the registry container is running
func (m *Manager) IsRunning(ctx context.Context) (bool, error) {
	name := m.cfg.GetName()
	cmd := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Running}}", name)
	output, err := cmd.Output()
	if err != nil {
		// Container doesn't exist
		return false, nil
	}
	return strings.TrimSpace(string(output)) == "true", nil
}

// Exists checks if the registry container exists (running or stopped)
func (m *Manager) Exists(ctx context.Context) (bool, error) {
	name := m.cfg.GetName()
	cmd := exec.CommandContext(ctx, "docker", "inspect", name)
	err := cmd.Run()
	return err == nil, nil
}

// Create creates and starts the registry container
func (m *Manager) Create(ctx context.Context) error {
	name := m.cfg.GetName()
	port := m.cfg.GetPort()

	// Check if already running
	running, err := m.IsRunning(ctx)
	if err != nil {
		return fmt.Errorf("failed to check registry status: %w", err)
	}
	if running {
		return nil // Already running
	}

	// Check if container exists but is stopped
	exists, err := m.Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check registry existence: %w", err)
	}
	if exists {
		// Start existing container
		cmd := exec.CommandContext(ctx, "docker", "start", name)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to start existing registry container: %w", err)
		}
		return nil
	}

	// Create new registry container
	cmd := exec.CommandContext(ctx, "docker", "run",
		"-d",
		"--restart=always",
		"-p", fmt.Sprintf("127.0.0.1:%d:%d", port, RegistryInternalPort),
		"--network", "bridge",
		"--name", name,
		DefaultRegistryImage,
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create registry container: %w", err)
	}

	return nil
}

// ConnectToNetwork connects the registry container to a Docker network
func (m *Manager) ConnectToNetwork(ctx context.Context, network string) error {
	name := m.cfg.GetName()

	// Check if already connected
	cmd := exec.CommandContext(ctx, "docker", "inspect",
		"-f", fmt.Sprintf("{{json .NetworkSettings.Networks.%s}}", network),
		name,
	)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to inspect registry container: %w", err)
	}

	// If not null, already connected
	if strings.TrimSpace(string(output)) != "null" {
		return nil
	}

	// Connect to network
	cmd = exec.CommandContext(ctx, "docker", "network", "connect", network, name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to connect registry to network %s: %w", network, err)
	}

	return nil
}

// ConfigureNodes configures Kind nodes to use the local registry
func (m *Manager) ConfigureNodes(ctx context.Context, clusterName string) error {
	name := m.cfg.GetName()
	port := m.cfg.GetPort()

	// Get Kubernetes client to list nodes
	kubeClient, err := kind.GetKubeClient(clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubernetes client: %w", err)
	}

	// List nodes using Kubernetes API with retry/backoff
	// The API server may still be bootstrapping, so we retry until nodes are available
	var nodesList *corev1.NodeList
	var lastErr error

	backoff := wait.Backoff{
		Duration: 500 * time.Millisecond,
		Factor:   1.5,
		Jitter:   0.1,
		Steps:    10,
		Cap:      10 * time.Second,
	}

	err = wait.ExponentialBackoffWithContext(ctx, backoff, func(ctx context.Context) (bool, error) {
		nodesList, lastErr = kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if lastErr != nil {
			// Retry on API errors (server may be bootstrapping)
			return false, nil
		}
		if len(nodesList.Items) == 0 {
			// Retry if no nodes found yet
			return false, nil
		}
		// Success: nodes found
		return true, nil
	})

	if err != nil {
		if lastErr != nil {
			return fmt.Errorf("node discovery timed out for cluster %s: %w", clusterName, lastErr)
		}
		return fmt.Errorf("node discovery timed out for cluster %s: no nodes found after retries", clusterName)
	}

	registryDir := fmt.Sprintf("/etc/containerd/certs.d/localhost:%d", port)

	// Configure each node
	for _, node := range nodesList.Items {
		nodeName := node.Name
		if nodeName == "" {
			continue
		}

		// Create registry config directory
		cmd := exec.CommandContext(ctx, "docker", "exec", nodeName, "mkdir", "-p", registryDir)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create registry config dir on node %s: %w", nodeName, err)
		}

		// Create hosts.toml configuration
		hostsToml := fmt.Sprintf(`[host."http://%s:%d"]
`, name, RegistryInternalPort)

		// Write hosts.toml to node
		cmd = exec.CommandContext(ctx, "docker", "exec", "-i", nodeName,
			"sh", "-c", fmt.Sprintf("cat > %s/hosts.toml", registryDir))
		cmd.Stdin = strings.NewReader(hostsToml)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to configure registry on node %s: %w", nodeName, err)
		}
	}

	return nil
}

// Remove removes the registry container
func (m *Manager) Remove(ctx context.Context) error {
	name := m.cfg.GetName()

	// Check if container exists
	exists, err := m.Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check registry existence: %w", err)
	}
	if !exists {
		return nil // Nothing to remove
	}

	// Stop container
	cmd := exec.CommandContext(ctx, "docker", "stop", name)
	_ = cmd.Run() // Ignore error if already stopped

	// Remove container
	cmd = exec.CommandContext(ctx, "docker", "rm", name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove registry container: %w", err)
	}

	return nil
}

// GetRegistryHost returns the registry host for use in pod specs
// This is the address that pods should use to pull images
func (m *Manager) GetRegistryHost() string {
	return fmt.Sprintf("localhost:%d", m.cfg.GetPort())
}

// GetInternalHost returns the internal registry host for use within the Kind network
// This is the address that containerd uses to pull images
func (m *Manager) GetInternalHost() string {
	return fmt.Sprintf("%s:%d", m.cfg.GetName(), RegistryInternalPort)
}
