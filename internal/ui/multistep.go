package ui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// -----------------------------------------------------------------------------
// Multi-step Options
// -----------------------------------------------------------------------------

// MultiStepOption configures a multi-step operation
type MultiStepOption func(*multiStepOptions)

type multiStepOptions struct {
	output io.Writer
}

func defaultMultiStepOptions() *multiStepOptions {
	return &multiStepOptions{
		output: nil, // will use defaultOutput
	}
}

// WithMultiStepOutput sets the output writer for the multi-step operation.
// If not set, defaults to the package's default output (usually os.Stdout).
func WithMultiStepOutput(w io.Writer) MultiStepOption {
	return func(o *multiStepOptions) {
		o.output = w
	}
}

func (o *multiStepOptions) getOutput() io.Writer {
	if o.output != nil {
		return o.output
	}
	return defaultOutput
}

// -----------------------------------------------------------------------------
// Multi-step Progress Component
// -----------------------------------------------------------------------------

// MultiStep represents a step in a multi-step operation
type MultiStep struct {
	Name   string
	Status StepStatus
}

// multiStepModel is a Bubble Tea model for showing progress through
// a multi-step operation with live updates.
type multiStepModel struct {
	spinner       spinner.Model
	title         string
	steps         map[string]*MultiStep
	stepOrder     []string // Maintain order of steps as they appear
	err           error
	done          bool
	cancelled     bool
	fn            func(ctx context.Context, updates chan<- StepUpdate, done <-chan struct{}) error
	state         *CancellableState
	started       bool
	updates       chan StepUpdate
	doneCh        chan struct{} // Closed when work is done so loggers stop sending
	workDone      chan error    // Channel to signal work completion with error
	updatesClosed bool          // Track if updates channel is closed
}

type (
	stepUpdateMsg struct {
		update StepUpdate
	}
	workPendingMsg   struct{}
	workCompletedMsg struct{}
	updatesClosedMsg struct{}
)

func (m multiStepModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m multiStepModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

		// Start the background work on first tick
		if !m.started {
			m.started = true
			return m, tea.Batch(cmd, m.startWork(), m.listenForUpdates(), m.checkWorkDone())
		}
		// Continue checking for work completion
		return m, tea.Batch(cmd, m.checkWorkDone())

	case stepUpdateMsg:
		// Update step status based on the update
		update := msg.update
		stepName := update.Step

		// Add step if it doesn't exist
		if _, exists := m.steps[stepName]; !exists {
			m.steps[stepName] = &MultiStep{
				Name:   stepName,
				Status: StepPending,
			}
			m.stepOrder = append(m.stepOrder, stepName)
		}

		step := m.steps[stepName]
		if update.Done {
			if update.Success {
				step.Status = StepComplete
			} else {
				step.Status = StepFailed
				m.err = fmt.Errorf("step failed: %s", stepName)
				m.done = true
				if m.state != nil && m.state.Cancel != nil {
					m.state.Cancel()
				}
				return m, tea.Quit
			}
		} else {
			step.Status = StepRunning
		}

		// Continue listening for updates (if channel still open)
		if !m.updatesClosed {
			return m, m.listenForUpdates()
		}
		return m, m.checkWorkDone()

	case error:
		// Check if this is a context error
		if errors.Is(msg, m.state.Ctx.Err()) {
			m.err = msg
			m.done = true
			m.cancelled = true
			return m, tea.Quit
		}
		// Work completed with error
		m.err = msg
		m.done = true
		return m, tea.Quit

	case updatesClosedMsg:
		// Updates channel has been closed, stop listening for updates
		m.updatesClosed = true
		return m, m.checkWorkDone()

	case workPendingMsg:
		// Work is still in progress, continue checking
		return m, m.checkWorkDone()

	case workCompletedMsg:
		// Work completed successfully - mark all running steps as complete
		for _, step := range m.steps {
			if step.Status == StepRunning {
				step.Status = StepComplete
			}
		}
		m.done = true
		return m, tea.Quit
	}

	return m, nil
}

func (m multiStepModel) startWork() tea.Cmd {
	return func() tea.Msg {
		go func() {
			var err error
			defer func() {
				if p := recover(); p != nil {
					if e, ok := p.(error); ok {
						err = e
					} else {
						err = fmt.Errorf("panic: %v", p)
					}
				}
				close(m.doneCh) // Signal loggers to stop sending before we close updates
				close(m.updates)
				m.workDone <- err
			}()
			err = m.fn(m.state.Ctx, m.updates, m.doneCh)
		}()
		return nil
	}
}

func (m multiStepModel) checkWorkDone() tea.Cmd {
	return func() tea.Msg {
		select {
		case <-m.state.Ctx.Done():
			return m.state.Ctx.Err()
		case err := <-m.workDone:
			if err != nil {
				return err
			}
			return workCompletedMsg{}
		default:
			return workPendingMsg{}
		}
	}
}

func (m multiStepModel) listenForUpdates() tea.Cmd {
	return func() tea.Msg {
		select {
		case <-m.state.Ctx.Done():
			return m.state.Ctx.Err()
		case update, ok := <-m.updates:
			if !ok {
				return updatesClosedMsg{}
			}
			return stepUpdateMsg{update: update}
		}
	}
}

func (m multiStepModel) View() string {
	if m.done {
		return ""
	}

	var sb strings.Builder

	// Title with spinner
	sb.WriteString(fmt.Sprintf("%s %s\n\n", m.spinner.View(), StyleBold.Render(m.title)))

	// Render steps in order
	if len(m.stepOrder) == 0 {
		sb.WriteString(StyleMuted.Render("  Waiting for operation to start..."))
		return sb.String()
	}

	for _, stepName := range m.stepOrder {
		step := m.steps[stepName]
		var icon string
		var style = step.Status.Style()

		switch step.Status {
		case StepRunning:
			// Show spinner animation for running steps
			icon = m.spinner.View()
		default:
			icon = step.Status.Icon()
		}

		line := "  " + style.Render(icon) + " " + step.Name
		sb.WriteString(line + "\n")
	}

	return sb.String()
}

// RunMultiStep shows an animated multi-step progress display.
// The function receives a context and a channel to send step updates.
//
// In non-TTY mode, this outputs simple text without animation:
//   - For each step started, prints the step name with a "running" icon
//   - On success for each step, prints a success icon
//   - On failure, prints the error and returns
//
// Options:
//   - WithMultiStepOutput(w): Set custom output writer (default: package default output)
func RunMultiStep(parentCtx context.Context, title string, fn func(ctx context.Context, updates chan<- StepUpdate, done <-chan struct{}) error, opts ...MultiStepOption) error {
	options := defaultMultiStepOptions()
	for _, opt := range opts {
		opt(options)
	}
	output := options.getOutput()

	if !IsTTY() {
		return runMultiStepNonTTY(parentCtx, title, fn, output)
	}

	state := NewCancellableState(parentCtx)
	updates := make(chan StepUpdate, 10)
	done := make(chan struct{})
	workDone := make(chan error, 1)

	m := multiStepModel{
		spinner:   NewDefaultSpinner(),
		title:     title,
		steps:     make(map[string]*MultiStep),
		stepOrder: []string{},
		fn:        fn,
		state:     state,
		updates:   updates,
		doneCh:    done,
		workDone:  workDone,
	}

	p := tea.NewProgram(m, tea.WithOutput(output))
	finalModel, err := p.Run()
	if err != nil {
		state.Cancel()
		return err
	}

	final := finalModel.(multiStepModel)
	if final.cancelled {
		_, _ = fmt.Fprintln(output, StyleWarning.Render(IconWarning)+" "+title+" (cancelled)")
		return ErrCancelled
	}

	if final.err != nil {
		_, _ = fmt.Fprintln(output, StyleError.Render(IconError)+" "+title)
		return final.err
	}

	// Print final success state with all completed steps
	_, _ = fmt.Fprintln(output, StyleSuccess.Render(IconSuccess)+" "+title)
	for _, stepName := range final.stepOrder {
		step := final.steps[stepName]
		if step.Status == StepComplete {
			_, _ = fmt.Fprintf(output, "  %s %s\n", StyleSuccess.Render(IconSuccess), step.Name)
		}
	}

	return nil
}

// runMultiStepNonTTY handles non-TTY fallback for multi-step operations
func runMultiStepNonTTY(parentCtx context.Context, title string, fn func(ctx context.Context, updates chan<- StepUpdate, done <-chan struct{}) error, output io.Writer) error {
	printNonTTYNoticeTo(output)
	_, _ = fmt.Fprintf(output, "%s %s...\n", IconRunning, title)

	updates := make(chan StepUpdate, 10)
	done := make(chan struct{})
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	// Start work in background
	workDone := make(chan error, 1)
	go func() {
		var err error
		defer func() {
			if p := recover(); p != nil {
				if e, ok := p.(error); ok {
					err = fmt.Errorf("panic in multi-step work: %w", e)
				} else {
					err = fmt.Errorf("panic in multi-step work: %v", p)
				}
			}
			close(done)
			close(updates)
			workDone <- err
		}()
		err = fn(ctx, updates, done)
	}()

	// Process updates
	steps := make(map[string]bool)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-workDone:
			if err != nil {
				return err
			}
			// Process any remaining updates
			for update := range updates {
				if update.Done {
					if update.Success {
						_, _ = fmt.Fprintf(output, "  %s %s\n", IconSuccess, update.Step)
					} else {
						_, _ = fmt.Fprintf(output, "  %s %s\n", IconError, update.Step)
					}
				} else {
					_, _ = fmt.Fprintf(output, "  %s %s...\n", IconRunning, update.Step)
				}
			}
			_, _ = fmt.Fprintf(output, "%s %s complete\n", IconSuccess, title)
			return nil
		case update, ok := <-updates:
			if !ok {
				updates = nil
				continue
			}
			stepName := update.Step
			if update.Done {
				if update.Success {
					if !steps[stepName] {
						_, _ = fmt.Fprintf(output, "  %s %s\n", IconSuccess, stepName)
						steps[stepName] = true
					}
				} else {
					_, _ = fmt.Fprintf(output, "  %s %s\n", IconError, stepName)
					return fmt.Errorf("step failed: %s", stepName)
				}
			} else {
				if !steps[stepName] {
					_, _ = fmt.Fprintf(output, "  %s %s...\n", IconRunning, stepName)
					steps[stepName] = true
				}
			}
		}
	}
}

// RunClusterCreate is a convenience wrapper for creating a Kind cluster.
// It provides a descriptive title and delegates to RunMultiStep.
//
// Options:
//   - WithMultiStepOutput(w): Set custom output writer (default: package default output)
func RunClusterCreate(parentCtx context.Context, clusterName string, fn func(ctx context.Context, updates chan<- StepUpdate, done <-chan struct{}) error, opts ...MultiStepOption) error {
	title := fmt.Sprintf("Creating Kind cluster '%s'", clusterName)
	return RunMultiStep(parentCtx, title, fn, opts...)
}
