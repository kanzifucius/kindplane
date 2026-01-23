package credentials

import (
	"context"
	"fmt"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kanzi/kindplane/internal/config"
	"github.com/kanzi/kindplane/internal/credentials"
	"github.com/kanzi/kindplane/internal/kind"
)

var (
	configureProvider string
	configureTimeout  time.Duration
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure credentials for cloud providers",
	Long: `Interactively configure credentials for Crossplane cloud providers.

This command will guide you through setting up credentials for the
selected cloud provider and create the necessary ProviderConfig
resources in the cluster.

Supported providers:
  - aws        AWS credentials
  - azure      Azure credentials
  - kubernetes Kubernetes provider credentials

Examples:
  # Configure AWS credentials interactively
  kindplane credentials configure --provider aws

  # Configure all providers
  kindplane credentials configure`,
	RunE: runConfigure,
}

func init() {
	configureCmd.Flags().StringVarP(&configureProvider, "provider", "p", "", "specific provider to configure (aws, azure, kubernetes)")
	configureCmd.Flags().DurationVar(&configureTimeout, "timeout", 5*time.Minute, "timeout for configuration")
}

func runConfigure(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load("")
	if err != nil {
		color.Red("✗ %v", err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), configureTimeout)
	defer cancel()

	// Check cluster exists
	exists, err := kind.ClusterExists(cfg.Cluster.Name)
	if err != nil {
		color.Red("✗ Failed to check cluster: %v", err)
		return err
	}
	if !exists {
		color.Red("✗ Cluster '%s' not found. Run 'kindplane up' first.", cfg.Cluster.Name)
		return fmt.Errorf("cluster not found")
	}

	// Get kube client
	kubeClient, err := kind.GetKubeClient(cfg.Cluster.Name)
	if err != nil {
		color.Red("✗ Failed to connect to cluster: %v", err)
		return err
	}

	credManager := credentials.NewManager(kubeClient)

	// Determine which providers to configure
	var providers []string
	if configureProvider != "" {
		providers = []string{configureProvider}
	} else {
		// Let user select providers
		providerOptions := []string{"aws", "azure", "kubernetes"}
		prompt := &survey.MultiSelect{
			Message: "Select providers to configure:",
			Options: providerOptions,
		}
		if err := survey.AskOne(prompt, &providers); err != nil {
			return err
		}
	}

	if len(providers) == 0 {
		color.Yellow("! No providers selected")
		return nil
	}

	for _, provider := range providers {
		fmt.Println()
		color.Cyan("Configuring %s credentials...", provider)

		switch provider {
		case "aws":
			if err := configureAWS(ctx, credManager); err != nil {
				color.Red("✗ Failed to configure AWS: %v", err)
				return err
			}
		case "azure":
			if err := configureAzure(ctx, credManager); err != nil {
				color.Red("✗ Failed to configure Azure: %v", err)
				return err
			}
		case "kubernetes":
			if err := configureKubernetes(ctx, credManager); err != nil {
				color.Red("✗ Failed to configure Kubernetes: %v", err)
				return err
			}
		default:
			color.Yellow("! Unknown provider: %s", provider)
		}
	}

	fmt.Println()
	color.Green("✓ Credentials configured successfully")
	return nil
}

func configureAWS(ctx context.Context, manager *credentials.Manager) error {
	var sourceChoice string
	sourcePrompt := &survey.Select{
		Message: "AWS credential source:",
		Options: []string{"Environment variables", "AWS CLI profile", "Enter manually"},
	}
	if err := survey.AskOne(sourcePrompt, &sourceChoice); err != nil {
		return err
	}

	switch sourceChoice {
	case "Environment variables":
		return manager.ConfigureAWSFromEnv(ctx)
	case "AWS CLI profile":
		var profile string
		profilePrompt := &survey.Input{
			Message: "AWS profile name:",
			Default: "default",
		}
		if err := survey.AskOne(profilePrompt, &profile); err != nil {
			return err
		}
		return manager.ConfigureAWSFromProfile(ctx, profile)
	case "Enter manually":
		var accessKey, secretKey string
		accessKeyPrompt := &survey.Input{Message: "AWS Access Key ID:"}
		secretKeyPrompt := &survey.Password{Message: "AWS Secret Access Key:"}

		if err := survey.AskOne(accessKeyPrompt, &accessKey); err != nil {
			return err
		}
		if err := survey.AskOne(secretKeyPrompt, &secretKey); err != nil {
			return err
		}

		return manager.ConfigureAWSManual(ctx, accessKey, secretKey)
	}

	return nil
}

func configureAzure(ctx context.Context, manager *credentials.Manager) error {
	var sourceChoice string
	sourcePrompt := &survey.Select{
		Message: "Azure credential source:",
		Options: []string{"Environment variables", "Enter manually"},
	}
	if err := survey.AskOne(sourcePrompt, &sourceChoice); err != nil {
		return err
	}

	switch sourceChoice {
	case "Environment variables":
		return manager.ConfigureAzureFromEnv(ctx)
	case "Enter manually":
		var tenantID, clientID, clientSecret, subscriptionID string
		questions := []*survey.Question{
			{Name: "tenantID", Prompt: &survey.Input{Message: "Azure Tenant ID:"}},
			{Name: "clientID", Prompt: &survey.Input{Message: "Azure Client ID:"}},
			{Name: "clientSecret", Prompt: &survey.Password{Message: "Azure Client Secret:"}},
			{Name: "subscriptionID", Prompt: &survey.Input{Message: "Azure Subscription ID:"}},
		}
		answers := struct {
			TenantID       string
			ClientID       string
			ClientSecret   string
			SubscriptionID string
		}{}
		if err := survey.Ask(questions, &answers); err != nil {
			return err
		}

		return manager.ConfigureAzureManual(ctx, tenantID, clientID, clientSecret, subscriptionID)
	}

	return nil
}

func configureKubernetes(ctx context.Context, manager *credentials.Manager) error {
	var sourceChoice string
	sourcePrompt := &survey.Select{
		Message: "Kubernetes credential source:",
		Options: []string{"In-cluster (ServiceAccount)", "Kubeconfig file"},
	}
	if err := survey.AskOne(sourcePrompt, &sourceChoice); err != nil {
		return err
	}

	switch sourceChoice {
	case "In-cluster (ServiceAccount)":
		return manager.ConfigureKubernetesInCluster(ctx)
	case "Kubeconfig file":
		return manager.ConfigureKubernetesFromKubeconfig(ctx)
	}

	return nil
}
