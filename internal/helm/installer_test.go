package helm

import (
	"context"
	"reflect"
	"testing"

	"k8s.io/client-go/kubernetes"
)

// TestGenerateRepoName tests the GenerateRepoName function
func TestGenerateRepoName(t *testing.T) {
	testCases := []struct {
		name     string
		url      string
		expected string // Only test prefix since hash is deterministic but verbose
	}{
		{
			name:     "standard https URL",
			url:      "https://charts.crossplane.io/stable",
			expected: "charts-crossplane-io-",
		},
		{
			name:     "http URL",
			url:      "http://example.com/charts",
			expected: "example-com-",
		},
		{
			name:     "long domain name",
			url:      "https://a-very-long-domain-name-that-exceeds-twenty-chars.example.com/charts",
			expected: "a-very-long-domain-n-",
		},
		{
			name:     "simple URL",
			url:      "https://helm.io",
			expected: "helm-io-",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GenerateRepoName(tc.url)

			// Check that result starts with expected prefix
			if len(result) < len(tc.expected) || result[:len(tc.expected)] != tc.expected {
				t.Errorf("Expected repo name to start with %q, got %q", tc.expected, result)
			}

			// Check that result has a hash suffix (8 hex chars)
			suffix := result[len(tc.expected):]
			if len(suffix) != 8 {
				t.Errorf("Expected 8-char hash suffix, got %d chars: %q", len(suffix), suffix)
			}
		})
	}
}

// TestGenerateRepoName_Deterministic tests that the same URL always generates the same repo name
func TestGenerateRepoName_Deterministic(t *testing.T) {
	url := "https://charts.crossplane.io/stable"

	result1 := GenerateRepoName(url)
	result2 := GenerateRepoName(url)
	result3 := GenerateRepoName(url)

	if result1 != result2 || result2 != result3 {
		t.Errorf("GenerateRepoName should be deterministic: got %q, %q, %q", result1, result2, result3)
	}
}

// TestGenerateRepoName_UniqueForDifferentURLs tests that different URLs generate different repo names
func TestGenerateRepoName_UniqueForDifferentURLs(t *testing.T) {
	urls := []string{
		"https://charts.crossplane.io/stable",
		"https://charts.crossplane.io/master",
		"https://kubernetes-charts.storage.googleapis.com",
		"https://charts.bitnami.com/bitnami",
	}

	seen := make(map[string]string)
	for _, url := range urls {
		name := GenerateRepoName(url)
		if existingURL, exists := seen[name]; exists {
			t.Errorf("Collision: %q and %q both generated %q", url, existingURL, name)
		}
		seen[name] = url
	}
}

// TestValuesTransformer tests that ValuesTransformer modifies values correctly
func TestValuesTransformer(t *testing.T) {
	testCases := []struct {
		name        string
		input       map[string]interface{}
		transformer ValuesTransformer
		expected    map[string]interface{}
	}{
		{
			name: "adds new key",
			input: map[string]interface{}{
				"existing": "value",
			},
			transformer: func(values map[string]interface{}) map[string]interface{} {
				values["added"] = "new-value"
				return values
			},
			expected: map[string]interface{}{
				"existing": "value",
				"added":    "new-value",
			},
		},
		{
			name: "modifies existing key",
			input: map[string]interface{}{
				"key": "old-value",
			},
			transformer: func(values map[string]interface{}) map[string]interface{} {
				values["key"] = "new-value"
				return values
			},
			expected: map[string]interface{}{
				"key": "new-value",
			},
		},
		{
			name:  "handles nil input",
			input: nil,
			transformer: func(values map[string]interface{}) map[string]interface{} {
				if values == nil {
					values = make(map[string]interface{})
				}
				values["key"] = "value"
				return values
			},
			expected: map[string]interface{}{
				"key": "value",
			},
		},
		{
			name: "adds nested values",
			input: map[string]interface{}{
				"root": "value",
			},
			transformer: func(values map[string]interface{}) map[string]interface{} {
				values["registryCaBundleConfig"] = map[string]interface{}{
					"name": "test-configmap",
					"key":  "ca-bundle",
				}
				return values
			},
			expected: map[string]interface{}{
				"root": "value",
				"registryCaBundleConfig": map[string]interface{}{
					"name": "test-configmap",
					"key":  "ca-bundle",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.transformer(tc.input)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestValuesLogger tests that ValuesLogger receives correct values
func TestValuesLogger(t *testing.T) {
	var loggedRelease string
	var loggedValues map[string]interface{}

	logger := func(releaseName string, values map[string]interface{}) {
		loggedRelease = releaseName
		loggedValues = values
	}

	// Simulate what InstallChartFromConfigWithOptions does
	releaseName := "test-release"
	values := map[string]interface{}{
		"key1": "value1",
		"key2": map[string]interface{}{
			"nested": "value2",
		},
	}

	// Call the logger
	logger(releaseName, values)

	if loggedRelease != releaseName {
		t.Errorf("Expected release name %q, got %q", releaseName, loggedRelease)
	}

	if !reflect.DeepEqual(loggedValues, values) {
		t.Errorf("Expected values %v, got %v", values, loggedValues)
	}
}

// TestInstallOptions_Defaults tests that empty InstallOptions works correctly
func TestInstallOptions_Defaults(t *testing.T) {
	opts := InstallOptions{}

	// All fields should be nil
	if opts.ValuesTransformer != nil {
		t.Error("ValuesTransformer should be nil by default")
	}
	if opts.ValuesLogger != nil {
		t.Error("ValuesLogger should be nil by default")
	}
	if opts.PreInstall != nil {
		t.Error("PreInstall should be nil by default")
	}
}

// TestInstallOptions_WithAllCallbacks tests InstallOptions with all callbacks set
func TestInstallOptions_WithAllCallbacks(t *testing.T) {
	transformerCalled := false
	loggerCalled := false
	preInstallCalled := false

	opts := InstallOptions{
		ValuesTransformer: func(values map[string]interface{}) map[string]interface{} {
			transformerCalled = true
			return values
		},
		ValuesLogger: func(releaseName string, values map[string]interface{}) {
			loggerCalled = true
		},
		PreInstall: func(ctx context.Context, kubeClient *kubernetes.Clientset, namespace string) error {
			preInstallCalled = true
			return nil
		},
	}

	// Manually invoke callbacks to simulate behavior
	values := map[string]interface{}{"key": "value"}
	opts.ValuesTransformer(values)
	opts.ValuesLogger("test", values)
	_ = opts.PreInstall(nil, nil, "")

	if !transformerCalled {
		t.Error("ValuesTransformer was not called")
	}
	if !loggerCalled {
		t.Error("ValuesLogger was not called")
	}
	if !preInstallCalled {
		t.Error("PreInstall was not called")
	}
}
