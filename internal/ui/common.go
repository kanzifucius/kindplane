package ui

import (
	"context"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// -----------------------------------------------------------------------------
// Shared Cancellation State
// -----------------------------------------------------------------------------

// CancellableState holds context and cancel function for graceful cancellation.
// This is shared across all interactive UI models to provide consistent
// cancellation behavior.
type CancellableState struct {
	Ctx    context.Context
	Cancel context.CancelFunc
}

// NewCancellableState creates a child context that can be cancelled.
func NewCancellableState(parentCtx context.Context) *CancellableState {
	ctx, cancel := context.WithCancel(parentCtx)
	return &CancellableState{
		Ctx:    ctx,
		Cancel: cancel,
	}
}

// -----------------------------------------------------------------------------
// Keyboard Handling
// -----------------------------------------------------------------------------

// KeyAction represents the result of processing a key press
type KeyAction struct {
	Handled   bool    // Whether the key was handled
	Cancelled bool    // Whether the operation was cancelled
	Cmd       tea.Cmd // Command to return (typically tea.Quit)
}

// HandleCancelKeys processes ctrl+c and q keys for graceful cancellation.
// Returns a KeyAction indicating whether the key was handled.
func HandleCancelKeys(msg tea.KeyMsg, state *CancellableState) KeyAction {
	switch msg.String() {
	case "ctrl+c", "q":
		if state != nil && state.Cancel != nil {
			state.Cancel()
		}
		return KeyAction{
			Handled:   true,
			Cancelled: true,
			Cmd:       tea.Quit,
		}
	}
	return KeyAction{Handled: false}
}

// -----------------------------------------------------------------------------
// Spinner Factory
// -----------------------------------------------------------------------------

// NewDefaultSpinner creates a spinner with the application's default style.
func NewDefaultSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)
	return s
}

// NewSpinnerWithStyle creates a spinner with a custom spinner style.
func NewSpinnerWithStyle(style spinner.Spinner) spinner.Model {
	s := spinner.New()
	s.Spinner = style
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)
	return s
}

// -----------------------------------------------------------------------------
// Step Status
// -----------------------------------------------------------------------------

// StepStatus represents the status of a progress step.
// This is used across spinner, progress, and multi-step components.
type StepStatus int

const (
	StepPending StepStatus = iota
	StepRunning
	StepComplete
	StepFailed
)

// String returns the string representation of the status
func (s StepStatus) String() string {
	switch s {
	case StepPending:
		return "pending"
	case StepRunning:
		return "running"
	case StepComplete:
		return "complete"
	case StepFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// Icon returns the appropriate icon for the status
func (s StepStatus) Icon() string {
	switch s {
	case StepPending:
		return IconPending
	case StepRunning:
		return IconRunning
	case StepComplete:
		return IconSuccess
	case StepFailed:
		return IconError
	default:
		return IconDot
	}
}

// Style returns the appropriate lipgloss style for the status
func (s StepStatus) Style() lipgloss.Style {
	switch s {
	case StepPending:
		return StyleStepPending
	case StepRunning:
		return StyleStepActive
	case StepComplete:
		return StyleStepComplete
	case StepFailed:
		return StyleStepFailed
	default:
		return StyleMuted
	}
}

// -----------------------------------------------------------------------------
// Step Update (for multi-step operations)
// -----------------------------------------------------------------------------

// StepUpdate represents a progress update for a multi-step operation.
// This is a generic type that can be used by any multi-step process.
type StepUpdate struct {
	Step    string // Name of the step
	Done    bool   // Whether the step is complete
	Success bool   // Whether the step succeeded (only valid when Done is true)
}
