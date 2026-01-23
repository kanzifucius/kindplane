package dump

import (
	"context"
	"fmt"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// Dumper handles fetching and processing resources from a Kubernetes cluster
type Dumper struct {
	// DynamicClient is the Kubernetes dynamic client
	DynamicClient dynamic.Interface

	// DiscoveryClient is used to discover available API resources
	DiscoveryClient discovery.DiscoveryInterface

	// Cleaner handles resource cleanup
	Cleaner *Cleaner

	// Options for the dump operation
	Options DumpOptions
}

// DumpOptions configures the dump operation
type DumpOptions struct {
	// Namespaces to dump from (empty = all namespaces)
	Namespaces []string

	// AllNamespaces dumps from all namespaces
	AllNamespaces bool

	// IncludeSystemNamespaces includes system namespaces in the dump
	IncludeSystemNamespaces bool

	// IncludeTypes is the list of resource types to include (empty = all)
	IncludeTypes []ResourceType

	// ExcludeTypes is the list of resource types to exclude
	ExcludeTypes []ResourceType

	// SkipSecrets skips secrets entirely
	SkipSecrets bool

	// DryRun only shows what would be dumped without fetching
	DryRun bool

	// DiscoverComposites attempts to discover and dump XR instances
	DiscoverComposites bool
}

// DumpResult contains the results of a dump operation
type DumpResult struct {
	// Resources contains all dumped resources grouped by type
	Resources map[ResourceType][]*unstructured.Unstructured

	// Errors contains any non-fatal errors encountered
	Errors []error

	// Stats contains statistics about the dump
	Stats DumpStats

	// DiscoveredXRDs contains XRD info for discovered composite resources
	DiscoveredXRDs []XRDInfo
}

// DumpStats contains statistics about the dump operation
type DumpStats struct {
	// TotalResources is the total number of resources dumped
	TotalResources int

	// ResourceCounts maps resource type to count
	ResourceCounts map[ResourceType]int

	// NamespacesScanned is the number of namespaces scanned
	NamespacesScanned int

	// SkippedResources is the number of resources skipped
	SkippedResources int
}

// XRDInfo contains information about a discovered XRD
type XRDInfo struct {
	Name     string
	Group    string
	Version  string
	Kind     string
	Plural   string
	Listable bool
}

// NewDumper creates a new Dumper with the given Kubernetes config
func NewDumper(config *rest.Config, opts DumpOptions) (*Dumper, error) {
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating dynamic client: %w", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating discovery client: %w", err)
	}

	cleaner := NewCleaner()
	if opts.SkipSecrets {
		cleaner.RedactSecrets = false // We'll skip them entirely
	}

	return &Dumper{
		DynamicClient:   dynamicClient,
		DiscoveryClient: discoveryClient,
		Cleaner:         cleaner,
		Options:         opts,
	}, nil
}

// Dump performs the dump operation
func (d *Dumper) Dump(ctx context.Context) (*DumpResult, error) {
	result := &DumpResult{
		Resources: make(map[ResourceType][]*unstructured.Unstructured),
		Stats: DumpStats{
			ResourceCounts: make(map[ResourceType]int),
		},
	}

	// Get the list of resources to dump
	resources := d.getResourcesToFetch()

	// Fetch each resource type
	for _, resourceInfo := range resources {
		if d.Options.DryRun {
			result.Stats.ResourceCounts[resourceInfo.Type] = 0
			continue
		}

		// Skip secrets if configured
		if d.Options.SkipSecrets && resourceInfo.Type == ResourceTypeSecrets {
			continue
		}

		objects, err := d.fetchResources(ctx, resourceInfo)
		if err != nil {
			// Non-fatal: resource type might not exist
			result.Errors = append(result.Errors, fmt.Errorf("fetching %s: %w", resourceInfo.DisplayName, err))
			continue
		}

		if len(objects) > 0 {
			result.Resources[resourceInfo.Type] = objects
			result.Stats.ResourceCounts[resourceInfo.Type] = len(objects)
			result.Stats.TotalResources += len(objects)
		}
	}

	// Discover and dump composite resources if enabled
	if d.Options.DiscoverComposites && !d.Options.DryRun {
		xrds, composites, err := d.discoverAndDumpComposites(ctx)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("discovering composites: %w", err))
		} else {
			result.DiscoveredXRDs = xrds
			if len(composites) > 0 {
				result.Resources[ResourceTypeCompositesClaims] = composites
				result.Stats.ResourceCounts[ResourceTypeCompositesClaims] = len(composites)
				result.Stats.TotalResources += len(composites)
			}
		}
	}

	return result, nil
}

// getResourcesToFetch returns the filtered list of resources to fetch
func (d *Dumper) getResourcesToFetch() []ResourceInfo {
	resources := DefaultResources()
	resources = FilterResources(resources, d.Options.IncludeTypes, d.Options.ExcludeTypes)

	// Sort by priority
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Priority < resources[j].Priority
	})

	return resources
}

// fetchResources fetches all resources of a given type
func (d *Dumper) fetchResources(ctx context.Context, info ResourceInfo) ([]*unstructured.Unstructured, error) {
	var allObjects []*unstructured.Unstructured

	if info.Namespaced {
		// Fetch from specified namespaces or all namespaces
		namespaces, err := d.getNamespacesToScan(ctx)
		if err != nil {
			return nil, err
		}

		for _, ns := range namespaces {
			objects, err := d.fetchFromNamespace(ctx, info.GVR, ns)
			if err != nil {
				// Continue with other namespaces
				continue
			}
			allObjects = append(allObjects, objects...)
		}
	} else {
		// Cluster-scoped resources
		objects, err := d.fetchClusterScoped(ctx, info.GVR)
		if err != nil {
			return nil, err
		}
		allObjects = objects
	}

	// Clean and filter the objects
	var cleaned []*unstructured.Unstructured
	for _, obj := range allObjects {
		// Skip system resources unless configured otherwise
		if !d.Options.IncludeSystemNamespaces && IsSystemResource(obj) {
			d.Options.IncludeSystemNamespaces = false // Just to avoid lint warning
			continue
		}

		// Skip certain resource types
		if ShouldSkipResource(obj) {
			continue
		}

		// Clean the resource
		cleanedObj, err := d.Cleaner.Clean(obj)
		if err != nil {
			continue
		}
		cleaned = append(cleaned, cleanedObj)
	}

	return cleaned, nil
}

// getNamespacesToScan returns the list of namespaces to scan
func (d *Dumper) getNamespacesToScan(ctx context.Context) ([]string, error) {
	if len(d.Options.Namespaces) > 0 {
		return d.Options.Namespaces, nil
	}

	if d.Options.AllNamespaces {
		// List all namespaces
		nsList, err := d.DynamicClient.Resource(schema.GroupVersionResource{
			Group:    "",
			Version:  "v1",
			Resource: "namespaces",
		}).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("listing namespaces: %w", err)
		}

		var namespaces []string
		for _, ns := range nsList.Items {
			nsName := ns.GetName()
			if !d.Options.IncludeSystemNamespaces && ShouldSkipNamespace(nsName) {
				continue
			}
			namespaces = append(namespaces, nsName)
		}
		return namespaces, nil
	}

	// Default to "default" namespace
	return []string{"default"}, nil
}

// fetchFromNamespace fetches resources from a specific namespace
func (d *Dumper) fetchFromNamespace(ctx context.Context, gvr schema.GroupVersionResource, namespace string) ([]*unstructured.Unstructured, error) {
	list, err := d.DynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	objects := make([]*unstructured.Unstructured, len(list.Items))
	for i := range list.Items {
		objects[i] = &list.Items[i]
	}
	return objects, nil
}

// fetchClusterScoped fetches cluster-scoped resources
func (d *Dumper) fetchClusterScoped(ctx context.Context, gvr schema.GroupVersionResource) ([]*unstructured.Unstructured, error) {
	list, err := d.DynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	objects := make([]*unstructured.Unstructured, len(list.Items))
	for i := range list.Items {
		objects[i] = &list.Items[i]
	}
	return objects, nil
}

// discoverAndDumpComposites discovers XRDs and dumps their instances
func (d *Dumper) discoverAndDumpComposites(ctx context.Context) ([]XRDInfo, []*unstructured.Unstructured, error) {
	// First, fetch all XRDs
	xrdGVR := schema.GroupVersionResource{
		Group:    "apiextensions.crossplane.io",
		Version:  "v1",
		Resource: "compositeresourcedefinitions",
	}

	xrdList, err := d.DynamicClient.Resource(xrdGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("listing XRDs: %w", err)
	}

	var xrdInfos []XRDInfo
	var allComposites []*unstructured.Unstructured

	for _, xrd := range xrdList.Items {
		// Extract XRD spec
		spec, found, _ := unstructured.NestedMap(xrd.Object, "spec")
		if !found {
			continue
		}

		group, _, _ := unstructured.NestedString(spec, "group")
		names, _, _ := unstructured.NestedMap(spec, "names")
		if names == nil {
			continue
		}

		kind, _ := names["kind"].(string)
		plural, _ := names["plural"].(string)

		// Get the served version
		versions, _, _ := unstructured.NestedSlice(spec, "versions")
		var servedVersion string
		for _, v := range versions {
			vMap, ok := v.(map[string]interface{})
			if !ok {
				continue
			}
			if served, _ := vMap["served"].(bool); served {
				if name, ok := vMap["name"].(string); ok {
					servedVersion = name
					break
				}
			}
		}

		if group == "" || plural == "" || servedVersion == "" {
			continue
		}

		xrdInfo := XRDInfo{
			Name:     xrd.GetName(),
			Group:    group,
			Version:  servedVersion,
			Kind:     kind,
			Plural:   plural,
			Listable: true,
		}
		xrdInfos = append(xrdInfos, xrdInfo)

		// Fetch instances of this composite resource
		compositeGVR := schema.GroupVersionResource{
			Group:    group,
			Version:  servedVersion,
			Resource: plural,
		}

		compositeList, err := d.DynamicClient.Resource(compositeGVR).List(ctx, metav1.ListOptions{})
		if err != nil {
			// Not all XRDs may have instances
			continue
		}

		for i := range compositeList.Items {
			obj := &compositeList.Items[i]

			// Clean the composite resource
			cleaned, err := d.Cleaner.Clean(obj)
			if err != nil {
				continue
			}
			allComposites = append(allComposites, cleaned)
		}
	}

	return xrdInfos, allComposites, nil
}

// FetchProviderConfigs fetches provider-specific ProviderConfig resources
func (d *Dumper) FetchProviderConfigs(ctx context.Context) ([]*unstructured.Unstructured, error) {
	var allConfigs []*unstructured.Unstructured

	for _, gvr := range ClusterScopedProviderConfigs() {
		configs, err := d.fetchClusterScoped(ctx, gvr)
		if err != nil {
			// Provider might not be installed
			continue
		}

		for _, config := range configs {
			cleaned, err := d.Cleaner.Clean(config)
			if err != nil {
				continue
			}
			allConfigs = append(allConfigs, cleaned)
		}
	}

	return allConfigs, nil
}

// DryRunResult returns information about what would be dumped
func (d *Dumper) DryRunResult() *DumpResult {
	result := &DumpResult{
		Resources: make(map[ResourceType][]*unstructured.Unstructured),
		Stats: DumpStats{
			ResourceCounts: make(map[ResourceType]int),
		},
	}

	resources := d.getResourcesToFetch()
	for _, r := range resources {
		result.Stats.ResourceCounts[r.Type] = -1 // -1 indicates "would be fetched"
	}

	return result
}

// GetResourceInfo returns information about the resources that would be dumped
func (d *Dumper) GetResourceInfo() []ResourceInfo {
	return d.getResourcesToFetch()
}
