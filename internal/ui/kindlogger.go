package ui

import (
	"fmt"
	"strings"
	"sync"

	"sigs.k8s.io/kind/pkg/log"
)

// KindLogger implements kind's log.Logger interface and sends step updates via channel.
// If done is non-nil, sends are guarded with a select so that no send occurs after done is closed (avoids panic on closed channel).
type KindLogger struct {
	mu      sync.Mutex
	updates chan<- StepUpdate
	done    <-chan struct{} // when closed, logger stops sending (optional); set to nil under mu once observed closed
	closed  bool            // true once done was observed closed; protected by mu; no sends after this
	level   log.Level
}

// NewKindLogger creates a new Logger that sends updates to the provided channel.
// If done is non-nil, it must be closed by the receiver when no more updates are expected; the logger will then avoid sending to a closed channel.
func NewKindLogger(updates chan<- StepUpdate, done <-chan struct{}) *KindLogger {
	return &KindLogger{
		updates: updates,
		done:    done,
		level:   1, // Capture V(0) and V(1) messages to get all progress updates
	}
}

// sendUpdate sends an update without panicking if the channel was closed (when done is used).
// Uses mu to guard l.done and l.closed: we read both into locals under lock. If closed is true we drop.
// If done is nil we re-check closed under lock and only then send; otherwise we select between send and <-done; if done fires we set l.done = nil and l.closed = true under mu.
func (l *KindLogger) sendUpdate(u StepUpdate) {
	l.mu.Lock()
	doneCh := l.done
	closed := l.closed
	l.mu.Unlock()

	if closed {
		return
	}
	if doneCh == nil {
		l.mu.Lock()
		if l.closed {
			l.mu.Unlock()
			return
		}
		l.mu.Unlock()
		l.updates <- u
		return
	}
	select {
	case l.updates <- u:
	case <-doneCh:
		l.mu.Lock()
		l.done = nil
		l.closed = true
		l.mu.Unlock()
	}
}

// Warn implements log.Logger
func (l *KindLogger) Warn(message string) {
	l.sendUpdate(StepUpdate{
		Step:    fmt.Sprintf("Warning: %s", message),
		Done:    true,
		Success: false,
	})
}

// Warnf implements log.Logger
func (l *KindLogger) Warnf(format string, args ...interface{}) {
	l.Warn(fmt.Sprintf(format, args...))
}

// Error implements log.Logger
func (l *KindLogger) Error(message string) {
	l.sendUpdate(StepUpdate{
		Step:    fmt.Sprintf("Error: %s", message),
		Done:    true,
		Success: false,
	})
}

// Errorf implements log.Logger
func (l *KindLogger) Errorf(format string, args ...interface{}) {
	l.Error(fmt.Sprintf(format, args...))
}

// V implements log.Logger and returns an InfoLogger for the given verbosity level
func (l *KindLogger) V(level log.Level) log.InfoLogger {
	return &kindInfoLogger{
		logger:  l,
		level:   level,
		enabled: level <= l.level, // Enable V(0) and V(1) for progress updates
	}
}

// kindInfoLogger implements log.InfoLogger
type kindInfoLogger struct {
	logger  *KindLogger
	level   log.Level
	enabled bool
}

// Enabled implements log.InfoLogger
func (i *kindInfoLogger) Enabled() bool {
	return i.enabled
}

// Info implements log.InfoLogger
func (i *kindInfoLogger) Info(message string) {
	if !i.enabled {
		return
	}

	// Parse kind's status messages
	// Kind typically sends messages like:
	// - " ✓ Ensuring node image (kindest/node:v1.27.1)"
	// - " • Preparing nodes  ..."
	// - " ✓ Preparing nodes"
	// - " ✓ Writing configuration 📜"
	// - " • Joining worker nodes  ..."
	// - " ✓ Joining worker nodes"

	step := parseKindMessage(message)
	if step == "" {
		// If parsing failed, try to extract any meaningful step name
		// Some messages might not have the standard format
		trimmed := strings.TrimSpace(message)
		// Skip empty messages or very short ones
		if len(trimmed) < 3 {
			return
		}

		// Check if this looks like a progress-related message
		// Look for common progress keywords
		lower := strings.ToLower(trimmed)
		isProgressMessage := strings.Contains(lower, "joining") ||
			strings.Contains(lower, "waiting") ||
			strings.Contains(lower, "starting") ||
			strings.Contains(lower, "preparing") ||
			strings.Contains(lower, "ensuring") ||
			strings.Contains(lower, "writing") ||
			strings.Contains(lower, "installing") ||
			strings.Contains(lower, "configuring") ||
			strings.Contains(lower, "creating") ||
			strings.Contains(lower, "setting up")

		if isProgressMessage {
			// Clean up the message to use as step name
			step = trimmed
			// Remove common prefixes/suffixes
			step = strings.TrimPrefix(step, "•")
			step = strings.TrimPrefix(step, "✓")
			step = strings.TrimPrefix(step, "✗")
			step = strings.TrimSuffix(step, "...")
			step = strings.TrimSpace(step)
			// Extract before parentheses if present
			if idx := strings.Index(step, "("); idx > 0 {
				step = strings.TrimSpace(step[:idx])
			}
		} else {
			// Skip messages that don't look like progress steps
			return
		}
	}

	// Check if this is a completion message (starts with ✓ or contains " ✓ ")
	// Also check for "..." which indicates in-progress
	trimmedMsg := strings.TrimSpace(message)
	isDone := strings.HasPrefix(trimmedMsg, "✓") ||
		strings.Contains(message, " ✓ ") ||
		(strings.HasSuffix(trimmedMsg, "✓") && !strings.Contains(trimmedMsg, "..."))
	isFailed := strings.HasPrefix(trimmedMsg, "✗") || strings.Contains(message, " ✗ ")
	isInProgress := strings.Contains(message, "...") || strings.Contains(message, " • ")
	isSuccess := isDone && !strings.Contains(strings.ToLower(message), "error")

	// Send update - mark as done only if it's a completion message
	i.logger.sendUpdate(StepUpdate{
		Step:    step,
		Done:    (isDone || isFailed) && !isInProgress, // Don't mark as done if it's still in progress
		Success: isSuccess && !isFailed,
	})
}

// Infof implements log.InfoLogger
func (i *kindInfoLogger) Infof(format string, args ...interface{}) {
	if i.enabled {
		i.Info(fmt.Sprintf(format, args...))
	}
}

// parseKindMessage extracts the step name from kind's log messages
// Kind messages can be:
// - " ✓ Ensuring node image (kindest/node:v1.27.1)"
// - " • Preparing nodes  ..."
// - " ✓ Preparing nodes"
// - " ✓ Writing configuration 📜"
// - " • Joining worker nodes  ..."
// - " ✓ Joining worker nodes"
func parseKindMessage(message string) string {
	// Remove leading/trailing whitespace
	message = strings.TrimSpace(message)

	// Skip empty messages
	if message == "" {
		return ""
	}

	// Remove status indicators (✓, ✗, •, etc.) from the beginning
	// Handle both " ✓ " and "✓" patterns
	message = strings.TrimPrefix(message, "✓")
	message = strings.TrimPrefix(message, "✗")
	message = strings.TrimPrefix(message, "•")
	// Also handle patterns like " ✓ " in the middle
	message = strings.ReplaceAll(message, " ✓ ", " ")
	message = strings.ReplaceAll(message, " ✗ ", " ")
	message = strings.ReplaceAll(message, " • ", " ")
	message = strings.TrimSpace(message)

	// Remove trailing "..." or " ..."
	message = strings.TrimSuffix(message, "...")
	message = strings.TrimSuffix(message, " ...")
	message = strings.TrimSpace(message)

	// Remove emojis (simple approach - remove common ones)
	emojis := []string{"📦", "📜", "🕹️", "🔌", "💾", "🚀", "⚙️"}
	for _, emoji := range emojis {
		message = strings.ReplaceAll(message, emoji, "")
	}
	message = strings.TrimSpace(message)

	// Extract the main step name (before any parenthetical info)
	if idx := strings.Index(message, "("); idx > 0 {
		message = strings.TrimSpace(message[:idx])
	}

	// Skip very short messages that are likely not step names
	if len(message) < 3 {
		return ""
	}

	return message
}

// -----------------------------------------------------------------------------
// Dashboard Logger (for TUI Dashboard mode)
// -----------------------------------------------------------------------------

// KindDashboardLogger implements kind's log.Logger interface and sends updates via callback
type KindDashboardLogger struct {
	updateFn func(step string)
	level    log.Level
}

// NewKindDashboardLogger creates a logger that sends updates to a callback function.
// If updateFn is nil, a no-op is used so methods never panic.
func NewKindDashboardLogger(updateFn func(step string)) *KindDashboardLogger {
	if updateFn == nil {
		updateFn = func(string) {}
	}
	return &KindDashboardLogger{
		updateFn: updateFn,
		level:    1,
	}
}

// Warn implements log.Logger
func (l *KindDashboardLogger) Warn(message string) {
	l.updateFn(fmt.Sprintf("Warning: %s", message))
}

// Warnf implements log.Logger
func (l *KindDashboardLogger) Warnf(format string, args ...interface{}) {
	l.Warn(fmt.Sprintf(format, args...))
}

// Error implements log.Logger
func (l *KindDashboardLogger) Error(message string) {
	l.updateFn(fmt.Sprintf("Error: %s", message))
}

// Errorf implements log.Logger
func (l *KindDashboardLogger) Errorf(format string, args ...interface{}) {
	l.Error(fmt.Sprintf(format, args...))
}

// V implements log.Logger
func (l *KindDashboardLogger) V(level log.Level) log.InfoLogger {
	return &kindDashboardInfoLogger{
		logger:  l,
		enabled: level <= l.level,
	}
}

// kindDashboardInfoLogger implements log.InfoLogger for KindDashboardLogger
type kindDashboardInfoLogger struct {
	logger  *KindDashboardLogger
	enabled bool
}

// Enabled implements log.InfoLogger
func (i *kindDashboardInfoLogger) Enabled() bool {
	return i.enabled
}

// Info implements log.InfoLogger
func (i *kindDashboardInfoLogger) Info(message string) {
	if !i.enabled {
		return
	}

	step := parseKindMessage(message)
	if step == "" {
		return
	}

	i.logger.updateFn(step)
}

// Infof implements log.InfoLogger
func (i *kindDashboardInfoLogger) Infof(format string, args ...interface{}) {
	if i.enabled {
		i.Info(fmt.Sprintf(format, args...))
	}
}
