package dump

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Cleaner handles cleanup of Kubernetes resources for GitOps-friendly output
type Cleaner struct {
	// StripAnnotations is the list of annotations to remove
	StripAnnotations []string

	// StripLabels is the list of labels to remove
	StripLabels []string

	// StripMetadataFields is the list of metadata fields to remove
	StripMetadataFields []string

	// StripStatus removes the status field entirely
	StripStatus bool

	// RedactSecrets replaces secret data with placeholders
	RedactSecrets bool

	// SecretRedactionPlaceholder is the placeholder text for redacted secrets
	SecretRedactionPlaceholder string

	// PreserveLabels is a list of label prefixes to preserve even if in StripLabels
	PreserveLabels []string

	// PreserveAnnotations is a list of annotation prefixes to preserve
	PreserveAnnotations []string
}

// NewCleaner creates a Cleaner with default settings
func NewCleaner() *Cleaner {
	return &Cleaner{
		StripAnnotations:           AnnotationsToStrip(),
		StripLabels:                LabelsToStrip(),
		StripMetadataFields:        MetadataFieldsToStrip(),
		StripStatus:                true,
		RedactSecrets:              true,
		SecretRedactionPlaceholder: "<REDACTED>",
		PreserveLabels:             []string{},
		PreserveAnnotations:        []string{},
	}
}

// Clean processes an unstructured object and returns a cleaned copy
func (c *Cleaner) Clean(obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	// Deep copy to avoid modifying the original
	cleaned := obj.DeepCopy()

	// Clean metadata
	if err := c.cleanMetadata(cleaned); err != nil {
		return nil, fmt.Errorf("cleaning metadata: %w", err)
	}

	// Strip status if configured
	if c.StripStatus {
		unstructured.RemoveNestedField(cleaned.Object, "status")
	}

	// Handle secrets specially
	if c.RedactSecrets && cleaned.GetKind() == "Secret" {
		if err := c.redactSecretData(cleaned); err != nil {
			return nil, fmt.Errorf("redacting secret data: %w", err)
		}
	}

	return cleaned, nil
}

// cleanMetadata removes cluster-specific metadata fields
func (c *Cleaner) cleanMetadata(obj *unstructured.Unstructured) error {
	metadata, found, err := unstructured.NestedMap(obj.Object, "metadata")
	if err != nil {
		return fmt.Errorf("getting metadata: %w", err)
	}
	if !found {
		return nil
	}

	// Remove specified metadata fields
	for _, field := range c.StripMetadataFields {
		delete(metadata, field)
	}

	// Clean annotations
	if annotations, ok := metadata["annotations"].(map[string]interface{}); ok {
		c.cleanAnnotations(annotations)
		if len(annotations) == 0 {
			delete(metadata, "annotations")
		} else {
			metadata["annotations"] = annotations
		}
	}

	// Clean labels
	if labels, ok := metadata["labels"].(map[string]interface{}); ok {
		c.cleanLabels(labels)
		if len(labels) == 0 {
			delete(metadata, "labels")
		} else {
			metadata["labels"] = labels
		}
	}

	// Remove ownerReferences (these are cluster-specific)
	delete(metadata, "ownerReferences")

	// Remove finalizers if empty
	if finalizers, ok := metadata["finalizers"].([]interface{}); ok && len(finalizers) == 0 {
		delete(metadata, "finalizers")
	}

	return unstructured.SetNestedMap(obj.Object, metadata, "metadata")
}

// cleanAnnotations removes unwanted annotations
func (c *Cleaner) cleanAnnotations(annotations map[string]interface{}) {
	for key := range annotations {
		if c.shouldStripAnnotation(key) {
			delete(annotations, key)
		}
	}
}

// shouldStripAnnotation checks if an annotation should be removed
func (c *Cleaner) shouldStripAnnotation(key string) bool {
	// Check preserve list first
	for _, prefix := range c.PreserveAnnotations {
		if strings.HasPrefix(key, prefix) {
			return false
		}
	}

	// Check strip list
	for _, stripKey := range c.StripAnnotations {
		if key == stripKey {
			return true
		}
	}

	return false
}

// cleanLabels removes unwanted labels
func (c *Cleaner) cleanLabels(labels map[string]interface{}) {
	for key := range labels {
		if c.shouldStripLabel(key) {
			delete(labels, key)
		}
	}
}

// shouldStripLabel checks if a label should be removed
func (c *Cleaner) shouldStripLabel(key string) bool {
	// Check preserve list first
	for _, prefix := range c.PreserveLabels {
		if strings.HasPrefix(key, prefix) {
			return false
		}
	}

	// Check strip list
	for _, stripKey := range c.StripLabels {
		if key == stripKey {
			return true
		}
	}

	return false
}

// redactSecretData replaces secret data with placeholders
func (c *Cleaner) redactSecretData(obj *unstructured.Unstructured) error {
	// Handle data field
	data, found, err := unstructured.NestedMap(obj.Object, "data")
	if err != nil {
		return fmt.Errorf("getting data: %w", err)
	}
	if found && len(data) > 0 {
		redacted := make(map[string]interface{})
		for key := range data {
			redacted[key] = c.SecretRedactionPlaceholder
		}
		if err := unstructured.SetNestedMap(obj.Object, redacted, "data"); err != nil {
			return fmt.Errorf("setting redacted data: %w", err)
		}
	}

	// Handle stringData field
	stringData, found, err := unstructured.NestedMap(obj.Object, "stringData")
	if err != nil {
		return fmt.Errorf("getting stringData: %w", err)
	}
	if found && len(stringData) > 0 {
		redacted := make(map[string]interface{})
		for key := range stringData {
			redacted[key] = c.SecretRedactionPlaceholder
		}
		if err := unstructured.SetNestedMap(obj.Object, redacted, "stringData"); err != nil {
			return fmt.Errorf("setting redacted stringData: %w", err)
		}
	}

	return nil
}

// CleanBatch processes multiple objects and returns cleaned copies
func (c *Cleaner) CleanBatch(objects []*unstructured.Unstructured) ([]*unstructured.Unstructured, error) {
	cleaned := make([]*unstructured.Unstructured, 0, len(objects))
	for _, obj := range objects {
		cleanedObj, err := c.Clean(obj)
		if err != nil {
			return nil, fmt.Errorf("cleaning %s/%s: %w", obj.GetKind(), obj.GetName(), err)
		}
		cleaned = append(cleaned, cleanedObj)
	}
	return cleaned, nil
}

// ShouldSkipNamespace checks if a namespace should be skipped
func ShouldSkipNamespace(namespace string) bool {
	// Check exact matches
	for _, ns := range SystemNamespaces() {
		if namespace == ns {
			return true
		}
	}

	// Check patterns
	for _, pattern := range SystemNamespacePatterns() {
		if strings.HasPrefix(namespace, pattern) {
			return true
		}
	}

	return false
}

// ShouldSkipResource checks if a resource should be skipped based on common criteria
func ShouldSkipResource(obj *unstructured.Unstructured) bool {
	// Skip resources owned by Helm
	annotations := obj.GetAnnotations()
	if annotations != nil {
		if _, ok := annotations["meta.helm.sh/release-name"]; ok {
			// This is a Helm-managed resource, might want to skip
			// For now, we'll include it but this could be configurable
			return false
		}
	}

	// Skip service account token secrets
	if obj.GetKind() == "Secret" {
		secretType, found, _ := unstructured.NestedString(obj.Object, "type")
		if found && secretType == "kubernetes.io/service-account-token" {
			return true
		}
	}

	return false
}

// IsSystemResource checks if a resource is a system resource that should typically be excluded
func IsSystemResource(obj *unstructured.Unstructured) bool {
	// Check namespace
	ns := obj.GetNamespace()
	if ns != "" && ShouldSkipNamespace(ns) {
		return true
	}

	// Check for system namespaces (for Namespace resources)
	if obj.GetKind() == "Namespace" {
		if ShouldSkipNamespace(obj.GetName()) {
			return true
		}
	}

	return false
}

// CleanerOption is a functional option for configuring a Cleaner
type CleanerOption func(*Cleaner)

// WithStripStatus configures whether to strip status fields
func WithStripStatus(strip bool) CleanerOption {
	return func(c *Cleaner) {
		c.StripStatus = strip
	}
}

// WithRedactSecrets configures whether to redact secret data
func WithRedactSecrets(redact bool) CleanerOption {
	return func(c *Cleaner) {
		c.RedactSecrets = redact
	}
}

// WithSecretPlaceholder sets the placeholder for redacted secrets
func WithSecretPlaceholder(placeholder string) CleanerOption {
	return func(c *Cleaner) {
		c.SecretRedactionPlaceholder = placeholder
	}
}

// WithPreserveAnnotations sets annotation prefixes to preserve
func WithPreserveAnnotations(prefixes []string) CleanerOption {
	return func(c *Cleaner) {
		c.PreserveAnnotations = prefixes
	}
}

// WithPreserveLabels sets label prefixes to preserve
func WithPreserveLabels(prefixes []string) CleanerOption {
	return func(c *Cleaner) {
		c.PreserveLabels = prefixes
	}
}

// WithAdditionalStripAnnotations adds more annotations to strip
func WithAdditionalStripAnnotations(annotations []string) CleanerOption {
	return func(c *Cleaner) {
		c.StripAnnotations = append(c.StripAnnotations, annotations...)
	}
}

// NewCleanerWithOptions creates a Cleaner with the given options
func NewCleanerWithOptions(opts ...CleanerOption) *Cleaner {
	c := NewCleaner()
	for _, opt := range opts {
		opt(c)
	}
	return c
}
