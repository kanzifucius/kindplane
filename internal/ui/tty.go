package ui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"golang.org/x/term"
)

// ErrCancelled is returned when the user cancels an operation
var ErrCancelled = errors.New("operation cancelled by user")

// -----------------------------------------------------------------------------
// TTY Detection (injectable for testing)
// -----------------------------------------------------------------------------

// TTYDetector is the function signature for TTY detection.
// By default this uses the real terminal check, but can be overridden for testing.
type TTYDetector func() bool

// defaultTTYDetector checks if stdout is a real terminal
var defaultTTYDetector TTYDetector = func() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// ttyDetector is the current TTY detector (can be overridden for testing)
var ttyDetector = defaultTTYDetector

// IsTTY returns true if stdout is a terminal.
// This function uses the configured TTY detector, which can be overridden
// with SetTTYDetector for testing purposes.
func IsTTY() bool {
	return ttyDetector()
}

// SetTTYDetector overrides the TTY detection function.
// This is primarily useful for testing. Call with nil to restore the default.
// Returns a function that restores the previous detector.
func SetTTYDetector(detector TTYDetector) func() {
	prev := ttyDetector
	if detector == nil {
		ttyDetector = defaultTTYDetector
	} else {
		ttyDetector = detector
	}
	return func() { ttyDetector = prev }
}

// -----------------------------------------------------------------------------
// Default Output Writer
// -----------------------------------------------------------------------------

// defaultOutput is the default output writer for the UI package.
// By default this is os.Stdout, but can be overridden for testing.
var defaultOutput io.Writer = os.Stdout

// SetDefaultOutput overrides the default output writer.
// This is primarily useful for testing. Call with nil to restore os.Stdout.
// Returns a function that restores the previous output writer.
func SetDefaultOutput(w io.Writer) func() {
	prev := defaultOutput
	if w == nil {
		defaultOutput = os.Stdout
	} else {
		defaultOutput = w
	}
	return func() { defaultOutput = prev }
}

// GetDefaultOutput returns the current default output writer.
func GetDefaultOutput() io.Writer {
	return defaultOutput
}

// -----------------------------------------------------------------------------
// Default Polling Intervals
// -----------------------------------------------------------------------------

// DefaultPollInterval is the default interval between status polls
var DefaultPollInterval = 2 * time.Second

// DefaultNonTTYPollInterval is the default interval for non-TTY polling
var DefaultNonTTYPollInterval = 3 * time.Second

// DefaultProviderPollInterval is the default interval for provider polling
var DefaultProviderPollInterval = 5 * time.Second

// -----------------------------------------------------------------------------
// Non-TTY Notice
// -----------------------------------------------------------------------------

// nonTTYNoticeOnce ensures the fallback notice is only printed once
var nonTTYNoticeOnce sync.Once

// ResetNonTTYNotice resets the non-TTY notice so it will be printed again.
// This is primarily useful for testing.
func ResetNonTTYNotice() {
	nonTTYNoticeOnce = sync.Once{}
}

// printNonTTYNoticeTo prints the notice to the specified writer
func printNonTTYNoticeTo(w io.Writer) {
	nonTTYNoticeOnce.Do(func() {
		_, _ = fmt.Fprintln(w, StyleMuted.Render("(non-interactive terminal detected, using static output)"))
	})
}

// NonTTYPrinter provides consistent output for non-interactive terminals.
// Use this to ensure consistent formatting across all non-TTY fallbacks.
//
// In non-TTY mode (e.g., piped output, CI environments), the printer:
// - Outputs a one-time notice about non-interactive mode
// - Uses simple text output with icons instead of animated UI
// - Writes to the configured output writer (defaults to stdout)
type NonTTYPrinter struct {
	title  string
	output io.Writer
}

// NonTTYPrinterOption configures a NonTTYPrinter
type NonTTYPrinterOption func(*NonTTYPrinter)

// WithOutput sets the output writer for the printer.
// If not set, defaults to the package's default output (usually os.Stdout).
func WithOutput(w io.Writer) NonTTYPrinterOption {
	return func(p *NonTTYPrinter) {
		p.output = w
	}
}

// NewNonTTYPrinter creates a printer and outputs the initial notice and title.
//
// In non-TTY mode, this immediately prints:
//   - A one-time notice about non-interactive mode (only on first call)
//   - The title with a "running" icon
//
// Options:
//   - WithOutput(w): Set custom output writer (default: package default output)
func NewNonTTYPrinter(title string, opts ...NonTTYPrinterOption) *NonTTYPrinter {
	p := &NonTTYPrinter{
		title:  title,
		output: defaultOutput,
	}
	for _, opt := range opts {
		opt(p)
	}
	printNonTTYNoticeTo(p.output)
	_, _ = fmt.Fprintf(p.output, "%s %s\n", IconRunning, title)
	return p
}

// Success prints a success completion message
func (p *NonTTYPrinter) Success() {
	_, _ = fmt.Fprintf(p.output, "%s %s\n", IconSuccess, p.title)
}

// SuccessWithMessage prints a success message with custom text
func (p *NonTTYPrinter) SuccessWithMessage(msg string) {
	_, _ = fmt.Fprintf(p.output, "%s %s\n", IconSuccess, msg)
}

// Error prints an error completion message
func (p *NonTTYPrinter) Error(err error) {
	_, _ = fmt.Fprintf(p.output, "%s %s: %v\n", IconError, p.title, err)
}

// Cancelled prints a cancellation message
func (p *NonTTYPrinter) Cancelled() {
	_, _ = fmt.Fprintf(p.output, "%s %s (cancelled)\n", IconWarning, p.title)
}

// Step prints an intermediate step with an icon
func (p *NonTTYPrinter) Step(icon, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	_, _ = fmt.Fprintf(p.output, "  %s %s\n", icon, msg)
}

// ItemProgress prints progress for an item (e.g., "  [1/5] item-name")
func (p *NonTTYPrinter) ItemProgress(current, total int, item string) {
	_, _ = fmt.Fprintf(p.output, "  [%d/%d] %s\n", current, total, item)
}

// ItemDone prints a completed item
func (p *NonTTYPrinter) ItemDone() {
	_, _ = fmt.Fprintf(p.output, "  %s Done\n", IconSuccess)
}

// ItemFailed prints a failed item
func (p *NonTTYPrinter) ItemFailed(err error) {
	_, _ = fmt.Fprintf(p.output, "  %s Failed: %v\n", IconError, err)
}

// Print outputs a plain message
func (p *NonTTYPrinter) Print(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(p.output, format+"\n", args...)
}
