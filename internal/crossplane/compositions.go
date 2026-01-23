package crossplane

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"

	"github.com/kanzi/kindplane/internal/config"
	"github.com/kanzi/kindplane/internal/git"
)

// ApplyCompositions applies compositions from the specified source
func (i *Installer) ApplyCompositions(ctx context.Context, source config.CompositionSource) error {
	var basePath string

	switch source.Type {
	case "local":
		basePath = source.Path
	case "git":
		// Clone the repository
		clonePath, err := git.CloneRepo(ctx, source.Repo, source.Branch)
		if err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
		basePath = filepath.Join(clonePath, source.Path)
	default:
		return fmt.Errorf("unknown source type: %s", source.Type)
	}

	// Check if path exists
	info, err := os.Stat(basePath)
	if err != nil {
		return fmt.Errorf("failed to access path: %w", err)
	}

	if !info.IsDir() {
		// Single file
		return i.applyFile(ctx, basePath)
	}

	// Walk directory and apply all YAML files
	return filepath.WalkDir(basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		return i.applyFile(ctx, path)
	})
}

// applyFile applies a single YAML file to the cluster
func (i *Installer) applyFile(ctx context.Context, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", path, err)
	}

	// Split multi-document YAML
	docs := strings.Split(string(data), "\n---")

	dynamicClient, err := i.getDynamicClient()
	if err != nil {
		return fmt.Errorf("failed to get dynamic client: %w", err)
	}

	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" || doc == "---" {
			continue
		}

		// Parse document
		obj := &unstructured.Unstructured{}
		if err := yaml.Unmarshal([]byte(doc), &obj.Object); err != nil {
			return fmt.Errorf("failed to parse YAML in %s: %w", path, err)
		}

		if obj.Object == nil {
			continue
		}

		// Apply object
		if err := i.applyObject(ctx, dynamicClient, obj); err != nil {
			return fmt.Errorf("failed to apply object from %s: %w", path, err)
		}
	}

	return nil
}

// applyObject applies a single Kubernetes object
func (i *Installer) applyObject(ctx context.Context, dynamicClient dynamic.Interface, obj *unstructured.Unstructured) error {
	gvk := obj.GroupVersionKind()
	gvr := schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: pluralize(gvk.Kind),
	}

	namespace := obj.GetNamespace()

	var err error
	if namespace != "" {
		// Namespaced resource
		_, err = dynamicClient.Resource(gvr).Namespace(namespace).Create(ctx, obj, metav1.CreateOptions{})
		if err != nil {
			// Try update if create fails
			_, err = dynamicClient.Resource(gvr).Namespace(namespace).Update(ctx, obj, metav1.UpdateOptions{})
		}
	} else {
		// Cluster-scoped resource
		_, err = dynamicClient.Resource(gvr).Create(ctx, obj, metav1.CreateOptions{})
		if err != nil {
			// Try update if create fails
			_, err = dynamicClient.Resource(gvr).Update(ctx, obj, metav1.UpdateOptions{})
		}
	}

	return err
}

// pluralize converts a Kind to its plural resource name
func pluralize(kind string) string {
	kind = strings.ToLower(kind)

	// Common irregular plurals
	irregulars := map[string]string{
		"compositeresourcedefinition": "compositeresourcedefinitions",
		"composition":                 "compositions",
		"provider":                    "providers",
		"providerconfig":              "providerconfigs",
		"configmap":                   "configmaps",
		"secret":                      "secrets",
		"namespace":                   "namespaces",
		"serviceaccount":              "serviceaccounts",
		"clusterrole":                 "clusterroles",
		"clusterrolebinding":          "clusterrolebindings",
		"role":                        "roles",
		"rolebinding":                 "rolebindings",
	}

	if plural, ok := irregulars[kind]; ok {
		return plural
	}

	// Default pluralization rules
	switch {
	case strings.HasSuffix(kind, "s"):
		return kind + "es"
	case strings.HasSuffix(kind, "y"):
		return kind[:len(kind)-1] + "ies"
	default:
		return kind + "s"
	}
}
