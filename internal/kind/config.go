package kind

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"gopkg.in/yaml.v3"

	"github.com/kanzi/kindplane/internal/config"
)

// KindConfig represents a Kind cluster configuration
type KindConfig struct {
	Kind       string     `yaml:"kind"`
	APIVersion string     `yaml:"apiVersion"`
	Name       string     `yaml:"name,omitempty"`
	Nodes      []KindNode `yaml:"nodes,omitempty"`
}

// KindNode represents a Kind node configuration
type KindNode struct {
	Role                 string            `yaml:"role"`
	Image                string            `yaml:"image,omitempty"`
	ExtraPortMappings    []KindPortMapping `yaml:"extraPortMappings,omitempty"`
	ExtraMounts          []KindMount       `yaml:"extraMounts,omitempty"`
	KubeadmConfigPatches []string          `yaml:"kubeadmConfigPatches,omitempty"`
	Labels               map[string]string `yaml:"labels,omitempty"`
}

// KindPortMapping represents a Kind port mapping
type KindPortMapping struct {
	ContainerPort int32  `yaml:"containerPort"`
	HostPort      int32  `yaml:"hostPort"`
	Protocol      string `yaml:"protocol,omitempty"`
}

// KindMount represents a Kind volume mount
type KindMount struct {
	HostPath      string `yaml:"hostPath"`
	ContainerPath string `yaml:"containerPath"`
	ReadOnly      bool   `yaml:"readOnly,omitempty"`
}

// BuildKindConfig creates a Kind configuration from kindplane config
func BuildKindConfig(cfg *config.Config) (string, error) {
	kindConfig := &KindConfig{
		Kind:       "Cluster",
		APIVersion: "kind.x-k8s.io/v1alpha4",
		Nodes:      []KindNode{},
	}

	// Load raw config if specified
	if cfg.Cluster.RawConfigPath != "" {
		rawConfig, err := loadRawKindConfig(cfg.Cluster.RawConfigPath)
		if err != nil {
			return "", fmt.Errorf("failed to load raw config: %w", err)
		}
		kindConfig = rawConfig
	}

	// Build node list (kindplane settings win)
	kindConfig.Nodes = buildNodes(cfg, kindConfig.Nodes)

	// Marshal to YAML
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(kindConfig); err != nil {
		return "", fmt.Errorf("failed to encode kind config: %w", err)
	}

	return buf.String(), nil
}

// loadRawKindConfig loads a Kind config from a file
func loadRawKindConfig(path string) (*KindConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config KindConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// buildNodes creates the node list based on config
func buildNodes(cfg *config.Config, existingNodes []KindNode) []KindNode {
	nodes := []KindNode{}

	// Port mappings for control plane (used for ingress)
	var controlPlanePortMappings []KindPortMapping
	for _, pm := range cfg.Cluster.PortMappings {
		protocol := pm.Protocol
		if protocol == "" {
			protocol = "TCP"
		}
		controlPlanePortMappings = append(controlPlanePortMappings, KindPortMapping{
			ContainerPort: pm.ContainerPort,
			HostPort:      pm.HostPort,
			Protocol:      protocol,
		})
	}

	// Extra mounts (applies to all nodes)
	var extraMounts []KindMount
	for _, em := range cfg.Cluster.ExtraMounts {
		extraMounts = append(extraMounts, KindMount{
			HostPath:      em.HostPath,
			ContainerPath: em.ContainerPath,
			ReadOnly:      em.ReadOnly,
		})
	}

	// Create control plane nodes
	for i := 0; i < cfg.Cluster.Nodes.ControlPlane; i++ {
		node := KindNode{
			Role:        "control-plane",
			ExtraMounts: extraMounts,
		}

		// Only first control plane gets port mappings
		if i == 0 {
			node.ExtraPortMappings = controlPlanePortMappings

			// Add ingress-ready configuration
			if cfg.Cluster.Ingress.Enabled {
				node.Labels = map[string]string{
					"ingress-ready": "true",
				}
				node.KubeadmConfigPatches = []string{
					ingressKubeadmPatch(),
				}
			}
		}

		nodes = append(nodes, node)
	}

	// Create worker nodes
	for i := 0; i < cfg.Cluster.Nodes.Workers; i++ {
		node := KindNode{
			Role:        "worker",
			ExtraMounts: extraMounts,
		}
		nodes = append(nodes, node)
	}

	return nodes
}

// ingressKubeadmPatch returns the kubeadm config patch for ingress readiness
func ingressKubeadmPatch() string {
	return `kind: InitConfiguration
nodeRegistration:
  kubeletExtraArgs:
    node-labels: "ingress-ready=true"`
}

// GenerateKindConfigFile creates a Kind config file from template
func GenerateKindConfigFile(cfg *config.Config, outputPath string) error {
	kindConfigYAML, err := BuildKindConfig(cfg)
	if err != nil {
		return err
	}

	if err := os.WriteFile(outputPath, []byte(kindConfigYAML), 0644); err != nil {
		return fmt.Errorf("failed to write kind config file: %w", err)
	}

	return nil
}

// Note: kindConfigTemplate is not currently used since we use YAML marshaling directly
// It's kept here for reference in case template-based generation is needed in the future
var _ = template.New("kindConfig") // Satisfy the import
