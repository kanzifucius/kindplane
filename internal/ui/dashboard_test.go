package ui

import (
	"context"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func TestNewDashboardModel(t *testing.T) {
	tracker := NewPhaseTracker("test", WithClusterInfo("my-cluster", "config.yaml"))
	tracker.AddPhase("Phase 1")
	tracker.AddPhase("Phase 2")

	model := NewDashboardModel(tracker, context.Background())

	if model.tracker != tracker {
		t.Error("expected tracker to be set")
	}
	if model.startTime.IsZero() {
		t.Error("expected startTime to be set")
	}
	if model.deadline.IsZero() {
		t.Error("expected deadline to be set")
	}
	if model.verbose {
		t.Error("expected verbose to be false initially")
	}
	if model.completed {
		t.Error("expected completed to be false initially")
	}
}

func TestNewDashboardModel_WithOptions(t *testing.T) {
	tracker := NewPhaseTracker("test")

	model := NewDashboardModel(tracker, context.Background(),
		WithTimeout(5*time.Minute),
		WithExtendAmount(2*time.Minute),
	)

	// Check timeout is approximately 5 minutes from now
	remaining := time.Until(model.deadline)
	if remaining < 4*time.Minute || remaining > 6*time.Minute {
		t.Errorf("expected deadline ~5 minutes from now, got %v remaining", remaining)
	}

	if model.extendAmount != 2*time.Minute {
		t.Errorf("expected extendAmount 2m, got %v", model.extendAmount)
	}
}

func TestDashboardModel_Init(t *testing.T) {
	tracker := NewPhaseTracker("test")
	model := NewDashboardModel(tracker, context.Background())

	cmd := model.Init()

	if cmd == nil {
		t.Error("expected Init to return a command")
	}
}

func TestDashboardController_NilSafe(t *testing.T) {
	// Controller with nil program (non-TTY mode) should not panic
	ctrl := &DashboardController{program: nil}

	// These should all be no-ops without panicking
	ctrl.StartPhase("test")
	ctrl.CompletePhase("test", "message")
	ctrl.SkipPhase("test", "reason")
	ctrl.FailPhase("test", nil)
	ctrl.UpdateOperation("step", 0.5)
	ctrl.Log("log line")
	ctrl.ExtendTimeout(time.Now().Add(5 * time.Minute))
	ctrl.Complete(true, "done", nil)
}

func TestDashboardModel_ViewRendersWithoutCrash(t *testing.T) {
	tracker := NewPhaseTracker("Bootstrap", WithClusterInfo("my-cluster", "kindplane.yaml"))
	tracker.AddPhase("Create cluster")
	tracker.AddPhase("Install Crossplane")
	tracker.AddPhase("Configure providers")

	model := NewDashboardModel(tracker, context.Background())
	model.width = 100
	model.height = 40

	// Should not panic
	view := model.View()

	if view == "" {
		t.Error("expected View to return non-empty string")
	}
}

func TestDashboardModel_ViewRendersPhaseTable(t *testing.T) {
	tracker := NewPhaseTracker("Bootstrap", WithClusterInfo("my-cluster", "kindplane.yaml"))
	tracker.AddPhase("Create cluster")
	tracker.AddPhase("Install Crossplane")

	model := NewDashboardModel(tracker, context.Background())
	model.width = 100
	model.height = 40

	view := model.View()

	// Should contain cluster info
	if !containsString(view, "my-cluster") {
		t.Error("expected view to contain cluster name")
	}

	// Should contain phase names
	if !containsString(view, "Create cluster") {
		t.Error("expected view to contain phase name")
	}
	if !containsString(view, "Install Crossplane") {
		t.Error("expected view to contain phase name")
	}
}

func TestDashboardModel_ViewRendersCurrentOperation(t *testing.T) {
	tracker := NewPhaseTracker("Bootstrap")
	tracker.AddPhase("Create cluster")

	model := NewDashboardModel(tracker, context.Background())
	model.width = 100
	model.height = 40

	// Start a phase
	tracker.MarkPhaseRunning("Create cluster")
	model.currentStep = "Pulling images..."

	view := model.View()

	// Should contain current step
	if !containsString(view, "Pulling images") {
		t.Error("expected view to contain current step")
	}
}

func TestDashboardModel_VerboseMode(t *testing.T) {
	tracker := NewPhaseTracker("Bootstrap")
	tracker.AddPhase("Phase 1")

	model := NewDashboardModel(tracker, context.Background())
	model.width = 100
	model.height = 40
	model.addLogLine("Test log line 1")
	model.addLogLine("Test log line 2")

	// Non-verbose mode should not show logs
	model.verbose = false
	viewNonVerbose := model.View()

	// Verbose mode should show logs
	model.verbose = true
	viewVerbose := model.View()

	// Verbose view should be longer (more content)
	if len(viewVerbose) <= len(viewNonVerbose) {
		t.Error("expected verbose view to contain more content")
	}

	// Should contain log lines in verbose mode
	if !containsString(viewVerbose, "Test log line") {
		t.Error("expected verbose view to contain log lines")
	}
}

func TestDashboardModel_Result(t *testing.T) {
	tracker := NewPhaseTracker("Bootstrap")
	model := NewDashboardModel(tracker, context.Background())

	model.result = BootstrapCompleteMsg{
		Success: true,
		Message: "All done",
	}
	model.success = true

	if !model.Success() {
		t.Error("expected Success() to return true")
	}

	result := model.Result()
	if !result.Success {
		t.Error("expected result.Success to be true")
	}
	if result.Message != "All done" {
		t.Errorf("expected result.Message 'All done', got '%s'", result.Message)
	}
}

func TestDashboardWidth(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{60, DashboardMinWidth},  // Below min, clamp to min
		{80, 80},                 // At min
		{100, 100},               // In range
		{120, DashboardMaxWidth}, // At max
		{150, DashboardMaxWidth}, // Above max, clamp to max
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := DashboardWidth(tt.input)
			if result != tt.expected {
				t.Errorf("DashboardWidth(%d) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPhaseStatusStyle(t *testing.T) {
	// Just verify these don't panic and return styles
	styles := []PhaseStatus{
		PhasePending,
		PhaseRunning,
		PhaseComplete,
		PhaseSkipped,
		PhaseFailed,
	}

	for _, status := range styles {
		style := PhaseStatusStyle(status)
		// Render something to verify style works
		_ = style.Render("test")
	}
}

func TestDashboardModel_CompletionView(t *testing.T) {
	tracker := NewPhaseTracker("Bootstrap", WithClusterInfo("my-cluster", "kindplane.yaml"))
	tracker.AddPhase("Create cluster")
	tracker.AddPhase("Install Crossplane")

	// Mark phases as complete
	tracker.MarkPhaseRunning("Create cluster")
	tracker.MarkPhaseComplete()
	tracker.MarkPhaseRunning("Install Crossplane")
	tracker.MarkPhaseComplete()

	model := NewDashboardModel(tracker, context.Background(),
		WithNextStepHint("kubectl cluster-info --context kindplane-test"),
	)
	model.width = 100
	model.height = 40

	// Set completed state
	model.completed = true
	model.success = true
	model.result = BootstrapCompleteMsg{
		Success: true,
		Message: "Bootstrap completed successfully",
	}

	view := model.View()

	// Should show completion view content
	if !containsString(view, "Bootstrap Complete") {
		t.Error("expected completion view to contain 'Bootstrap Complete'")
	}
	if !containsString(view, "Cluster ready") {
		t.Error("expected completion view to contain 'Cluster ready'")
	}
	if !containsString(view, "my-cluster") {
		t.Error("expected completion view to contain cluster name")
	}
	if !containsString(view, "kubectl cluster-info") {
		t.Error("expected completion view to contain next step hint")
	}
	if !containsString(view, "Press any key to exit") {
		t.Error("expected completion view to contain exit instruction")
	}
}

func TestDashboardModel_CompletionView_Failure(t *testing.T) {
	tracker := NewPhaseTracker("Bootstrap", WithClusterInfo("my-cluster", "kindplane.yaml"))
	tracker.AddPhase("Create cluster")
	tracker.AddPhase("Install Crossplane")

	// Mark first phase complete, second failed
	tracker.MarkPhaseRunning("Create cluster")
	tracker.MarkPhaseComplete()
	tracker.MarkPhaseRunning("Install Crossplane")

	model := NewDashboardModel(tracker, context.Background())
	model.width = 100
	model.height = 40

	// Set failed state
	model.completed = true
	model.success = false
	model.result = BootstrapCompleteMsg{
		Success: false,
		Message: "Bootstrap failed",
		Error:   context.DeadlineExceeded,
	}

	view := model.View()

	// Should show failure content
	if !containsString(view, "Bootstrap Failed") {
		t.Error("expected completion view to contain 'Bootstrap Failed'")
	}
	if !containsString(view, "Press any key to exit") {
		t.Error("expected completion view to contain exit instruction")
	}
}

func TestDashboardModel_WithNextStepHint(t *testing.T) {
	tracker := NewPhaseTracker("test")

	model := NewDashboardModel(tracker, context.Background(),
		WithNextStepHint("kubectl get pods"),
	)

	if model.nextStepHint != "kubectl get pods" {
		t.Errorf("expected nextStepHint 'kubectl get pods', got '%s'", model.nextStepHint)
	}
}

func TestInsertBorderTitleColorHandling(t *testing.T) {
	borderFg := lipgloss.Color("#9CA3AF")
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))

	tests := []struct {
		name     string
		box      string // first line is the border
		title    string
		wantLast rune // expected last border rune on the first line
	}{
		{
			name:     "plain border short title",
			box:      "┌──────────────┐\n│ content      │\n└──────────────┘",
			title:    "Hi",
			wantLast: '┐',
		},
		{
			name:     "plain border longer title",
			box:      "┌──────────────┐\n│ content      │\n└──────────────┘",
			title:    "Pods",
			wantLast: '┐',
		},
		{
			name:     "ANSI-colored border short title",
			box:      "\x1b[38;2;156;163;175m┌──────────────┐\x1b[0m\n│ content      │\n└──────────────┘",
			title:    "Hi",
			wantLast: '┐',
		},
		{
			name:     "ANSI-colored border longer title",
			box:      "\x1b[38;2;156;163;175m┌──────────────┐\x1b[0m\n│ content      │\n└──────────────┘",
			title:    "Pods",
			wantLast: '┐',
		},
		{
			name:     "ANSI-colored border wide title",
			box:      "\x1b[38;2;156;163;175m┌────────────────────┐\x1b[0m\n│ content            │\n└────────────────────┘",
			title:    "Long Title Here",
			wantLast: '┐',
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := insertBorderTitle(tt.box, tt.title, style, borderFg)
			lines := splitLines(result)
			if len(lines) == 0 {
				t.Fatal("insertBorderTitle returned empty result")
			}

			firstLine := lines[0]
			stripped := ansi.Strip(firstLine)
			runes := []rune(stripped)
			if len(runes) == 0 {
				t.Fatal("first line is empty after stripping ANSI")
			}

			lastRune := runes[len(runes)-1]
			if lastRune != tt.wantLast {
				t.Errorf("last rune = %q (%U), want %q (%U)",
					lastRune, lastRune, tt.wantLast, tt.wantLast)
			}

			// The title should appear in the first line
			if !findSubstring(stripped, tt.title) {
				t.Errorf("title %q not found in first line %q", tt.title, stripped)
			}

			// No stray 'm' from ANSI escape at the end
			if lastRune == 'm' {
				t.Error("first line ends with 'm', likely an ANSI escape leak")
			}
		})
	}
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
