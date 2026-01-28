package ui

import (
	"context"
	"testing"
	"time"
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
