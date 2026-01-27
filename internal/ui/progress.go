package ui

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// -----------------------------------------------------------------------------
// Progress Options
// -----------------------------------------------------------------------------

// ProgressOption configures a progress operation
type ProgressOption func(*progressOptions)

type progressOptions struct {
	output io.Writer
}

func defaultProgressOptions() *progressOptions {
	return &progressOptions{
		output: nil, // will use defaultOutput
	}
}

// WithProgressOutput sets the output writer for the progress bar.
// If not set, defaults to the package's default output (usually os.Stdout).
func WithProgressOutput(w io.Writer) ProgressOption {
	return func(o *progressOptions) {
		o.output = w
	}
}

func (o *progressOptions) getOutput() io.Writer {
	if o.output != nil {
		return o.output
	}
	return defaultOutput
}

// -----------------------------------------------------------------------------
// Progress Bar Component
// -----------------------------------------------------------------------------

// progressModel is a Bubble Tea model for showing animated progress
// through a list of items.
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
	state     *CancellableState
}

// progressItemDoneMsg is sent when processing of an item completes
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
		if action := HandleCancelKeys(msg, m.state); action.Handled {
			m.done = true
			m.cancelled = action.Cancelled
			if action.Cancelled {
				m.err = ErrCancelled
			}
			return m, action.Cmd
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
		err := m.fn(m.state.Ctx, item)
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
//
// In non-TTY mode, this outputs simple text without animation:
//   - For each item, prints "[current/total] item-name"
//   - On success for each item, prints a success message
//   - On failure, prints the error and returns
//
// Options:
//   - WithProgressOutput(w): Set custom output writer (default: package default output)
func RunProgress(title string, items []string, fn func(item string) error, opts ...ProgressOption) error {
	return RunProgressWithContext(context.Background(), title, items, func(ctx context.Context, item string) error {
		return fn(item)
	}, opts...)
}

// RunProgressWithContext shows animated progress through a list of items.
// The context is cancelled if the user presses ctrl+c or q.
//
// In non-TTY mode, this outputs simple text without animation:
//   - For each item, prints "[current/total] item-name"
//   - On success for each item, prints a success message
//   - On failure, prints the error and returns
//
// Options:
//   - WithProgressOutput(w): Set custom output writer (default: package default output)
func RunProgressWithContext(parentCtx context.Context, title string, items []string, fn func(ctx context.Context, item string) error, opts ...ProgressOption) error {
	options := defaultProgressOptions()
	for _, opt := range opts {
		opt(options)
	}
	output := options.getOutput()

	if len(items) == 0 {
		return nil
	}

	if !IsTTY() {
		printer := NewNonTTYPrinter(title, WithOutput(output))
		for i, item := range items {
			printer.ItemProgress(i+1, len(items), item)
			if err := fn(parentCtx, item); err != nil {
				printer.ItemFailed(err)
				return err
			}
			printer.ItemDone()
		}
		return nil
	}

	state := NewCancellableState(parentCtx)

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

	p := tea.NewProgram(m, tea.WithOutput(output))
	finalModel, err := p.Run()
	if err != nil {
		state.Cancel()
		return err
	}

	final := finalModel.(progressModel)
	if final.cancelled {
		fmt.Fprintln(output, StyleWarning.Render(IconWarning)+" "+title+" (cancelled)")
		return ErrCancelled
	}

	if final.err != nil {
		fmt.Fprintln(output, StyleError.Render(IconError)+" "+title+" failed")
		return final.err
	}

	fmt.Fprintln(output, StyleSuccess.Render(IconSuccess)+" "+title+" complete")
	return nil
}

// -----------------------------------------------------------------------------
// Simple Progress Bar (static, for non-interactive use)
// -----------------------------------------------------------------------------

// ProgressBar renders a simple static progress bar using bubbles/progress.
// This is useful for displaying progress in non-interactive contexts.
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
// Multi-step Progress Display (static rendering)
// -----------------------------------------------------------------------------

// ProgressStep represents a step in a multi-step process for static rendering.
type ProgressStep struct {
	Name   string
	Status StepStatus
	Error  error
}

// RenderProgressSteps renders a list of steps with their status.
// This is useful for displaying a summary of completed steps.
func RenderProgressSteps(steps []ProgressStep) string {
	var sb strings.Builder

	for _, step := range steps {
		icon := step.Status.Icon()
		style := step.Status.Style()

		line := style.Render(icon) + " " + step.Name
		if step.Error != nil {
			line += StyleError.Render(" (" + step.Error.Error() + ")")
		}
		sb.WriteString(line + "\n")
	}

	return sb.String()
}
