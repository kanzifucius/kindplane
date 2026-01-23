package diagnostics

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ProviderDiagnostic contains diagnostic information about a Crossplane provider
type ProviderDiagnostic struct {
	Name       string
	Package    string
	Revision   string
	Healthy    bool
	Installed  bool
	Conditions []ProviderCondition
}

// ProviderCondition represents a provider condition
type ProviderCondition struct {
	Type               string
	Status             string
	Reason             string
	Message            string
	LastTransitionTime string
}

// providerGVR is the GroupVersionResource for Crossplane providers
var providerGVR = schema.GroupVersionResource{
	Group:    "pkg.crossplane.io",
	Version:  "v1",
	Resource: "providers",
}

// CollectProviderDiagnostics collects diagnostics for Crossplane providers
func (c *Collector) CollectProviderDiagnostics(ctx context.Context, diagCtx Context) ([]ProviderDiagnostic, error) {
	if c.dynamicClient == nil {
		return nil, fmt.Errorf("dynamic client not available")
	}

	providers, err := c.dynamicClient.Resource(providerGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing providers: %w", err)
	}

	var diagnostics []ProviderDiagnostic

	for _, p := range providers.Items {
		// Filter by provider names if specified
		if len(diagCtx.ProviderNames) > 0 {
			found := false
			for _, name := range diagCtx.ProviderNames {
				if p.GetName() == name {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		diag := collectProviderDiagnostic(&p)
		diagnostics = append(diagnostics, diag)
	}

	return diagnostics, nil
}

// collectProviderDiagnostic extracts diagnostic information from a provider
func collectProviderDiagnostic(p *unstructured.Unstructured) ProviderDiagnostic {
	diag := ProviderDiagnostic{
		Name: p.GetName(),
	}

	// Get package from spec
	if spec, found, _ := unstructured.NestedMap(p.Object, "spec"); found {
		if pkg, ok := spec["package"].(string); ok {
			diag.Package = pkg
		}
	}

	// Get status
	if status, found, _ := unstructured.NestedMap(p.Object, "status"); found {
		// Get current revision
		if revision, ok := status["currentRevision"].(string); ok {
			diag.Revision = revision
		}

		// Get conditions
		if conditions, found, _ := unstructured.NestedSlice(status, "conditions"); found {
			for _, c := range conditions {
				cond, ok := c.(map[string]interface{})
				if !ok {
					continue
				}

				pc := ProviderCondition{
					Type:   getString(cond, "type"),
					Status: getString(cond, "status"),
					Reason: getString(cond, "reason"),
				}

				// Get message (may be long)
				if msg, ok := cond["message"].(string); ok {
					pc.Message = msg
				}

				// Get last transition time
				if ltt, ok := cond["lastTransitionTime"].(string); ok {
					pc.LastTransitionTime = ltt
				}

				diag.Conditions = append(diag.Conditions, pc)

				// Check health status
				if pc.Type == "Healthy" {
					diag.Healthy = pc.Status == "True"
				}
				if pc.Type == "Installed" {
					diag.Installed = pc.Status == "True"
				}
			}
		}
	}

	return diag
}

// getString safely extracts a string from a map
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// GetUnhealthyProviders filters and returns only unhealthy providers
func GetUnhealthyProviders(providers []ProviderDiagnostic) []ProviderDiagnostic {
	var unhealthy []ProviderDiagnostic
	for _, p := range providers {
		if !p.Healthy {
			unhealthy = append(unhealthy, p)
		}
	}
	return unhealthy
}

// GetProviderErrors extracts error messages from provider conditions
func GetProviderErrors(providers []ProviderDiagnostic) []string {
	var errors []string
	for _, p := range providers {
		for _, c := range p.Conditions {
			if c.Status == "False" && c.Message != "" {
				errors = append(errors, fmt.Sprintf("%s: %s - %s", p.Name, c.Type, c.Message))
			}
		}
	}
	return errors
}
