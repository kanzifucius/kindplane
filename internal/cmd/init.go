package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/kanzi/kindplane/internal/config"
)

var (
	initForce   bool
	initMinimal bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new kindplane configuration file",
	Long: `Initialize a new kindplane.yaml configuration file with sensible defaults.

This command creates a configuration file that defines your Kind cluster setup,
Crossplane installation, providers, and other components.

Examples:
  # Create default config file
  kindplane init

  # Overwrite existing config file
  kindplane init --force

  # Create minimal config without comments
  kindplane init --minimal`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "overwrite existing configuration file")
	initCmd.Flags().BoolVar(&initMinimal, "minimal", false, "generate minimal config without comments")
}

func runInit(cmd *cobra.Command, args []string) error {
	configPath := cfgFile
	if configPath == "" {
		configPath = config.DefaultConfigFile
	}

	// Check if config already exists
	if config.Exists(configPath) && !initForce {
		printError("Configuration file already exists: %s", configPath)
		printStep("Use --force to overwrite")
		return nil
	}

	// Write the config file
	var err error
	if initMinimal {
		// Generate minimal config without comments
		cfg := config.DefaultConfig()
		err = cfg.Save(configPath)
	} else {
		// Generate config with comments
		content := config.DefaultConfigWithComments()
		err = os.WriteFile(configPath, []byte(content), 0644)
	}

	if err != nil {
		printError("Failed to create configuration file: %v", err)
		return err
	}

	printSuccess("Created %s", configPath)
	printStep("")
	printStep("Next steps:")
	printStep("  1. Edit %s to customize your setup", configPath)
	printStep("  2. Run 'kindplane up' to create your cluster")

	return nil
}
