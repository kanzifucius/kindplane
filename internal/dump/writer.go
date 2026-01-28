package dump

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// OutputFormat specifies the output format for the dump
type OutputFormat string

const (
	// OutputFormatFiles writes each resource to a separate file
	OutputFormatFiles OutputFormat = "files"

	// OutputFormatSingle writes all resources to a single multi-document YAML
	OutputFormatSingle OutputFormat = "single"
)

// Writer handles writing dump results to disk or stdout
type Writer struct {
	// OutputDir is the base directory for file output
	OutputDir string

	// Format specifies the output format
	Format OutputFormat

	// Stdout is used for stdout output
	Stdout io.Writer

	// GenerateReadme generates a README file with dump info
	GenerateReadme bool

	// ResourcesByType is used for organizing output
	ResourcesByType map[ResourceType]ResourceInfo
}

// NewWriter creates a new Writer with default settings
func NewWriter(outputDir string, format OutputFormat) *Writer {
	resourcesByType := ResourcesByType()
	return &Writer{
		OutputDir:       outputDir,
		Format:          format,
		Stdout:          os.Stdout,
		GenerateReadme:  true,
		ResourcesByType: resourcesByType,
	}
}

// Write writes the dump result to the configured output
func (w *Writer) Write(result *DumpResult) error {
	switch w.Format {
	case OutputFormatFiles:
		return w.writeFiles(result)
	case OutputFormatSingle:
		return w.writeSingleFile(result)
	default:
		return fmt.Errorf("unknown output format: %s", w.Format)
	}
}

// WriteToStdout writes the dump result to stdout
func (w *Writer) WriteToStdout(result *DumpResult) error {
	return w.writeToWriter(result, w.Stdout)
}

// writeFiles writes each resource to a separate file
func (w *Writer) writeFiles(result *DumpResult) error {
	// Create base output directory
	if err := os.MkdirAll(w.OutputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Write Kind config if present
	if result.KindConfig != "" {
		kindConfigPath := filepath.Join(w.OutputDir, "kind-config.yaml")
		if err := os.WriteFile(kindConfigPath, []byte(result.KindConfig), 0644); err != nil {
			return fmt.Errorf("writing kind-config.yaml: %w", err)
		}
	}

	// Get sorted resource types for consistent output
	resourceTypes := w.getSortedResourceTypes(result)

	for _, rt := range resourceTypes {
		objects := result.Resources[rt]
		if len(objects) == 0 {
			continue
		}

		// Get the output subdirectory for this resource type
		subdir := w.getSubdirForType(rt)
		dir := filepath.Join(w.OutputDir, subdir)

		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}

		for _, obj := range objects {
			filename := w.generateFilename(obj)
			filepath := filepath.Join(dir, filename)

			if err := w.writeResourceToFile(obj, filepath); err != nil {
				return fmt.Errorf("writing %s: %w", filepath, err)
			}
		}
	}

	// Write README if configured
	if w.GenerateReadme {
		if err := w.writeReadme(result); err != nil {
			return fmt.Errorf("writing README: %w", err)
		}
	}

	return nil
}

// writeSingleFile writes all resources to a single multi-document YAML file
func (w *Writer) writeSingleFile(result *DumpResult) error {
	// Create base output directory
	if err := os.MkdirAll(w.OutputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	filepath := filepath.Join(w.OutputDir, "dump.yaml")
	f, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if err := w.writeToWriter(result, f); err != nil {
		return err
	}

	// Write README if configured
	if w.GenerateReadme {
		if err := w.writeReadme(result); err != nil {
			return fmt.Errorf("writing README: %w", err)
		}
	}

	return nil
}

// writeToWriter writes all resources to a writer as multi-document YAML
func (w *Writer) writeToWriter(result *DumpResult, writer io.Writer) error {
	first := true

	// Write Kind config first if present
	if result.KindConfig != "" {
		header := "# Kind cluster configuration\n# Use: kind create cluster --config kind-config.yaml\n"
		if _, err := writer.Write([]byte(header + result.KindConfig)); err != nil {
			return err
		}
		first = false
	}

	resourceTypes := w.getSortedResourceTypes(result)

	for _, rt := range resourceTypes {
		objects := result.Resources[rt]
		if len(objects) == 0 {
			continue
		}

		// Sort objects by name for consistent output
		sortedObjects := w.sortObjects(objects)

		for _, obj := range sortedObjects {
			if !first {
				if _, err := writer.Write([]byte("---\n")); err != nil {
					return err
				}
			}
			first = false

			yamlBytes, err := yaml.Marshal(obj.Object)
			if err != nil {
				return fmt.Errorf("marshaling %s/%s: %w", obj.GetKind(), obj.GetName(), err)
			}

			if _, err := writer.Write(yamlBytes); err != nil {
				return err
			}
		}
	}

	return nil
}

// writeResourceToFile writes a single resource to a file
func (w *Writer) writeResourceToFile(obj *unstructured.Unstructured, filepath string) error {
	yamlBytes, err := yaml.Marshal(obj.Object)
	if err != nil {
		return fmt.Errorf("marshaling: %w", err)
	}

	return os.WriteFile(filepath, yamlBytes, 0644)
}

// generateFilename generates a filename for a resource
func (w *Writer) generateFilename(obj *unstructured.Unstructured) string {
	name := obj.GetName()
	namespace := obj.GetNamespace()

	// Sanitize the name for filesystem
	name = sanitizeFilename(name)

	if namespace != "" {
		return fmt.Sprintf("%s.%s.yaml", namespace, name)
	}
	return fmt.Sprintf("%s.yaml", name)
}

// getSubdirForType returns the output subdirectory for a resource type
func (w *Writer) getSubdirForType(rt ResourceType) string {
	if info, ok := w.ResourcesByType[rt]; ok {
		return info.OutputSubdir
	}
	return string(rt)
}

// getSortedResourceTypes returns resource types sorted by priority
func (w *Writer) getSortedResourceTypes(result *DumpResult) []ResourceType {
	var types []ResourceType
	for rt := range result.Resources {
		types = append(types, rt)
	}

	// Sort by priority
	sort.Slice(types, func(i, j int) bool {
		pi, pj := 999, 999
		if info, ok := w.ResourcesByType[types[i]]; ok {
			pi = info.Priority
		}
		if info, ok := w.ResourcesByType[types[j]]; ok {
			pj = info.Priority
		}
		return pi < pj
	})

	return types
}

// sortObjects sorts objects by namespace and name
func (w *Writer) sortObjects(objects []*unstructured.Unstructured) []*unstructured.Unstructured {
	sorted := make([]*unstructured.Unstructured, len(objects))
	copy(sorted, objects)

	sort.Slice(sorted, func(i, j int) bool {
		// Sort by namespace first
		nsI, nsJ := sorted[i].GetNamespace(), sorted[j].GetNamespace()
		if nsI != nsJ {
			return nsI < nsJ
		}
		// Then by name
		return sorted[i].GetName() < sorted[j].GetName()
	})

	return sorted
}

// writeReadme generates a README file with dump information
func (w *Writer) writeReadme(result *DumpResult) error {
	readmePath := filepath.Join(w.OutputDir, "README.md")

	var sb strings.Builder

	sb.WriteString("# Kindplane Dump\n\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC3339)))

	sb.WriteString("## Summary\n\n")
	sb.WriteString(fmt.Sprintf("Total resources: %d\n\n", result.Stats.TotalResources))

	sb.WriteString("## Resources by Type\n\n")
	sb.WriteString("| Type | Count |\n")
	sb.WriteString("|------|-------|\n")

	resourceTypes := w.getSortedResourceTypes(result)
	for _, rt := range resourceTypes {
		count := result.Stats.ResourceCounts[rt]
		if count > 0 {
			displayName := string(rt)
			if info, ok := w.ResourcesByType[rt]; ok {
				displayName = info.DisplayName
			}
			sb.WriteString(fmt.Sprintf("| %s | %d |\n", displayName, count))
		}
	}

	sb.WriteString("\n## Directory Structure\n\n")
	sb.WriteString("```\n")
	sb.WriteString(w.OutputDir + "/\n")

	// Show kind-config.yaml if present
	if result.KindConfig != "" {
		sb.WriteString("├── kind-config.yaml\n")
	}

	// List directories
	dirs := make(map[string]bool)
	for _, rt := range resourceTypes {
		if len(result.Resources[rt]) > 0 {
			subdir := w.getSubdirForType(rt)
			dirs[subdir] = true
		}
	}

	var sortedDirs []string
	for d := range dirs {
		sortedDirs = append(sortedDirs, d)
	}
	sort.Strings(sortedDirs)

	for _, d := range sortedDirs {
		sb.WriteString(fmt.Sprintf("├── %s/\n", d))
	}
	sb.WriteString("└── README.md\n")
	sb.WriteString("```\n")

	// Add Kind config section if present
	if result.KindConfig != "" {
		sb.WriteString("\n## Kind Cluster Configuration\n\n")
		sb.WriteString("The `kind-config.yaml` file contains the Kind cluster configuration derived from `kindplane.yaml`.\n\n")
		sb.WriteString("To recreate the cluster using Kind directly:\n\n")
		sb.WriteString("```bash\n")
		sb.WriteString("kind create cluster --config kind-config.yaml\n")
		sb.WriteString("```\n")
	}

	// Add discovered XRDs if any
	if len(result.DiscoveredXRDs) > 0 {
		sb.WriteString("\n## Discovered Composite Resource Definitions\n\n")
		sb.WriteString("| Name | Group | Kind |\n")
		sb.WriteString("|------|-------|------|\n")
		for _, xrd := range result.DiscoveredXRDs {
			sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", xrd.Name, xrd.Group, xrd.Kind))
		}
	}

	// Add errors if any
	if len(result.Errors) > 0 {
		sb.WriteString("\n## Warnings\n\n")
		sb.WriteString("The following non-fatal errors occurred during dump:\n\n")
		for _, err := range result.Errors {
			sb.WriteString(fmt.Sprintf("- %s\n", err.Error()))
		}
	}

	sb.WriteString("\n## Usage\n\n")
	sb.WriteString("To apply these resources to a cluster:\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("kubectl apply -R -f %s/\n", w.OutputDir))
	sb.WriteString("```\n")

	sb.WriteString("\n**Note:** Secrets contain redacted data and must be replaced with actual values before applying.\n")

	return os.WriteFile(readmePath, []byte(sb.String()), 0644)
}

// sanitizeFilename removes or replaces characters not suitable for filenames
func sanitizeFilename(name string) string {
	// Replace problematic characters
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "-",
		"\"", "-",
		"<", "-",
		">", "-",
		"|", "-",
	)
	return replacer.Replace(name)
}

// WriteDryRunReport writes a report of what would be dumped
func WriteDryRunReport(w io.Writer, resources []ResourceInfo) {
	_, _ = fmt.Fprintln(w, "Dry run - the following resource types would be dumped:")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "| Type | Category | Namespaced |")
	_, _ = fmt.Fprintln(w, "|------|----------|------------|")

	for _, r := range resources {
		namespaced := "No"
		if r.Namespaced {
			namespaced = "Yes"
		}
		_, _ = fmt.Fprintf(w, "| %s | %s | %s |\n", r.DisplayName, r.Category, namespaced)
	}
	_, _ = fmt.Fprintln(w, "")
}

// WriteStats writes dump statistics to a writer
func WriteStats(w io.Writer, stats DumpStats) {
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintf(w, "Total resources dumped: %d\n", stats.TotalResources)
	if stats.SkippedResources > 0 {
		_, _ = fmt.Fprintf(w, "Resources skipped: %d\n", stats.SkippedResources)
	}
	if stats.NamespacesScanned > 0 {
		_, _ = fmt.Fprintf(w, "Namespaces scanned: %d\n", stats.NamespacesScanned)
	}
}
