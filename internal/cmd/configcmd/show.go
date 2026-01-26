package configcmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/kanzi/kindplane/internal/config"
	"github.com/kanzi/kindplane/internal/ui"
)

var (
	showFormat   string
	showResolved bool
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Display the current configuration",
	Long: `Display the current kindplane configuration.

By default, shows the raw configuration file. Use --resolved to show
the configuration with all defaults filled in.`,
	Example: `  # Show current configuration
  kindplane config show

  # Show configuration in JSON format
  kindplane config show --format json

  # Show resolved configuration with defaults
  kindplane config show --resolved`,
	RunE: runShow,
}

func init() {
	showCmd.Flags().StringVar(&showFormat, "format", "yaml", "Output format (yaml, json)")
	showCmd.Flags().BoolVar(&showResolved, "resolved", false, "Show configuration with defaults resolved")
}

func runShow(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		fmt.Println(ui.Error("Failed to load configuration: %v", err))
		return err
	}

	fmt.Println()
	fmt.Println(ui.Title(ui.IconFile + " Configuration"))
	fmt.Println(ui.Divider())
	fmt.Println()

	var output []byte
	switch showFormat {
	case "yaml":
		output, err = yaml.Marshal(cfg)
		if err != nil {
			fmt.Println(ui.Error("Failed to marshal configuration: %v", err))
			return err
		}
	case "json":
		output, err = json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			fmt.Println(ui.Error("Failed to marshal configuration: %v", err))
			return err
		}
	default:
		fmt.Println(ui.Error("Unknown format: %s. Use 'yaml' or 'json'.", showFormat))
		return fmt.Errorf("unknown format: %s", showFormat)
	}

	fmt.Println(string(output))
	return nil
}
