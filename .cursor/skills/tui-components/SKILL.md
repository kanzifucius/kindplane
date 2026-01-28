---
name: tui-components
description: Create and use interactive TUI components (spinners, progress bars, live tables). Use when adding animated UI or creating new Bubble Tea components.
---

# TUI Components

## Quick Reference

| Component | Function | Use Case |
|-----------|----------|----------|
| Spinner | `ui.RunSpinner` | Single async operation |
| Progress | `ui.RunProgress` | Processing list of items |
| Multi-step | `ui.RunMultiStep` | Multiple steps with live updates |
| Pod table | `ui.RunPodTable` | Live pod status monitoring |
| Provider table | `ui.RunProviderTable` | Live provider status monitoring |
| Table select | `ui.RunTableSelect` | Interactive row selection |

## Package Architecture


## Using Components

### Spinners

```go
// Simple - no context
err := ui.RunSpinner("Installing component", func() error {
    return doInstall()
})

// With context cancellation
err := ui.RunSpinnerWithContext(ctx, "Installing", func(ctx context.Context) error {
    return doInstall(ctx)
})

// With custom output (for testing)
err := ui.RunSpinner("Installing", fn, ui.WithSpinnerOutput(buf))
```

### Progress Bar

```go
items := []string{"item1", "item2", "item3"}
err := ui.RunProgress("Processing", items, func(item string) error {
    return process(item)
})

// With options
err := ui.RunProgressWithContext(ctx, "Processing", items, fn,
    ui.WithProgressOutput(buf))
```

### Multi-step Operations

```go
err := ui.RunMultiStep(ctx, "Creating cluster", func(ctx context.Context, updates chan<- ui.StepUpdate) error {
    updates <- ui.StepUpdate{Step: "Creating network", Done: false}
    if err := createNetwork(ctx); err != nil {
        return err
    }
    updates <- ui.StepUpdate{Step: "Creating network", Done: true, Success: true}
    
    updates <- ui.StepUpdate{Step: "Starting nodes", Done: false}
    // ...
    return nil
})
```

### Live Tables

```go
// Pod status table
err := ui.RunPodTable(ctx, "Waiting for pods", func(ctx context.Context) ([]ui.PodInfo, bool, error) {
    pods, err := getPods(ctx)
    allReady := checkAllReady(pods)
    return pods, allReady, err
}, ui.WithPodTablePollInterval(5*time.Second))

// Provider status table  
err := ui.RunProviderTable(ctx, "Waiting for providers", pollFn,
    ui.WithProviderTablePollInterval(10*time.Second))
```

### Available Options

| Component | Options |
|-----------|---------|
| Spinner | `WithSpinnerOutput(w)` |
| Progress | `WithProgressOutput(w)` |
| MultiStep | `WithMultiStepOutput(w)` |
| PodTable | `WithPodTableOutput(w)`, `WithPodTablePollInterval(d)` |
| ProviderTable | `WithProviderTableOutput(w)`, `WithProviderTablePollInterval(d)` |
| Table | `WithTableOutput(w)`, `WithTableHeight(h)`, `WithTableWidth(w)` |

## Creating New Components

Template for a new interactive component:

```go
// 1. Options struct
type MyComponentOption func(*myComponentOptions)

type myComponentOptions struct {
    output       io.Writer
    pollInterval time.Duration
}

func defaultMyComponentOptions() *myComponentOptions {
    return &myComponentOptions{
        output:       nil, // uses defaultOutput
        pollInterval: DefaultPollInterval,
    }
}

// 2. Public runner with TTY check
func RunMyComponent(ctx context.Context, title string, opts ...MyComponentOption) error {
    options := defaultMyComponentOptions()
    for _, opt := range opts {
        opt(options)
    }
    output := options.getOutput()

    if !IsTTY() {
        return runMyComponentNonTTY(ctx, title, output)
    }

    // Bubble Tea model + program
    state := NewCancellableState(ctx)
    m := myModel{state: state, ...}
    p := tea.NewProgram(m, tea.WithOutput(output))
    // ...
}

// 3. Non-TTY fallback
func runMyComponentNonTTY(ctx context.Context, title string, output io.Writer) error {
    printNonTTYNoticeTo(output)
    fmt.Fprintf(output, "%s %s\n", IconRunning, title)
    // Simple polling/output loop
}
```

Key patterns:
- Use `CancellableState` for context + cancel handling
- Use `HandleCancelKeys(msg, state)` in Update for ctrl+c/q
- Use `NewDefaultSpinner()` for consistent spinner styling
- Always implement non-TTY fallback

## Testing

```go
func TestMyComponent(t *testing.T) {
    // 1. Setup test environment (disables TTY, resets notice)
    cleanup := setupTestEnvironment(false)
    defer cleanup()

    // 2. Capture output
    buf := &bytes.Buffer{}
    
    // 3. Run with injected output
    err := ui.RunMyComponent(ctx, "Test", ui.WithMyComponentOutput(buf))
    
    // 4. Assert on captured output
    if !strings.Contains(buf.String(), "expected text") {
        t.Error("...")
    }
}

// setupTestEnvironment helper (in ui_test.go)
func setupTestEnvironment(ttyEnabled bool) func() {
    restoreTTY := SetTTYDetector(func() bool { return ttyEnabled })
    ResetNonTTYNotice()
    return func() {
        restoreTTY()
        ResetNonTTYNotice()
    }
}
```

## Checklist

- [ ] Component file created in `internal/ui/`
- [ ] Options struct with `With*` functions
- [ ] TTY fallback with `printNonTTYNoticeTo(output)`
- [ ] Output writer injectable via option
- [ ] Uses `CancellableState` and `HandleCancelKeys`
- [ ] Unit tests in `ui_test.go`
- [ ] `go build ./...` passes
- [ ] `go test ./internal/ui/...` passes

## Related Skills

- **kindplane-commands**: CLI command creation (uses these components)
- **vhs-tapes**: Recording command demos
