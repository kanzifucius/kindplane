package kind

import (
	"reflect"
	"testing"

	"github.com/kanzi/kindplane/internal/config"
)

func TestPreloadResult(t *testing.T) {
	tests := []struct {
		name          string
		result        *PreloadResult
		wantLoaded    int
		wantMissing   int
		wantMissingAt int
		wantImage     string
	}{
		{
			name: "empty result",
			result: &PreloadResult{
				LoadedCount:   0,
				MissingImages: []string{},
			},
			wantLoaded:  0,
			wantMissing: 0,
		},
		{
			name: "some loaded, some missing",
			result: &PreloadResult{
				LoadedCount:   2,
				MissingImages: []string{"image1:v1", "image2:v2"},
			},
			wantLoaded:    2,
			wantMissing:   2,
			wantMissingAt: 0,
			wantImage:     "image1:v1",
		},
		{
			name: "all loaded",
			result: &PreloadResult{
				LoadedCount:   5,
				MissingImages: []string{},
			},
			wantLoaded:  5,
			wantMissing: 0,
		},
		{
			name: "all missing",
			result: &PreloadResult{
				LoadedCount:   0,
				MissingImages: []string{"img1:v1", "img2:v2", "img3:v3"},
			},
			wantLoaded:  0,
			wantMissing: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.LoadedCount != tt.wantLoaded {
				t.Errorf("LoadedCount = %d, want %d", tt.result.LoadedCount, tt.wantLoaded)
			}
			if len(tt.result.MissingImages) != tt.wantMissing {
				t.Errorf("len(MissingImages) = %d, want %d", len(tt.result.MissingImages), tt.wantMissing)
			}
			if tt.wantMissing > 0 && tt.wantImage != "" {
				if tt.result.MissingImages[tt.wantMissingAt] != tt.wantImage {
					t.Errorf("MissingImages[%d] = %s, want %s",
						tt.wantMissingAt, tt.result.MissingImages[tt.wantMissingAt], tt.wantImage)
				}
			}
		})
	}
}

func TestDeriveProviderImages(t *testing.T) {
	tests := []struct {
		name     string
		pkg      string
		expected []string
	}{
		{
			name: "standard upbound provider",
			pkg:  "xpkg.upbound.io/upbound/provider-aws:v1.1.0",
			expected: []string{
				"xpkg.upbound.io/upbound/provider-aws:v1.1.0",
				"xpkg.upbound.io/upbound/provider-aws-controller:v1.1.0",
			},
		},
		{
			name: "family provider",
			pkg:  "xpkg.upbound.io/upbound/provider-aws-s3:v1.1.0",
			expected: []string{
				"xpkg.upbound.io/upbound/provider-aws-s3:v1.1.0",
				"xpkg.upbound.io/upbound/provider-aws-s3-controller:v1.1.0",
			},
		},
		{
			name: "community provider",
			pkg:  "xpkg.upbound.io/crossplane-contrib/provider-kubernetes:v0.12.0",
			expected: []string{
				"xpkg.upbound.io/crossplane-contrib/provider-kubernetes:v0.12.0",
				"xpkg.upbound.io/crossplane-contrib/provider-kubernetes-controller:v0.12.0",
			},
		},
		{
			name: "provider with complex name",
			pkg:  "registry.example.com/team/provider-my-service:v2.0.0",
			expected: []string{
				"registry.example.com/team/provider-my-service:v2.0.0",
				"registry.example.com/team/provider-my-service-controller:v2.0.0",
			},
		},
		{
			name:     "malformed package (no tag)",
			pkg:      "xpkg.upbound.io/upbound/provider-aws",
			expected: []string{"xpkg.upbound.io/upbound/provider-aws"},
		},
		{
			name:     "malformed package (too short)",
			pkg:      "provider-aws:v1.0.0",
			expected: []string{"provider-aws:v1.0.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deriveProviderImages(tt.pkg)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("deriveProviderImages(%q) = %v, want %v", tt.pkg, result, tt.expected)
			}
		})
	}
}

func TestDeriveCrossplaneImages(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected []string
	}{
		{
			name:    "version with v prefix",
			version: "v1.15.0",
			expected: []string{
				"crossplane/crossplane:v1.15.0",
				"crossplane/crossplane-rbac-manager:v1.15.0",
			},
		},
		{
			name:    "version without v prefix",
			version: "1.15.0",
			expected: []string{
				"crossplane/crossplane:v1.15.0",
				"crossplane/crossplane-rbac-manager:v1.15.0",
			},
		},
		{
			name:    "version with patch",
			version: "1.14.5",
			expected: []string{
				"crossplane/crossplane:v1.14.5",
				"crossplane/crossplane-rbac-manager:v1.14.5",
			},
		},
		{
			name:    "version with rc",
			version: "v1.16.0-rc.0",
			expected: []string{
				"crossplane/crossplane:v1.16.0-rc.0",
				"crossplane/crossplane-rbac-manager:v1.16.0-rc.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deriveCrossplaneImages(tt.version)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("deriveCrossplaneImages(%q) = %v, want %v", tt.version, result, tt.expected)
			}
		})
	}
}

func TestRetagForRegistry(t *testing.T) {
	tests := []struct {
		name         string
		imageName    string
		registryHost string
		expected     string
	}{
		{
			name:         "xpkg with registry prefix",
			imageName:    "xpkg.upbound.io/upbound/provider-aws:v1.1.0",
			registryHost: "localhost:5001",
			expected:     "localhost:5001/upbound/provider-aws:v1.1.0",
		},
		{
			name:         "docker hub image",
			imageName:    "crossplane/crossplane:v1.15.0",
			registryHost: "localhost:5001",
			expected:     "localhost:5001/crossplane/crossplane:v1.15.0",
		},
		{
			name:         "custom registry",
			imageName:    "registry.example.com/team/image:v1.0.0",
			registryHost: "localhost:5001",
			expected:     "localhost:5001/team/image:v1.0.0",
		},
		{
			name:         "simple image name",
			imageName:    "alpine:latest",
			registryHost: "localhost:5001",
			expected:     "localhost:5001/alpine:latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := retagForRegistry(tt.imageName, tt.registryHost)
			if result != tt.expected {
				t.Errorf("retagForRegistry(%q, %q) = %q, want %q", tt.imageName, tt.registryHost, result, tt.expected)
			}
		})
	}
}

func TestGetShortImageName(t *testing.T) {
	tests := []struct {
		name      string
		imageName string
		expected  string
	}{
		{
			name:      "xpkg image",
			imageName: "xpkg.upbound.io/upbound/provider-aws:v1.1.0",
			expected:  "upbound/provider-aws:v1.1.0",
		},
		{
			name:      "docker hub image",
			imageName: "crossplane/crossplane:v1.15.0",
			expected:  "crossplane/crossplane:v1.15.0",
		},
		{
			name:      "custom registry",
			imageName: "registry.example.com/team/image:v1.0.0",
			expected:  "team/image:v1.0.0",
		},
		{
			name:      "simple image",
			imageName: "alpine:latest",
			expected:  "alpine:latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getShortImageName(tt.imageName)
			if result != tt.expected {
				t.Errorf("getShortImageName(%q) = %q, want %q", tt.imageName, result, tt.expected)
			}
		})
	}
}

func TestCollectImagesToPreload(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name     string
		cfg      *config.Config
		expected []string
	}{
		{
			name: "default config with provider",
			cfg: &config.Config{
				Crossplane: config.CrossplaneConfig{
					Version: "1.15.0",
					Providers: []config.ProviderConfig{
						{
							Name:    "provider-kubernetes",
							Package: "xpkg.upbound.io/crossplane-contrib/provider-kubernetes:v0.12.0",
						},
					},
				},
			},
			expected: []string{
				"crossplane/crossplane:v1.15.0",
				"crossplane/crossplane-rbac-manager:v1.15.0",
				"xpkg.upbound.io/crossplane-contrib/provider-kubernetes:v0.12.0",
				"xpkg.upbound.io/crossplane-contrib/provider-kubernetes-controller:v0.12.0",
			},
		},
		{
			name: "with additional images",
			cfg: &config.Config{
				Crossplane: config.CrossplaneConfig{
					Version: "1.15.0",
					ImageCache: &config.ImageCacheConfig{
						Enabled: &trueVal,
						AdditionalImages: []string{
							"custom-image:v1.0.0",
							"another-image:v2.0.0",
						},
					},
					Providers: []config.ProviderConfig{
						{
							Name:    "provider-aws",
							Package: "xpkg.upbound.io/upbound/provider-aws:v1.1.0",
						},
					},
				},
			},
			expected: []string{
				"crossplane/crossplane:v1.15.0",
				"crossplane/crossplane-rbac-manager:v1.15.0",
				"xpkg.upbound.io/upbound/provider-aws:v1.1.0",
				"xpkg.upbound.io/upbound/provider-aws-controller:v1.1.0",
				"custom-image:v1.0.0",
				"another-image:v2.0.0",
			},
		},
		{
			name: "with image overrides",
			cfg: &config.Config{
				Crossplane: config.CrossplaneConfig{
					Version: "1.15.0",
					ImageCache: &config.ImageCacheConfig{
						Enabled: &trueVal,
						ImageOverrides: map[string][]string{
							"xpkg.upbound.io/custom/provider:v1.0.0": {
								"registry.example.com/custom/controller:v1.0.0",
								"registry.example.com/custom/webhook:v1.0.0",
							},
						},
					},
					Providers: []config.ProviderConfig{
						{
							Name:    "custom-provider",
							Package: "xpkg.upbound.io/custom/provider:v1.0.0",
						},
					},
				},
			},
			expected: []string{
				"crossplane/crossplane:v1.15.0",
				"crossplane/crossplane-rbac-manager:v1.15.0",
				"registry.example.com/custom/controller:v1.0.0",
				"registry.example.com/custom/webhook:v1.0.0",
			},
		},
		{
			name: "crossplane disabled",
			cfg: &config.Config{
				Crossplane: config.CrossplaneConfig{
					Version: "1.15.0",
					ImageCache: &config.ImageCacheConfig{
						Enabled:           &trueVal,
						PreloadCrossplane: &falseVal,
					},
					Providers: []config.ProviderConfig{
						{
							Name:    "provider-kubernetes",
							Package: "xpkg.upbound.io/crossplane-contrib/provider-kubernetes:v0.12.0",
						},
					},
				},
			},
			expected: []string{
				"xpkg.upbound.io/crossplane-contrib/provider-kubernetes:v0.12.0",
				"xpkg.upbound.io/crossplane-contrib/provider-kubernetes-controller:v0.12.0",
			},
		},
		{
			name: "providers disabled",
			cfg: &config.Config{
				Crossplane: config.CrossplaneConfig{
					Version: "1.15.0",
					ImageCache: &config.ImageCacheConfig{
						Enabled:          &trueVal,
						PreloadProviders: &falseVal,
					},
					Providers: []config.ProviderConfig{
						{
							Name:    "provider-kubernetes",
							Package: "xpkg.upbound.io/crossplane-contrib/provider-kubernetes:v0.12.0",
						},
					},
				},
			},
			expected: []string{
				"crossplane/crossplane:v1.15.0",
				"crossplane/crossplane-rbac-manager:v1.15.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collectImagesToPreload(tt.cfg)

			// Sort both slices for comparison (order doesn't matter)
			if len(result) != len(tt.expected) {
				t.Errorf("collectImagesToPreload() returned %d images, want %d", len(result), len(tt.expected))
			}

			// Check that all expected images are present
			resultMap := make(map[string]bool)
			for _, img := range result {
				resultMap[img] = true
			}

			for _, expectedImg := range tt.expected {
				if !resultMap[expectedImg] {
					t.Errorf("collectImagesToPreload() missing expected image: %q", expectedImg)
				}
			}
		})
	}
}

func TestImageCacheConfig_IsEnabled(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name     string
		cfg      *config.ImageCacheConfig
		expected bool
	}{
		{
			name:     "nil config",
			cfg:      nil,
			expected: true, // Default is enabled
		},
		{
			name:     "empty config",
			cfg:      &config.ImageCacheConfig{},
			expected: true, // Default is enabled
		},
		{
			name: "explicitly enabled",
			cfg: &config.ImageCacheConfig{
				Enabled: &trueVal,
			},
			expected: true,
		},
		{
			name: "explicitly disabled",
			cfg: &config.ImageCacheConfig{
				Enabled: &falseVal,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cfg.IsEnabled()
			if result != tt.expected {
				t.Errorf("IsEnabled() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestImageCacheConfig_ShouldPreloadProviders(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name     string
		cfg      *config.ImageCacheConfig
		expected bool
	}{
		{
			name:     "nil config",
			cfg:      nil,
			expected: true, // Default is enabled
		},
		{
			name:     "empty config",
			cfg:      &config.ImageCacheConfig{},
			expected: true, // Default is enabled
		},
		{
			name: "explicitly enabled",
			cfg: &config.ImageCacheConfig{
				PreloadProviders: &trueVal,
			},
			expected: true,
		},
		{
			name: "explicitly disabled",
			cfg: &config.ImageCacheConfig{
				PreloadProviders: &falseVal,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cfg.ShouldPreloadProviders()
			if result != tt.expected {
				t.Errorf("ShouldPreloadProviders() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestImageCacheConfig_ShouldPreloadCrossplane(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name     string
		cfg      *config.ImageCacheConfig
		expected bool
	}{
		{
			name:     "nil config",
			cfg:      nil,
			expected: true, // Default is enabled
		},
		{
			name:     "empty config",
			cfg:      &config.ImageCacheConfig{},
			expected: true, // Default is enabled
		},
		{
			name: "explicitly enabled",
			cfg: &config.ImageCacheConfig{
				PreloadCrossplane: &trueVal,
			},
			expected: true,
		},
		{
			name: "explicitly disabled",
			cfg: &config.ImageCacheConfig{
				PreloadCrossplane: &falseVal,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cfg.ShouldPreloadCrossplane()
			if result != tt.expected {
				t.Errorf("ShouldPreloadCrossplane() = %v, want %v", result, tt.expected)
			}
		})
	}
}
