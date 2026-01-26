// Package main provides a command to generate CLI documentation.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra/doc"

	cmd "github.com/kanzi/kindplane/internal/cmd"
)

func main() {
	// Default output directory
	outputDir := "./docs/cli-reference"
	if len(os.Args) > 1 {
		outputDir = os.Args[1]
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Disable auto-generation notices to keep docs clean
	cmd.RootCmd.DisableAutoGenTag = true

	// Generate markdown documentation
	// Using custom file prepender and link handler for mkdocs compatibility
	err := doc.GenMarkdownTreeCustom(cmd.RootCmd, outputDir, filePrepender, linkHandler)
	if err != nil {
		log.Fatalf("Failed to generate documentation: %v", err)
	}

	// Generate an index file for the CLI reference
	if err := generateIndex(outputDir); err != nil {
		log.Fatalf("Failed to generate index: %v", err)
	}

	fmt.Printf("Documentation generated in %s\n", outputDir)
}

// filePrepender adds frontmatter or headers to generated files
func filePrepender(filename string) string {
	// Currently returns empty string - frontmatter is handled by mkdocs
	_ = filename // unused but required by interface
	return ""
}

// linkHandler converts links to be mkdocs compatible
func linkHandler(name string) string {
	// Convert underscores to the actual filename format
	base := strings.TrimSuffix(name, ".md")
	return base + ".md"
}

// generateIndex creates an index.md file for the CLI reference section
func generateIndex(outputDir string) error {
	content := `# CLI Reference

This section contains automatically generated documentation for all kindplane commands.

!!! info "Auto-generated"
    This documentation is automatically generated from the CLI source code to ensure accuracy.

## Commands

| Command | Description |
|---------|-------------|
| [kindplane](kindplane.md) | Root command and global flags |
| [kindplane init](kindplane_init.md) | Initialise a new kindplane configuration |
| [kindplane validate](kindplane_validate.md) | Validate configuration file |
| [kindplane up](kindplane_up.md) | Create and bootstrap a cluster |
| [kindplane down](kindplane_down.md) | Delete a cluster |
| [kindplane status](kindplane_status.md) | Show cluster status |
| [kindplane dump](kindplane_dump.md) | Export cluster resources |
| [kindplane provider](kindplane_provider.md) | Manage Crossplane providers |
| [kindplane chart](kindplane_chart.md) | Manage Helm charts |
| [kindplane credentials](kindplane_credentials.md) | Manage cloud credentials |

## Global Flags

All commands support these global flags:

| Flag | Description |
|------|-------------|
| ` + "`-c, --config`" + ` | Configuration file (default: ` + "`kindplane.yaml`" + `) |
| ` + "`-V, --verbose`" + ` | Enable verbose output |
| ` + "`-h, --help`" + ` | Show help for any command |

## Getting Help

Use ` + "`--help`" + ` with any command to see available options:

` + "```bash" + `
kindplane --help
kindplane up --help
kindplane provider add --help
` + "```" + `
`

	indexPath := filepath.Join(outputDir, "index.md")
	return os.WriteFile(indexPath, []byte(content), 0644)
}
