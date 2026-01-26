package kind

import (
	"fmt"
	"strings"

	"sigs.k8s.io/kind/pkg/log"
)

// StepUpdate represents a progress step update from kind
type StepUpdate struct {
	Step    string
	Done    bool
	Success bool
}

// Logger implements kind's log.Logger interface and sends step updates via channel
type Logger struct {
	updates chan<- StepUpdate
	level   log.Level
}

// NewLogger creates a new Logger that sends updates to the provided channel
func NewLogger(updates chan<- StepUpdate) *Logger {
	return &Logger{
		updates: updates,
		level:   1, // Capture V(0) and V(1) messages to get all progress updates
	}
}

// Warn implements log.Logger
func (l *Logger) Warn(message string) {
	// Warnings are typically not progress steps, but we can send them
	l.updates <- StepUpdate{
		Step:    fmt.Sprintf("Warning: %s", message),
		Done:    true,
		Success: false,
	}
}

// Warnf implements log.Logger
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Warn(fmt.Sprintf(format, args...))
}

// Error implements log.Logger
func (l *Logger) Error(message string) {
	l.updates <- StepUpdate{
		Step:    fmt.Sprintf("Error: %s", message),
		Done:    true,
		Success: false,
	}
}

// Errorf implements log.Logger
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Error(fmt.Sprintf(format, args...))
}

// V implements log.Logger and returns an InfoLogger for the given verbosity level
func (l *Logger) V(level log.Level) log.InfoLogger {
	return &infoLogger{
		logger:  l,
		level:   level,
		enabled: level <= l.level, // Only enable V(0) and below
	}
}

// infoLogger implements log.InfoLogger
type infoLogger struct {
	logger  *Logger
	level   log.Level
	enabled bool
}

// Enabled implements log.InfoLogger
func (i *infoLogger) Enabled() bool {
	return i.enabled
}

// Info implements log.InfoLogger
func (i *infoLogger) Info(message string) {
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
	isInProgress := strings.Contains(message, "...") || strings.Contains(message, " • ")
	isSuccess := isDone && !strings.Contains(strings.ToLower(message), "error")
	
	// Send update - mark as done only if it's a completion message
	i.logger.updates <- StepUpdate{
		Step:    step,
		Done:    isDone && !isInProgress, // Don't mark as done if it's still in progress
		Success: isSuccess,
	}
}

// Infof implements log.InfoLogger
func (i *infoLogger) Infof(format string, args ...interface{}) {
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
