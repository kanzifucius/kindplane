package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// -----------------------------------------------------------------------------
// Dashboard Key Bindings
// -----------------------------------------------------------------------------

// dashboardKeyMap defines the key bindings for the dashboard
type dashboardKeyMap struct {
	Verbose key.Binding
	Pods    key.Binding
	Extend  key.Binding
	Quit    key.Binding
}

// ShortHelp returns key bindings for the short help view (footer)
func (k dashboardKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Verbose, k.Pods, k.Extend, k.Quit}
}

// FullHelp returns key bindings for the full help view (not used)
func (k dashboardKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Verbose, k.Pods}, {k.Extend, k.Quit}}
}

// -----------------------------------------------------------------------------
// Dashboard Messages
// -----------------------------------------------------------------------------

// DashboardMsg is the base interface for dashboard messages
type DashboardMsg interface {
	isDashboardMsg()
}

// PhaseStartedMsg indicates a phase has started
type PhaseStartedMsg struct {
	PhaseName string
}

func (PhaseStartedMsg) isDashboardMsg() {}

// PhaseCompletedMsg indicates a phase has completed
type PhaseCompletedMsg struct {
	PhaseName string
	Message   string // Optional completion message
}

func (PhaseCompletedMsg) isDashboardMsg() {}

// PhaseSkippedMsg indicates a phase was skipped
type PhaseSkippedMsg struct {
	PhaseName string
	Reason    string
}

func (PhaseSkippedMsg) isDashboardMsg() {}

// PhaseFailedMsg indicates a phase has failed
type PhaseFailedMsg struct {
	PhaseName string
	Error     error
}

func (PhaseFailedMsg) isDashboardMsg() {}

// OperationUpdateMsg updates the current operation display
type OperationUpdateMsg struct {
	Step     string  // Current step description
	Progress float64 // 0.0 to 1.0, or -1 for indeterminate
}

func (OperationUpdateMsg) isDashboardMsg() {}

// LogLineMsg adds a line to the log buffer
type LogLineMsg struct {
	Line string
}

func (LogLineMsg) isDashboardMsg() {}

// TimeoutExtendedMsg indicates the timeout was extended
type TimeoutExtendedMsg struct {
	NewDeadline time.Time
}

func (TimeoutExtendedMsg) isDashboardMsg() {}

// BootstrapCompleteMsg indicates the entire bootstrap is complete
type BootstrapCompleteMsg struct {
	Success      bool
	Message      string
	Error        error
	NextStepHint string // Command hint for next steps (e.g., "kubectl cluster-info --context ...")
}

func (BootstrapCompleteMsg) isDashboardMsg() {}

// PodStatusUpdateMsg updates the pod status display
type PodStatusUpdateMsg struct {
	Pods []PodInfo
}

func (PodStatusUpdateMsg) isDashboardMsg() {}

// tickMsg is sent periodically to update elapsed time and timeout
type tickMsg time.Time

// -----------------------------------------------------------------------------
// Dashboard Model
// -----------------------------------------------------------------------------

// DashboardModel is the Bubble Tea model for the bootstrap dashboard
type DashboardModel struct {
	// Phase tracking
	tracker *PhaseTracker

	// Timeout management
	startTime    time.Time
	deadline     time.Time
	showExtend   bool // Show "press e to extend" prompt
	extendAmount time.Duration

	// Current operation state
	currentStep     string  // Current sub-step being executed
	currentProgress float64 // -1 for spinner, 0-1 for progress bar

	// UI components
	spinner  spinner.Model
	progress progress.Model
	viewport viewport.Model // For verbose log view

	// Log buffer
	logLines      []string
	logAutoScroll bool // Auto-scroll to bottom on new logs (disabled when user scrolls up)

	// UI state
	verbose   bool // Show detailed log output
	showPods  bool // Show pods status panel
	width     int  // Terminal width
	height    int  // Terminal height
	quitting  bool
	completed bool
	success   bool
	result    BootstrapCompleteMsg

	// Completion display
	nextStepHint string // Hint command to show on success (e.g., "kubectl cluster-info --context ...")

	// Pod status
	pods []PodInfo

	// Help component
	help   help.Model
	keyMap dashboardKeyMap

	// Cancellation
	state *CancellableState
}

// DashboardOption configures the dashboard
type DashboardOption func(*DashboardModel)

// WithTimeout sets the bootstrap timeout
func WithTimeout(timeout time.Duration) DashboardOption {
	return func(m *DashboardModel) {
		m.deadline = m.startTime.Add(timeout)
	}
}

// WithExtendAmount sets the amount of time to extend when user presses 'e'
func WithExtendAmount(amount time.Duration) DashboardOption {
	return func(m *DashboardModel) {
		m.extendAmount = amount
	}
}

// WithNextStepHint sets the command hint to display on successful completion
func WithNextStepHint(hint string) DashboardOption {
	return func(m *DashboardModel) {
		m.nextStepHint = hint
	}
}

// NewDashboardModel creates a new dashboard model
func NewDashboardModel(tracker *PhaseTracker, parentCtx context.Context, opts ...DashboardOption) DashboardModel {
	state := NewCancellableState(parentCtx)

	// Create spinner
	s := NewDefaultSpinner()

	// Create progress bar with gradient
	prog := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
	)

	// Create viewport for log scrolling with default dimensions
	vp := viewport.New(DashboardMinWidth-4, DashboardLogBuffer)

	// Create help model with styled keybindings
	h := help.New()
	h.ShortSeparator = "  "
	h.Styles.ShortKey = lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Bold(true)
	h.Styles.ShortDesc = lipgloss.NewStyle().
		Foreground(ColorMuted)
	h.Styles.ShortSeparator = lipgloss.NewStyle().
		Foreground(ColorDim)

	// Create key bindings
	keys := dashboardKeyMap{
		Verbose: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "verbose"),
		),
		Pods: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "pods"),
		),
		Extend: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "extend"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
	}
	// Start with extend disabled (enabled when timeout approaching)
	keys.Extend.SetEnabled(false)

	m := DashboardModel{
		tracker:         tracker,
		startTime:       time.Now(),
		deadline:        time.Now().Add(10 * time.Minute), // Default 10 min timeout
		extendAmount:    5 * time.Minute,                  // Default extend by 5 min
		currentProgress: -1,                               // Start with spinner
		spinner:         s,
		progress:        prog,
		viewport:        vp,
		logLines:        make([]string, 0, DashboardLogBuffer),
		logAutoScroll:   true, // Start with auto-scroll enabled
		help:            h,
		keyMap:          keys,
		state:           state,
		width:           DashboardMinWidth,
		height:          24,
	}

	for _, opt := range opts {
		opt(&m)
	}

	return m
}

// Init implements tea.Model
func (m DashboardModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.tickCmd(),
	)
}

func (m DashboardModel) tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Update implements tea.Model
func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If completed, any key exits
		if m.completed {
			m.quitting = true
			return m, tea.Quit
		}

		switch msg.String() {
		case "ctrl+c", "q":
			if m.state != nil && m.state.Cancel != nil {
				m.state.Cancel()
			}
			m.quitting = true
			return m, tea.Quit

		case "v":
			m.verbose = !m.verbose
			// Update key binding description
			if m.verbose {
				m.keyMap.Verbose.SetHelp("v", "compact")
			} else {
				m.keyMap.Verbose.SetHelp("v", "verbose")
			}
			return m, nil

		case "e":
			// Extend timeout
			if !m.completed && time.Until(m.deadline) < 2*time.Minute {
				m.deadline = m.deadline.Add(m.extendAmount)
				m.showExtend = false
				m.addLogLine(fmt.Sprintf("Timeout extended by %s", m.extendAmount))
			}
			return m, nil

		case "p":
			m.showPods = !m.showPods
			// Update key binding description
			if m.showPods {
				m.keyMap.Pods.SetHelp("p", "hide pods")
			} else {
				m.keyMap.Pods.SetHelp("p", "pods")
			}
			return m, nil

		// Log viewport scrolling (only when verbose mode is active)
		case "up", "k":
			if m.verbose {
				m.viewport.LineUp(1)
				m.logAutoScroll = m.viewport.AtBottom()
				return m, nil
			}

		case "down", "j":
			if m.verbose {
				m.viewport.LineDown(1)
				m.logAutoScroll = m.viewport.AtBottom()
				return m, nil
			}

		case "pgup", "ctrl+u":
			if m.verbose {
				m.viewport.HalfViewUp()
				m.logAutoScroll = m.viewport.AtBottom()
				return m, nil
			}

		case "pgdown", "ctrl+d":
			if m.verbose {
				m.viewport.HalfViewDown()
				m.logAutoScroll = m.viewport.AtBottom()
				return m, nil
			}

		case "home", "g":
			if m.verbose {
				m.viewport.GotoTop()
				m.logAutoScroll = false
				return m, nil
			}

		case "end", "G":
			if m.verbose {
				m.viewport.GotoBottom()
				m.logAutoScroll = true
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Update viewport for verbose mode
		m.viewport.Width = DashboardWidth(m.width) - 4
		m.viewport.Height = DashboardLogBuffer
		return m, nil

	case tickMsg:
		// Check if timeout is approaching
		remaining := time.Until(m.deadline)
		if remaining < 2*time.Minute && remaining > 0 {
			m.showExtend = true
			m.keyMap.Extend.SetEnabled(true)
		} else {
			m.showExtend = false
			m.keyMap.Extend.SetEnabled(false)
		}

		// Check if timed out
		if remaining <= 0 && !m.completed {
			// Cancel background work context before quitting
			if m.state != nil && m.state.Cancel != nil {
				m.state.Cancel()
			}
			m.result = BootstrapCompleteMsg{
				Success: false,
				Message: "Bootstrap timed out",
				Error:   fmt.Errorf("operation timed out after %s", time.Since(m.startTime).Round(time.Second)),
			}
			m.completed = true
			m.success = false
			return m, tea.Quit
		}

		return m, m.tickCmd()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		cmds = append(cmds, cmd)

	// Dashboard-specific messages
	case PhaseStartedMsg:
		m.tracker.MarkPhaseRunning(msg.PhaseName)
		m.currentStep = ""
		m.currentProgress = -1
		m.addLogLine(fmt.Sprintf("Starting: %s", msg.PhaseName))
		cmds = append(cmds, m.spinner.Tick)

	case PhaseCompletedMsg:
		if msg.Message != "" {
			m.tracker.MarkPhaseCompleteWithMessage(msg.Message)
			m.addLogLine(fmt.Sprintf("Completed: %s (%s)", msg.PhaseName, msg.Message))
		} else {
			m.tracker.MarkPhaseComplete()
			m.addLogLine(fmt.Sprintf("Completed: %s", msg.PhaseName))
		}
		m.currentStep = ""
		m.currentProgress = -1

	case PhaseSkippedMsg:
		m.tracker.MarkPhaseSkipped(msg.PhaseName, msg.Reason)
		m.addLogLine(fmt.Sprintf("Skipped: %s (%s)", msg.PhaseName, msg.Reason))

	case PhaseFailedMsg:
		m.tracker.MarkPhaseFailed(msg.Error)
		m.addLogLine(fmt.Sprintf("Failed: %s - %v", msg.PhaseName, msg.Error))
		m.result = BootstrapCompleteMsg{
			Success: false,
			Message: fmt.Sprintf("Phase '%s' failed", msg.PhaseName),
			Error:   msg.Error,
		}
		m.completed = true
		m.success = false
		return m, tea.Quit

	case OperationUpdateMsg:
		m.currentStep = msg.Step
		m.currentProgress = msg.Progress
		if msg.Progress >= 0 && msg.Progress <= 1 {
			cmds = append(cmds, m.progress.SetPercent(msg.Progress))
		}
		if msg.Step != "" {
			m.addLogLine(fmt.Sprintf("  %s", msg.Step))
		}

	case LogLineMsg:
		m.addLogLine(msg.Line)

	case PodStatusUpdateMsg:
		m.pods = msg.Pods

	case TimeoutExtendedMsg:
		m.deadline = msg.NewDeadline
		m.showExtend = false
		m.addLogLine(fmt.Sprintf("Timeout extended to %s", msg.NewDeadline.Format("15:04:05")))

	case BootstrapCompleteMsg:
		m.result = msg
		m.completed = true
		m.success = msg.Success
		if msg.Success {
			m.addLogLine("Bootstrap completed successfully!")
		} else if msg.Error != nil {
			m.addLogLine(fmt.Sprintf("Bootstrap failed: %v", msg.Error))
		}
		// Don't quit immediately - show completion summary view
		// User will press any key to exit
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

func (m *DashboardModel) addLogLine(line string) {
	m.logLines = append(m.logLines, line)
	// Keep buffer size limited (larger to allow scroll history)
	maxBuffer := DashboardLogBuffer * 10 // Store up to 150 lines of history
	if len(m.logLines) > maxBuffer {
		m.logLines = m.logLines[len(m.logLines)-maxBuffer:]
	}
	// Update viewport content with wrapped lines
	wrappedLines := make([]string, len(m.logLines))
	wrapStyle := lipgloss.NewStyle().Width(m.viewport.Width)
	for i, l := range m.logLines {
		wrappedLines[i] = wrapStyle.Render(l)
	}
	m.viewport.SetContent(strings.Join(wrappedLines, "\n"))
	// Only auto-scroll if user hasn't scrolled up manually
	if m.logAutoScroll {
		m.viewport.GotoBottom()
	}
}

// View implements tea.Model
func (m DashboardModel) View() string {
	if m.quitting {
		return ""
	}

	width := DashboardWidth(m.width)

	// Show completion view if bootstrap is complete
	if m.completed {
		return m.renderCompletionView(width)
	}

	var sections []string

	// Header section
	sections = append(sections, m.renderHeader(width))

	// Phase table
	sections = append(sections, m.renderPhaseTable(width))

	// Current operation panel (only if a phase is running)
	if current := m.tracker.CurrentPhase(); current != nil && current.Status == PhaseRunning {
		sections = append(sections, m.renderCurrentOperation(width))
	}

	// Verbose log panel
	if m.verbose {
		sections = append(sections, m.renderLogPanel(width))
	}

	// Pods status panel
	if m.showPods {
		sections = append(sections, m.renderPodsPanel(width))
	}

	// Footer
	sections = append(sections, m.renderFooter(width))

	return strings.Join(sections, "\n")
}

func (m DashboardModel) renderHeader(width int) string {
	// Title line
	titleIcon := IconRocket
	title := fmt.Sprintf("%s Bootstrap Cluster", titleIcon)

	// Cluster info
	clusterLine := ""
	if m.tracker.ClusterName() != "" || m.tracker.ConfigFile() != "" {
		parts := []string{}
		if m.tracker.ClusterName() != "" {
			parts = append(parts, StyleDashboardLabel.Render("Cluster: ")+StyleDashboardValue.Render(m.tracker.ClusterName()))
		}
		if m.tracker.ConfigFile() != "" {
			parts = append(parts, StyleDashboardLabel.Render("Config: ")+StyleDashboardValue.Render(m.tracker.ConfigFile()))
		}
		clusterLine = strings.Join(parts, "    ")
	}

	// Timeout line
	remaining := time.Until(m.deadline)
	var timeoutStr string
	var timeoutStyle lipgloss.Style
	if remaining < 0 {
		timeoutStr = "EXPIRED"
		timeoutStyle = StyleDashboardTimeoutCritical
	} else if remaining < 1*time.Minute {
		timeoutStr = fmt.Sprintf("%s remaining", remaining.Round(time.Second))
		timeoutStyle = StyleDashboardTimeoutCritical
	} else if remaining < 2*time.Minute {
		timeoutStr = fmt.Sprintf("%s remaining", remaining.Round(time.Second))
		timeoutStyle = StyleDashboardTimeoutWarning
	} else {
		mins := int(remaining.Minutes())
		secs := int(remaining.Seconds()) % 60
		timeoutStr = fmt.Sprintf("%dm %ds remaining", mins, secs)
		timeoutStyle = StyleDashboardTimeoutOk
	}
	timeoutLine := StyleDashboardLabel.Render("Timeout: ") + timeoutStyle.Render(timeoutStr)

	if m.showExtend {
		timeoutLine += StyleWarning.Render(" (press 'e' to extend)")
	}

	// Build content
	content := StyleBold.Render(title) + "\n"
	if clusterLine != "" {
		content += clusterLine + "\n"
	}
	content += timeoutLine

	// Apply box style
	box := StyleDashboardHeader.Width(width - 2).Render(content)
	return box
}

func (m DashboardModel) renderPhaseTable(width int) string {
	phases := m.tracker.Phases()
	if len(phases) == 0 {
		return ""
	}

	// Column widths
	numCol := 3
	statusCol := 12
	timeCol := 8
	messageCol := 20
	phaseCol := width - numCol - statusCol - timeCol - messageCol - 12 // padding

	// Header
	header := fmt.Sprintf(
		"%s  %s  %s  %s  %s",
		StyleDashboardPhaseHeader.Width(numCol).Render("#"),
		StyleDashboardPhaseHeader.Width(phaseCol).Render("Phase"),
		StyleDashboardPhaseHeader.Width(statusCol).Render("Status"),
		StyleDashboardPhaseHeader.Width(timeCol).Render("Time"),
		StyleDashboardPhaseHeader.Width(messageCol).Render("Message"),
	)

	// Rows
	var rows []string
	rows = append(rows, header)
	rows = append(rows, strings.Repeat("-", width-4))

	for i, phase := range phases {
		num := fmt.Sprintf("%d", i+1)

		// Status with icon
		var statusIcon, statusText string
		switch phase.Status {
		case PhasePending:
			statusIcon = IconPending
			statusText = "Pending"
		case PhaseRunning:
			statusIcon = m.spinner.View()
			statusText = "Running"
		case PhaseComplete:
			statusIcon = StyleSuccess.Render(IconSuccess)
			statusText = "Complete"
		case PhaseSkipped:
			statusIcon = StyleWarning.Render(IconWarning)
			statusText = "Skipped"
		case PhaseFailed:
			statusIcon = StyleError.Render(IconError)
			statusText = "Failed"
		}
		status := fmt.Sprintf("%s %s", statusIcon, statusText)

		// Time
		timeStr := phase.FormatDuration()

		// Message
		message := phase.Message
		if phase.Status == PhaseSkipped && phase.SkipReason != "" {
			message = phase.SkipReason
		}
		if len(message) > messageCol {
			message = message[:messageCol-3] + "..."
		}

		// Apply row style based on status
		rowStyle := PhaseStatusStyle(phase.Status)
		row := fmt.Sprintf(
			"%s  %s  %s  %s  %s",
			rowStyle.Width(numCol).Render(num),
			rowStyle.Width(phaseCol).Render(phase.Name),
			lipgloss.NewStyle().Width(statusCol).Render(status),
			rowStyle.Width(timeCol).Render(timeStr),
			StyleMuted.Width(messageCol).Render(message),
		)
		rows = append(rows, row)
	}

	content := strings.Join(rows, "\n")
	box := StyleDashboardBox.Width(width - 2).Render(content)
	return box
}

func (m DashboardModel) renderCurrentOperation(width int) string {
	current := m.tracker.CurrentPhase()
	if current == nil {
		return ""
	}

	title := current.Name

	var content string
	if m.currentStep != "" {
		content = fmt.Sprintf("%s %s", m.spinner.View(), m.currentStep)
	} else {
		content = fmt.Sprintf("%s Working...", m.spinner.View())
	}

	// Progress bar or spinner
	if m.currentProgress >= 0 && m.currentProgress <= 1 {
		// Show progress bar
		content += "\n\n" + m.progress.ViewAs(m.currentProgress)
	}

	// Render box with consistent width
	box := StyleDashboardOperationBox.Width(width - 2).Render(content)

	// Add title to border
	box = insertBorderTitle(box, title, StyleBold)

	return box
}

func (m DashboardModel) renderLogPanel(width int) string {
	// Use viewport for scrollable log view
	content := m.viewport.View()
	if len(m.logLines) == 0 {
		content = StyleMuted.Render("No log output yet...")
	}

	// Render box with consistent width
	box := StyleDashboardLogBox.Width(width - 2).Render(content)

	// Build title with scroll indicator
	title := "Logs"
	if len(m.logLines) > 0 && !m.viewport.AtBottom() {
		// Show indicator that there are more logs below
		title = fmt.Sprintf("Logs [%d/%d]", m.viewport.YOffset+DashboardLogBuffer, len(m.logLines))
	} else if !m.viewport.AtTop() && len(m.logLines) > DashboardLogBuffer {
		// Show total count when at bottom but there's history above
		title = fmt.Sprintf("Logs [%d lines]", len(m.logLines))
	}

	// Add title to border
	box = insertBorderTitle(box, title, StyleMuted)

	return box
}

func (m DashboardModel) renderPodsPanel(width int) string {
	if len(m.pods) == 0 {
		content := StyleMuted.Render("No pods yet...")
		box := StyleDashboardLogBox.Width(width - 2).Render(content)
		box = insertBorderTitle(box, "Pods", StyleMuted)
		return box
	}

	// Column widths
	podCol := width - 36 // Remaining space for pod name
	statusCol := 12
	readyCol := 12

	// Header
	header := fmt.Sprintf(
		"%s  %s  %s",
		StyleDashboardPhaseHeader.Width(podCol).Render("Pod"),
		StyleDashboardPhaseHeader.Width(statusCol).Render("Status"),
		StyleDashboardPhaseHeader.Width(readyCol).Render("Ready"),
	)

	// Rows
	var rows []string
	rows = append(rows, header)
	rows = append(rows, strings.Repeat("─", width-6))

	readyCount := 0
	for _, pod := range m.pods {
		if pod.Ready {
			readyCount++
		}

		// Format pod name (truncate if too long)
		podName := pod.Name
		if len(podName) > podCol {
			podName = podName[:podCol-3] + "..."
		}

		// Format status
		status := pod.Status
		if len(status) > statusCol {
			status = status[:statusCol-3] + "..."
		}

		// Format ready indicator
		var readyStr string
		if pod.Ready {
			readyStr = StyleSuccess.Render(IconSuccess) + " Ready"
		} else if pod.Status == "Failed" {
			readyStr = StyleError.Render(IconError) + " Failed"
		} else if pod.Status == "Pending" {
			readyStr = IconPending + " Pending"
		} else {
			readyStr = m.spinner.View() + " Waiting"
		}

		row := fmt.Sprintf(
			"%s  %s  %s",
			lipgloss.NewStyle().Width(podCol).Render(podName),
			lipgloss.NewStyle().Width(statusCol).Render(status),
			lipgloss.NewStyle().Width(readyCol).Render(readyStr),
		)
		rows = append(rows, row)
	}

	// Add summary line
	rows = append(rows, "")
	summaryLine := StyleMuted.Render(fmt.Sprintf("%d/%d pods ready", readyCount, len(m.pods)))
	rows = append(rows, summaryLine)

	content := strings.Join(rows, "\n")
	box := StyleDashboardLogBox.Width(width - 2).Render(content)
	box = insertBorderTitle(box, "Pods", StyleMuted)

	return box
}

// insertBorderTitle inserts a title into the top border of a box
func insertBorderTitle(box, title string, style lipgloss.Style) string {
	lines := strings.Split(box, "\n")
	if len(lines) == 0 {
		return box
	}

	firstLine := lines[0]
	titleText := " " + title + " "
	styledTitle := style.Render(titleText)
	titleWidth := lipgloss.Width(titleText)

	// We need to find where to insert the title (after the first border character)
	// The first line is the top border, typically: ┌──────────────┐
	// We want to replace some dashes after position 1: ┌─ Title ─────┐

	if len(firstLine) > titleWidth+4 {
		// Find the visual width of the first line
		lineWidth := lipgloss.Width(firstLine)
		if lineWidth > titleWidth+4 {
			// Build new first line: border char + title + remaining border
			// Since the border might have ANSI codes, we need to be careful
			// Simple approach: take first rune, add title, pad with border chars
			runes := []rune(firstLine)
			if len(runes) > 2 {
				// Get first border character
				firstChar := string(runes[0])
				lastChar := string(runes[len(runes)-1])

				// Calculate how many border chars we need after the title
				remainingWidth := lineWidth - 2 - titleWidth // -2 for first and last border chars

				// Build the new line
				lines[0] = firstChar + styledTitle + strings.Repeat("─", remainingWidth) + lastChar
			}
		}
	}

	return strings.Join(lines, "\n")
}

func (m DashboardModel) renderFooter(width int) string {
	// Elapsed time
	elapsed := time.Since(m.startTime).Round(time.Second)
	elapsedStr := StyleDashboardLabel.Render("Total: ") + StyleDashboardValue.Render(elapsed.String())

	// Use help component for hotkeys
	m.help.Width = width - lipgloss.Width(elapsedStr) - 6
	hotkeys := m.help.ShortHelpView(m.keyMap.ShortHelp())

	// Build footer line
	padding := width - lipgloss.Width(elapsedStr) - lipgloss.Width(hotkeys) - 4
	if padding < 1 {
		padding = 1
	}

	footer := fmt.Sprintf(" %s%s%s ", elapsedStr, strings.Repeat(" ", padding), hotkeys)
	return StyleDashboardFooter.Render(footer)
}

// renderCompletionView renders the completion summary panel after bootstrap finishes
func (m DashboardModel) renderCompletionView(width int) string {
	var sections []string

	// Header - success or failure
	var headerIcon, headerText string
	var headerStyle lipgloss.Style
	if m.success {
		headerIcon = IconSuccess
		headerText = "Bootstrap Complete"
		headerStyle = StyleSuccess
	} else {
		headerIcon = IconError
		headerText = "Bootstrap Failed"
		headerStyle = StyleError
	}

	// Build header content
	elapsed := time.Since(m.startTime).Round(time.Second)
	headerLine := headerStyle.Render(fmt.Sprintf("%s %s", headerIcon, headerText))

	clusterLine := ""
	if m.tracker.ClusterName() != "" {
		clusterLine = StyleDashboardLabel.Render("Cluster: ") + StyleDashboardValue.Render(m.tracker.ClusterName())
	}

	durationLine := StyleDashboardLabel.Render("Duration: ") + StyleDashboardValue.Render(elapsed.String())

	headerContent := headerLine + "\n"
	if clusterLine != "" {
		headerContent += clusterLine + "    " + durationLine
	} else {
		headerContent += durationLine
	}

	// Header box style based on success/failure
	var headerBoxStyle lipgloss.Style
	if m.success {
		headerBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorSuccess).
			Padding(0, 1).
			Width(width - 2)
	} else {
		headerBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorError).
			Padding(0, 1).
			Width(width - 2)
	}

	sections = append(sections, headerBoxStyle.Render(headerContent))

	// Phase summary table
	var phaseLines []string
	phases := m.tracker.Phases()
	for _, phase := range phases {
		icon := phase.Status.Icon()
		style := PhaseStatusStyle(phase.Status)

		// Format duration
		duration := phase.FormatDuration()
		durationStr := StyleMuted.Render(duration)

		// Build line with proper alignment
		nameWidth := width - 12 // Leave room for icon, duration, borders
		name := phase.Name
		if len(name) > nameWidth {
			name = name[:nameWidth-3] + "..."
		}

		// Add message or skip reason if present
		detail := ""
		if phase.Message != "" {
			detail = StyleMuted.Render(" (" + phase.Message + ")")
		} else if phase.SkipReason != "" {
			detail = StyleMuted.Render(" (" + phase.SkipReason + ")")
		}

		line := fmt.Sprintf(" %s %s%s", style.Render(icon), style.Render(name), detail)

		// Add duration aligned to the right
		lineWidth := lipgloss.Width(line)
		padding := width - lineWidth - len(duration) - 4
		if padding < 1 {
			padding = 1
		}
		line += strings.Repeat(" ", padding) + durationStr

		phaseLines = append(phaseLines, line)
	}

	phaseContent := strings.Join(phaseLines, "\n")
	phaseBox := StyleDashboardBox.Width(width - 2).Render(phaseContent)
	phaseBox = insertBorderTitle(phaseBox, "Phases", StyleDashboardPhaseHeader)
	sections = append(sections, phaseBox)

	// Result message and next steps (only for success)
	if m.success {
		var resultLines []string

		// Success message
		resultLines = append(resultLines, StyleSuccess.Render(IconSuccess)+" "+StyleBold.Render("Cluster ready!"))
		resultLines = append(resultLines, "")

		// Next step hint - prefer result hint, fall back to model hint
		hint := m.result.NextStepHint
		if hint == "" {
			hint = m.nextStepHint
		}
		if hint != "" {
			resultLines = append(resultLines, StyleMuted.Render("Next step:"))
			resultLines = append(resultLines, "  "+StyleCode.Render(hint))
		}

		resultContent := strings.Join(resultLines, "\n")
		resultBox := StyleBoxSuccess.Width(width - 2).Render(resultContent)
		sections = append(sections, resultBox)
	} else {
		// Error message
		var errorLines []string
		errorLines = append(errorLines, StyleError.Render(IconError)+" "+StyleBold.Render("Bootstrap failed"))
		if m.result.Error != nil {
			errorLines = append(errorLines, "")
			errorLines = append(errorLines, StyleMuted.Render("Error: ")+m.result.Error.Error())
		}

		errorContent := strings.Join(errorLines, "\n")
		errorBox := StyleBoxError.Width(width - 2).Render(errorContent)
		sections = append(sections, errorBox)
	}

	// Footer with "press any key to exit"
	footerText := StyleMuted.Render("Press any key to exit")
	footerPadding := (width - lipgloss.Width(footerText)) / 2
	footer := strings.Repeat(" ", footerPadding) + footerText

	sections = append(sections, footer)

	return strings.Join(sections, "\n")
}

// Result returns the final result after the dashboard completes
func (m DashboardModel) Result() BootstrapCompleteMsg {
	return m.result
}

// Success returns whether the bootstrap was successful
func (m DashboardModel) Success() bool {
	return m.success
}

// Context returns the cancellable context
func (m DashboardModel) Context() context.Context {
	return m.state.Ctx
}

// -----------------------------------------------------------------------------
// Dashboard Runner
// -----------------------------------------------------------------------------

// DashboardController provides methods to send updates to the running dashboard
type DashboardController struct {
	program *tea.Program
}

// StartPhase notifies the dashboard that a phase has started
func (c *DashboardController) StartPhase(name string) {
	if c.program != nil {
		c.program.Send(PhaseStartedMsg{PhaseName: name})
	}
}

// CompletePhase notifies the dashboard that a phase has completed
func (c *DashboardController) CompletePhase(name string, message string) {
	if c.program != nil {
		c.program.Send(PhaseCompletedMsg{PhaseName: name, Message: message})
	}
}

// SkipPhase notifies the dashboard that a phase was skipped
func (c *DashboardController) SkipPhase(name string, reason string) {
	if c.program != nil {
		c.program.Send(PhaseSkippedMsg{PhaseName: name, Reason: reason})
	}
}

// FailPhase notifies the dashboard that a phase has failed
func (c *DashboardController) FailPhase(name string, err error) {
	if c.program != nil {
		c.program.Send(PhaseFailedMsg{PhaseName: name, Error: err})
	}
}

// UpdateOperation updates the current operation display
func (c *DashboardController) UpdateOperation(step string, progress float64) {
	if c.program != nil {
		c.program.Send(OperationUpdateMsg{Step: step, Progress: progress})
	}
}

// Log adds a line to the log buffer
func (c *DashboardController) Log(line string) {
	if c.program != nil {
		c.program.Send(LogLineMsg{Line: line})
	}
}

// UpdatePods updates the pods status display
func (c *DashboardController) UpdatePods(pods []PodInfo) {
	if c.program != nil {
		c.program.Send(PodStatusUpdateMsg{Pods: pods})
	}
}

// ExtendTimeout extends the timeout by the configured amount
func (c *DashboardController) ExtendTimeout(newDeadline time.Time) {
	if c.program != nil {
		c.program.Send(TimeoutExtendedMsg{NewDeadline: newDeadline})
	}
}

// Complete signals that the bootstrap is complete
func (c *DashboardController) Complete(success bool, message string, err error) {
	c.CompleteWithHint(success, message, err, "")
}

// CompleteWithHint signals that the bootstrap is complete with a hint for next steps
func (c *DashboardController) CompleteWithHint(success bool, message string, err error, hint string) {
	if c.program != nil {
		c.program.Send(BootstrapCompleteMsg{Success: success, Message: message, Error: err, NextStepHint: hint})
	}
}

// RunBootstrapDashboard runs the dashboard with the given tracker and work function.
// The work function receives a controller to update the dashboard and should return
// when the bootstrap is complete.
//
// Returns the final result message.
func RunBootstrapDashboard(
	parentCtx context.Context,
	tracker *PhaseTracker,
	workFn func(ctx context.Context, ctrl *DashboardController) error,
	opts ...DashboardOption,
) (BootstrapCompleteMsg, error) {
	if !IsTTY() {
		// Non-TTY fallback: run work directly with simple output
		return runNonTTYBootstrap(parentCtx, tracker, workFn)
	}

	model := NewDashboardModel(tracker, parentCtx, opts...)
	program := tea.NewProgram(model, tea.WithAltScreen())

	ctrl := &DashboardController{program: program}

	// Run the work function in a goroutine
	go func() {
		err := workFn(model.state.Ctx, ctrl)
		if err != nil {
			if errors.Is(err, ErrCancelled) {
				ctrl.Complete(false, "Bootstrap cancelled", err)
			} else {
				ctrl.Complete(false, "Bootstrap failed", err)
			}
		} else {
			ctrl.Complete(true, "Bootstrap completed successfully", nil)
		}
	}()

	// Run the TUI
	finalModel, err := program.Run()
	if err != nil {
		return BootstrapCompleteMsg{}, err
	}

	result := finalModel.(DashboardModel).Result()
	return result, result.Error
}

// runNonTTYBootstrap provides a simple text-based fallback for non-TTY environments
func runNonTTYBootstrap(
	parentCtx context.Context,
	tracker *PhaseTracker,
	workFn func(ctx context.Context, ctrl *DashboardController) error,
) (BootstrapCompleteMsg, error) {
	// Print header
	tracker.PrintHeader()

	// Run the work in print mode (nil controller)
	err := workFn(parentCtx, nil)

	if err != nil {
		result := BootstrapCompleteMsg{Success: false, Error: err}
		if errors.Is(err, ErrCancelled) {
			result.Message = "Bootstrap cancelled"
		} else {
			result.Message = "Bootstrap failed"
		}
		// Print summary
		tracker.PrintSummary()
		return result, err
	}

	// Print success
	tracker.PrintSummary()
	tracker.PrintSuccess("Bootstrap complete!")

	return BootstrapCompleteMsg{Success: true, Message: "Bootstrap completed successfully"}, nil
}
