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
// Pod Table Options
// -----------------------------------------------------------------------------

// PodTableOption configures a pod table operation
type PodTableOption func(*podTableOptions)

type podTableOptions struct {
	output       io.Writer
	pollInterval time.Duration
}

func defaultPodTableOptions() *podTableOptions {
	return &podTableOptions{
		output:       nil, // will use defaultOutput
		pollInterval: DefaultPollInterval,
	}
}

// WithPodTableOutput sets the output writer for the pod table.
// If not set, defaults to the package's default output (usually os.Stdout).
func WithPodTableOutput(w io.Writer) PodTableOption {
	return func(o *podTableOptions) {
		o.output = w
	}
}

// WithPodTablePollInterval sets the polling interval for pod status updates.
// If not set, defaults to DefaultPollInterval (2 seconds).
func WithPodTablePollInterval(d time.Duration) PodTableOption {
	return func(o *podTableOptions) {
		o.pollInterval = d
	}
}

func (o *podTableOptions) getOutput() io.Writer {
	if o.output != nil {
		return o.output
	}
	return defaultOutput
}

// -----------------------------------------------------------------------------
// Pod Status Table Component
// -----------------------------------------------------------------------------

// PodInfo represents a pod's status for display
type PodInfo struct {
	Name    string
	Status  string // Pod phase (Pending, Running, Succeeded, Failed)
	Ready   bool
	Message string // Optional status message
}

// podTableModel is a Bubble Tea model for showing a live-updating
// table of pod statuses.
type podTableModel struct {
	spinner      spinner.Model
	pods         []PodInfo
	title        string
	err          error
	done         bool
	cancelled    bool
	fn           func(ctx context.Context) ([]PodInfo, bool, error)
	state        *CancellableState
	started      bool
	pollInterval time.Duration
}

// podTableUpdateMsg is sent when pod status is updated
type podTableUpdateMsg struct {
	pods     []PodInfo
	allReady bool
	err      error
}

func (m podTableModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m podTableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			return m, tea.Batch(cmd, m.pollPods())
		}
		return m, cmd

	case podTableUpdateMsg:
		if msg.err != nil {
			m.err = msg.err
			m.done = true
			return m, tea.Quit
		}

		m.pods = msg.pods

		if msg.allReady {
			m.done = true
			return m, tea.Quit
		}

		// Continue polling after a delay
		return m, m.pollPodsDelayed()
	}

	return m, nil
}

func (m podTableModel) pollPods() tea.Cmd {
	return func() tea.Msg {
		pods, allReady, err := m.fn(m.state.Ctx)
		return podTableUpdateMsg{
			pods:     pods,
			allReady: allReady,
			err:      err,
		}
	}
}

func (m podTableModel) pollPodsDelayed() tea.Cmd {
	return tea.Tick(m.pollInterval, func(t time.Time) tea.Msg {
		pods, allReady, err := m.fn(m.state.Ctx)
		return podTableUpdateMsg{
			pods:     pods,
			allReady: allReady,
			err:      err,
		}
	})
}

func (m podTableModel) View() string {
	if m.done {
		return ""
	}

	var sb strings.Builder

	// Title with spinner
	sb.WriteString(fmt.Sprintf("%s %s\n\n", m.spinner.View(), StyleBold.Render(m.title)))

	if len(m.pods) == 0 {
		sb.WriteString(StyleMuted.Render("  Checking pods..."))
		return sb.String()
	}

	// Build and render the table
	rows := make([]StatusRow, len(m.pods))
	for i, p := range m.pods {
		status := formatPodStatus(p)
		ready := formatPodReady(p)
		rows[i] = StatusRow{Cells: []string{p.Name, status, ready}}
	}

	table := RenderStatusTable(StatusTableConfig{
		Columns: []StatusColumn{
			{Title: "Pod", MinWidth: 10},
			{Title: "Status", MinWidth: 12},
			{Title: "Ready", MinWidth: 10},
		},
		Rows: rows,
	})
	sb.WriteString(table)
	sb.WriteString("\n")

	// Summary
	readyCount := 0
	for _, p := range m.pods {
		if p.Ready {
			readyCount++
		}
	}
	sb.WriteString("\n")
	sb.WriteString(StyleMuted.Render(fmt.Sprintf("%d/%d pods ready", readyCount, len(m.pods))))

	return sb.String()
}

// formatPodStatus formats a pod's status for display
func formatPodStatus(p PodInfo) string {
	status := p.Status
	if p.Message != "" && p.Status != "Running" {
		// Truncate long messages
		if len(p.Message) > 30 {
			status = p.Status + " (" + p.Message[:27] + "...)"
		} else {
			status = p.Status + " (" + p.Message + ")"
		}
	}
	return status
}

// formatPodReady formats a pod's ready status for display
func formatPodReady(p PodInfo) string {
	if p.Ready {
		return IconSuccess + " Ready"
	}
	if p.Status == "Failed" {
		return IconError + " Failed"
	}
	if p.Status == "Pending" {
		return IconPending + " Pending"
	}
	return IconRunning + " Waiting"
}

// RunPodTable shows an animated table of pods with their status.
// The pollFn is called periodically to get the current pod status.
// It should return the list of pods, whether all are ready, and any error.
//
// In non-TTY mode, this outputs simple text without animation:
//   - Periodically polls and prints pod statuses
//   - Shows icons for ready/pending/failed states
//   - Returns when all pods are ready or an error occurs
//
// Options:
//   - WithPodTableOutput(w): Set custom output writer (default: package default output)
//   - WithPodTablePollInterval(d): Set polling interval (default: 2 seconds)
func RunPodTable(parentCtx context.Context, title string, pollFn func(ctx context.Context) ([]PodInfo, bool, error), opts ...PodTableOption) error {
	options := defaultPodTableOptions()
	for _, opt := range opts {
		opt(options)
	}
	output := options.getOutput()

	if !IsTTY() {
		return runPodTableNonTTY(parentCtx, title, pollFn, output, options.pollInterval)
	}

	state := NewCancellableState(parentCtx)

	m := podTableModel{
		spinner:      NewDefaultSpinner(),
		title:        title,
		pods:         []PodInfo{},
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

	final := finalModel.(podTableModel)
	if final.cancelled {
		_, _ = fmt.Fprintln(output, StyleWarning.Render(IconWarning)+" "+title+" (cancelled)")
		return ErrCancelled
	}

	if final.err != nil {
		_, _ = fmt.Fprintln(output, StyleError.Render(IconError)+" "+title)
		return final.err
	}

	// Print final success state with all pods
	_, _ = fmt.Fprintln(output, StyleSuccess.Render(IconSuccess)+" "+title)
	for _, p := range final.pods {
		_, _ = fmt.Fprintf(output, "  %s %s: %s\n", StyleSuccess.Render(IconSuccess), p.Name, p.Status)
	}

	return nil
}

// runPodTableNonTTY handles non-TTY fallback for pod table
func runPodTableNonTTY(parentCtx context.Context, title string, pollFn func(ctx context.Context) ([]PodInfo, bool, error), output io.Writer, pollInterval time.Duration) error {
	printNonTTYNoticeTo(output)
	_, _ = fmt.Fprintf(output, "%s %s\n", IconRunning, title)

	// Poll until all ready or error
	for {
		select {
		case <-parentCtx.Done():
			return parentCtx.Err()
		default:
			pods, allReady, err := pollFn(parentCtx)
			if err != nil {
				_, _ = fmt.Fprintf(output, "%s %s: %v\n", IconError, title, err)
				return err
			}

			// Print current status
			for _, p := range pods {
				status := p.Status
				icon := IconRunning
				if p.Ready {
					status = "ready"
					icon = IconSuccess
				} else if p.Status == "Failed" {
					icon = IconError
				} else if p.Status == "Pending" {
					icon = IconPending
				}
				_, _ = fmt.Fprintf(output, "  %s %s: %s", icon, p.Name, status)
				if p.Message != "" {
					_, _ = fmt.Fprintf(output, " (%s)", p.Message)
				}
				_, _ = fmt.Fprintln(output)
			}

			if allReady {
				_, _ = fmt.Fprintf(output, "%s %s\n", IconSuccess, title)
				return nil
			}

			time.Sleep(pollInterval)
		}
	}
}
