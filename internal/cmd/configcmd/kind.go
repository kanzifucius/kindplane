package configcmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kanzi/kindplane/internal/config"
	"github.com/kanzi/kindplane/internal/kind"
	"github.com/kanzi/kindplane/internal/ui"
)

var (
	kindOutput string
)

var kindCmd = &cobra.Command{
	Use:   "kind",
	Short: "Output the Kind cluster configuration",
	Long: `Generate and output the Kind cluster configuration derived from kindplane.yaml.

This allows you to use the configuration with the kind CLI directly:

  kindplane config kind > kind-config.yaml
  kind create cluster --config kind-config.yaml

The output includes all settings from kindplane.yaml translated to Kind's
configuration format, including node images, port mappings, mounts, and
containerd patches for registries.`,
	Example: `  # Output to stdout
  kindplane config kind

  # Save to file
  kindplane config kind -o kind-config.yaml

  # Use directly with kind CLI (stdin)
  kindplane config kind | kind create cluster --config -`,
	RunE: runKindConfig,
}

func init() {
	kindCmd.Flags().StringVarP(&kindOutput, "output", "o", "", "Write output to file instead of stdout")
}

func runKindConfig(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		fmt.Println(ui.Error("Failed to load configuration: %v", err))
		return err
	}

	// Build Kind config
	kindConfig, err := kind.BuildKindConfig(cfg)
	if err != nil {
		fmt.Println(ui.Error("Failed to build Kind configuration: %v", err))
		return err
	}

	// Output to file or stdout
	if kindOutput != "" {
		if err := os.WriteFile(kindOutput, []byte(kindConfig), 0644); err != nil {
			fmt.Println(ui.Error("Failed to write file: %v", err))
			return err
		}
		fmt.Println(ui.Success("Kind configuration written to %s", kindOutput))
	} else {
		fmt.Print(kindConfig)
	}

	return nil
}
