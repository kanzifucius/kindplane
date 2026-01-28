//nolint:staticcheck // Using fake.NewSimpleClientset which is deprecated but still works
package helm

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

// TestEnsureNamespace_CreatesNewNamespace tests that EnsureNamespace creates a namespace when it doesn't exist
func TestEnsureNamespace_CreatesNewNamespace(t *testing.T) {
	ctx := context.Background()
	kubeClient := fake.NewSimpleClientset()
	namespace := "test-namespace"

	// Verify namespace doesn't exist
	_, err := kubeClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if !apierrors.IsNotFound(err) {
		t.Fatalf("Expected namespace to not exist, but got: %v", err)
	}

	// Call EnsureNamespace
	err = EnsureNamespace(ctx, kubeClient, namespace)
	if err != nil {
		t.Fatalf("EnsureNamespace failed: %v", err)
	}

	// Verify namespace was created
	ns, err := kubeClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get namespace after EnsureNamespace: %v", err)
	}

	if ns.Name != namespace {
		t.Errorf("Expected namespace name %s, got %s", namespace, ns.Name)
	}
}

// TestEnsureNamespace_ExistingNamespace tests that EnsureNamespace succeeds when namespace already exists
func TestEnsureNamespace_ExistingNamespace(t *testing.T) {
	ctx := context.Background()
	namespace := "existing-namespace"

	// Create namespace beforehand
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	kubeClient := fake.NewSimpleClientset(ns)

	// Verify namespace exists
	_, err := kubeClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Namespace should exist: %v", err)
	}

	// Call EnsureNamespace - should succeed without error
	err = EnsureNamespace(ctx, kubeClient, namespace)
	if err != nil {
		t.Fatalf("EnsureNamespace should succeed for existing namespace: %v", err)
	}
}

// TestEnsureNamespace_GetError tests error handling when Get fails (not NotFound)
func TestEnsureNamespace_GetError(t *testing.T) {
	ctx := context.Background()
	kubeClient := fake.NewSimpleClientset()
	namespace := "test-namespace"

	// Add a reactor that returns an error for namespace get operations
	kubeClient.PrependReactor("get", "namespaces", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, apierrors.NewServiceUnavailable("API server unavailable")
	})

	// Call EnsureNamespace - should return error
	err := EnsureNamespace(ctx, kubeClient, namespace)
	if err == nil {
		t.Fatal("EnsureNamespace should fail when Get returns error")
	}

	// Verify error message contains expected context
	if !strings.Contains(err.Error(), "failed to check namespace") {
		t.Errorf("Error message should contain 'failed to check namespace', got: %v", err)
	}
}

// TestEnsureNamespace_CreateError tests error handling when Create fails
func TestEnsureNamespace_CreateError(t *testing.T) {
	ctx := context.Background()
	kubeClient := fake.NewSimpleClientset()
	namespace := "test-namespace"

	// Add a reactor that returns an error for namespace create operations
	kubeClient.PrependReactor("create", "namespaces", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, apierrors.NewForbidden(corev1.Resource("namespaces"), namespace, nil)
	})

	// Call EnsureNamespace - should return error
	err := EnsureNamespace(ctx, kubeClient, namespace)
	if err == nil {
		t.Fatal("EnsureNamespace should fail when Create returns error")
	}

	// Verify error message contains expected context
	if !strings.Contains(err.Error(), "failed to create namespace") {
		t.Errorf("Error message should contain 'failed to create namespace', got: %v", err)
	}
}

// TestEnsureNamespace_MultipleNamespaces tests creating multiple namespaces
func TestEnsureNamespace_MultipleNamespaces(t *testing.T) {
	ctx := context.Background()
	kubeClient := fake.NewSimpleClientset()

	namespaces := []string{"namespace-1", "namespace-2", "namespace-3"}

	for _, ns := range namespaces {
		err := EnsureNamespace(ctx, kubeClient, ns)
		if err != nil {
			t.Fatalf("EnsureNamespace failed for %s: %v", ns, err)
		}
	}

	// Verify all namespaces were created
	for _, ns := range namespaces {
		_, err := kubeClient.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
		if err != nil {
			t.Errorf("Expected namespace %s to exist: %v", ns, err)
		}
	}
}

// TestEnsureNamespace_Idempotency tests that calling EnsureNamespace multiple times is safe
func TestEnsureNamespace_Idempotency(t *testing.T) {
	ctx := context.Background()
	kubeClient := fake.NewSimpleClientset()
	namespace := "idempotent-namespace"

	// Call EnsureNamespace multiple times
	for i := 0; i < 3; i++ {
		err := EnsureNamespace(ctx, kubeClient, namespace)
		if err != nil {
			t.Fatalf("EnsureNamespace call %d failed: %v", i+1, err)
		}
	}

	// Verify namespace exists exactly once
	nsList, err := kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatalf("Failed to list namespaces: %v", err)
	}

	count := 0
	for _, ns := range nsList.Items {
		if ns.Name == namespace {
			count++
		}
	}

	if count != 1 {
		t.Errorf("Expected exactly 1 namespace with name %s, found %d", namespace, count)
	}
}
