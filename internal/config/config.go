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
	Charts       []ChartConfig      `yaml:"charts,omitempty" doc:"Additional Helm charts to install\nPhases: pre-crossplane, post-crossplane, post-providers, final (default), post-eso (deprecated)"`
	Compositions CompositionsConfig `yaml:"compositions" comment:"Crossplane compositions and XRDs"`
}

// ClusterConfig contains Kind cluster configuration
type ClusterConfig struct {
	Name              string           `yaml:"name" comment:"Name of the Kind cluster"`
	KubernetesVersion string           `yaml:"kubernetesVersion" comment:"Kubernetes version to use (optional, uses Kind default if not specified)"`
	Nodes             NodesConfig      `yaml:"nodes" comment:"Node configuration"`
	PortMappings      []PortMapping    `yaml:"portMappings,omitempty" comment:"Port mappings from container to host" doc:"Format: containerPort:hostPort/protocol"`
	ExtraMounts       []ExtraMount     `yaml:"extraMounts,omitempty" comment:"Mount host paths into Kind nodes"`
	Ingress           IngressConfig    `yaml:"ingress" comment:"Ingress controller readiness configuration" doc:"When enabled, adds required labels and port mappings for ingress controllers"`
	Registry          RegistryConfig   `yaml:"registry,omitempty"`
	TrustedCAs        TrustedCAsConfig `yaml:"trustedCAs,omitempty" comment:"Trusted CA certificates for private registries and workloads"`
	RawConfigPath     string           `yaml:"rawConfigPath,omitempty" comment:"Optional: path to a raw Kind config file" doc:"Settings from kindplane.yaml will be merged on top (kindplane wins)"`
	NodeImage         string           `yaml:"nodeImage,omitempty" comment:"Full Kind node image path (optional)" doc:"Use when pulling images through a proxy registry like Artifactory\nIf not specified, defaults to \"kindest/node:v<kubernetesVersion>\"\nExamples:\n  - \"artifactory.example.com/kindest/node:v1.29.0\"\n  - \"artifactory.example.com/docker.io/kindest/node:v1.29.0\"\nNote: Ensure the proxy registry is configured in trustedCAs if using custom certificates"`
}

// RegistryConfig contains local container registry configuration
type RegistryConfig struct {
	Enabled    bool   `yaml:"enabled"`              // Enable local container registry
	Port       int    `yaml:"port,omitempty"`       // Host port for the registry (default: 5001)
	Persistent bool   `yaml:"persistent,omitempty"` // Keep registry container after kindplane down
	Name       string `yaml:"name,omitempty"`       // Registry container name (default: kind-registry)
}

// GetPort returns the registry port, defaulting to 5001
func (r *RegistryConfig) GetPort() int {
	if r.Port == 0 {
		return 5001
	}
	return r.Port
}

// GetName returns the registry container name, defaulting to kind-registry
func (r *RegistryConfig) GetName() string {
	if r.Name == "" {
		return "kind-registry"
	}
	return r.Name
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

// TrustedCAsConfig contains trusted CA certificate configuration
type TrustedCAsConfig struct {
	Registries []RegistryCA `yaml:"registries,omitempty" comment:"CA certificates for private container registries (trusted by containerd)"`
	Workloads  []WorkloadCA `yaml:"workloads,omitempty" comment:"CA certificates mounted for workloads to trust"`
}

// RegistryCA defines a CA certificate for a private container registry
type RegistryCA struct {
	Host   string `yaml:"host" comment:"Registry host (e.g., \"registry.example.com:5000\" or \"*.internal.company.com\")"`
	CAFile string `yaml:"caFile" comment:"Path to CA certificate file on the host"`
}

// WorkloadCA defines a CA certificate to mount for workloads
type WorkloadCA struct {
	Name   string `yaml:"name" comment:"Identifier for the CA (used in the mount path)"`
	CAFile string `yaml:"caFile" comment:"Path to CA certificate file on the host"`
}

// RegistryCaBundleConfig configures CA bundle for Crossplane registry access
// Multiple certificates can be specified and will be bundled together
type RegistryCaBundleConfig struct {
	CAFiles        []string `yaml:"caFiles,omitempty" comment:"Direct paths to CA files (multiple allowed)"`
	WorkloadCARefs []string `yaml:"workloadCARefs,omitempty" comment:"Reference existing workload CAs by name (must be defined in cluster.trustedCAs.workloads)"`
}

// ResolveCAFiles resolves all CA file paths from both direct paths and workload CA references
// Returns a list of CA file paths and an error if any reference is not found
func (r *RegistryCaBundleConfig) ResolveCAFiles(workloadCAs []WorkloadCA) ([]string, error) {
	var caFiles []string

	// Add direct CA files
	caFiles = append(caFiles, r.CAFiles...)

	// Resolve workload CA references
	for _, ref := range r.WorkloadCARefs {
		found := false
		for _, wl := range workloadCAs {
			if wl.Name == ref {
				caFiles = append(caFiles, wl.CAFile)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("workload CA '%s' not found", ref)
		}
	}

	if len(caFiles) == 0 {
		return nil, fmt.Errorf("no CA files specified (use caFiles or workloadCARefs)")
	}

	return caFiles, nil
}

// CrossplaneConfig contains Crossplane installation settings
type CrossplaneConfig struct {
	Version          string                  `yaml:"version" comment:"Crossplane version to install"`
	Repo             string                  `yaml:"repo,omitempty" comment:"Custom Helm repository URL (optional)" doc:"Use when pulling charts from a private registry or air-gapped environment\nIf not specified, defaults to https://charts.crossplane.io/stable"`
	Values           map[string]interface{}  `yaml:"values,omitempty" comment:"Inline Helm values for Crossplane installation (optional)" doc:"These values will be merged with any values files specified below"`
	ValuesFiles      []string                `yaml:"valuesFiles,omitempty" comment:"External values files (optional)" doc:"Values from files are loaded first, then inline values override them"`
	Providers        []ProviderConfig        `yaml:"providers,omitempty" comment:"Crossplane providers to install" doc:"Use full OCI package path with version tag"`
	RegistryCaBundle *RegistryCaBundleConfig `yaml:"registryCaBundle,omitempty" comment:"Registry CA bundle for Crossplane (optional)" doc:"Required when pulling Configuration and Provider packages from private registries with custom certificates\nMultiple certificates can be specified and will be bundled together into one ConfigMap"`
	ImageCache       *ImageCacheConfig       `yaml:"imageCache,omitempty" comment:"Image caching configuration"`
}

// DefaultCrossplaneRepo is the default Helm repository URL for Crossplane
const DefaultCrossplaneRepo = "https://charts.crossplane.io/stable"

// GetRepo returns the Helm repository URL, defaulting to the official Crossplane repo
func (c *CrossplaneConfig) GetRepo() string {
	if c.Repo == "" {
		return DefaultCrossplaneRepo
	}
	return c.Repo
}

// ImageCacheConfig configures local image caching behavior
type ImageCacheConfig struct {
	// Enabled controls whether to pre-load images from local Docker
	// Default: true (enabled by default)
	Enabled *bool `yaml:"enabled,omitempty"`

	// PreloadProviders attempts to load provider images into Kind before installation
	// Default: true
	PreloadProviders *bool `yaml:"preloadProviders,omitempty"`

	// PreloadCrossplane attempts to load Crossplane core images
	// Default: true
	PreloadCrossplane *bool `yaml:"preloadCrossplane,omitempty"`

	// AdditionalImages is a list of extra images to pre-load (e.g., functions, custom images)
	AdditionalImages []string `yaml:"additionalImages,omitempty"`

	// ImageOverrides maps provider packages to their actual container images
	// Useful for providers that don't follow standard naming conventions
	// Key: provider package (e.g., "xpkg.upbound.io/upbound/provider-aws:v1.1.0")
	// Value: list of actual container images
	ImageOverrides map[string][]string `yaml:"imageOverrides,omitempty"`
}

// IsEnabled returns whether image caching is enabled (default: true)
func (i *ImageCacheConfig) IsEnabled() bool {
	if i == nil || i.Enabled == nil {
		return true // Enabled by default
	}
	return *i.Enabled
}

// ShouldPreloadProviders returns whether to preload provider images (default: true)
func (i *ImageCacheConfig) ShouldPreloadProviders() bool {
	if i == nil || i.PreloadProviders == nil {
		return true
	}
	return *i.PreloadProviders
}

// ShouldPreloadCrossplane returns whether to preload Crossplane images (default: true)
func (i *ImageCacheConfig) ShouldPreloadCrossplane() bool {
	if i == nil || i.PreloadCrossplane == nil {
		return true
	}
	return *i.PreloadCrossplane
}

// ProviderConfig defines a Crossplane provider
type ProviderConfig struct {
	Name    string `yaml:"name"`
	Package string `yaml:"package" comment:"Full OCI package path with version tag (e.g., xpkg.upbound.io/upbound/provider-aws:v1.1.0)"`
}

// CredentialsConfig contains cloud provider credentials configuration
type CredentialsConfig struct {
	AWS        AWSCredentials        `yaml:"aws,omitempty" comment:"AWS credentials configuration (uncomment if using provider-aws)"`
	Azure      AzureCredentials      `yaml:"azure,omitempty" comment:"Azure credentials configuration (uncomment if using provider-azure)"`
	Kubernetes KubernetesCredentials `yaml:"kubernetes,omitempty" comment:"Kubernetes provider credentials"`
}

// AWSCredentials defines AWS credential source
type AWSCredentials struct {
	Source  string `yaml:"source" comment:"Source: env (environment variables), file (credentials file), profile (AWS CLI profile)"`
	Profile string `yaml:"profile,omitempty"`
}

// AzureCredentials defines Azure credential source
type AzureCredentials struct {
	Source string `yaml:"source" comment:"Source: env (environment variables), file (credentials file)"`
}

// KubernetesCredentials defines Kubernetes credential source
type KubernetesCredentials struct {
	Source string `yaml:"source" comment:"Source: incluster (use service account), kubeconfig (use kubeconfig file)"`
}

// CompositionsConfig contains composition sources configuration
type CompositionsConfig struct {
	Sources []CompositionSource `yaml:"sources,omitempty" comment:"Sources to load compositions from"`
}

// CompositionSource defines where to load compositions from
type CompositionSource struct {
	Type   string `yaml:"type" comment:"local, git"`
	Path   string `yaml:"path" comment:"local path or path within git repo"`
	Repo   string `yaml:"repo,omitempty" comment:"git repository URL"`
	Branch string `yaml:"branch,omitempty" comment:"git branch"`
}

// ChartConfig defines a Helm chart to install
type ChartConfig struct {
	Name            string                 `yaml:"name" comment:"Helm release name"`
	Repo            string                 `yaml:"repo" comment:"Helm repository URL"`
	Chart           string                 `yaml:"chart" comment:"Chart name in the repository"`
	Version         string                 `yaml:"version,omitempty" comment:"Chart version (optional, latest if omitted)"`
	Namespace       string                 `yaml:"namespace" comment:"Target namespace"`
	CreateNamespace *bool                  `yaml:"createNamespace,omitempty" comment:"Create namespace if not exists (default: true)"`
	Phase           string                 `yaml:"phase,omitempty" comment:"Installation phase: pre-crossplane, post-crossplane, post-providers, post-eso (default)"`
	Wait            *bool                  `yaml:"wait,omitempty" comment:"Wait for resources to be ready (default: true)"`
	Timeout         string                 `yaml:"timeout,omitempty" comment:"Installation timeout (default: 5m)"`
	Values          map[string]interface{} `yaml:"values,omitempty" comment:"Inline values"`
	ValuesFiles     []string               `yaml:"valuesFiles,omitempty" comment:"Paths to values files"`
}

// ChartPhase constants
const (
	ChartPhasePrecrossplane  = "pre-crossplane"
	ChartPhasePostCrossplane = "post-crossplane"
	ChartPhasePostProviders  = "post-providers"
	ChartPhaseFinal          = "final"    // Preferred name for final installation phase
	ChartPhasePostESO        = "post-eso" // Deprecated: kept for backwards compatibility, use "final" instead
)

// GetPhase returns the chart phase, defaulting to final
func (c *ChartConfig) GetPhase() string {
	if c.Phase == "" {
		return ChartPhaseFinal
	}
	// Map deprecated post-eso to final for backwards compatibility
	if c.Phase == ChartPhasePostESO {
		return ChartPhaseFinal
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

	// Create parent directories if they don't exist
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
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
