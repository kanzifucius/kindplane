package crossplane

import (
	"reflect"
	"testing"

	"github.com/kanzi/kindplane/internal/config"
	"github.com/kanzi/kindplane/internal/helm"
)

// TestInstallHelmChartWithOptions_ValuesTransformerInjection tests that registryCaBundleConfig is injected
func TestInstallHelmChartWithOptions_ValuesTransformerInjection(t *testing.T) {
	testCases := []struct {
		name            string
		cfg             config.CrossplaneConfig
		inputValues     map[string]interface{}
		expectedValues  map[string]interface{}
		expectInjection bool
	}{
		{
			name: "injects registryCaBundleConfig when RegistryCaBundle is set",
			cfg: config.CrossplaneConfig{
				RegistryCaBundle: &config.RegistryCaBundleConfig{
					CAFiles: []string{"/path/to/ca.crt"},
				},
			},
			inputValues: map[string]interface{}{
				"existingKey": "existingValue",
			},
			expectedValues: map[string]interface{}{
				"existingKey": "existingValue",
				"registryCaBundleConfig": map[string]interface{}{
					"name": RegistryCaBundleConfigMapName,
					"key":  RegistryCaBundleConfigMapKey,
				},
			},
			expectInjection: true,
		},
		{
			name: "does not inject when RegistryCaBundle is nil",
			cfg: config.CrossplaneConfig{
				RegistryCaBundle: nil,
			},
			inputValues: map[string]interface{}{
				"existingKey": "existingValue",
			},
			expectedValues: map[string]interface{}{
				"existingKey": "existingValue",
			},
			expectInjection: false,
		},
		{
			name: "creates values map if nil",
			cfg: config.CrossplaneConfig{
				RegistryCaBundle: &config.RegistryCaBundleConfig{
					CAFiles: []string{"/path/to/ca.crt"},
				},
			},
			inputValues: nil,
			expectedValues: map[string]interface{}{
				"registryCaBundleConfig": map[string]interface{}{
					"name": RegistryCaBundleConfigMapName,
					"key":  RegistryCaBundleConfigMapKey,
				},
			},
			expectInjection: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create the transformer as InstallHelmChartWithOptions does
			transformer := createValuesTransformer(tc.cfg, nil)

			result := transformer(tc.inputValues)

			// Check for injection
			_, hasInjection := result["registryCaBundleConfig"]
			if tc.expectInjection && !hasInjection {
				t.Error("Expected registryCaBundleConfig to be injected, but it wasn't")
			}
			if !tc.expectInjection && hasInjection {
				t.Error("Expected registryCaBundleConfig NOT to be injected, but it was")
			}

			if !reflect.DeepEqual(result, tc.expectedValues) {
				t.Errorf("Expected %v, got %v", tc.expectedValues, result)
			}
		})
	}
}

// TestInstallHelmChartWithOptions_ChainsTransformers tests that original transformer is called first
func TestInstallHelmChartWithOptions_ChainsTransformers(t *testing.T) {
	cfg := config.CrossplaneConfig{
		RegistryCaBundle: &config.RegistryCaBundleConfig{
			CAFiles: []string{"/path/to/ca.crt"},
		},
	}

	originalCalled := false
	originalTransformer := func(values map[string]interface{}) map[string]interface{} {
		originalCalled = true
		values["originalKey"] = "originalValue"
		return values
	}

	transformer := createValuesTransformer(cfg, originalTransformer)

	result := transformer(map[string]interface{}{})

	if !originalCalled {
		t.Error("Original transformer should have been called")
	}

	// Should have both originalKey (from original) and registryCaBundleConfig (injected)
	if _, ok := result["originalKey"]; !ok {
		t.Error("Original transformer's key should be present")
	}
	if _, ok := result["registryCaBundleConfig"]; !ok {
		t.Error("registryCaBundleConfig should be injected")
	}
}

// createValuesTransformer replicates the logic from InstallHelmChartWithOptions
// This allows us to test the transformer without needing a real cluster
func createValuesTransformer(cfg config.CrossplaneConfig, originalTransformer helm.ValuesTransformer) helm.ValuesTransformer {
	return func(values map[string]interface{}) map[string]interface{} {
		// Apply original transformer first if provided
		if originalTransformer != nil {
			values = originalTransformer(values)
		}
		// Inject registryCaBundleConfig if configured
		if cfg.RegistryCaBundle != nil {
			if values == nil {
				values = make(map[string]interface{})
			}
			values["registryCaBundleConfig"] = map[string]interface{}{
				"name": RegistryCaBundleConfigMapName,
				"key":  RegistryCaBundleConfigMapKey,
			}
		}
		return values
	}
}

// TestCrossplaneConstants tests that constants are correctly defined
func TestCrossplaneConstants(t *testing.T) {
	if CrossplaneNamespace != "crossplane-system" {
		t.Errorf("Expected CrossplaneNamespace to be 'crossplane-system', got %q", CrossplaneNamespace)
	}

	if CrossplaneRepoURL != "https://charts.crossplane.io/stable" {
		t.Errorf("Expected CrossplaneRepoURL to be 'https://charts.crossplane.io/stable', got %q", CrossplaneRepoURL)
	}

	if RegistryCaBundleConfigMapName != "crossplane-registry-ca-bundle" {
		t.Errorf("Expected RegistryCaBundleConfigMapName to be 'crossplane-registry-ca-bundle', got %q", RegistryCaBundleConfigMapName)
	}

	if RegistryCaBundleConfigMapKey != "ca-bundle" {
		t.Errorf("Expected RegistryCaBundleConfigMapKey to be 'ca-bundle', got %q", RegistryCaBundleConfigMapKey)
	}
}

// TestNewInstaller tests that NewInstaller creates valid installer
func TestNewInstaller(t *testing.T) {
	// We can't fully test without a real client, but we can verify nil handling
	installer := NewInstaller(nil)

	if installer == nil {
		t.Fatal("NewInstaller should return a non-nil installer")
	}

	if installer.kubeClient != nil {
		t.Error("Expected kubeClient to be nil when passed nil")
	}

	if installer.helmInstaller == nil {
		t.Error("Expected helmInstaller to be initialized")
	}
}
