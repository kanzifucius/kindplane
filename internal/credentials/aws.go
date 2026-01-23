package credentials

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"gopkg.in/ini.v1"
)

const (
	// AWSSecretName is the name of the AWS credentials secret
	AWSSecretName = "aws-creds"
	// AWSProviderConfigName is the name of the AWS ProviderConfig
	AWSProviderConfigName = "default"
)

// ConfigureAWSFromEnv creates AWS credentials from environment variables
func (m *Manager) ConfigureAWSFromEnv(ctx context.Context) error {
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	sessionToken := os.Getenv("AWS_SESSION_TOKEN")

	if accessKey == "" || secretKey == "" {
		return fmt.Errorf("AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables are required")
	}

	return m.createAWSCredentials(ctx, accessKey, secretKey, sessionToken)
}

// ConfigureAWSFromProfile creates AWS credentials from an AWS CLI profile
func (m *Manager) ConfigureAWSFromProfile(ctx context.Context, profile string) error {
	// Find credentials file
	credentialsPath := os.Getenv("AWS_SHARED_CREDENTIALS_FILE")
	if credentialsPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		credentialsPath = filepath.Join(home, ".aws", "credentials")
	}

	// Load credentials file
	cfg, err := ini.Load(credentialsPath)
	if err != nil {
		return fmt.Errorf("failed to load AWS credentials file: %w", err)
	}

	section, err := cfg.GetSection(profile)
	if err != nil {
		return fmt.Errorf("profile '%s' not found in credentials file", profile)
	}

	accessKey := section.Key("aws_access_key_id").String()
	secretKey := section.Key("aws_secret_access_key").String()
	sessionToken := section.Key("aws_session_token").String()

	if accessKey == "" || secretKey == "" {
		return fmt.Errorf("profile '%s' does not contain aws_access_key_id and aws_secret_access_key", profile)
	}

	return m.createAWSCredentials(ctx, accessKey, secretKey, sessionToken)
}

// ConfigureAWSManual creates AWS credentials from provided values
func (m *Manager) ConfigureAWSManual(ctx context.Context, accessKey, secretKey string) error {
	return m.createAWSCredentials(ctx, accessKey, secretKey, "")
}

// createAWSCredentials creates the AWS credentials secret and ProviderConfig
func (m *Manager) createAWSCredentials(ctx context.Context, accessKey, secretKey, sessionToken string) error {
	// Create credentials content in AWS credentials file format
	credsContent := fmt.Sprintf("[default]\naws_access_key_id = %s\naws_secret_access_key = %s", accessKey, secretKey)
	if sessionToken != "" {
		credsContent += fmt.Sprintf("\naws_session_token = %s", sessionToken)
	}

	// Create secret
	if err := m.createSecret(ctx, AWSSecretName, CrossplaneSystemNamespace, map[string][]byte{
		"creds": []byte(credsContent),
	}); err != nil {
		return fmt.Errorf("failed to create AWS credentials secret: %w", err)
	}

	// Create ProviderConfig
	if err := m.createAWSProviderConfig(ctx); err != nil {
		return fmt.Errorf("failed to create AWS ProviderConfig: %w", err)
	}

	return nil
}

// createAWSProviderConfig creates the AWS ProviderConfig resource
func (m *Manager) createAWSProviderConfig(ctx context.Context) error {
	providerConfig := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "aws.upbound.io/v1beta1",
			"kind":       "ProviderConfig",
			"metadata": map[string]interface{}{
				"name": AWSProviderConfigName,
			},
			"spec": map[string]interface{}{
				"credentials": map[string]interface{}{
					"source": "Secret",
					"secretRef": map[string]interface{}{
						"namespace": CrossplaneSystemNamespace,
						"name":      AWSSecretName,
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
		Group:    "aws.upbound.io",
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
