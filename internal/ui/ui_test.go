package ui

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// -----------------------------------------------------------------------------
// Test Helpers
// -----------------------------------------------------------------------------

// setupTestEnvironment configures the UI package for testing:
// - Sets TTY detection to return false (non-interactive mode)
// - Resets the non-TTY notice
// - Returns a cleanup function to restore defaults
func setupTestEnvironment(ttyEnabled bool) func() {
	restoreTTY := SetTTYDetector(func() bool { return ttyEnabled })
	ResetNonTTYNotice()
	return func() {
		restoreTTY()
		ResetNonTTYNotice()
	}
}

// createKeyMsg creates a tea.KeyMsg for testing
func createKeyMsg(key string) tea.KeyMsg {
	switch key {
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	case "q":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
}

// -----------------------------------------------------------------------------
// TTY Detection Tests
// -----------------------------------------------------------------------------

func TestIsTTY_DefaultsToTerminalCheck(t *testing.T) {
	// This test verifies the function doesn't panic
	// The actual result depends on the test environment
	_ = IsTTY()
}

func TestSetTTYDetector_OverridesDetection(t *testing.T) {
	// Override to return true
	restore := SetTTYDetector(func() bool { return true })
	if !IsTTY() {
		t.Error("expected IsTTY() to return true after override")
	}
	restore()

	// Override to return false
	restore = SetTTYDetector(func() bool { return false })
	if IsTTY() {
		t.Error("expected IsTTY() to return false after override")
	}
	restore()
}

func TestSetTTYDetector_NilRestoresDefault(t *testing.T) {
	// First override
	SetTTYDetector(func() bool { return true })
	// Then restore with nil
	SetTTYDetector(nil)
	// Should use default detector now (doesn't panic)
	_ = IsTTY()
}

// -----------------------------------------------------------------------------
// Output Writer Tests
// -----------------------------------------------------------------------------

func TestSetDefaultOutput_OverridesOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	restore := SetDefaultOutput(buf)
	defer restore()

	if GetDefaultOutput() != buf {
		t.Error("expected GetDefaultOutput() to return the custom buffer")
	}
}

func TestSetDefaultOutput_NilRestoresStdout(t *testing.T) {
	buf := &bytes.Buffer{}
	SetDefaultOutput(buf)
	SetDefaultOutput(nil)

	// GetDefaultOutput should return os.Stdout (not nil)
	if GetDefaultOutput() == nil {
		t.Error("expected GetDefaultOutput() to not be nil after restoring")
	}
}

// -----------------------------------------------------------------------------
// NonTTYPrinter Tests
// -----------------------------------------------------------------------------

func TestNonTTYPrinter_Success(t *testing.T) {
	cleanup := setupTestEnvironment(false)
	defer cleanup()

	buf := &bytes.Buffer{}
	printer := NewNonTTYPrinter("Test Operation", WithOutput(buf))
	printer.Success()

	output := buf.String()
	if !strings.Contains(output, "Test Operation") {
		t.Errorf("expected output to contain 'Test Operation', got: %s", output)
	}
	if !strings.Contains(output, IconSuccess) {
		t.Errorf("expected output to contain success icon, got: %s", output)
	}
}

func TestNonTTYPrinter_Error(t *testing.T) {
	cleanup := setupTestEnvironment(false)
	defer cleanup()

	buf := &bytes.Buffer{}
	printer := NewNonTTYPrinter("Test Operation", WithOutput(buf))
	printer.Error(errors.New("test error"))

	output := buf.String()
	if !strings.Contains(output, "test error") {
		t.Errorf("expected output to contain error message, got: %s", output)
	}
	if !strings.Contains(output, IconError) {
		t.Errorf("expected output to contain error icon, got: %s", output)
	}
}

func TestNonTTYPrinter_ItemProgress(t *testing.T) {
	cleanup := setupTestEnvironment(false)
	defer cleanup()

	buf := &bytes.Buffer{}
	printer := NewNonTTYPrinter("Test Operation", WithOutput(buf))
	printer.ItemProgress(2, 5, "item-name")

	output := buf.String()
	if !strings.Contains(output, "[2/5]") {
		t.Errorf("expected output to contain '[2/5]', got: %s", output)
	}
	if !strings.Contains(output, "item-name") {
		t.Errorf("expected output to contain 'item-name', got: %s", output)
	}
}

// -----------------------------------------------------------------------------
// RunSpinner Tests (non-TTY mode)
// -----------------------------------------------------------------------------

func TestRunSpinner_NonTTY_Success(t *testing.T) {
	cleanup := setupTestEnvironment(false)
	defer cleanup()

	buf := &bytes.Buffer{}
	err := RunSpinner("Test Task", func() error {
		return nil
	}, WithSpinnerOutput(buf))

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Test Task") {
		t.Errorf("expected output to contain 'Test Task', got: %s", output)
	}
	if !strings.Contains(output, IconSuccess) {
		t.Errorf("expected output to contain success icon, got: %s", output)
	}
}

func TestRunSpinner_NonTTY_Error(t *testing.T) {
	cleanup := setupTestEnvironment(false)
	defer cleanup()

	buf := &bytes.Buffer{}
	expectedErr := errors.New("test error")
	err := RunSpinner("Test Task", func() error {
		return expectedErr
	}, WithSpinnerOutput(buf))

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got: %v", expectedErr, err)
	}

	output := buf.String()
	if !strings.Contains(output, "test error") {
		t.Errorf("expected output to contain error, got: %s", output)
	}
}

func TestRunSpinnerWithContext_NonTTY_ContextCancelled(t *testing.T) {
	cleanup := setupTestEnvironment(false)
	defer cleanup()

	buf := &bytes.Buffer{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := RunSpinnerWithContext(ctx, "Test Task", func(ctx context.Context) error {
		return ctx.Err()
	}, WithSpinnerOutput(buf))

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got: %v", err)
	}
}

// -----------------------------------------------------------------------------
// RunProgress Tests (non-TTY mode)
// -----------------------------------------------------------------------------

func TestRunProgress_NonTTY_Success(t *testing.T) {
	cleanup := setupTestEnvironment(false)
	defer cleanup()

	buf := &bytes.Buffer{}
	items := []string{"item1", "item2", "item3"}
	processedItems := []string{}

	err := RunProgress("Processing Items", items, func(item string) error {
		processedItems = append(processedItems, item)
		return nil
	}, WithProgressOutput(buf))

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if len(processedItems) != len(items) {
		t.Errorf("expected %d items processed, got %d", len(items), len(processedItems))
	}

	output := buf.String()
	for _, item := range items {
		if !strings.Contains(output, item) {
			t.Errorf("expected output to contain '%s', got: %s", item, output)
		}
	}
}

func TestRunProgress_NonTTY_EmptyItems(t *testing.T) {
	cleanup := setupTestEnvironment(false)
	defer cleanup()

	buf := &bytes.Buffer{}
	err := RunProgress("Processing Items", []string{}, func(item string) error {
		return errors.New("should not be called")
	}, WithProgressOutput(buf))

	if err != nil {
		t.Errorf("expected no error for empty items, got: %v", err)
	}
}

func TestRunProgress_NonTTY_FailsOnFirstError(t *testing.T) {
	cleanup := setupTestEnvironment(false)
	defer cleanup()

	buf := &bytes.Buffer{}
	items := []string{"item1", "item2", "item3"}
	expectedErr := errors.New("item2 failed")

	err := RunProgress("Processing Items", items, func(item string) error {
		if item == "item2" {
			return expectedErr
		}
		return nil
	}, WithProgressOutput(buf))

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got: %v", expectedErr, err)
	}
}

// -----------------------------------------------------------------------------
// Table Validation Tests
// -----------------------------------------------------------------------------

func TestValidateTableData_Valid(t *testing.T) {
	headers := []string{"Name", "Status", "Age"}
	rows := [][]string{
		{"pod-1", "Running", "5m"},
		{"pod-2", "Pending", "2m"},
	}

	err := ValidateTableData(headers, rows)
	if err != nil {
		t.Errorf("expected no error for valid data, got: %v", err)
	}
}

func TestValidateTableData_EmptyHeaders(t *testing.T) {
	err := ValidateTableData([]string{}, [][]string{})
	if err == nil {
		t.Error("expected error for empty headers")
	}
}

func TestValidateTableData_RowColumnMismatch(t *testing.T) {
	headers := []string{"Name", "Status", "Age"}
	rows := [][]string{
		{"pod-1", "Running", "5m"},
		{"pod-2", "Pending"}, // Missing Age column
	}

	err := ValidateTableData(headers, rows)
	if err == nil {
		t.Error("expected error for column mismatch")
	}

	// Check it's the right error type
	var validErr TableValidationError
	if !errors.As(err, &validErr) {
		t.Errorf("expected TableValidationError, got: %T", err)
	}
}

func TestValidateStatusTableConfig_Valid(t *testing.T) {
	cfg := StatusTableConfig{
		Columns: []StatusColumn{
			{Title: "Name", MinWidth: 10},
			{Title: "Status", MinWidth: 8},
		},
		Rows: []StatusRow{
			{Cells: []string{"pod-1", "Running"}},
			{Cells: []string{"pod-2", "Pending"}},
		},
	}

	err := ValidateStatusTableConfig(cfg)
	if err != nil {
		t.Errorf("expected no error for valid config, got: %v", err)
	}
}

func TestValidateStatusTableConfig_EmptyColumns(t *testing.T) {
	cfg := StatusTableConfig{
		Columns: []StatusColumn{},
		Rows:    []StatusRow{},
	}

	err := ValidateStatusTableConfig(cfg)
	if err == nil {
		t.Error("expected error for empty columns")
	}
}

func TestValidateStatusTableConfig_CellCountMismatch(t *testing.T) {
	cfg := StatusTableConfig{
		Columns: []StatusColumn{
			{Title: "Name", MinWidth: 10},
			{Title: "Status", MinWidth: 8},
		},
		Rows: []StatusRow{
			{Cells: []string{"pod-1", "Running"}},
			{Cells: []string{"pod-2"}}, // Missing second cell
		},
	}

	err := ValidateStatusTableConfig(cfg)
	if err == nil {
		t.Error("expected error for cell count mismatch")
	}
}

// -----------------------------------------------------------------------------
// StepStatus Tests
// -----------------------------------------------------------------------------

func TestStepStatus_String(t *testing.T) {
	tests := []struct {
		status   StepStatus
		expected string
	}{
		{StepPending, "pending"},
		{StepRunning, "running"},
		{StepComplete, "complete"},
		{StepFailed, "failed"},
		{StepStatus(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("StepStatus.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestStepStatus_Icon(t *testing.T) {
	tests := []struct {
		status       StepStatus
		expectedIcon string
	}{
		{StepPending, IconPending},
		{StepRunning, IconRunning},
		{StepComplete, IconSuccess},
		{StepFailed, IconError},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			if got := tt.status.Icon(); got != tt.expectedIcon {
				t.Errorf("StepStatus.Icon() = %q, want %q", got, tt.expectedIcon)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// CancellableState Tests
// -----------------------------------------------------------------------------

func TestCancellableState_Cancel(t *testing.T) {
	state := NewCancellableState(context.Background())

	// Context should not be cancelled initially
	select {
	case <-state.Ctx.Done():
		t.Error("context should not be cancelled initially")
	default:
		// OK
	}

	// Cancel and verify
	state.Cancel()
	select {
	case <-state.Ctx.Done():
		// OK - context is cancelled
	default:
		t.Error("context should be cancelled after Cancel()")
	}
}

func TestCancellableState_InheritsParentCancellation(t *testing.T) {
	parentCtx, parentCancel := context.WithCancel(context.Background())
	state := NewCancellableState(parentCtx)

	// Cancel parent
	parentCancel()

	// Child context should also be cancelled
	select {
	case <-state.Ctx.Done():
		// OK - context is cancelled
	default:
		t.Error("child context should be cancelled when parent is cancelled")
	}
}

// -----------------------------------------------------------------------------
// HandleCancelKeys Tests
// -----------------------------------------------------------------------------

func TestHandleCancelKeys_CtrlC(t *testing.T) {
	state := NewCancellableState(context.Background())

	action := HandleCancelKeys(createKeyMsg("ctrl+c"), state)

	if !action.Handled {
		t.Error("expected ctrl+c to be handled")
	}
	if !action.Cancelled {
		t.Error("expected ctrl+c to trigger cancellation")
	}

	// Verify context was cancelled
	select {
	case <-state.Ctx.Done():
		// OK
	default:
		t.Error("context should be cancelled")
	}
}

func TestHandleCancelKeys_Q(t *testing.T) {
	state := NewCancellableState(context.Background())

	action := HandleCancelKeys(createKeyMsg("q"), state)

	if !action.Handled {
		t.Error("expected 'q' to be handled")
	}
	if !action.Cancelled {
		t.Error("expected 'q' to trigger cancellation")
	}
}

func TestHandleCancelKeys_OtherKey(t *testing.T) {
	state := NewCancellableState(context.Background())

	action := HandleCancelKeys(createKeyMsg("a"), state)

	if action.Handled {
		t.Error("expected 'a' to not be handled")
	}
	if action.Cancelled {
		t.Error("expected 'a' to not trigger cancellation")
	}

	// Context should not be cancelled
	select {
	case <-state.Ctx.Done():
		t.Error("context should not be cancelled for other keys")
	default:
		// OK
	}
}

func TestHandleCancelKeys_NilState(t *testing.T) {
	// Should not panic with nil state
	action := HandleCancelKeys(createKeyMsg("ctrl+c"), nil)

	if !action.Handled {
		t.Error("expected ctrl+c to be handled even with nil state")
	}
	if !action.Cancelled {
		t.Error("expected ctrl+c to trigger cancellation flag even with nil state")
	}
}

// -----------------------------------------------------------------------------
// Default Intervals Tests
// -----------------------------------------------------------------------------

func TestDefaultPollIntervals(t *testing.T) {
	if DefaultPollInterval != 2*time.Second {
		t.Errorf("expected DefaultPollInterval to be 2s, got %v", DefaultPollInterval)
	}
	if DefaultNonTTYPollInterval != 3*time.Second {
		t.Errorf("expected DefaultNonTTYPollInterval to be 3s, got %v", DefaultNonTTYPollInterval)
	}
	if DefaultProviderPollInterval != 5*time.Second {
		t.Errorf("expected DefaultProviderPollInterval to be 5s, got %v", DefaultProviderPollInterval)
	}
}

// -----------------------------------------------------------------------------
// RenderProgressSteps Tests
// -----------------------------------------------------------------------------

func TestRenderProgressSteps(t *testing.T) {
	steps := []ProgressStep{
		{Name: "Step 1", Status: StepComplete},
		{Name: "Step 2", Status: StepRunning},
		{Name: "Step 3", Status: StepPending},
		{Name: "Step 4", Status: StepFailed, Error: errors.New("something went wrong")},
	}

	output := RenderProgressSteps(steps)

	if !strings.Contains(output, "Step 1") {
		t.Error("expected output to contain 'Step 1'")
	}
	if !strings.Contains(output, "Step 2") {
		t.Error("expected output to contain 'Step 2'")
	}
	if !strings.Contains(output, "something went wrong") {
		t.Error("expected output to contain error message")
	}
}

// -----------------------------------------------------------------------------
// Table Rendering Tests
// -----------------------------------------------------------------------------

func TestRenderTable(t *testing.T) {
	headers := []string{"Name", "Status"}
	rows := [][]string{
		{"pod-1", "Running"},
		{"pod-2", "Pending"},
	}

	output := RenderTable(headers, rows)

	if !strings.Contains(output, "Name") {
		t.Error("expected output to contain 'Name' header")
	}
	if !strings.Contains(output, "Status") {
		t.Error("expected output to contain 'Status' header")
	}
	if !strings.Contains(output, "pod-1") {
		t.Error("expected output to contain 'pod-1'")
	}
}

func TestRenderStatusTable(t *testing.T) {
	cfg := StatusTableConfig{
		Columns: []StatusColumn{
			{Title: "Pod", MinWidth: 10},
			{Title: "Status", MinWidth: 12},
		},
		Rows: []StatusRow{
			{Cells: []string{"nginx", "Running"}},
		},
	}

	output := RenderStatusTable(cfg)

	// The table should contain headers
	if !strings.Contains(output, "Pod") {
		t.Errorf("expected output to contain 'Pod' header, got: %s", output)
	}
	// The table should contain data rows (cells are rendered in table format)
	// Note: The exact output format depends on the table component
	if output == "" {
		t.Error("expected non-empty table output")
	}
}

// -----------------------------------------------------------------------------
// ProgressBar Tests
// -----------------------------------------------------------------------------

func TestProgressBar(t *testing.T) {
	output := ProgressBar(50, 100, 20)

	if output == "" {
		t.Error("expected non-empty progress bar output")
	}
	if !strings.Contains(output, "50%") {
		t.Error("expected output to contain '50%'")
	}
}

func TestProgressBar_ZeroTotal(t *testing.T) {
	output := ProgressBar(0, 0, 20)

	if output != "" {
		t.Errorf("expected empty string for zero total, got: %s", output)
	}
}

func TestProgressBarWithLabel(t *testing.T) {
	output := ProgressBarWithLabel("Download", 75, 100, 20)

	if !strings.Contains(output, "Download") {
		t.Error("expected output to contain label 'Download'")
	}
	if !strings.Contains(output, "75%") {
		t.Error("expected output to contain '75%'")
	}
}
