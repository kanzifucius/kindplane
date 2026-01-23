package ui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// ErrCancelled is returned when the user cancels an operation
var ErrCancelled = errors.New("operation cancelled by user")

// -----------------------------------------------------------------------------
// TTY Detection
// -----------------------------------------------------------------------------

// IsTTY returns true if stdout is a terminal
func IsTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// nonTTYNoticeOnce ensures the fallback notice is only printed once
var nonTTYNoticeOnce sync.Once

// printNonTTYNotice prints a one-time notice that interactive UI is disabled
func printNonTTYNotice() {
	nonTTYNoticeOnce.Do(func() {
		fmt.Println(StyleMuted.Render("(non-interactive terminal detected, using static output)"))
	})
}

// -----------------------------------------------------------------------------
// Spinner Component
// -----------------------------------------------------------------------------

// spinnerState holds shared state that persists across model copies
type spinnerState struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type spinnerModel struct {
	spinner   spinner.Model
	title     string
	err       error
	done      bool
	cancelled bool
	fn        func(ctx context.Context) error
	state     *spinnerState
	started   bool
}

type spinnerDoneMsg struct {
	err error
}

func (m spinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			// Cancel the background work
			if m.state != nil && m.state.cancel != nil {
				m.state.cancel()
			}
			m.done = true
			m.cancelled = true
			m.err = ErrCancelled
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)

		// Start the background work on first tick
		if !m.started {
			m.started = true
			return m, tea.Batch(cmd, m.runWork())
		}
		return m, cmd

	case spinnerDoneMsg:
		m.err = msg.err
		m.done = true
		return m, tea.Quit
	}

	return m, nil
}

func (m spinnerModel) runWork() tea.Cmd {
	return func() tea.Msg {
		err := m.fn(m.state.ctx)
		return spinnerDoneMsg{err: err}
	}
}

func (m spinnerModel) View() string {
	if m.done {
		return ""
	}
	return fmt.Sprintf("%s %s", m.spinner.View(), m.title)
}

// RunSpinner shows an animated spinner while executing a function.
// The function receives a context that is cancelled if the user presses ctrl+c or q.
// Falls back to static output if not running in a TTY.
func RunSpinner(title string, fn func() error) error {
	return RunSpinnerWithContext(context.Background(), title, func(ctx context.Context) error {
		return fn()
	})
}

// RunSpinnerWithContext shows an animated spinner while executing a context-aware function.
// The context is cancelled if the user presses ctrl+c or q.
// Falls back to static output if not running in a TTY.
func RunSpinnerWithContext(parentCtx context.Context, title string, fn func(ctx context.Context) error) error {
	if !IsTTY() {
		// Fallback for non-TTY
		printNonTTYNotice()
		fmt.Printf("%s %s\n", IconRunning, title)
		err := fn(parentCtx)
		if err != nil {
			fmt.Printf("%s %s: %v\n", IconError, title, err)
		} else {
			fmt.Printf("%s %s\n", IconSuccess, title)
		}
		return err
	}

	// Create a cancellable context - shared via pointer so cancel works
	ctx, cancel := context.WithCancel(parentCtx)
	state := &spinnerState{ctx: ctx, cancel: cancel}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)

	m := spinnerModel{
		spinner: s,
		title:   title,
		fn:      fn,
		state:   state,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		cancel()
		return err
	}

	final := finalModel.(spinnerModel)
	if final.cancelled {
		fmt.Println(StyleWarning.Render(IconWarning) + " " + title + " (cancelled)")
		return ErrCancelled
	}

	if final.err != nil {
		fmt.Println(StyleError.Render(IconError) + " " + title)
		return final.err
	}

	fmt.Println(StyleSuccess.Render(IconSuccess) + " " + title)
	return nil
}

// -----------------------------------------------------------------------------
// Progress Bar Component
// -----------------------------------------------------------------------------

// progressState holds shared state that persists across model copies
type progressState struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type progressModel struct {
	progress  progress.Model
	title     string
	items     []string
	current   int
	err       error
	done      bool
	finishing bool // animating to 100% before quitting
	cancelled bool
	fn        func(ctx context.Context, item string) error
	state     *progressState
}

type progressItemDoneMsg struct {
	err error
}

func (m progressModel) Init() tea.Cmd {
	if len(m.items) == 0 {
		return tea.Quit
	}
	// Set initial progress to 0% to trigger first render, then start work
	return tea.Batch(
		m.progress.SetPercent(0),
		m.processNext(),
	)
}

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			// Cancel the background work
			if m.state != nil && m.state.cancel != nil {
				m.state.cancel()
			}
			m.done = true
			m.cancelled = true
			m.err = ErrCancelled
			return m, tea.Quit
		}

	case progressItemDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			m.done = true
			return m, tea.Quit
		}

		m.current++
		if m.current >= len(m.items) {
			// Set to 100% and mark as finishing (not done yet, so view still renders)
			cmd := m.progress.SetPercent(1.0)
			m.finishing = true
			return m, cmd
		}

		// Animate to new percentage, then process next item
		cmd := m.progress.SetPercent(float64(m.current) / float64(len(m.items)))
		return m, tea.Batch(cmd, m.processNext())

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		// If finishing and animation complete (no more frames), quit
		if m.finishing && cmd == nil {
			m.done = true
			return m, tea.Quit
		}
		return m, cmd
	}

	return m, nil
}

func (m progressModel) processNext() tea.Cmd {
	item := m.items[m.current]
	return func() tea.Msg {
		err := m.fn(m.state.ctx, item)
		return progressItemDoneMsg{err: err}
	}
}

func (m progressModel) View() string {
	if m.done {
		return ""
	}

	// Show completion status when finishing
	if m.finishing {
		return fmt.Sprintf(
			"%s\n%s\n%s",
			StyleBold.Render(m.title),
			m.progress.View(),
			StyleMuted.Render(fmt.Sprintf("  Complete (%d/%d)", len(m.items), len(m.items))),
		)
	}

	currentItem := ""
	if m.current < len(m.items) {
		currentItem = m.items[m.current]
	}

	return fmt.Sprintf(
		"%s\n%s\n%s",
		StyleBold.Render(m.title),
		m.progress.View(),
		StyleMuted.Render(fmt.Sprintf("  %s (%d/%d)", currentItem, m.current+1, len(m.items))),
	)
}

// RunProgress shows animated progress through a list of items.
// The function is cancelled if the user presses ctrl+c or q.
// Falls back to static output if not running in a TTY.
func RunProgress(title string, items []string, fn func(item string) error) error {
	return RunProgressWithContext(context.Background(), title, items, func(ctx context.Context, item string) error {
		return fn(item)
	})
}

// RunProgressWithContext shows animated progress through a list of items.
// The context is cancelled if the user presses ctrl+c or q.
// Falls back to static output if not running in a TTY.
func RunProgressWithContext(parentCtx context.Context, title string, items []string, fn func(ctx context.Context, item string) error) error {
	if len(items) == 0 {
		return nil
	}

	if !IsTTY() {
		// Fallback for non-TTY
		printNonTTYNotice()
		fmt.Println(title)
		for i, item := range items {
			fmt.Printf("  [%d/%d] %s\n", i+1, len(items), item)
			if err := fn(parentCtx, item); err != nil {
				fmt.Printf("  %s Failed: %v\n", IconError, err)
				return err
			}
			fmt.Printf("  %s Done\n", IconSuccess)
		}
		return nil
	}

	// Create a cancellable context - shared via pointer so cancel works
	ctx, cancel := context.WithCancel(parentCtx)
	state := &progressState{ctx: ctx, cancel: cancel}

	prog := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
		progress.WithSpringOptions(0.1, 0.8), // damping, frequency for smooth animation
	)

	m := progressModel{
		progress: prog,
		title:    title,
		items:    items,
		fn:       fn,
		state:    state,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		cancel()
		return err
	}

	final := finalModel.(progressModel)
	if final.cancelled {
		fmt.Println(StyleWarning.Render(IconWarning) + " " + title + " (cancelled)")
		return ErrCancelled
	}

	if final.err != nil {
		fmt.Println(StyleError.Render(IconError) + " " + title + " failed")
		return final.err
	}

	fmt.Println(StyleSuccess.Render(IconSuccess) + " " + title + " complete")
	return nil
}

// -----------------------------------------------------------------------------
// Table Component
// -----------------------------------------------------------------------------

// TableOption configures the table
type TableOption func(*tableConfig)

type tableConfig struct {
	width       int
	height      int
	focused     bool
	interactive bool
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

	t.SetStyles(s)

	return t
}

// RenderTable renders a simple static table (non-interactive).
// This is a convenience function that creates a table and returns its view.
func RenderTable(headers []string, rows [][]string, opts ...TableOption) string {
	t := NewBubblesTable(headers, rows, opts...)
	return t.View()
}

// -----------------------------------------------------------------------------
// Interactive Table (for selection)
// -----------------------------------------------------------------------------

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
func RunTableSelect(headers []string, rows [][]string, opts ...TableOption) (int, error) {
	if !IsTTY() {
		// Fallback: just print the table, return first row
		printNonTTYNotice()
		fmt.Println(RenderTable(headers, rows, opts...))
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

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return -1, err
	}

	final := finalModel.(tableModel)
	return final.selected, nil
}

// -----------------------------------------------------------------------------
// Spinner Styles (for reference, replacing SpinnerFrames)
// -----------------------------------------------------------------------------

// Available spinner styles from bubbles/spinner:
// - spinner.Line
// - spinner.Dot
// - spinner.MiniDot
// - spinner.Jump
// - spinner.Pulse
// - spinner.Points
// - spinner.Globe
// - spinner.Moon
// - spinner.Monkey
// - spinner.Meter
// - spinner.Hamburger
// - spinner.Ellipsis

// DefaultSpinnerStyle is the default spinner style used by RunSpinner
var DefaultSpinnerStyle = spinner.Dot

// SpinnerStyles provides access to all available spinner styles
var SpinnerStyles = struct {
	Line      spinner.Spinner
	Dot       spinner.Spinner
	MiniDot   spinner.Spinner
	Jump      spinner.Spinner
	Pulse     spinner.Spinner
	Points    spinner.Spinner
	Globe     spinner.Spinner
	Moon      spinner.Spinner
	Monkey    spinner.Spinner
	Meter     spinner.Spinner
	Hamburger spinner.Spinner
	Ellipsis  spinner.Spinner
}{
	Line:      spinner.Line,
	Dot:       spinner.Dot,
	MiniDot:   spinner.MiniDot,
	Jump:      spinner.Jump,
	Pulse:     spinner.Pulse,
	Points:    spinner.Points,
	Globe:     spinner.Globe,
	Moon:      spinner.Moon,
	Monkey:    spinner.Monkey,
	Meter:     spinner.Meter,
	Hamburger: spinner.Hamburger,
	Ellipsis:  spinner.Ellipsis,
}

// -----------------------------------------------------------------------------
// Simple Progress Bar (static, for non-interactive use)
// -----------------------------------------------------------------------------

// ProgressBar renders a simple static progress bar using bubbles/progress.
// This replaces the manual block-character implementation.
func ProgressBar(current, total int, width int) string {
	if total == 0 {
		return ""
	}

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(width),
		progress.WithoutPercentage(),
	)

	percent := float64(current) / float64(total)
	bar := p.ViewAs(percent)

	percentStr := fmt.Sprintf("%d%%", int(percent*100))
	percentStyle := lipgloss.NewStyle().Foreground(ColorMuted).Width(6).Align(lipgloss.Right)

	return bar + " " + percentStyle.Render(percentStr)
}

// ProgressBarWithLabel renders a progress bar with a label
func ProgressBarWithLabel(label string, current, total int, width int) string {
	bar := ProgressBar(current, total, width)
	labelStyle := lipgloss.NewStyle().Foreground(ColorText).Width(20)
	return labelStyle.Render(label) + " " + bar
}

// -----------------------------------------------------------------------------
// Multi-step Progress Display
// -----------------------------------------------------------------------------

// ProgressStepStatus represents the status of a progress step
type ProgressStepStatus int

const (
	ProgressStepPending ProgressStepStatus = iota
	ProgressStepRunning
	ProgressStepComplete
	ProgressStepFailed
)

// ProgressStep represents a step in a multi-step process
type ProgressStep struct {
	Name   string
	Status ProgressStepStatus
	Error  error
}

// RenderProgressSteps renders a list of steps with their status
func RenderProgressSteps(steps []ProgressStep) string {
	var sb strings.Builder

	for _, step := range steps {
		var icon string
		var style lipgloss.Style

		switch step.Status {
		case ProgressStepPending:
			icon = IconPending
			style = StyleStepPending
		case ProgressStepRunning:
			icon = IconRunning
			style = StyleStepActive
		case ProgressStepComplete:
			icon = IconSuccess
			style = StyleStepComplete
		case ProgressStepFailed:
			icon = IconError
			style = StyleStepFailed
		}

		line := style.Render(icon) + " " + step.Name
		if step.Error != nil {
			line += StyleError.Render(" (" + step.Error.Error() + ")")
		}
		sb.WriteString(line + "\n")
	}

	return sb.String()
}

// -----------------------------------------------------------------------------
// Provider Status Table Component
// -----------------------------------------------------------------------------

// ProviderInfo represents a provider's status for display
type ProviderInfo struct {
	Name    string
	Package string
	Healthy bool
	Message string
}

// providerTableState holds shared state that persists across model copies
type providerTableState struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type providerTableModel struct {
	spinner   spinner.Model
	table     table.Model
	providers []ProviderInfo
	title     string
	err       error
	done      bool
	cancelled bool
	fn        func(ctx context.Context) ([]ProviderInfo, bool, error)
	state     *providerTableState
	started   bool
}

type providerTableUpdateMsg struct {
	providers []ProviderInfo
	allReady  bool
	err       error
}

func (m providerTableModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m providerTableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			// Cancel the background work
			if m.state != nil && m.state.cancel != nil {
				m.state.cancel()
			}
			m.done = true
			m.cancelled = true
			m.err = ErrCancelled
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)

		// Start the background polling on first tick
		if !m.started {
			m.started = true
			return m, tea.Batch(cmd, m.pollProviders())
		}
		return m, cmd

	case providerTableUpdateMsg:
		if msg.err != nil {
			m.err = msg.err
			m.done = true
			return m, tea.Quit
		}

		m.providers = msg.providers

		if msg.allReady {
			m.done = true
			return m, tea.Quit
		}

		// Continue polling after a delay
		return m, m.pollProvidersDelayed()
	}

	return m, nil
}

func (m providerTableModel) pollProviders() tea.Cmd {
	return func() tea.Msg {
		providers, allReady, err := m.fn(m.state.ctx)
		return providerTableUpdateMsg{
			providers: providers,
			allReady:  allReady,
			err:       err,
		}
	}
}

func (m providerTableModel) pollProvidersDelayed() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		providers, allReady, err := m.fn(m.state.ctx)
		return providerTableUpdateMsg{
			providers: providers,
			allReady:  allReady,
			err:       err,
		}
	})
}

// providerTableBaseStyle defines the border style for the provider table
var providerTableBaseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

// buildProviderTable creates a new table with the given providers
func buildProviderTable(providers []ProviderInfo) table.Model {
	// Calculate column widths based on content
	nameWidth := 10
	for _, p := range providers {
		if len(p.Name) > nameWidth {
			nameWidth = len(p.Name)
		}
	}
	nameWidth += 2

	columns := []table.Column{
		{Title: "Provider", Width: nameWidth},
		{Title: "Status", Width: 12},
	}

	rows := make([]table.Row, len(providers))
	for i, p := range providers {
		status := IconRunning + " Waiting"
		if p.Healthy {
			status = IconSuccess + " Healthy"
		}
		rows[i] = table.Row{p.Name, status}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithHeight(len(providers)),
	)

	// Apply styles
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
	t.SetStyles(s)

	return t
}

func (m providerTableModel) View() string {
	if m.done {
		return ""
	}

	var sb strings.Builder

	// Title with spinner
	sb.WriteString(fmt.Sprintf("%s %s\n\n", m.spinner.View(), StyleBold.Render(m.title)))

	if len(m.providers) == 0 {
		sb.WriteString(StyleMuted.Render("  Checking providers..."))
		return sb.String()
	}

	// Build and render the table with borders
	t := buildProviderTable(m.providers)
	sb.WriteString(providerTableBaseStyle.Render(t.View()))
	sb.WriteString("\n")

	// Summary
	readyCount := 0
	for _, p := range m.providers {
		if p.Healthy {
			readyCount++
		}
	}
	sb.WriteString("\n")
	sb.WriteString(StyleMuted.Render(fmt.Sprintf("%d/%d providers ready", readyCount, len(m.providers))))

	return sb.String()
}

// RunProviderTable shows an animated table of providers with their status.
// The pollFn is called periodically to get the current provider status.
// It should return the list of providers, whether all are ready, and any error.
// Falls back to static output if not running in a TTY.
func RunProviderTable(parentCtx context.Context, title string, pollFn func(ctx context.Context) ([]ProviderInfo, bool, error)) error {
	if !IsTTY() {
		// Fallback for non-TTY
		printNonTTYNotice()
		fmt.Printf("%s %s\n", IconRunning, title)

		// Poll until all ready or error
		for {
			select {
			case <-parentCtx.Done():
				return parentCtx.Err()
			default:
				providers, allReady, err := pollFn(parentCtx)
				if err != nil {
					fmt.Printf("%s %s: %v\n", IconError, title, err)
					return err
				}

				// Print current status
				for _, p := range providers {
					status := "waiting"
					icon := IconRunning
					if p.Healthy {
						status = "healthy"
						icon = IconSuccess
					}
					fmt.Printf("  %s %s: %s\n", icon, p.Name, status)
				}

				if allReady {
					fmt.Printf("%s %s\n", IconSuccess, title)
					return nil
				}

				time.Sleep(5 * time.Second)
			}
		}
	}

	// Create a cancellable context - shared via pointer so cancel works
	ctx, cancel := context.WithCancel(parentCtx)
	state := &providerTableState{ctx: ctx, cancel: cancel}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)

	m := providerTableModel{
		spinner:   s,
		title:     title,
		providers: []ProviderInfo{},
		fn:        pollFn,
		state:     state,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		cancel()
		return err
	}

	final := finalModel.(providerTableModel)
	if final.cancelled {
		fmt.Println(StyleWarning.Render(IconWarning) + " " + title + " (cancelled)")
		return ErrCancelled
	}

	if final.err != nil {
		fmt.Println(StyleError.Render(IconError) + " " + title)
		return final.err
	}

	// Print final success state with all providers
	fmt.Println(StyleSuccess.Render(IconSuccess) + " " + title)
	for _, p := range final.providers {
		fmt.Printf("  %s %s\n", StyleSuccess.Render(IconSuccess), p.Name)
	}

	return nil
}
