package cmd

import (
	"fmt"
	"os"
	"sync"

	"github.com/spf13/cobra"

	"github.com/kanzi/kindplane/internal/cmd/chart"
	"github.com/kanzi/kindplane/internal/cmd/cluster"
	"github.com/kanzi/kindplane/internal/cmd/configcmd"
	"github.com/kanzi/kindplane/internal/cmd/credentials"
	"github.com/kanzi/kindplane/internal/cmd/provider"
	"github.com/kanzi/kindplane/internal/config"
	"github.com/kanzi/kindplane/internal/ui"
	"github.com/kanzi/kindplane/internal/version"
)

var (
	// Version is set by main.go from build-time ldflags
	Version = "dev"

	// Global flags
	cfgFile string
	verbose bool

	// Global config
	cfg *config.Config

	// versionCheckOnce ensures version check only runs once
	versionCheckOnce sync.Once
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "kindplane",
	Short: "Bootstrap Kind clusters with Crossplane",
	Long: `kindplane is a CLI tool that helps developers quickly spin up
Kind (Kubernetes in Docker) clusters pre-configured with Crossplane,
cloud providers, and other essential components.

It automates the tedious process of setting up a local Kubernetes
development environment with Crossplane for infrastructure management.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Run version check asynchronously (only once per CLI invocation)
		versionCheckOnce.Do(func() {
			go checkForUpdates()
		})
	},
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	// Note: Using -V for verbose to avoid conflict with fang's -v/--version
	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is ./kindplane.yaml)")
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "V", false, "verbose output")

	// Add subcommands
	RootCmd.AddCommand(initCmd)
	RootCmd.AddCommand(validateCmd)
	RootCmd.AddCommand(upCmd)
	RootCmd.AddCommand(downCmd)
	RootCmd.AddCommand(statusCmd)
	RootCmd.AddCommand(dumpCmd)
	RootCmd.AddCommand(diagnosticsCmd)
	RootCmd.AddCommand(logsCmd)
	RootCmd.AddCommand(applyCmd)
	RootCmd.AddCommand(doctorCmd)
	RootCmd.AddCommand(cluster.ClusterCmd)
	RootCmd.AddCommand(configcmd.ConfigCmd)
	RootCmd.AddCommand(provider.ProviderCmd)
	RootCmd.AddCommand(chart.ChartCmd)
	RootCmd.AddCommand(credentials.CredentialsCmd)
}

// initConfig reads in config file if set
func initConfig() {
	// Config is loaded on-demand by commands that need it
}

// loadConfig loads the configuration file, returns error if not found
func loadConfig() error {
	path := cfgFile
	if path == "" {
		path = config.DefaultConfigFile
	}

	var err error
	cfg, err = config.Load(path)
	if err != nil {
		return err
	}
	return nil
}

// requireConfig ensures config is loaded, exits if not found
func requireConfig() {
	if err := loadConfig(); err != nil {
		printError("failed to load config: %v", err)
		os.Exit(1)
	}
}

// Helper print functions using the new UI package
func printSuccess(format string, a ...interface{}) {
	fmt.Println(ui.Success(format, a...))
}

func printError(format string, a ...interface{}) {
	fmt.Println(ui.Error(format, a...))
}

func printWarn(format string, a ...interface{}) {
	fmt.Println(ui.Warning(format, a...))
}

func printInfo(format string, a ...interface{}) {
	fmt.Println(ui.Info(format, a...))
}

func printStep(format string, a ...interface{}) {
	fmt.Println(ui.Step(format, a...))
}

func printVerbose(format string, a ...interface{}) {
	if verbose {
		fmt.Println(ui.Muted("[debug] "+format, a...))
	}
}

// checkForUpdates checks for a newer version and prints a warning if available
func checkForUpdates() {
	// Skip version check for dev builds
	if Version == "dev" || Version == "none" || Version == "" {
		return
	}

	var result *version.CheckResult

	// Try to use cached result first
	if !version.ShouldCheck() {
		result = version.GetCachedResult(Version)
	}

	// If no valid cache, fetch from GitHub
	if result == nil {
		var err error
		result, err = version.CheckForUpdate(Version)
		if err != nil {
			// Fail silently - network issues shouldn't affect CLI usage
			return
		}

		// Cache the result for next time
		_ = version.SaveCache(result.LatestVersion, result.ReleaseURL)
	}

	// Show warning if update is available
	if result != nil && result.UpdateAvailable {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, ui.Warning("A new version of kindplane is available: %s → %s", Version, result.LatestVersion))
		fmt.Fprintln(os.Stderr, ui.Muted("  Update with: curl -fsSL https://raw.githubusercontent.com/kanzifucius/kindplane/main/install.sh | bash"))
		fmt.Fprintln(os.Stderr, "")
	}
}
