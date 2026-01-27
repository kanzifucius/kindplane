package ui

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// -----------------------------------------------------------------------------
// Provider Table Options
// -----------------------------------------------------------------------------

// ProviderTableOption configures a provider table operation
type ProviderTableOption func(*providerTableOptions)

type providerTableOptions struct {
	output       io.Writer
	pollInterval time.Duration
}

func defaultProviderTableOptions() *providerTableOptions {
	return &providerTableOptions{
		output:       nil, // will use defaultOutput
		pollInterval: DefaultProviderPollInterval,
	}
}

// WithProviderTableOutput sets the output writer for the provider table.
// If not set, defaults to the package's default output (usually os.Stdout).
func WithProviderTableOutput(w io.Writer) ProviderTableOption {
	return func(o *providerTableOptions) {
		o.output = w
	}
}

// WithProviderTablePollInterval sets the polling interval for provider status updates.
// If not set, defaults to DefaultProviderPollInterval (5 seconds).
func WithProviderTablePollInterval(d time.Duration) ProviderTableOption {
	return func(o *providerTableOptions) {
		o.pollInterval = d
	}
}

func (o *providerTableOptions) getOutput() io.Writer {
	if o.output != nil {
		return o.output
	}
	return defaultOutput
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

// providerTableModel is a Bubble Tea model for showing a live-updating
// table of provider statuses.
type providerTableModel struct {
	spinner      spinner.Model
	providers    []ProviderInfo
	title        string
	err          error
	done         bool
	cancelled    bool
	fn           func(ctx context.Context) ([]ProviderInfo, bool, error)
	state        *CancellableState
	started      bool
	pollInterval time.Duration
}

// providerTableUpdateMsg is sent when provider status is updated
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
		if action := HandleCancelKeys(msg, m.state); action.Handled {
			m.done = true
			m.cancelled = action.Cancelled
			if action.Cancelled {
				m.err = ErrCancelled
			}
			return m, action.Cmd
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
		providers, allReady, err := m.fn(m.state.Ctx)
		return providerTableUpdateMsg{
			providers: providers,
			allReady:  allReady,
			err:       err,
		}
	}
}

func (m providerTableModel) pollProvidersDelayed() tea.Cmd {
	return tea.Tick(m.pollInterval, func(t time.Time) tea.Msg {
		providers, allReady, err := m.fn(m.state.Ctx)
		return providerTableUpdateMsg{
			providers: providers,
			allReady:  allReady,
			err:       err,
		}
	})
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

	// Build and render the table
	rows := make([]StatusRow, len(m.providers))
	for i, p := range m.providers {
		status := formatProviderStatus(p)
		rows[i] = StatusRow{Cells: []string{p.Name, status}}
	}

	table := RenderStatusTable(StatusTableConfig{
		Columns: []StatusColumn{
			{Title: "Provider", MinWidth: 10},
			{Title: "Status", MinWidth: 12},
		},
		Rows: rows,
	})
	sb.WriteString(table)
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

// formatProviderStatus formats a provider's status for display
func formatProviderStatus(p ProviderInfo) string {
	if p.Healthy {
		return IconSuccess + " Healthy"
	}
	return IconRunning + " Waiting"
}

// RunProviderTable shows an animated table of providers with their status.
// The pollFn is called periodically to get the current provider status.
// It should return the list of providers, whether all are ready, and any error.
//
// In non-TTY mode, this outputs simple text without animation:
//   - Periodically polls and prints provider statuses
//   - Shows icons for healthy/waiting states
//   - Returns when all providers are ready or an error occurs
//
// Options:
//   - WithProviderTableOutput(w): Set custom output writer (default: package default output)
//   - WithProviderTablePollInterval(d): Set polling interval (default: 5 seconds)
func RunProviderTable(parentCtx context.Context, title string, pollFn func(ctx context.Context) ([]ProviderInfo, bool, error), opts ...ProviderTableOption) error {
	options := defaultProviderTableOptions()
	for _, opt := range opts {
		opt(options)
	}
	output := options.getOutput()

	if !IsTTY() {
		return runProviderTableNonTTY(parentCtx, title, pollFn, output, options.pollInterval)
	}

	state := NewCancellableState(parentCtx)

	m := providerTableModel{
		spinner:      NewDefaultSpinner(),
		title:        title,
		providers:    []ProviderInfo{},
		fn:           pollFn,
		state:        state,
		pollInterval: options.pollInterval,
	}

	p := tea.NewProgram(m, tea.WithOutput(output))
	finalModel, err := p.Run()
	if err != nil {
		state.Cancel()
		return err
	}

	final := finalModel.(providerTableModel)
	if final.cancelled {
		fmt.Fprintln(output, StyleWarning.Render(IconWarning)+" "+title+" (cancelled)")
		return ErrCancelled
	}

	if final.err != nil {
		fmt.Fprintln(output, StyleError.Render(IconError)+" "+title)
		return final.err
	}

	// Print final success state with all providers
	fmt.Fprintln(output, StyleSuccess.Render(IconSuccess)+" "+title)
	for _, p := range final.providers {
		fmt.Fprintf(output, "  %s %s\n", StyleSuccess.Render(IconSuccess), p.Name)
	}

	return nil
}

// runProviderTableNonTTY handles non-TTY fallback for provider table
func runProviderTableNonTTY(parentCtx context.Context, title string, pollFn func(ctx context.Context) ([]ProviderInfo, bool, error), output io.Writer, pollInterval time.Duration) error {
	printNonTTYNoticeTo(output)
	fmt.Fprintf(output, "%s %s\n", IconRunning, title)

	// Poll until all ready or error
	for {
		select {
		case <-parentCtx.Done():
			return parentCtx.Err()
		default:
			providers, allReady, err := pollFn(parentCtx)
			if err != nil {
				fmt.Fprintf(output, "%s %s: %v\n", IconError, title, err)
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
				fmt.Fprintf(output, "  %s %s: %s\n", icon, p.Name, status)
			}

			if allReady {
				fmt.Fprintf(output, "%s %s\n", IconSuccess, title)
				return nil
			}

			time.Sleep(pollInterval)
		}
	}
}
