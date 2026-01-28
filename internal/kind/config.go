package kind

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"

	"github.com/kanzi/kindplane/internal/config"
)

// KindConfig represents a Kind cluster configuration
type KindConfig struct {
	Kind                    string     `yaml:"kind"`
	APIVersion              string     `yaml:"apiVersion"`
	Name                    string     `yaml:"name,omitempty"`
	Nodes                   []KindNode `yaml:"nodes,omitempty"`
	ContainerdConfigPatches []string   `yaml:"containerdConfigPatches,omitempty"`
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

	// Add containerd config patches for local registry
	if cfg.Cluster.Registry.Enabled {
		kindConfig.ContainerdConfigPatches = append(kindConfig.ContainerdConfigPatches,
			registryContainerdPatch(),
		)
	}

	// Build node list (kindplane settings win)
	kindConfig.Nodes = buildNodes(cfg, kindConfig.Nodes)

	// Add containerd config patches for trusted registry CAs
	kindConfig.ContainerdConfigPatches = append(kindConfig.ContainerdConfigPatches, buildContainerdPatches(cfg)...)

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

// GetNodeImage determines the node image that will be used for cluster creation
// Returns the image path and a description of how it was determined
func GetNodeImage(cfg *config.Config) (image string, source string) {
	// Trim whitespace to handle any YAML parsing issues
	nodeImage := strings.TrimSpace(cfg.Cluster.NodeImage)
	if nodeImage != "" {
		return nodeImage, "explicitly configured"
	}
	if cfg.Cluster.KubernetesVersion != "" {
		image = fmt.Sprintf("kindest/node:v%s", strings.TrimPrefix(cfg.Cluster.KubernetesVersion, "v"))
		return image, fmt.Sprintf("derived from kubernetesVersion (%s)", cfg.Cluster.KubernetesVersion)
	}
	return "", "Kind default (not specified)"
}

// buildNodes creates the node list based on config
func buildNodes(cfg *config.Config, existingNodes []KindNode) []KindNode {
	nodes := []KindNode{}

	// Determine node image - prioritize explicit nodeImage from config
	// This must be set to ensure the user's nodeImage setting is always respected
	var nodeImage string
	if cfg.Cluster.NodeImage != "" {
		// Use explicit nodeImage from config (highest priority)
		// Trim whitespace to handle any YAML parsing issues
		nodeImage = strings.TrimSpace(cfg.Cluster.NodeImage)
	} else if cfg.Cluster.KubernetesVersion != "" {
		// Fall back to constructing from kubernetesVersion if nodeImage not specified
		nodeImage = fmt.Sprintf("kindest/node:v%s", strings.TrimPrefix(cfg.Cluster.KubernetesVersion, "v"))
	}
	// If nodeImage is still empty here, it means neither NodeImage nor KubernetesVersion
	// were set, and Kind will use its default image (Image field will be omitted due to omitempty)

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

	// Add mounts for trusted CA certificates
	extraMounts = append(extraMounts, buildCAMounts(cfg)...)

	// Create control plane nodes
	for i := 0; i < cfg.Cluster.Nodes.ControlPlane; i++ {
		node := KindNode{
			Role:        "control-plane",
			Image:       nodeImage,
			ExtraMounts: extraMounts,
		}

		// Only first control plane gets port mappings
		if i == 0 {
			node.ExtraPortMappings = controlPlanePortMappings

			// Add ingress-ready and kindplane marker configuration
			if cfg.Cluster.Ingress.Enabled {
				node.Labels = map[string]string{
					"ingress-ready": "true",
				}
				node.KubeadmConfigPatches = []string{
					ingressKubeadmPatch(),
				}
			} else {
				// Add kindplane marker label via kubeadm patch
				node.KubeadmConfigPatches = []string{
					kindplaneKubeadmPatch(),
				}
			}
		} else {
			// Add kindplane marker label via kubeadm patch for additional control planes
			node.KubeadmConfigPatches = []string{
				kindplaneKubeadmPatch(),
			}
		}

		nodes = append(nodes, node)
	}

	// Create worker nodes
	for i := 0; i < cfg.Cluster.Nodes.Workers; i++ {
		node := KindNode{
			Role:        "worker",
			Image:       nodeImage,
			ExtraMounts: extraMounts,
			// Add kindplane marker label via kubeadm patch
			KubeadmConfigPatches: []string{
				kindplaneKubeadmPatch(),
			},
		}
		nodes = append(nodes, node)
	}

	return nodes
}

// kindplaneKubeadmPatch returns the kubeadm config patch for kindplane marker label
func kindplaneKubeadmPatch() string {
	return `kind: InitConfiguration
nodeRegistration:
  kubeletExtraArgs:
    node-labels: "kindplane.io/managed-by=kindplane"`
}

// ingressKubeadmPatch returns the kubeadm config patch for ingress readiness
func ingressKubeadmPatch() string {
	return `kind: InitConfiguration
nodeRegistration:
  kubeletExtraArgs:
    node-labels: "ingress-ready=true,kindplane.io/managed-by=kindplane"`
}

// registryContainerdPatch returns the containerd config patch for local registry support
func registryContainerdPatch() string {
	return `[plugins."io.containerd.grpc.v1.cri".registry]
  config_path = "/etc/containerd/certs.d"`
}

// buildCAMounts creates extra mounts for trusted CA certificates
func buildCAMounts(cfg *config.Config) []KindMount {
	var mounts []KindMount

	// Mount registry CA certificates
	for _, reg := range cfg.Cluster.TrustedCAs.Registries {
		// Get absolute path for the CA file
		absPath, err := filepath.Abs(reg.CAFile)
		if err != nil {
			absPath = reg.CAFile
		}

		// Mount to containerd certs directory
		containerPath := fmt.Sprintf("/etc/containerd/certs.d/%s/ca.crt", reg.Host)
		mounts = append(mounts, KindMount{
			HostPath:      absPath,
			ContainerPath: containerPath,
			ReadOnly:      true,
		})
	}

	// Mount workload CA certificates
	for _, wl := range cfg.Cluster.TrustedCAs.Workloads {
		// Get absolute path for the CA file
		absPath, err := filepath.Abs(wl.CAFile)
		if err != nil {
			absPath = wl.CAFile
		}

		// Mount to system CA certificates directory for automatic trust
		// After cluster creation, update-ca-certificates must be run on each node
		containerPath := fmt.Sprintf("/usr/local/share/ca-certificates/%s.crt", wl.Name)
		mounts = append(mounts, KindMount{
			HostPath:      absPath,
			ContainerPath: containerPath,
			ReadOnly:      true,
		})
	}

	return mounts
}

// buildContainerdPatches creates containerd config patches for trusted registry CAs
func buildContainerdPatches(cfg *config.Config) []string {
	if len(cfg.Cluster.TrustedCAs.Registries) == 0 {
		return nil
	}

	var patches []string

	// Build containerd config patch for each registry
	for _, reg := range cfg.Cluster.TrustedCAs.Registries {
		patch := fmt.Sprintf(`[plugins."io.containerd.grpc.v1.cri".registry.configs."%s".tls]
  ca_file = "/etc/containerd/certs.d/%s/ca.crt"`, reg.Host, reg.Host)
		patches = append(patches, patch)
	}

	// Combine all patches into a single patch string
	if len(patches) > 0 {
		var combined string
		for i, patch := range patches {
			if i > 0 {
				combined += "\n"
			}
			combined += patch
		}
		return []string{combined}
	}

	return nil
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

// TrustedCAsSummary contains the result of CA validation
type TrustedCAsSummary struct {
	RegistryCount int
	WorkloadCount int
}

// HasTrustedCAs returns true if any trusted CAs are configured
func HasTrustedCAs(cfg *config.Config) bool {
	return len(cfg.Cluster.TrustedCAs.Registries) > 0 ||
		len(cfg.Cluster.TrustedCAs.Workloads) > 0
}

// ValidateTrustedCAs validates that all configured CA files exist and are readable
// Returns a summary of the configured CAs or an error if validation fails
func ValidateTrustedCAs(cfg *config.Config) (*TrustedCAsSummary, error) {
	summary := &TrustedCAsSummary{}

	// Validate registry CAs
	for _, reg := range cfg.Cluster.TrustedCAs.Registries {
		absPath, err := filepath.Abs(reg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("invalid path for registry CA '%s': %w", reg.Host, err)
		}
		if _, err := os.Stat(absPath); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("registry CA file not found for '%s': %s", reg.Host, absPath)
			}
			return nil, fmt.Errorf("cannot access registry CA file for '%s': %w", reg.Host, err)
		}
		summary.RegistryCount++
	}

	// Validate workload CAs
	for _, wl := range cfg.Cluster.TrustedCAs.Workloads {
		absPath, err := filepath.Abs(wl.CAFile)
		if err != nil {
			return nil, fmt.Errorf("invalid path for workload CA '%s': %w", wl.Name, err)
		}
		if _, err := os.Stat(absPath); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("workload CA file not found for '%s': %s", wl.Name, absPath)
			}
			return nil, fmt.Errorf("cannot access workload CA file for '%s': %w", wl.Name, err)
		}
		summary.WorkloadCount++
	}

	return summary, nil
}
