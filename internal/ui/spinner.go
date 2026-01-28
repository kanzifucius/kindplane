package ui

import (
	"context"
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// -----------------------------------------------------------------------------
// Spinner Options
// -----------------------------------------------------------------------------

// SpinnerOption configures a spinner operation
type SpinnerOption func(*spinnerOptions)

type spinnerOptions struct {
	output io.Writer
}

func defaultSpinnerOptions() *spinnerOptions {
	return &spinnerOptions{
		output: nil, // will use defaultOutput
	}
}

// WithSpinnerOutput sets the output writer for the spinner.
// If not set, defaults to the package's default output (usually os.Stdout).
func WithSpinnerOutput(w io.Writer) SpinnerOption {
	return func(o *spinnerOptions) {
		o.output = w
	}
}

func (o *spinnerOptions) getOutput() io.Writer {
	if o.output != nil {
		return o.output
	}
	return defaultOutput
}

// -----------------------------------------------------------------------------
// Spinner Component
// -----------------------------------------------------------------------------

// spinnerModel is a Bubble Tea model for showing an animated spinner
// while executing a background task.
type spinnerModel struct {
	spinner   spinner.Model
	title     string
	err       error
	done      bool
	cancelled bool
	fn        func(ctx context.Context) error
	state     *CancellableState
	started   bool
}

// spinnerDoneMsg is sent when the background work completes
type spinnerDoneMsg struct {
	err error
}

func (m spinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		// Run the work function - it should handle context cancellation internally
		err := m.fn(m.state.Ctx)
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
//
// In non-TTY mode, this outputs simple text without animation:
//   - Prints the title with a "running" icon
//   - On success, prints the title with a success icon
//   - On error, prints the error message
//
// Options:
//   - WithSpinnerOutput(w): Set custom output writer (default: package default output)
func RunSpinner(title string, fn func() error, opts ...SpinnerOption) error {
	return RunSpinnerWithContext(context.Background(), title, func(ctx context.Context) error {
		return fn()
	}, opts...)
}

// RunSpinnerWithContext shows an animated spinner while executing a context-aware function.
// The context is cancelled if the user presses ctrl+c or q.
//
// In non-TTY mode, this outputs simple text without animation:
//   - Prints the title with a "running" icon
//   - On success, prints the title with a success icon
//   - On error, prints the error message
//
// Options:
//   - WithSpinnerOutput(w): Set custom output writer (default: package default output)
func RunSpinnerWithContext(parentCtx context.Context, title string, fn func(ctx context.Context) error, opts ...SpinnerOption) error {
	options := defaultSpinnerOptions()
	for _, opt := range opts {
		opt(options)
	}
	output := options.getOutput()

	if !IsTTY() {
		printer := NewNonTTYPrinter(title, WithOutput(output))
		err := fn(parentCtx)
		if err != nil {
			printer.Error(err)
		} else {
			printer.Success()
		}
		return err
	}

	state := NewCancellableState(parentCtx)

	m := spinnerModel{
		spinner: NewDefaultSpinner(),
		title:   title,
		fn:      fn,
		state:   state,
	}

	p := tea.NewProgram(m, tea.WithOutput(output))
	finalModel, err := p.Run()
	if err != nil {
		state.Cancel()
		return err
	}

	final := finalModel.(spinnerModel)
	if final.cancelled {
		_, _ = fmt.Fprintln(output, StyleWarning.Render(IconWarning)+" "+title+" (cancelled)")
		return ErrCancelled
	}

	if final.err != nil {
		_, _ = fmt.Fprintln(output, StyleError.Render(IconError)+" "+title)
		return final.err
	}

	_, _ = fmt.Fprintln(output, StyleSuccess.Render(IconSuccess)+" "+title)
	return nil
}

// -----------------------------------------------------------------------------
// Spinner Styles (available spinner types)
// -----------------------------------------------------------------------------

// DefaultSpinnerStyle is the default spinner style used by RunSpinner
var DefaultSpinnerStyle = spinner.Dot

// SpinnerStyles provides access to all available spinner styles from bubbles/spinner
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
