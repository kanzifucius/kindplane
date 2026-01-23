package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Banner renders a styled banner/header for the application
func Banner() string {
	banner := `
   _    _           _       _                  
  | | _(_)_ __   __| |_ __ | | __ _ _ __   ___ 
  | |/ / | '_ \ / _' | '_ \| |/ _' | '_ \ / _ \
  |   <| | | | | (_| | |_) | | (_| | | | |  __/
  |_|\_\_|_| |_|\__,_| .__/|_|\__,_|_| |_|\___|
                     |_|                       `

	style := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true)

	return style.Render(banner)
}

// SmallBanner renders a compact banner
func SmallBanner() string {
	style := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true)

	return style.Render("kindplane") + StyleMuted.Render(" - Bootstrap Kind clusters with Crossplane")
}

// Header renders a section header
func Header(title string) string {
	return StyleSectionHeader.Render(title)
}

// Title renders a main title
func Title(title string) string {
	return StyleTitle.Render(title)
}

// Subtitle renders a subtitle
func Subtitle(text string) string {
	return StyleSubtitle.Render(text)
}

// Success renders a success message
func Success(format string, a ...interface{}) string {
	msg := fmt.Sprintf(format, a...)
	return StyleSuccess.Render(IconSuccess+" ") + msg
}

// Error renders an error message
func Error(format string, a ...interface{}) string {
	msg := fmt.Sprintf(format, a...)
	return StyleError.Render(IconError+" ") + msg
}

// Warning renders a warning message
func Warning(format string, a ...interface{}) string {
	msg := fmt.Sprintf(format, a...)
	return StyleWarning.Render(IconWarning+" ") + msg
}

// Info renders an info message
func Info(format string, a ...interface{}) string {
	msg := fmt.Sprintf(format, a...)
	return StyleInfo.Render(IconInfo+" ") + msg
}

// Step renders a step/progress item
func Step(format string, a ...interface{}) string {
	msg := fmt.Sprintf(format, a...)
	return StyleIndent1.Render(StyleMuted.Render(IconBullet+" ") + msg)
}

// Muted renders muted/dim text
func Muted(format string, a ...interface{}) string {
	msg := fmt.Sprintf(format, a...)
	return StyleMuted.Render(msg)
}

// Code renders inline code
func Code(text string) string {
	return StyleCode.Render(text)
}

// Box renders content in a bordered box
func Box(title, content string) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		MarginBottom(1)

	box := StyleBox.Copy()
	if title != "" {
		return box.Render(titleStyle.Render(title) + "\n" + content)
	}
	return box.Render(content)
}

// SuccessBox renders content in a success-styled box
func SuccessBox(title, content string) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorSuccess).
		MarginBottom(1)

	box := StyleBoxSuccess.Copy()
	if title != "" {
		return box.Render(titleStyle.Render(IconSuccess+" "+title) + "\n" + content)
	}
	return box.Render(content)
}

// ErrorBox renders content in an error-styled box
func ErrorBox(title, content string) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorError).
		MarginBottom(1)

	box := StyleBoxError.Copy()
	if title != "" {
		return box.Render(titleStyle.Render(IconError+" "+title) + "\n" + content)
	}
	return box.Render(content)
}

// WarningBox renders content in a warning-styled box
func WarningBox(title, content string) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorWarning).
		MarginBottom(1)

	box := StyleBoxWarning.Copy()
	if title != "" {
		return box.Render(titleStyle.Render(IconWarning+" "+title) + "\n" + content)
	}
	return box.Render(content)
}

// InfoBox renders content in an info-styled box
func InfoBox(title, content string) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorInfo).
		MarginBottom(1)

	box := StyleBoxInfo.Copy()
	if title != "" {
		return box.Render(titleStyle.Render(IconInfo+" "+title) + "\n" + content)
	}
	return box.Render(content)
}

// DiagnosticBox renders a diagnostic report box
func DiagnosticBox(content string) string {
	header := StyleDiagnosticHeader.Render(" DIAGNOSTICS ")
	box := StyleDiagnosticBox.Render(content)
	return header + "\n" + box
}

// Table represents a simple table
type Table struct {
	Headers []string
	Rows    [][]string
	Width   int
}

// NewTable creates a new table
func NewTable(headers ...string) *Table {
	return &Table{
		Headers: headers,
		Rows:    [][]string{},
	}
}

// AddRow adds a row to the table
func (t *Table) AddRow(cells ...string) *Table {
	t.Rows = append(t.Rows, cells)
	return t
}

// SetWidth sets the table width
func (t *Table) SetWidth(width int) *Table {
	t.Width = width
	return t
}

// Render renders the table
func (t *Table) Render() string {
	if len(t.Headers) == 0 && len(t.Rows) == 0 {
		return ""
	}

	// Calculate column widths
	colWidths := make([]int, len(t.Headers))
	for i, h := range t.Headers {
		colWidths[i] = len(h)
	}
	for _, row := range t.Rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// Add padding
	for i := range colWidths {
		colWidths[i] += 2
	}

	var sb strings.Builder

	// Render header
	if len(t.Headers) > 0 {
		headerStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary)

		var headerCells []string
		for i, h := range t.Headers {
			cellStyle := lipgloss.NewStyle().Width(colWidths[i])
			headerCells = append(headerCells, cellStyle.Render(h))
		}
		sb.WriteString(headerStyle.Render(strings.Join(headerCells, " ")))
		sb.WriteString("\n")

		// Separator
		var sepParts []string
		for _, w := range colWidths {
			sepParts = append(sepParts, strings.Repeat("─", w))
		}
		sb.WriteString(StyleMuted.Render(strings.Join(sepParts, " ")))
		sb.WriteString("\n")
	}

	// Render rows
	for _, row := range t.Rows {
		var cells []string
		for i, cell := range row {
			width := 10
			if i < len(colWidths) {
				width = colWidths[i]
			}
			cellStyle := lipgloss.NewStyle().Width(width)
			cells = append(cells, cellStyle.Render(cell))
		}
		sb.WriteString(strings.Join(cells, " "))
		sb.WriteString("\n")
	}

	return sb.String()
}

// KeyValue renders a key-value pair
func KeyValue(key, value string) string {
	keyStyle := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Width(16)
	return keyStyle.Render(key+":") + " " + value
}

// KeyValueIndented renders an indented key-value pair
func KeyValueIndented(key, value string, indent int) string {
	padding := strings.Repeat(" ", indent)
	keyStyle := lipgloss.NewStyle().
		Foreground(ColorMuted)
	return padding + keyStyle.Render(key+":") + " " + value
}

// List renders a bullet list
func List(items ...string) string {
	var lines []string
	for _, item := range items {
		lines = append(lines, StyleListItem.Render(StyleMuted.Render(IconBullet)+" "+item))
	}
	return strings.Join(lines, "\n")
}

// NumberedList renders a numbered list
func NumberedList(items ...string) string {
	var lines []string
	for i, item := range items {
		num := lipgloss.NewStyle().
			Foreground(ColorMuted).
			Width(3).
			Align(lipgloss.Right).
			Render(fmt.Sprintf("%d.", i+1))
		lines = append(lines, StyleListItem.Render(num+" "+item))
	}
	return strings.Join(lines, "\n")
}

// TreeItem represents an item in a tree view
type TreeItem struct {
	Label    string
	Children []TreeItem
}

// Tree renders a tree structure
func Tree(items []TreeItem) string {
	return renderTree(items, "", true)
}

func renderTree(items []TreeItem, prefix string, isRoot bool) string {
	var sb strings.Builder

	for i, item := range items {
		isLast := i == len(items)-1
		connector := IconTee
		if isLast {
			connector = IconCorner
		}

		if !isRoot {
			sb.WriteString(prefix)
			sb.WriteString(StyleMuted.Render(connector + IconDash + " "))
		}
		sb.WriteString(item.Label)
		sb.WriteString("\n")

		if len(item.Children) > 0 {
			childPrefix := prefix
			if !isRoot {
				if isLast {
					childPrefix += "    "
				} else {
					childPrefix += StyleMuted.Render(IconPipe) + "   "
				}
			}
			sb.WriteString(renderTree(item.Children, childPrefix, false))
		}
	}

	return sb.String()
}

// ProgressBar renders a simple progress bar
func ProgressBar(current, total int, width int) string {
	if total == 0 {
		return ""
	}

	percentage := float64(current) / float64(total)
	filled := int(percentage * float64(width))
	if filled > width {
		filled = width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	barStyle := lipgloss.NewStyle().Foreground(ColorPrimary)
	percentStyle := lipgloss.NewStyle().Foreground(ColorMuted).Width(6).Align(lipgloss.Right)

	return barStyle.Render(bar) + " " + percentStyle.Render(fmt.Sprintf("%d%%", int(percentage*100)))
}

// Spinner characters for animation
var SpinnerFrames = []string{"◐", "◓", "◑", "◒"}

// Divider renders a horizontal divider
func Divider() string {
	return StyleMuted.Render(strings.Repeat("─", 60))
}

// DividerWithText renders a divider with centered text
func DividerWithText(text string) string {
	textLen := len(text) + 2 // Add padding
	totalWidth := 60
	if textLen >= totalWidth {
		return StyleMuted.Render(" " + text + " ")
	}

	leftWidth := (totalWidth - textLen) / 2
	rightWidth := totalWidth - textLen - leftWidth

	left := strings.Repeat("─", leftWidth)
	right := strings.Repeat("─", rightWidth)

	return StyleMuted.Render(left) + " " + StyleBold.Render(text) + " " + StyleMuted.Render(right)
}

// StatusLine renders a status line with icon, label, and value
func StatusLine(status, label, value string) string {
	icon := StatusIcon(status)
	style := StatusStyle(status)

	iconPart := style.Render(icon)
	labelPart := lipgloss.NewStyle().Width(20).Render(label + ":")
	valuePart := value

	return iconPart + " " + labelPart + " " + valuePart
}

// CompactStatus renders a compact status indicator
func CompactStatus(status, label string) string {
	icon := StatusIcon(status)
	style := StatusStyle(status)
	return style.Render(icon) + " " + label
}

// Indent returns an indented string
func Indent(text string, level int) string {
	padding := strings.Repeat("  ", level)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = padding + line
	}
	return strings.Join(lines, "\n")
}

// TruncateWithEllipsis truncates text and adds ellipsis if needed
func TruncateWithEllipsis(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	if maxLen <= 3 {
		return text[:maxLen]
	}
	return text[:maxLen-3] + "..."
}

// WrapText wraps text to a specified width
func WrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	var currentLine strings.Builder

	words := strings.Fields(text)
	for i, word := range words {
		if currentLine.Len()+len(word)+1 > width && currentLine.Len() > 0 {
			result.WriteString(currentLine.String())
			result.WriteString("\n")
			currentLine.Reset()
		}

		if currentLine.Len() > 0 {
			currentLine.WriteString(" ")
		}
		currentLine.WriteString(word)

		if i == len(words)-1 {
			result.WriteString(currentLine.String())
		}
	}

	return result.String()
}
