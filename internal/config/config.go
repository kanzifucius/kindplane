package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultConfigFile is the default configuration file name
	DefaultConfigFile = "kindplane.yaml"
)

// Config represents the main configuration structure
type Config struct {
	Cluster      ClusterConfig      `yaml:"cluster"`
	Crossplane   CrossplaneConfig   `yaml:"crossplane"`
	Credentials  CredentialsConfig  `yaml:"credentials"`
	ESO          ESOConfig          `yaml:"eso"`
	Charts       []ChartConfig      `yaml:"charts,omitempty"`
	Compositions CompositionsConfig `yaml:"compositions"`
}

// ClusterConfig contains Kind cluster configuration
type ClusterConfig struct {
	Name              string        `yaml:"name"`
	KubernetesVersion string        `yaml:"kubernetesVersion"`
	Nodes             NodesConfig   `yaml:"nodes"`
	PortMappings      []PortMapping `yaml:"portMappings,omitempty"`
	ExtraMounts       []ExtraMount  `yaml:"extraMounts,omitempty"`
	Ingress           IngressConfig `yaml:"ingress"`
	RawConfigPath     string        `yaml:"rawConfigPath,omitempty"`
}

// NodesConfig defines the number of nodes in the cluster
type NodesConfig struct {
	ControlPlane int `yaml:"controlPlane"`
	Workers      int `yaml:"workers"`
}

// PortMapping defines a port mapping from container to host
type PortMapping struct {
	ContainerPort int32  `yaml:"containerPort"`
	HostPort      int32  `yaml:"hostPort"`
	Protocol      string `yaml:"protocol,omitempty"`
}

// ExtraMount defines a volume mount from host to container
type ExtraMount struct {
	HostPath      string `yaml:"hostPath"`
	ContainerPath string `yaml:"containerPath"`
	ReadOnly      bool   `yaml:"readOnly,omitempty"`
}

// IngressConfig defines ingress controller readiness settings
type IngressConfig struct {
	Enabled bool `yaml:"enabled"`
}

// CrossplaneConfig contains Crossplane installation settings
type CrossplaneConfig struct {
	Version   string           `yaml:"version"`
	Providers []ProviderConfig `yaml:"providers,omitempty"`
}

// ProviderConfig defines a Crossplane provider
type ProviderConfig struct {
	Name    string `yaml:"name"`
	Package string `yaml:"package"` // Full OCI package path with version tag (e.g., xpkg.upbound.io/upbound/provider-aws:v1.1.0)
}

// CredentialsConfig contains cloud provider credentials configuration
type CredentialsConfig struct {
	AWS        AWSCredentials        `yaml:"aws,omitempty"`
	Azure      AzureCredentials      `yaml:"azure,omitempty"`
	Kubernetes KubernetesCredentials `yaml:"kubernetes,omitempty"`
}

// AWSCredentials defines AWS credential source
type AWSCredentials struct {
	Source  string `yaml:"source"` // env, file, profile
	Profile string `yaml:"profile,omitempty"`
}

// AzureCredentials defines Azure credential source
type AzureCredentials struct {
	Source string `yaml:"source"` // env, file
}

// KubernetesCredentials defines Kubernetes credential source
type KubernetesCredentials struct {
	Source string `yaml:"source"` // incluster, kubeconfig
}

// ESOConfig contains External Secrets Operator configuration
type ESOConfig struct {
	Enabled bool   `yaml:"enabled"`
	Version string `yaml:"version"`
}

// CompositionsConfig contains composition sources configuration
type CompositionsConfig struct {
	Sources []CompositionSource `yaml:"sources,omitempty"`
}

// CompositionSource defines where to load compositions from
type CompositionSource struct {
	Type   string `yaml:"type"`             // local, git
	Path   string `yaml:"path"`             // local path or path within git repo
	Repo   string `yaml:"repo,omitempty"`   // git repository URL
	Branch string `yaml:"branch,omitempty"` // git branch
}

// ChartConfig defines a Helm chart to install
type ChartConfig struct {
	Name            string                 `yaml:"name"`                      // Helm release name
	Repo            string                 `yaml:"repo"`                      // Helm repository URL
	Chart           string                 `yaml:"chart"`                     // Chart name in the repository
	Version         string                 `yaml:"version,omitempty"`         // Chart version (optional, latest if omitted)
	Namespace       string                 `yaml:"namespace"`                 // Target namespace
	CreateNamespace *bool                  `yaml:"createNamespace,omitempty"` // Create namespace if not exists (default: true)
	Phase           string                 `yaml:"phase,omitempty"`           // Installation phase: pre-crossplane, post-crossplane, post-providers, post-eso (default)
	Wait            *bool                  `yaml:"wait,omitempty"`            // Wait for resources to be ready (default: true)
	Timeout         string                 `yaml:"timeout,omitempty"`         // Installation timeout (default: 5m)
	Values          map[string]interface{} `yaml:"values,omitempty"`          // Inline values
	ValuesFiles     []string               `yaml:"valuesFiles,omitempty"`     // Paths to values files
}

// ChartPhase constants
const (
	ChartPhasePrecrossplane  = "pre-crossplane"
	ChartPhasePostCrossplane = "post-crossplane"
	ChartPhasePostProviders  = "post-providers"
	ChartPhasePostESO        = "post-eso"
)

// GetPhase returns the chart phase, defaulting to post-eso
func (c *ChartConfig) GetPhase() string {
	if c.Phase == "" {
		return ChartPhasePostESO
	}
	return c.Phase
}

// ShouldCreateNamespace returns whether to create the namespace
func (c *ChartConfig) ShouldCreateNamespace() bool {
	if c.CreateNamespace == nil {
		return true
	}
	return *c.CreateNamespace
}

// ShouldWait returns whether to wait for the chart to be ready
func (c *ChartConfig) ShouldWait() bool {
	if c.Wait == nil {
		return true
	}
	return *c.Wait
}

// GetTimeout returns the timeout duration string, defaulting to "5m"
func (c *ChartConfig) GetTimeout() string {
	if c.Timeout == "" {
		return "5m"
	}
	return c.Timeout
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigFile
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("configuration file not found: %s\n\nRun 'kindplane init' to create a configuration file", path)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// Save writes the configuration to a file
func (c *Config) Save(path string) error {
	if path == "" {
		path = DefaultConfigFile
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Exists checks if a configuration file exists at the given path
func Exists(path string) bool {
	if path == "" {
		path = DefaultConfigFile
	}
	_, err := os.Stat(path)
	return err == nil
}

// GetKubeContext returns the kubectl context name for the Kind cluster
func (c *Config) GetKubeContext() string {
	return fmt.Sprintf("kind-%s", c.Cluster.Name)
}

// GetKubeconfigPath returns the path to the kubeconfig file
func (c *Config) GetKubeconfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".kube", "config")
}
