package ui

import (
	"context"
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmOption is a functional option for Confirm
type ConfirmOption func(*confirmOptions)

// confirmOptions holds configuration for the confirmation prompt
type confirmOptions struct {
	output     io.Writer
	defaultYes bool
}

// defaultConfirmOptions returns the default options
func defaultConfirmOptions() *confirmOptions {
	return &confirmOptions{
		output:     nil, // will use defaultOutput
		defaultYes: false,
	}
}

// WithConfirmOutput sets the output writer for the confirmation prompt
func WithConfirmOutput(w io.Writer) ConfirmOption {
	return func(o *confirmOptions) {
		o.output = w
	}
}

// WithConfirmDefault sets the default selection (true = Yes, false = No)
func WithConfirmDefault(defaultYes bool) ConfirmOption {
	return func(o *confirmOptions) {
		o.defaultYes = defaultYes
	}
}

// getOutput returns the configured output or defaultOutput
func (o *confirmOptions) getOutput() io.Writer {
	if o.output != nil {
		return o.output
	}
	return defaultOutput
}

// confirmModel is the Bubble Tea model for confirmation prompts
type confirmModel struct {
	message     string
	yesSelected bool
	confirmed   bool
	cancelled   bool
	state       *CancellableState
}

// confirmResultMsg is sent when the user makes a choice

// Init initializes the model
func (m confirmModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			if action := HandleCancelKeys(msg, m.state); action.Handled {
				m.cancelled = true
				return m, tea.Quit
			}

		case "left", "h":
			m.yesSelected = true
			return m, nil

		case "right", "l", "tab":
			m.yesSelected = false
			return m, nil

		case "y":
			m.yesSelected = true
			m.confirmed = true
			return m, tea.Quit

		case "n":
			m.yesSelected = false
			m.confirmed = true
			return m, tea.Quit

		case "enter":
			m.confirmed = true
			return m, tea.Quit
		}
	}

	return m, nil
}

// View renders the confirmation prompt
func (m confirmModel) View() string {
	if m.confirmed || m.cancelled {
		return "" // Clear the screen on exit
	}

	var b strings.Builder

	// Render the message
	b.WriteString(StyleBold.Render(m.message))
	b.WriteString("\n\n  ")

	// Render Yes option
	yesStyle := lipgloss.NewStyle()
	if m.yesSelected {
		yesStyle = yesStyle.Foreground(ColorPrimary).Bold(true)
		b.WriteString(yesStyle.Render("▸ Yes"))
	} else {
		yesStyle = yesStyle.Foreground(ColorMuted)
		b.WriteString(yesStyle.Render("  Yes"))
	}

	b.WriteString("  ")

	// Render No option
	noStyle := lipgloss.NewStyle()
	if !m.yesSelected {
		noStyle = noStyle.Foreground(ColorPrimary).Bold(true)
		b.WriteString(noStyle.Render("▸ No"))
	} else {
		noStyle = noStyle.Foreground(ColorMuted)
		b.WriteString(noStyle.Render("  No"))
	}

	b.WriteString("\n")

	return b.String()
}

// Confirm displays a yes/no confirmation prompt
func Confirm(message string, opts ...ConfirmOption) (bool, error) {
	return ConfirmWithContext(context.Background(), message, opts...)
}

// ConfirmWithContext displays a yes/no confirmation prompt with context support
func ConfirmWithContext(parentCtx context.Context, message string, opts ...ConfirmOption) (bool, error) {
	// Apply options
	options := defaultConfirmOptions()
	for _, opt := range opts {
		opt(options)
	}

	output := options.getOutput()

	// Non-TTY fallback: return false (safe default) with a warning
	if !IsTTY() {
		_, _ = fmt.Fprintf(output, "%s %s (defaulting to No in non-TTY mode)\n", IconWarning, message)
		return false, nil
	}

	// Create cancellable state
	state := NewCancellableState(parentCtx)

	// Create model
	model := confirmModel{
		message:     message,
		yesSelected: options.defaultYes,
		confirmed:   false,
		cancelled:   false,
		state:       state,
	}

	// Run the program
	p := tea.NewProgram(model, tea.WithOutput(output))
	finalModel, err := p.Run()
	if err != nil {
		return false, fmt.Errorf("failed to read confirmation: %w", err)
	}

	// Get the final state
	final := finalModel.(confirmModel)

	// Check if cancelled
	if final.cancelled {
		return false, ErrCancelled
	}

	// Return the user's choice
	return final.yesSelected, nil
}
