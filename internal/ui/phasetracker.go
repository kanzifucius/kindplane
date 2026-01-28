package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// -----------------------------------------------------------------------------
// Phase Status
// -----------------------------------------------------------------------------

// PhaseStatus represents the status of a phase in a multi-phase operation
type PhaseStatus int

const (
	// PhasePending indicates the phase has not started yet
	PhasePending PhaseStatus = iota
	// PhaseRunning indicates the phase is currently executing
	PhaseRunning
	// PhaseComplete indicates the phase completed successfully
	PhaseComplete
	// PhaseSkipped indicates the phase was skipped
	PhaseSkipped
	// PhaseFailed indicates the phase failed
	PhaseFailed
)

// String returns the string representation of the status
func (s PhaseStatus) String() string {
	switch s {
	case PhasePending:
		return "pending"
	case PhaseRunning:
		return "running"
	case PhaseComplete:
		return "complete"
	case PhaseSkipped:
		return "skipped"
	case PhaseFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// Icon returns the appropriate icon for the status
func (s PhaseStatus) Icon() string {
	switch s {
	case PhasePending:
		return IconPending
	case PhaseRunning:
		return IconRunning
	case PhaseComplete:
		return IconSuccess
	case PhaseSkipped:
		return IconWarning
	case PhaseFailed:
		return IconError
	default:
		return IconDot
	}
}

// -----------------------------------------------------------------------------
// Phase
// -----------------------------------------------------------------------------

// Phase represents a single phase in a multi-phase operation
type Phase struct {
	Name       string
	Status     PhaseStatus
	SkipReason string    // Reason if skipped (e.g., "already exists", "none configured")
	Error      error     // Error if failed
	Message    string    // Completion message (e.g., "2 registry CAs configured")
	StartTime  time.Time // When phase started
	EndTime    time.Time // When phase ended
}

// Duration returns the duration of the phase (or elapsed time if still running)
func (p *Phase) Duration() time.Duration {
	if p.StartTime.IsZero() {
		return 0
	}
	if p.EndTime.IsZero() {
		return time.Since(p.StartTime)
	}
	return p.EndTime.Sub(p.StartTime)
}

// FormatDuration returns a human-readable duration string
func (p *Phase) FormatDuration() string {
	d := p.Duration()
	if d == 0 {
		return "-"
	}
	if d < time.Second {
		return "<1s"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
}

// -----------------------------------------------------------------------------
// PhaseTracker Options
// -----------------------------------------------------------------------------

// PhaseTrackerOption configures a PhaseTracker
type PhaseTrackerOption func(*phaseTrackerOptions)

type phaseTrackerOptions struct {
	output          io.Writer
	icon            string
	showUpfrontList bool
	clusterName     string
	configFile      string
}

func defaultPhaseTrackerOptions() *phaseTrackerOptions {
	return &phaseTrackerOptions{
		output:          os.Stdout,
		icon:            IconRocket,
		showUpfrontList: true,
	}
}

// WithPhaseTrackerOutput sets the output writer for the phase tracker
func WithPhaseTrackerOutput(w io.Writer) PhaseTrackerOption {
	return func(o *phaseTrackerOptions) {
		o.output = w
	}
}

// WithPhaseTrackerIcon sets the icon for the phase tracker header
func WithPhaseTrackerIcon(icon string) PhaseTrackerOption {
	return func(o *phaseTrackerOptions) {
		o.icon = icon
	}
}

// WithShowUpfrontList controls whether to show the phase list upfront
func WithShowUpfrontList(show bool) PhaseTrackerOption {
	return func(o *phaseTrackerOptions) {
		o.showUpfrontList = show
	}
}

// WithClusterInfo sets cluster name and config file for display in header
func WithClusterInfo(clusterName, configFile string) PhaseTrackerOption {
	return func(o *phaseTrackerOptions) {
		o.clusterName = clusterName
		o.configFile = configFile
	}
}

// -----------------------------------------------------------------------------
// PhaseTracker
// -----------------------------------------------------------------------------

// PhaseTracker manages multi-phase operation progress with visual feedback
type PhaseTracker struct {
	title   string
	phases  []*Phase
	current int // Index of current phase (-1 if not started)
	options *phaseTrackerOptions
}

// NewPhaseTracker creates a new phase tracker with the given title
func NewPhaseTracker(title string, opts ...PhaseTrackerOption) *PhaseTracker {
	options := defaultPhaseTrackerOptions()
	for _, opt := range opts {
		opt(options)
	}

	return &PhaseTracker{
		title:   title,
		phases:  make([]*Phase, 0),
		current: -1,
		options: options,
	}
}

// AddPhase adds a new phase to the tracker
func (pt *PhaseTracker) AddPhase(name string) *Phase {
	phase := &Phase{
		Name:   name,
		Status: PhasePending,
	}
	pt.phases = append(pt.phases, phase)
	return phase
}

// AddPhaseIf adds a phase only if the condition is true
// Returns nil if the condition is false
func (pt *PhaseTracker) AddPhaseIf(condition bool, name string) *Phase {
	if !condition {
		return nil
	}
	return pt.AddPhase(name)
}

// PrintHeader prints the command header with title, divider, and optional phase list
func (pt *PhaseTracker) PrintHeader() {
	output := pt.options.output

	fmt.Fprintln(output)
	fmt.Fprintln(output, Title(pt.options.icon+" "+pt.title))
	fmt.Fprintln(output, Divider())
	fmt.Fprintln(output)

	// Print cluster info if provided
	if pt.options.clusterName != "" || pt.options.configFile != "" {
		if pt.options.clusterName != "" {
			fmt.Fprintln(output, KeyValue("Cluster", pt.options.clusterName))
		}
		if pt.options.configFile != "" {
			fmt.Fprintln(output, KeyValue("Config", pt.options.configFile))
		}
		fmt.Fprintln(output)
	}

	// Print upfront phase list
	if pt.options.showUpfrontList && len(pt.phases) > 0 {
		fmt.Fprintln(output, StyleBold.Render("Phases:"))
		for _, phase := range pt.phases {
			icon := phase.Status.Icon()
			fmt.Fprintf(output, "  %s %s\n", StyleMuted.Render(icon), phase.Name)
		}
		fmt.Fprintln(output)
	}
}

// StartPhase marks a phase as running and prints the phase header
// Returns false if the phase is not found
func (pt *PhaseTracker) StartPhase(name string) bool {
	for i, phase := range pt.phases {
		if phase.Name == name {
			phase.Status = PhaseRunning
			phase.StartTime = time.Now()
			pt.current = i

			// Print phase header with [N/M] prefix
			prefix := pt.formatPrefix()
			fmt.Fprintf(pt.options.output, "%s %s\n", StyleBold.Render(prefix), phase.Name)
			return true
		}
	}
	return false
}

// CompletePhase marks the current phase as complete
func (pt *PhaseTracker) CompletePhase() {
	if pt.current >= 0 && pt.current < len(pt.phases) {
		phase := pt.phases[pt.current]
		phase.Status = PhaseComplete
		phase.EndTime = time.Now()
		fmt.Fprintf(pt.options.output, "%s %s\n\n",
			StyleSuccess.Render(IconSuccess),
			phase.Name)
	}
}

// CompletePhaseWithMessage marks the current phase as complete with a custom message
func (pt *PhaseTracker) CompletePhaseWithMessage(message string) {
	if pt.current >= 0 && pt.current < len(pt.phases) {
		phase := pt.phases[pt.current]
		phase.Status = PhaseComplete
		fmt.Fprintf(pt.options.output, "%s %s\n",
			StyleSuccess.Render(IconSuccess),
			phase.Name)
		if message != "" {
			fmt.Fprintf(pt.options.output, "  %s\n", StyleMuted.Render(message))
		}
		fmt.Fprintln(pt.options.output)
	}
}

// -----------------------------------------------------------------------------
// State-Only Methods (for Dashboard integration)
// -----------------------------------------------------------------------------
// These methods update phase state without printing anything.
// Visual feedback is handled by the Dashboard or other UI components.

// MarkPhaseRunning marks a phase as running without printing.
// Use this when visual feedback is handled by another component (e.g., Dashboard).
// Returns false if the phase is not found.
func (pt *PhaseTracker) MarkPhaseRunning(name string) bool {
	for i, phase := range pt.phases {
		if phase.Name == name {
			phase.Status = PhaseRunning
			phase.StartTime = time.Now()
			pt.current = i
			return true
		}
	}
	return false
}

// MarkPhaseComplete marks the current phase as complete without printing.
// Use this when visual feedback is handled by another component.
func (pt *PhaseTracker) MarkPhaseComplete() {
	if pt.current >= 0 && pt.current < len(pt.phases) {
		phase := pt.phases[pt.current]
		phase.Status = PhaseComplete
		phase.EndTime = time.Now()
	}
}

// MarkPhaseCompleteWithMessage marks the current phase as complete with a message, without printing.
// The message is stored and can be displayed by the Dashboard.
func (pt *PhaseTracker) MarkPhaseCompleteWithMessage(message string) {
	if pt.current >= 0 && pt.current < len(pt.phases) {
		phase := pt.phases[pt.current]
		phase.Status = PhaseComplete
		phase.Message = message
		phase.EndTime = time.Now()
	}
}

// MarkPhaseSkipped marks a phase as skipped without printing.
// Use this when visual feedback is handled by another component.
func (pt *PhaseTracker) MarkPhaseSkipped(name string, reason string) bool {
	for i, phase := range pt.phases {
		if phase.Name == name {
			phase.Status = PhaseSkipped
			phase.SkipReason = reason
			phase.EndTime = time.Now()
			pt.current = i
			return true
		}
	}
	return false
}

// MarkPhaseFailed marks the current phase as failed without printing.
// Use this when visual feedback is handled by another component.
func (pt *PhaseTracker) MarkPhaseFailed(err error) {
	if pt.current >= 0 && pt.current < len(pt.phases) {
		phase := pt.phases[pt.current]
		phase.Status = PhaseFailed
		phase.Error = err
		phase.EndTime = time.Now()
	}
}

// SkipPhase marks a phase as skipped with a reason
func (pt *PhaseTracker) SkipPhase(name string, reason string) bool {
	for i, phase := range pt.phases {
		if phase.Name == name {
			phase.Status = PhaseSkipped
			phase.SkipReason = reason
			pt.current = i

			// Print skip message
			prefix := pt.formatPrefix()
			fmt.Fprintf(pt.options.output, "%s %s\n", StyleBold.Render(prefix), phase.Name)
			fmt.Fprintf(pt.options.output, "  %s Skipped (%s)\n\n",
				StyleWarning.Render(IconWarning),
				reason)
			return true
		}
	}
	return false
}

// FailPhase marks the current phase as failed
func (pt *PhaseTracker) FailPhase(err error) {
	if pt.current >= 0 && pt.current < len(pt.phases) {
		phase := pt.phases[pt.current]
		phase.Status = PhaseFailed
		phase.Error = err
		fmt.Fprintf(pt.options.output, "%s %s failed: %v\n\n",
			StyleError.Render(IconError),
			phase.Name,
			err)
	}
}

// PrintSummary prints a summary of all phases with their final status
func (pt *PhaseTracker) PrintSummary() {
	output := pt.options.output

	fmt.Fprintln(output)
	fmt.Fprintln(output, StyleBold.Render("Summary:"))

	for _, phase := range pt.phases {
		var icon, status string
		switch phase.Status {
		case PhaseComplete:
			icon = StyleSuccess.Render(IconSuccess)
			status = phase.Name
		case PhaseSkipped:
			icon = StyleWarning.Render(IconWarning)
			status = fmt.Sprintf("%s (skipped: %s)", phase.Name, phase.SkipReason)
		case PhaseFailed:
			icon = StyleError.Render(IconError)
			status = fmt.Sprintf("%s (failed)", phase.Name)
		case PhaseRunning:
			icon = StyleInfo.Render(IconRunning)
			status = fmt.Sprintf("%s (interrupted)", phase.Name)
		default:
			icon = StyleMuted.Render(IconPending)
			status = fmt.Sprintf("%s (not started)", phase.Name)
		}
		fmt.Fprintf(output, "  %s %s\n", icon, status)
	}
	fmt.Fprintln(output)
}

// PrintSuccess prints the final success message
func (pt *PhaseTracker) PrintSuccess(message string) {
	output := pt.options.output
	fmt.Fprintln(output, Divider())
	fmt.Fprintf(output, "%s %s\n", StyleSuccess.Render(IconSuccess), StyleBold.Render(message))
}

// PrintSuccessWithHint prints the final success message with a hint command
func (pt *PhaseTracker) PrintSuccessWithHint(message, hint string) {
	output := pt.options.output
	fmt.Fprintln(output, Divider())
	fmt.Fprintf(output, "%s %s\n\n", StyleSuccess.Render(IconSuccess), StyleBold.Render(message))
	if hint != "" {
		fmt.Fprintf(output, "  %s\n", Code(hint))
	}
	fmt.Fprintln(output)
}

// -----------------------------------------------------------------------------
// Helper Methods
// -----------------------------------------------------------------------------

// formatPrefix returns the [N/M] prefix for the current phase
func (pt *PhaseTracker) formatPrefix() string {
	activeIndex := pt.activeIndex()
	activeCount := pt.ActiveCount()
	return fmt.Sprintf("[%d/%d]", activeIndex, activeCount)
}

// ActiveCount returns the count of non-skipped phases
func (pt *PhaseTracker) ActiveCount() int {
	count := 0
	for _, phase := range pt.phases {
		if phase.Status != PhaseSkipped {
			count++
		}
	}
	return count
}

// activeIndex returns the 1-based index of the current phase among active phases
func (pt *PhaseTracker) activeIndex() int {
	if pt.current < 0 {
		return 0
	}

	index := 0
	for i := 0; i <= pt.current && i < len(pt.phases); i++ {
		if pt.phases[i].Status != PhaseSkipped {
			index++
		}
	}
	return index
}

// CurrentPhase returns the current phase, or nil if none
func (pt *PhaseTracker) CurrentPhase() *Phase {
	if pt.current >= 0 && pt.current < len(pt.phases) {
		return pt.phases[pt.current]
	}
	return nil
}

// GetPhase returns a phase by name, or nil if not found
func (pt *PhaseTracker) GetPhase(name string) *Phase {
	for _, phase := range pt.phases {
		if phase.Name == name {
			return phase
		}
	}
	return nil
}

// HasFailed returns true if any phase has failed
func (pt *PhaseTracker) HasFailed() bool {
	for _, phase := range pt.phases {
		if phase.Status == PhaseFailed {
			return true
		}
	}
	return false
}

// AllComplete returns true if all phases are complete or skipped
func (pt *PhaseTracker) AllComplete() bool {
	for _, phase := range pt.phases {
		if phase.Status == PhasePending || phase.Status == PhaseRunning {
			return false
		}
		if phase.Status == PhaseFailed {
			return false
		}
	}
	return true
}

// String returns a string representation of all phases
func (pt *PhaseTracker) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("PhaseTracker: %s\n", pt.title))
	for i, phase := range pt.phases {
		marker := " "
		if i == pt.current {
			marker = ">"
		}
		sb.WriteString(fmt.Sprintf("  %s [%s] %s\n", marker, phase.Status, phase.Name))
	}
	return sb.String()
}

// -----------------------------------------------------------------------------
// Accessor Methods (for Dashboard integration)
// -----------------------------------------------------------------------------

// Phases returns all phases (for Dashboard rendering)
func (pt *PhaseTracker) Phases() []*Phase {
	return pt.phases
}

// PhaseCount returns the total number of phases
func (pt *PhaseTracker) PhaseCount() int {
	return len(pt.phases)
}

// CurrentIndex returns the 0-based index of the current phase
func (pt *PhaseTracker) CurrentIndex() int {
	return pt.current
}

// Title returns the tracker title
func (pt *PhaseTracker) Title() string {
	return pt.title
}

// ClusterName returns the cluster name (if set)
func (pt *PhaseTracker) ClusterName() string {
	return pt.options.clusterName
}

// ConfigFile returns the config file path (if set)
func (pt *PhaseTracker) ConfigFile() string {
	return pt.options.configFile
}

// PhaseIndex returns the 1-based phase number for display (includes skipped phases)
func (pt *PhaseTracker) PhaseIndex(phase *Phase) int {
	for i, p := range pt.phases {
		if p == phase {
			return i + 1
		}
	}
	return 0
}
