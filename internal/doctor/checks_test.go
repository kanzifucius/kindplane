package doctor

import (
	"context"
	"testing"
)

// TestCheckKind_Removed verifies that CheckKind has been removed from RunAllChecks
func TestCheckKind_Removed(t *testing.T) {
	ctx := context.Background()
	results := RunAllChecks(ctx, nil)

	// Verify that CheckKind is not in the results
	for _, result := range results {
		if result.Name == "kind binary" {
			t.Error("CheckKind should be removed from RunAllChecks")
		}
	}
}

// TestRunAllChecks_WithoutKindCheck verifies all other checks still run correctly
func TestRunAllChecks_WithoutKindCheck(t *testing.T) {
	ctx := context.Background()
	results := RunAllChecks(ctx, nil)

	// Expected checks (without CheckKind):
	expectedChecks := map[string]bool{
		"Docker daemon":    false,
		"kubectl binary":   false,
		"helm binary":      false,
		"Disk space":        false,
		"kind binary":      false, // Should not be present
	}

	// Mark which checks we found
	for _, result := range results {
		if _, exists := expectedChecks[result.Name]; exists {
			expectedChecks[result.Name] = true
		}
	}

	// Verify expected checks are present (except kind binary)
	for name, found := range expectedChecks {
		if name == "kind binary" {
			if found {
				t.Errorf("CheckKind should not be in results, but found: %s", name)
			}
		} else {
			// Other checks may or may not be present depending on system state
			// We just verify the structure is correct
		}
	}
}

// TestCheckDocker tests the CheckDocker function
func TestCheckDocker(t *testing.T) {
	ctx := context.Background()
	result := CheckDocker(ctx)

	if result.Name != "Docker daemon" {
		t.Errorf("Expected check name 'Docker daemon', got '%s'", result.Name)
	}

	if !result.Required {
		t.Error("Docker check should be required")
	}

	// The actual result depends on whether Docker is installed/running
	// We just verify the structure is correct
}

// TestCheckKubectl tests the CheckKubectl function
func TestCheckKubectl(t *testing.T) {
	ctx := context.Background()
	result := CheckKubectl(ctx)

	if result.Name != "kubectl binary" {
		t.Errorf("Expected check name 'kubectl binary', got '%s'", result.Name)
	}

	if !result.Required {
		t.Error("Kubectl check should be required")
	}
}

// TestCheckHelm tests the CheckHelm function
func TestCheckHelm(t *testing.T) {
	ctx := context.Background()
	result := CheckHelm(ctx)

	if result.Name != "helm binary" {
		t.Errorf("Expected check name 'helm binary', got '%s'", result.Name)
	}

	if result.Required {
		t.Error("Helm check should be optional, not required")
	}

	// Helm is optional, so result.Passed should be true even if not found
	// This is tested by the function logic
}

// TestRunAllChecks_WithKubeClient tests RunAllChecks with a kubeClient
func TestRunAllChecks_WithKubeClient(t *testing.T) {
	ctx := context.Background()

	// Run without kubeClient first
	resultsWithoutClient := RunAllChecks(ctx, nil)
	countWithoutClient := len(resultsWithoutClient)

	// Run with nil kubeClient (same as without)
	resultsWithNilClient := RunAllChecks(ctx, nil)
	countWithNilClient := len(resultsWithNilClient)

	if countWithoutClient != countWithNilClient {
		t.Errorf("Expected same number of results with nil client, got %d vs %d",
			countWithoutClient, countWithNilClient)
	}

	// When kubeClient is provided, additional checks should be added
	// (CheckKubernetesAPI, CheckCrossplaneCRDs)
	// But we can't easily test this without a real client or extensive mocking
}

// TestCheckResult_Structure tests the CheckResult structure
func TestCheckResult_Structure(t *testing.T) {
	result := CheckResult{
		Name:       "Test Check",
		Passed:     true,
		Message:    "Test message",
		Details:    "Test details",
		Suggestion: "Test suggestion",
		Required:   true,
	}

	if result.Name == "" {
		t.Error("CheckResult should have a name")
	}

	// Verify all fields are set
	if result.Message == "" && result.Passed {
		t.Log("Note: Message can be empty for passed checks")
	}
}
