package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/kanzi/kindplane/internal/config"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the configuration file",
	Long: `Validate the kindplane.yaml configuration file for errors.

This command checks the configuration file for syntax errors, missing required
fields, invalid values, and other issues before running 'kindplane up'.

The validation includes:
  - YAML syntax validation
  - Required field checks (cluster.name, crossplane.version, etc.)
  - Value constraints (port ranges, enum values, etc.)
  - File existence checks (rawConfigPath, valuesFiles)`,
	Example: `  # Validate default config file
  kindplane validate

  # Validate a specific config file
  kindplane validate --config custom.yaml`,
	RunE: runValidate,
}

func runValidate(cmd *cobra.Command, args []string) error {
	configPath := cfgFile
	if configPath == "" {
		configPath = config.DefaultConfigFile
	}

	// Check if config file exists
	if !config.Exists(configPath) {
		printError("Configuration file not found: %s", configPath)
		printStep("Run 'kindplane init' to create a configuration file")
		os.Exit(1)
	}

	// Load and validate the config
	_, err := config.Load(configPath)
	if err != nil {
		printError("Configuration validation failed: %s", configPath)
		printStep("")
		printStep("%v", err)
		os.Exit(1)
	}

	printSuccess("Configuration is valid: %s", configPath)
	return nil
}
