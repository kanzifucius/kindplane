package credentials

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// AzureSecretName is the name of the Azure credentials secret
	AzureSecretName = "azure-creds"
	// AzureProviderConfigName is the name of the Azure ProviderConfig
	AzureProviderConfigName = "default"
)

// AzureCredentials represents Azure service principal credentials
type AzureCredentials struct {
	ClientID       string `json:"clientId"`
	ClientSecret   string `json:"clientSecret"`
	TenantID       string `json:"tenantId"`
	SubscriptionID string `json:"subscriptionId"`
}

// ConfigureAzureFromEnv creates Azure credentials from environment variables
func (m *Manager) ConfigureAzureFromEnv(ctx context.Context) error {
	clientID := os.Getenv("AZURE_CLIENT_ID")
	clientSecret := os.Getenv("AZURE_CLIENT_SECRET")
	tenantID := os.Getenv("AZURE_TENANT_ID")
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")

	if clientID == "" || clientSecret == "" || tenantID == "" || subscriptionID == "" {
		return fmt.Errorf("AZURE_CLIENT_ID, AZURE_CLIENT_SECRET, AZURE_TENANT_ID, and AZURE_SUBSCRIPTION_ID environment variables are required")
	}

	return m.createAzureCredentials(ctx, tenantID, clientID, clientSecret, subscriptionID)
}

// ConfigureAzureManual creates Azure credentials from provided values
func (m *Manager) ConfigureAzureManual(ctx context.Context, tenantID, clientID, clientSecret, subscriptionID string) error {
	return m.createAzureCredentials(ctx, tenantID, clientID, clientSecret, subscriptionID)
}

// createAzureCredentials creates the Azure credentials secret and ProviderConfig
func (m *Manager) createAzureCredentials(ctx context.Context, tenantID, clientID, clientSecret, subscriptionID string) error {
	creds := AzureCredentials{
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		TenantID:       tenantID,
		SubscriptionID: subscriptionID,
	}

	credsJSON, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal Azure credentials: %w", err)
	}

	// Create secret
	if err := m.createSecret(ctx, AzureSecretName, CrossplaneSystemNamespace, map[string][]byte{
		"creds": credsJSON,
	}); err != nil {
		return fmt.Errorf("failed to create Azure credentials secret: %w", err)
	}

	// Create ProviderConfig
	if err := m.createAzureProviderConfig(ctx); err != nil {
		return fmt.Errorf("failed to create Azure ProviderConfig: %w", err)
	}

	return nil
}

// createAzureProviderConfig creates the Azure ProviderConfig resource
func (m *Manager) createAzureProviderConfig(ctx context.Context) error {
	providerConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "azure.upbound.io/v1beta1",
			"kind":       "ProviderConfig",
			"metadata": map[string]interface{}{
				"name": AzureProviderConfigName,
			},
			"spec": map[string]interface{}{
				"credentials": map[string]interface{}{
					"source": "Secret",
					"secretRef": map[string]interface{}{
						"namespace": CrossplaneSystemNamespace,
						"name":      AzureSecretName,
						"key":       "creds",
					},
				},
			},
		},
	}

	dynamicClient, err := m.getDynamicClient()
	if err != nil {
		return err
	}

	gvr := schema.GroupVersionResource{
		Group:    "azure.upbound.io",
		Version:  "v1beta1",
		Resource: "providerconfigs",
	}

	_, err = dynamicClient.Resource(gvr).Create(ctx, providerConfig, metav1.CreateOptions{})
	if err != nil {
		// Try update if create fails
		_, err = dynamicClient.Resource(gvr).Update(ctx, providerConfig, metav1.UpdateOptions{})
	}

	return err
}
