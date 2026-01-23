package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kanzi/kindplane/internal/cmd/chart"
	"github.com/kanzi/kindplane/internal/cmd/credentials"
	"github.com/kanzi/kindplane/internal/cmd/provider"
	"github.com/kanzi/kindplane/internal/config"
	"github.com/kanzi/kindplane/internal/ui"
)

var (
	// Version info set at build time
	version   = "dev"
	commit    = "none"
	buildTime = "unknown"

	// Global flags
	cfgFile string
	verbose bool

	// Global config
	cfg *config.Config
)

// SetVersionInfo sets the version information from build flags
func SetVersionInfo(v, c, b string) {
	version = v
	commit = c
	buildTime = b
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kindplane",
	Short: "Bootstrap Kind clusters with Crossplane",
	Long: `kindplane is a CLI tool that helps developers quickly spin up
Kind (Kubernetes in Docker) clusters pre-configured with Crossplane,
cloud providers, and other essential components.

It automates the tedious process of setting up a local Kubernetes
development environment with Crossplane for infrastructure management.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is ./kindplane.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Add subcommands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(dumpCmd)
	rootCmd.AddCommand(provider.ProviderCmd)
	rootCmd.AddCommand(chart.ChartCmd)
	rootCmd.AddCommand(credentials.CredentialsCmd)
	rootCmd.AddCommand(versionCmd)
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
		printError(err.Error())
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

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(ui.Title("kindplane"))
		fmt.Println()
		fmt.Println(ui.KeyValue("Version", version))
		fmt.Println(ui.KeyValue("Commit", commit))
		fmt.Println(ui.KeyValue("Built", buildTime))
	},
}
