package kind

import (
	"strings"
	"testing"

	"github.com/kanzi/kindplane/internal/config"
)

func TestGetNodeImage(t *testing.T) {
	tests := []struct {
		name           string
		cfg            *config.Config
		expectedImage  string
		expectedSource string
	}{
		{
			name: "explicit nodeImage takes priority",
			cfg: &config.Config{
				Cluster: config.ClusterConfig{
					NodeImage:         "nn-docker-remote.artifactory.insim.biz/kindest/node:v1.34.0",
					KubernetesVersion: "1.34.0",
				},
			},
			expectedImage:  "nn-docker-remote.artifactory.insim.biz/kindest/node:v1.34.0",
			expectedSource: "explicitly configured",
		},
		{
			name: "nodeImage without kubernetesVersion",
			cfg: &config.Config{
				Cluster: config.ClusterConfig{
					NodeImage: "artifactory.example.com/kindest/node:v1.29.0",
				},
			},
			expectedImage:  "artifactory.example.com/kindest/node:v1.29.0",
			expectedSource: "explicitly configured",
		},
		{
			name: "derived from kubernetesVersion when nodeImage not set",
			cfg: &config.Config{
				Cluster: config.ClusterConfig{
					KubernetesVersion: "1.34.0",
				},
			},
			expectedImage:  "kindest/node:v1.34.0",
			expectedSource: "derived from kubernetesVersion (1.34.0)",
		},
		{
			name: "Kind default when neither set",
			cfg: &config.Config{
				Cluster: config.ClusterConfig{},
			},
			expectedImage:  "",
			expectedSource: "Kind default (not specified)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			image, source := GetNodeImage(tt.cfg)
			if image != tt.expectedImage {
				t.Errorf("expected image %q, got %q", tt.expectedImage, image)
			}
			if source != tt.expectedSource {
				t.Errorf("expected source %q, got %q", tt.expectedSource, source)
			}
		})
	}
}

func TestBuildKindConfig_NodeImage(t *testing.T) {
	tests := []struct {
		name           string
		cfg            *config.Config
		expectedInYAML string
	}{
		{
			name: "explicit nodeImage should appear in YAML",
			cfg: &config.Config{
				Cluster: config.ClusterConfig{
					Name:              "test-cluster",
					NodeImage:         "nn-docker-remote.artifactory.insim.biz/kindest/node:v1.34.0",
					KubernetesVersion: "1.34.0",
					Nodes: config.NodesConfig{
						ControlPlane: 1,
						Workers:      0,
					},
					Ingress: config.IngressConfig{
						Enabled: false,
					},
				},
			},
			expectedInYAML: "nn-docker-remote.artifactory.insim.biz/kindest/node:v1.34.0",
		},
		{
			name: "nodeImage should be used even with kubernetesVersion set",
			cfg: &config.Config{
				Cluster: config.ClusterConfig{
					Name:              "test-cluster",
					NodeImage:         "artifactory.example.com/kindest/node:v1.29.0",
					KubernetesVersion: "1.34.0", // Different version to verify nodeImage takes priority
					Nodes: config.NodesConfig{
						ControlPlane: 1,
						Workers:      1,
					},
					Ingress: config.IngressConfig{
						Enabled: false,
					},
				},
			},
			expectedInYAML: "artifactory.example.com/kindest/node:v1.29.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kindConfig, err := BuildKindConfig(tt.cfg)
			if err != nil {
				t.Fatalf("BuildKindConfig failed: %v", err)
			}

			if !strings.Contains(kindConfig, tt.expectedInYAML) {
				t.Errorf("expected YAML to contain %q, but it didn't. YAML:\n%s", tt.expectedInYAML, kindConfig)
			}

			// Also verify it appears in the image field
			if !strings.Contains(kindConfig, "image: "+tt.expectedInYAML) {
				t.Errorf("expected YAML to contain 'image: %s', but it didn't. YAML:\n%s", tt.expectedInYAML, kindConfig)
			}
		})
	}
}
