package ui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// -----------------------------------------------------------------------------
// Table Configuration
// -----------------------------------------------------------------------------

// TableOption configures the table
type TableOption func(*tableConfig)

type tableConfig struct {
	width       int
	height      int
	focused     bool
	interactive bool
	output      io.Writer
}

// WithTableWidth sets the table width
func WithTableWidth(w int) TableOption {
	return func(c *tableConfig) {
		c.width = w
	}
}

// WithTableHeight sets the table height (for scrolling)
func WithTableHeight(h int) TableOption {
	return func(c *tableConfig) {
		c.height = h
	}
}

// WithTableFocused sets whether the table is focused
func WithTableFocused(f bool) TableOption {
	return func(c *tableConfig) {
		c.focused = f
	}
}

// WithTableInteractive enables keyboard navigation
func WithTableInteractive(i bool) TableOption {
	return func(c *tableConfig) {
		c.interactive = i
	}
}

// WithTableOutput sets the output writer for the table.
// If not set, defaults to the package's default output (usually os.Stdout).
func WithTableOutput(w io.Writer) TableOption {
	return func(c *tableConfig) {
		c.output = w
	}
}

func (c *tableConfig) getOutput() io.Writer {
	if c.output != nil {
		return c.output
	}
	return defaultOutput
}

// -----------------------------------------------------------------------------
// Table Validation
// -----------------------------------------------------------------------------

// TableValidationError is returned when table data is invalid
type TableValidationError struct {
	Message string
}

func (e TableValidationError) Error() string {
	return e.Message
}

// ValidateTableData checks that all rows have the same number of columns as headers.
// Returns an error if validation fails, nil otherwise.
func ValidateTableData(headers []string, rows [][]string) error {
	if len(headers) == 0 {
		return TableValidationError{Message: "table must have at least one header"}
	}

	expectedCols := len(headers)
	for i, row := range rows {
		if len(row) != expectedCols {
			return TableValidationError{
				Message: fmt.Sprintf("row %d has %d columns, expected %d (matching headers)", i, len(row), expectedCols),
			}
		}
	}

	return nil
}

// -----------------------------------------------------------------------------
// Table Creation
// -----------------------------------------------------------------------------

// NewBubblesTable creates a styled table using bubbles/table.
// Returns the table model which can be rendered with .View()
func NewBubblesTable(headers []string, rows [][]string, opts ...TableOption) table.Model {
	cfg := &tableConfig{
		width:   80,
		height:  10,
		focused: false,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	// Calculate column widths based on content
	colWidths := make([]int, len(headers))
	for i, h := range headers {
		colWidths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// Add padding and ensure minimum width
	for i := range colWidths {
		colWidths[i] += 2
		if colWidths[i] < 8 {
			colWidths[i] = 8
		}
	}

	// Create columns
	columns := make([]table.Column, len(headers))
	for i, h := range headers {
		columns[i] = table.Column{
			Title: h,
			Width: colWidths[i],
		}
	}

	// Create rows
	tableRows := make([]table.Row, len(rows))
	for i, row := range rows {
		tableRows[i] = row
	}

	// Create table
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(tableRows),
		table.WithFocused(cfg.focused),
		table.WithHeight(cfg.height),
	)

	// Style the table
	t.SetStyles(defaultTableStyles())

	return t
}

// RenderTable renders a simple static table (non-interactive).
// This is a convenience function that creates a table and returns its view.
func RenderTable(headers []string, rows [][]string, opts ...TableOption) string {
	t := NewBubblesTable(headers, rows, opts...)
	return t.View()
}

// defaultTableStyles returns the standard table styling used by NewBubblesTable
func defaultTableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ColorBorder).
		BorderBottom(true).
		Bold(true).
		Foreground(ColorSecondary)

	s.Selected = s.Selected.
		Foreground(ColorText).
		Background(lipgloss.AdaptiveColor{Light: "#E0E7FF", Dark: "#312E81"}).
		Bold(false)

	s.Cell = s.Cell.
		Foreground(ColorText)

	return s
}

// -----------------------------------------------------------------------------
// Interactive Table (for selection)
// -----------------------------------------------------------------------------

// tableModel is a Bubble Tea model for interactive table selection
type tableModel struct {
	table    table.Model
	selected int
	done     bool
}

func (m tableModel) Init() tea.Cmd {
	return nil
}

func (m tableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.done = true
			m.selected = -1
			return m, tea.Quit
		case "enter":
			m.done = true
			m.selected = m.table.Cursor()
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m tableModel) View() string {
	if m.done {
		return ""
	}
	return m.table.View() + "\n" + StyleMuted.Render("↑/↓: navigate • enter: select • q: quit")
}

// RunTableSelect shows an interactive table and returns the selected row index.
// Returns -1 if cancelled.
//
// In non-TTY mode, this just prints the table and returns the first row (or -1 if empty):
//   - Prints a notice about non-interactive mode
//   - Renders the table as static text
//   - Returns index 0 if there are rows, -1 otherwise
//
// Options:
//   - WithTableOutput(w): Set custom output writer (default: package default output)
//   - WithTableFocused(f): Set whether the table is focused (default: false, set to true for selection)
//   - WithTableHeight(h): Set visible rows (default: 10)
//   - WithTableWidth(w): Set table width (default: 80)
func RunTableSelect(headers []string, rows [][]string, opts ...TableOption) (int, error) {
	cfg := &tableConfig{
		width:   80,
		height:  10,
		focused: false,
		output:  nil,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	output := cfg.getOutput()

	if !IsTTY() {
		// Fallback: just print the table, return first row
		printNonTTYNoticeTo(output)
		fmt.Fprintln(output, RenderTable(headers, rows, opts...))
		if len(rows) > 0 {
			return 0, nil
		}
		return -1, nil
	}

	opts = append(opts, WithTableFocused(true))
	t := NewBubblesTable(headers, rows, opts...)

	m := tableModel{
		table:    t,
		selected: -1,
	}

	p := tea.NewProgram(m, tea.WithOutput(output))
	finalModel, err := p.Run()
	if err != nil {
		return -1, err
	}

	final := finalModel.(tableModel)
	return final.selected, nil
}

// -----------------------------------------------------------------------------
// Status Table Builder (shared between pod and provider tables)
// -----------------------------------------------------------------------------

// StatusTableConfig configures a status table
type StatusTableConfig struct {
	Columns []StatusColumn
	Rows    []StatusRow
	Height  int // Optional, defaults to len(Rows)
}

// StatusColumn defines a table column
type StatusColumn struct {
	Title    string
	MinWidth int
}

// StatusRow represents a row in a status table
type StatusRow struct {
	Cells []string
}

// BuildStatusTable creates a consistently-styled table for status display.
// This is used by both pod and provider tables to ensure consistent appearance.
//
// Note: This function does not validate row/column consistency. Use ValidateStatusTableConfig
// if you need validation before rendering.
func BuildStatusTable(cfg StatusTableConfig) table.Model {
	// Calculate column widths based on content
	colWidths := make([]int, len(cfg.Columns))
	for i, col := range cfg.Columns {
		colWidths[i] = max(len(col.Title), col.MinWidth)
	}
	for _, row := range cfg.Rows {
		for i, cell := range row.Cells {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// Add padding
	for i := range colWidths {
		colWidths[i] += 2
	}

	// Create columns
	columns := make([]table.Column, len(cfg.Columns))
	for i, col := range cfg.Columns {
		columns[i] = table.Column{
			Title: col.Title,
			Width: colWidths[i],
		}
	}

	// Create rows
	tableRows := make([]table.Row, len(cfg.Rows))
	for i, row := range cfg.Rows {
		tableRows[i] = row.Cells
	}

	// Determine height
	height := cfg.Height
	if height == 0 {
		height = len(cfg.Rows)
	}

	// Create table
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(tableRows),
		table.WithFocused(false),
		table.WithHeight(height),
	)

	// Apply consistent styles
	t.SetStyles(statusTableStyles())

	return t
}

// statusTableStyles returns the standard styling for status tables
func statusTableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	s.Cell = s.Cell.
		Foreground(ColorText)
	return s
}

// statusTableBaseStyle defines the border style for status tables
var statusTableBaseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

// RenderStatusTable renders a status table with borders
func RenderStatusTable(cfg StatusTableConfig) string {
	t := BuildStatusTable(cfg)
	return statusTableBaseStyle.Render(t.View())
}

// ValidateStatusTableConfig validates that all rows have the same number of cells
// as there are columns defined. Returns an error if validation fails.
func ValidateStatusTableConfig(cfg StatusTableConfig) error {
	if len(cfg.Columns) == 0 {
		return TableValidationError{Message: "status table must have at least one column"}
	}

	expectedCols := len(cfg.Columns)
	for i, row := range cfg.Rows {
		if len(row.Cells) != expectedCols {
			return TableValidationError{
				Message: fmt.Sprintf("row %d has %d cells, expected %d (matching columns)", i, len(row.Cells), expectedCols),
			}
		}
	}

	return nil
}
