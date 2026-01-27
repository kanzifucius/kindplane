package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestPhaseStatus_String(t *testing.T) {
	tests := []struct {
		status   PhaseStatus
		expected string
	}{
		{PhasePending, "pending"},
		{PhaseRunning, "running"},
		{PhaseComplete, "complete"},
		{PhaseSkipped, "skipped"},
		{PhaseFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("PhaseStatus.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPhaseStatus_Icon(t *testing.T) {
	tests := []struct {
		status   PhaseStatus
		expected string
	}{
		{PhasePending, IconPending},
		{PhaseRunning, IconRunning},
		{PhaseComplete, IconSuccess},
		{PhaseSkipped, IconWarning},
		{PhaseFailed, IconError},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			if got := tt.status.Icon(); got != tt.expected {
				t.Errorf("PhaseStatus.Icon() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewPhaseTracker(t *testing.T) {
	pt := NewPhaseTracker("test")

	if pt.title != "test" {
		t.Errorf("expected title 'test', got '%s'", pt.title)
	}
	if len(pt.phases) != 0 {
		t.Errorf("expected 0 phases, got %d", len(pt.phases))
	}
	if pt.current != -1 {
		t.Errorf("expected current -1, got %d", pt.current)
	}
}

func TestPhaseTracker_AddPhase(t *testing.T) {
	pt := NewPhaseTracker("test")

	phase1 := pt.AddPhase("Phase 1")
	phase2 := pt.AddPhase("Phase 2")

	if len(pt.phases) != 2 {
		t.Errorf("expected 2 phases, got %d", len(pt.phases))
	}
	if phase1.Name != "Phase 1" {
		t.Errorf("expected phase name 'Phase 1', got '%s'", phase1.Name)
	}
	if phase2.Name != "Phase 2" {
		t.Errorf("expected phase name 'Phase 2', got '%s'", phase2.Name)
	}
	if phase1.Status != PhasePending {
		t.Errorf("expected phase status Pending, got %v", phase1.Status)
	}
}

func TestPhaseTracker_AddPhaseIf(t *testing.T) {
	pt := NewPhaseTracker("test")

	phase1 := pt.AddPhaseIf(true, "Phase 1")
	phase2 := pt.AddPhaseIf(false, "Phase 2")

	if len(pt.phases) != 1 {
		t.Errorf("expected 1 phase, got %d", len(pt.phases))
	}
	if phase1 == nil {
		t.Error("expected phase1 to not be nil")
	}
	if phase2 != nil {
		t.Error("expected phase2 to be nil")
	}
}

func TestPhaseTracker_StartPhase(t *testing.T) {
	var buf bytes.Buffer
	pt := NewPhaseTracker("test", WithPhaseTrackerOutput(&buf))

	pt.AddPhase("Phase 1")
	pt.AddPhase("Phase 2")

	started := pt.StartPhase("Phase 1")

	if !started {
		t.Error("expected StartPhase to return true")
	}
	if pt.current != 0 {
		t.Errorf("expected current 0, got %d", pt.current)
	}
	if pt.phases[0].Status != PhaseRunning {
		t.Errorf("expected phase status Running, got %v", pt.phases[0].Status)
	}

	output := buf.String()
	if !strings.Contains(output, "[1/2]") {
		t.Errorf("expected output to contain '[1/2]', got: %s", output)
	}
	if !strings.Contains(output, "Phase 1") {
		t.Errorf("expected output to contain 'Phase 1', got: %s", output)
	}
}

func TestPhaseTracker_CompletePhase(t *testing.T) {
	var buf bytes.Buffer
	pt := NewPhaseTracker("test", WithPhaseTrackerOutput(&buf))

	pt.AddPhase("Phase 1")
	pt.StartPhase("Phase 1")
	buf.Reset() // Clear start output

	pt.CompletePhase()

	if pt.phases[0].Status != PhaseComplete {
		t.Errorf("expected phase status Complete, got %v", pt.phases[0].Status)
	}

	output := buf.String()
	if !strings.Contains(output, IconSuccess) {
		t.Errorf("expected output to contain success icon, got: %s", output)
	}
	if !strings.Contains(output, "Phase 1") {
		t.Errorf("expected output to contain 'Phase 1', got: %s", output)
	}
}

func TestPhaseTracker_SkipPhase(t *testing.T) {
	var buf bytes.Buffer
	pt := NewPhaseTracker("test", WithPhaseTrackerOutput(&buf))

	pt.AddPhase("Phase 1")
	pt.AddPhase("Phase 2")

	skipped := pt.SkipPhase("Phase 1", "already exists")

	if !skipped {
		t.Error("expected SkipPhase to return true")
	}
	if pt.phases[0].Status != PhaseSkipped {
		t.Errorf("expected phase status Skipped, got %v", pt.phases[0].Status)
	}
	if pt.phases[0].SkipReason != "already exists" {
		t.Errorf("expected skip reason 'already exists', got '%s'", pt.phases[0].SkipReason)
	}

	output := buf.String()
	if !strings.Contains(output, "Skipped") {
		t.Errorf("expected output to contain 'Skipped', got: %s", output)
	}
	if !strings.Contains(output, "already exists") {
		t.Errorf("expected output to contain 'already exists', got: %s", output)
	}
}

func TestPhaseTracker_FailPhase(t *testing.T) {
	var buf bytes.Buffer
	pt := NewPhaseTracker("test", WithPhaseTrackerOutput(&buf))

	pt.AddPhase("Phase 1")
	pt.StartPhase("Phase 1")
	buf.Reset()

	testErr := errForTest{"test error"}
	pt.FailPhase(testErr)

	if pt.phases[0].Status != PhaseFailed {
		t.Errorf("expected phase status Failed, got %v", pt.phases[0].Status)
	}
	if pt.phases[0].Error == nil {
		t.Error("expected phase error to be set")
	}

	output := buf.String()
	if !strings.Contains(output, IconError) {
		t.Errorf("expected output to contain error icon, got: %s", output)
	}
	if !strings.Contains(output, "failed") {
		t.Errorf("expected output to contain 'failed', got: %s", output)
	}
}

type errForTest struct {
	msg string
}

func (e errForTest) Error() string {
	return e.msg
}

func TestPhaseTracker_ActiveCount(t *testing.T) {
	pt := NewPhaseTracker("test")

	pt.AddPhase("Phase 1")
	pt.AddPhase("Phase 2")
	pt.AddPhase("Phase 3")

	// All pending, count = 3
	if count := pt.ActiveCount(); count != 3 {
		t.Errorf("expected ActiveCount 3, got %d", count)
	}

	// Skip one, count = 2
	pt.phases[1].Status = PhaseSkipped
	if count := pt.ActiveCount(); count != 2 {
		t.Errorf("expected ActiveCount 2 after skip, got %d", count)
	}
}

func TestPhaseTracker_PrintHeader(t *testing.T) {
	var buf bytes.Buffer
	pt := NewPhaseTracker("up",
		WithPhaseTrackerOutput(&buf),
		WithPhaseTrackerIcon(IconRocket),
		WithClusterInfo("my-cluster", "kindplane.yaml"),
		WithShowUpfrontList(true),
	)

	pt.AddPhase("Phase 1")
	pt.AddPhase("Phase 2")

	pt.PrintHeader()

	output := buf.String()

	// Check title
	if !strings.Contains(output, "up") {
		t.Errorf("expected output to contain 'up', got: %s", output)
	}

	// Check cluster info
	if !strings.Contains(output, "my-cluster") {
		t.Errorf("expected output to contain 'my-cluster', got: %s", output)
	}
	if !strings.Contains(output, "kindplane.yaml") {
		t.Errorf("expected output to contain 'kindplane.yaml', got: %s", output)
	}

	// Check phase list
	if !strings.Contains(output, "Phases:") {
		t.Errorf("expected output to contain 'Phases:', got: %s", output)
	}
	if !strings.Contains(output, "Phase 1") {
		t.Errorf("expected output to contain 'Phase 1', got: %s", output)
	}
	if !strings.Contains(output, "Phase 2") {
		t.Errorf("expected output to contain 'Phase 2', got: %s", output)
	}
}

func TestPhaseTracker_PrintSummary(t *testing.T) {
	var buf bytes.Buffer
	pt := NewPhaseTracker("test", WithPhaseTrackerOutput(&buf))

	pt.AddPhase("Phase 1")
	pt.AddPhase("Phase 2")
	pt.AddPhase("Phase 3")
	pt.AddPhase("Phase 4")

	pt.phases[0].Status = PhaseComplete
	pt.phases[1].Status = PhaseSkipped
	pt.phases[1].SkipReason = "not needed"
	pt.phases[2].Status = PhaseFailed
	pt.phases[3].Status = PhasePending

	pt.PrintSummary()

	output := buf.String()

	if !strings.Contains(output, "Summary:") {
		t.Errorf("expected output to contain 'Summary:', got: %s", output)
	}
	if !strings.Contains(output, "Phase 1") {
		t.Errorf("expected output to contain 'Phase 1', got: %s", output)
	}
	if !strings.Contains(output, "skipped") {
		t.Errorf("expected output to contain 'skipped', got: %s", output)
	}
	if !strings.Contains(output, "failed") {
		t.Errorf("expected output to contain 'failed', got: %s", output)
	}
	if !strings.Contains(output, "not started") {
		t.Errorf("expected output to contain 'not started', got: %s", output)
	}
}

func TestPhaseTracker_HasFailed(t *testing.T) {
	pt := NewPhaseTracker("test")

	pt.AddPhase("Phase 1")
	pt.AddPhase("Phase 2")

	if pt.HasFailed() {
		t.Error("expected HasFailed to return false initially")
	}

	pt.phases[1].Status = PhaseFailed
	if !pt.HasFailed() {
		t.Error("expected HasFailed to return true after failure")
	}
}

func TestPhaseTracker_AllComplete(t *testing.T) {
	pt := NewPhaseTracker("test")

	pt.AddPhase("Phase 1")
	pt.AddPhase("Phase 2")

	if pt.AllComplete() {
		t.Error("expected AllComplete to return false initially")
	}

	pt.phases[0].Status = PhaseComplete
	pt.phases[1].Status = PhaseSkipped

	if !pt.AllComplete() {
		t.Error("expected AllComplete to return true when all complete/skipped")
	}

	pt.phases[1].Status = PhaseFailed
	if pt.AllComplete() {
		t.Error("expected AllComplete to return false when one failed")
	}
}

func TestPhaseTracker_IndexCalculation(t *testing.T) {
	var buf bytes.Buffer
	pt := NewPhaseTracker("test", WithPhaseTrackerOutput(&buf))

	pt.AddPhase("Phase 1")
	pt.AddPhase("Phase 2")
	pt.AddPhase("Phase 3")

	// Skip phase 1
	pt.SkipPhase("Phase 1", "skipped")
	buf.Reset()

	// Start phase 2 - should be [1/2] not [2/3]
	pt.StartPhase("Phase 2")

	output := buf.String()
	if !strings.Contains(output, "[1/2]") {
		t.Errorf("expected output to contain '[1/2]' (active index), got: %s", output)
	}
}
