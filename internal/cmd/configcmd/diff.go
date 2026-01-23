package configcmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kanzi/kindplane/internal/config"
	"github.com/kanzi/kindplane/internal/ui"
)

var (
	diffFormat string
)

var diffCmd = &cobra.Command{
	Use:   "diff <file1> [file2]",
	Short: "Compare two configuration files",
	Long: `Compare two kindplane configuration files and show the differences.

If only one file is provided, it compares against the current configuration.`,
	Example: `  # Compare current config with another file
  kindplane config diff ./other-kindplane.yaml

  # Compare two configuration files
  kindplane config diff ./config1.yaml ./config2.yaml`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runDiff,
}

func init() {
	diffCmd.Flags().StringVar(&diffFormat, "format", "unified", "Diff format (unified, side-by-side)")
}

func runDiff(cmd *cobra.Command, args []string) error {
	var file1, file2 string

	if len(args) == 1 {
		// Compare with current config
		file1 = config.DefaultConfigFile
		file2 = args[0]
	} else {
		file1 = args[0]
		file2 = args[1]
	}

	// Read both files
	content1, err := os.ReadFile(file1)
	if err != nil {
		fmt.Println(ui.Error("Failed to read %s: %v", file1, err))
		return err
	}

	content2, err := os.ReadFile(file2)
	if err != nil {
		fmt.Println(ui.Error("Failed to read %s: %v", file2, err))
		return err
	}

	fmt.Println()
	fmt.Println(ui.Title(ui.IconMagnifier + " Configuration Diff"))
	fmt.Println(ui.Divider())
	fmt.Println()

	lines1 := splitLines(string(content1))
	lines2 := splitLines(string(content2))

	// Simple line-by-line diff
	switch diffFormat {
	case "unified":
		printUnifiedDiff(file1, file2, lines1, lines2)
	case "side-by-side":
		printSideBySideDiff(file1, file2, lines1, lines2)
	default:
		fmt.Println(ui.Error("Unknown format: %s. Use 'unified' or 'side-by-side'.", diffFormat))
		return fmt.Errorf("unknown format: %s", diffFormat)
	}

	return nil
}

func splitLines(s string) []string {
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

func printUnifiedDiff(file1, file2 string, lines1, lines2 []string) {
	// Header
	fmt.Println(ui.StyleInfo.Render("--- " + file1))
	fmt.Println(ui.StyleInfo.Render("+++ " + file2))
	fmt.Println()

	// Simple diff using longest common subsequence concept
	i, j := 0, 0
	for i < len(lines1) || j < len(lines2) {
		if i >= len(lines1) {
			// Remaining lines in file2 are additions
			fmt.Println(ui.StyleSuccess.Render("+ " + lines2[j]))
			j++
		} else if j >= len(lines2) {
			// Remaining lines in file1 are deletions
			fmt.Println(ui.StyleError.Render("- " + lines1[i]))
			i++
		} else if lines1[i] == lines2[j] {
			// Lines match
			fmt.Printf("  %s\n", lines1[i])
			i++
			j++
		} else {
			// Lines differ - look ahead to find match
			matchI := findMatch(lines1, i, lines2[j], 5)
			matchJ := findMatch(lines2, j, lines1[i], 5)

			if matchI >= 0 && (matchJ < 0 || matchI-i <= matchJ-j) {
				// Delete lines from file1 until we reach the match
				for k := i; k < matchI; k++ {
					fmt.Println(ui.StyleError.Render("- " + lines1[k]))
				}
				i = matchI
			} else if matchJ >= 0 {
				// Add lines from file2 until we reach the match
				for k := j; k < matchJ; k++ {
					fmt.Println(ui.StyleSuccess.Render("+ " + lines2[k]))
				}
				j = matchJ
			} else {
				// No match found nearby, show as change
				fmt.Println(ui.StyleError.Render("- " + lines1[i]))
				fmt.Println(ui.StyleSuccess.Render("+ " + lines2[j]))
				i++
				j++
			}
		}
	}
}

func findMatch(lines []string, start int, target string, maxLookahead int) int {
	end := start + maxLookahead
	if end > len(lines) {
		end = len(lines)
	}
	for k := start; k < end; k++ {
		if lines[k] == target {
			return k
		}
	}
	return -1
}

func printSideBySideDiff(file1, file2 string, lines1, lines2 []string) {
	// Calculate column width
	width := 40

	// Header
	fmt.Printf("%-*s | %s\n", width, file1, file2)
	fmt.Println(ui.StyleMuted.Render(strings.Repeat("-", width) + "-+-" + strings.Repeat("-", width)))

	maxLines := len(lines1)
	if len(lines2) > maxLines {
		maxLines = len(lines2)
	}

	for i := 0; i < maxLines; i++ {
		var left, right string
		if i < len(lines1) {
			left = truncate(lines1[i], width)
		}
		if i < len(lines2) {
			right = truncate(lines2[i], width)
		}

		// Determine if lines are different
		marker := " "
		if i < len(lines1) && i < len(lines2) {
			if lines1[i] != lines2[i] {
				marker = "*"
				left = ui.StyleWarning.Render(fmt.Sprintf("%-*s", width, left))
				right = ui.StyleWarning.Render(right)
			} else {
				left = fmt.Sprintf("%-*s", width, left)
			}
		} else if i < len(lines1) {
			marker = "<"
			left = ui.StyleError.Render(fmt.Sprintf("%-*s", width, left))
		} else {
			marker = ">"
			right = ui.StyleSuccess.Render(right)
			left = fmt.Sprintf("%-*s", width, "")
		}

		fmt.Printf("%s %s%s\n", left, marker, right)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
