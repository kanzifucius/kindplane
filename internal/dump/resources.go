package dump

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ResourceType represents a category of resources to dump
type ResourceType string

const (
	// Crossplane core resources
	ResourceTypeProviders        ResourceType = "providers"
	ResourceTypeProviderConfigs  ResourceType = "providerconfigs"
	ResourceTypeConfigurations   ResourceType = "configurations"
	ResourceTypeFunctions        ResourceType = "functions"
	ResourceTypeCompositions     ResourceType = "compositions"
	ResourceTypeXRDs             ResourceType = "xrds"
	ResourceTypeCompositesClaims ResourceType = "composites"

	// ESO resources
	ResourceTypeSecretStores    ResourceType = "secretstores"
	ResourceTypeExternalSecrets ResourceType = "externalsecrets"

	// Kubernetes core resources
	ResourceTypeNamespaces   ResourceType = "namespaces"
	ResourceTypeSecrets      ResourceType = "secrets"
	ResourceTypeConfigMaps   ResourceType = "configmaps"
	ResourceTypeHelmReleases ResourceType = "helmreleases"
)

// ResourceInfo contains information about a resource type for dumping
type ResourceInfo struct {
	// Type is the resource type identifier
	Type ResourceType

	// DisplayName is a human-readable name
	DisplayName string

	// GVR is the GroupVersionResource for the Kubernetes API
	GVR schema.GroupVersionResource

	// Namespaced indicates if the resource is namespaced
	Namespaced bool

	// Category groups related resources together
	Category string

	// Priority determines dump order (lower = first)
	Priority int

	// OutputSubdir is the subdirectory for file output
	OutputSubdir string
}

// DefaultResources returns the default set of resources to dump
func DefaultResources() []ResourceInfo {
	return []ResourceInfo{
		// Crossplane Core
		{
			Type:         ResourceTypeProviders,
			DisplayName:  "Providers",
			GVR:          schema.GroupVersionResource{Group: "pkg.crossplane.io", Version: "v1", Resource: "providers"},
			Namespaced:   false,
			Category:     "crossplane",
			Priority:     10,
			OutputSubdir: "crossplane/providers",
		},
		{
			Type:         ResourceTypeProviderConfigs,
			DisplayName:  "Provider Configs",
			GVR:          schema.GroupVersionResource{Group: "pkg.crossplane.io", Version: "v1alpha1", Resource: "providerconfigs"},
			Namespaced:   false,
			Category:     "crossplane",
			Priority:     15,
			OutputSubdir: "crossplane/providerconfigs",
		},
		{
			Type:         ResourceTypeConfigurations,
			DisplayName:  "Configurations",
			GVR:          schema.GroupVersionResource{Group: "pkg.crossplane.io", Version: "v1", Resource: "configurations"},
			Namespaced:   false,
			Category:     "crossplane",
			Priority:     20,
			OutputSubdir: "crossplane/configurations",
		},
		{
			Type:         ResourceTypeFunctions,
			DisplayName:  "Functions",
			GVR:          schema.GroupVersionResource{Group: "pkg.crossplane.io", Version: "v1beta1", Resource: "functions"},
			Namespaced:   false,
			Category:     "crossplane",
			Priority:     25,
			OutputSubdir: "crossplane/functions",
		},
		{
			Type:         ResourceTypeXRDs,
			DisplayName:  "Composite Resource Definitions",
			GVR:          schema.GroupVersionResource{Group: "apiextensions.crossplane.io", Version: "v1", Resource: "compositeresourcedefinitions"},
			Namespaced:   false,
			Category:     "crossplane",
			Priority:     30,
			OutputSubdir: "crossplane/xrds",
		},
		{
			Type:         ResourceTypeCompositions,
			DisplayName:  "Compositions",
			GVR:          schema.GroupVersionResource{Group: "apiextensions.crossplane.io", Version: "v1", Resource: "compositions"},
			Namespaced:   false,
			Category:     "crossplane",
			Priority:     35,
			OutputSubdir: "crossplane/compositions",
		},

		// ESO Resources
		{
			Type:         ResourceTypeSecretStores,
			DisplayName:  "Secret Stores",
			GVR:          schema.GroupVersionResource{Group: "external-secrets.io", Version: "v1beta1", Resource: "secretstores"},
			Namespaced:   true,
			Category:     "eso",
			Priority:     50,
			OutputSubdir: "eso/secretstores",
		},
		{
			Type:         ResourceTypeExternalSecrets,
			DisplayName:  "External Secrets",
			GVR:          schema.GroupVersionResource{Group: "external-secrets.io", Version: "v1beta1", Resource: "externalsecrets"},
			Namespaced:   true,
			Category:     "eso",
			Priority:     55,
			OutputSubdir: "eso/externalsecrets",
		},

		// Kubernetes Core
		{
			Type:         ResourceTypeNamespaces,
			DisplayName:  "Namespaces",
			GVR:          schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"},
			Namespaced:   false,
			Category:     "kubernetes",
			Priority:     5,
			OutputSubdir: "namespaces",
		},
		{
			Type:         ResourceTypeSecrets,
			DisplayName:  "Secrets",
			GVR:          schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"},
			Namespaced:   true,
			Category:     "kubernetes",
			Priority:     60,
			OutputSubdir: "secrets",
		},
		{
			Type:         ResourceTypeConfigMaps,
			DisplayName:  "ConfigMaps",
			GVR:          schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"},
			Namespaced:   true,
			Category:     "kubernetes",
			Priority:     65,
			OutputSubdir: "configmaps",
		},
	}
}

// ClusterScopedProviderConfigs returns GVRs for common provider-specific ProviderConfig types
// These are dynamically registered by providers and are cluster-scoped
func ClusterScopedProviderConfigs() []schema.GroupVersionResource {
	return []schema.GroupVersionResource{
		// AWS
		{Group: "aws.upbound.io", Version: "v1beta1", Resource: "providerconfigs"},
		// Azure
		{Group: "azure.upbound.io", Version: "v1beta1", Resource: "providerconfigs"},
		// GCP
		{Group: "gcp.upbound.io", Version: "v1beta1", Resource: "providerconfigs"},
		// Kubernetes
		{Group: "kubernetes.crossplane.io", Version: "v1alpha1", Resource: "providerconfigs"},
		// Helm
		{Group: "helm.crossplane.io", Version: "v1beta1", Resource: "providerconfigs"},
	}
}

// SystemNamespaces returns namespaces that should be excluded from dumps by default
func SystemNamespaces() []string {
	return []string{
		"kube-system",
		"kube-public",
		"kube-node-lease",
		"local-path-storage",
	}
}

// SystemNamespacePatterns returns patterns for system namespaces to exclude
func SystemNamespacePatterns() []string {
	return []string{
		"crossplane-system",
	}
}

// AnnotationsToStrip returns annotations that should be removed during cleanup
func AnnotationsToStrip() []string {
	return []string{
		"kubectl.kubernetes.io/last-applied-configuration",
		"deployment.kubernetes.io/revision",
		"meta.helm.sh/release-name",
		"meta.helm.sh/release-namespace",
		"helm.sh/hook",
		"helm.sh/hook-delete-policy",
		"helm.sh/hook-weight",
	}
}

// LabelsToStrip returns labels that should be removed during cleanup
func LabelsToStrip() []string {
	return []string{
		"app.kubernetes.io/managed-by",
	}
}

// MetadataFieldsToStrip returns metadata fields that should be removed
func MetadataFieldsToStrip() []string {
	return []string{
		"managedFields",
		"resourceVersion",
		"uid",
		"creationTimestamp",
		"generation",
		"selfLink",
	}
}

// StatusFieldsToStrip returns status-related fields to remove for GitOps compatibility
func StatusFieldsToStrip() []string {
	return []string{
		"status",
	}
}

// ResourcesByType returns a map of resource type to ResourceInfo for quick lookup
func ResourcesByType() map[ResourceType]ResourceInfo {
	resources := DefaultResources()
	result := make(map[ResourceType]ResourceInfo, len(resources))
	for _, r := range resources {
		result[r.Type] = r
	}
	return result
}

// ResourcesByCategory returns resources grouped by category
func ResourcesByCategory() map[string][]ResourceInfo {
	resources := DefaultResources()
	result := make(map[string][]ResourceInfo)
	for _, r := range resources {
		result[r.Category] = append(result[r.Category], r)
	}
	return result
}

// FilterResources filters the resource list based on include/exclude lists
func FilterResources(resources []ResourceInfo, include, exclude []ResourceType) []ResourceInfo {
	// If include list is provided, only include those
	if len(include) > 0 {
		includeSet := make(map[ResourceType]bool)
		for _, t := range include {
			includeSet[t] = true
		}
		var filtered []ResourceInfo
		for _, r := range resources {
			if includeSet[r.Type] {
				filtered = append(filtered, r)
			}
		}
		resources = filtered
	}

	// Apply exclusions
	if len(exclude) > 0 {
		excludeSet := make(map[ResourceType]bool)
		for _, t := range exclude {
			excludeSet[t] = true
		}
		var filtered []ResourceInfo
		for _, r := range resources {
			if !excludeSet[r.Type] {
				filtered = append(filtered, r)
			}
		}
		resources = filtered
	}

	return resources
}

// ParseResourceTypes parses a comma-separated string into resource types
func ParseResourceTypes(s string) []ResourceType {
	if s == "" {
		return nil
	}

	var types []ResourceType
	// Simple split - in production you'd use strings.Split
	current := ""
	for _, c := range s {
		if c == ',' {
			if current != "" {
				types = append(types, ResourceType(current))
				current = ""
			}
		} else if c != ' ' {
			current += string(c)
		}
	}
	if current != "" {
		types = append(types, ResourceType(current))
	}
	return types
}

// AllResourceTypes returns all available resource type identifiers
func AllResourceTypes() []ResourceType {
	resources := DefaultResources()
	types := make([]ResourceType, len(resources))
	for i, r := range resources {
		types[i] = r.Type
	}
	return types
}
