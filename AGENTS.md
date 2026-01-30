# Agent Guidelines for kindplane

This document provides guidelines for AI coding agents working in the kindplane repository.
kindplane is a CLI tool that bootstraps Kind clusters with Crossplane, cloud providers, and Helm charts.

## Project Overview

- **Language**: Go 1.25+
- **Build Tool**: [Taskfile](https://taskfile.dev/) (see `Taskfile.yaml`)
- **CLI Framework**: [Cobra](https://github.com/spf13/cobra)
- **TUI Library**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lipgloss](https://github.com/charmbracelet/lipgloss)
- **Module Path**: `github.com/kanzi/kindplane`

## Build Commands

```bash
# Build the binary
task build

# Build and run any command
task dev:run -- <command>    # e.g., task dev:run -- status

# Install to $GOPATH/bin
task install

# Tidy and verify dependencies
task build:tidy

# Clean build artifacts
task build:clean
```

## Testing Commands

```bash
# Run all tests with race detection
task test

# Run a single test
go test -v -race ./internal/config -run TestValidate

# Run tests in a specific package
go test -v -race ./internal/helm/...

# Run short tests only (skip slow/E2E tests)
task test:short

# Run tests with coverage report
task test:coverage
```

### E2E Tests

Tests requiring a Kind cluster check for `KIND_E2E` environment variable:

```go
if _, ok := os.LookupEnv("KIND_E2E"); !ok {
    t.Skip("skipping Kind E2E tests")
}
```

## Linting and Formatting

```bash
# Run linter
task lint

# Auto-fix linting issues
task lint:fix

# Format code (gofmt + goimports)
task fmt

# Run all quality checks (fmt, lint, test)
task check
```

Configuration: `.golangci.yml`
Key linters enabled: `errcheck`, `govet`, `staticcheck`, `errorlint`, `misspell`, `bodyclose`

## Code Style Guidelines

### Import Organization

Group imports in this order, separated by blank lines:

1. Standard library
2. External packages
3. Internal packages (`github.com/kanzi/kindplane/...`)

```go
import (
    "context"
    "fmt"
    "os"

    "github.com/spf13/cobra"
    "k8s.io/client-go/kubernetes"

    "github.com/kanzi/kindplane/internal/config"
    "github.com/kanzi/kindplane/internal/ui"
)
```

### Error Handling

- Always wrap errors with context using `fmt.Errorf("context: %w", err)`
- Check error returns immediately after function calls
- Handle context cancellation explicitly in long-running operations

```go
// Good
if err := doSomething(); err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// Bad
err := doSomething()
// ... other code ...
if err != nil { ... }
```

### Naming Conventions

- Use `camelCase` for unexported, `PascalCase` for exported
- Receiver names: short (1-2 letters), consistent within type
- Interfaces: verb-er suffix when describing behavior (e.g., `Installer`, `Validator`)
- Avoid stuttering: `config.Config` not `config.ConfigConfig`

### Function Signatures

- Use `context.Context` as first parameter for cancellable operations
- Return `error` as the last return value
- Keep parameter lists short; use option structs for many parameters

```go
func (i *Installer) Install(ctx context.Context, spec ChartSpec) error
```

### Testing Patterns

- Use table-driven tests with descriptive case names
- Test files: `*_test.go` in the same package
- Use subtests with `t.Run()` for multiple cases

```go
func TestGetContextName(t *testing.T) {
    testCases := []struct {
        name        string
        clusterName string
        expectedCtx string
    }{
        {
            name:        "standard cluster name",
            clusterName: "my-cluster",
            expectedCtx: "kind-my-cluster",
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            got := GetContextName(tc.clusterName)
            if got != tc.expectedCtx {
                t.Errorf("got %s, want %s", got, tc.expectedCtx)
            }
        })
    }
}
```

### Concurrency Patterns

- When a receive operation returns `ok==false`, the receiver should set its channel variable to `nil` to prevent that select case from firing (avoids busy-loops); the closer does not set the channel to nil—the receiver does
- Use `k8s.io/apimachinery/pkg/util/wait` for retry/backoff patterns
- Avoid spawning goroutines in loops without proper lifecycle management

### Kubernetes Client Patterns

- Use retry/backoff when calling APIs that may still be bootstrapping
- Prefer typed clients over dynamic clients where possible
- Always pass context to Kubernetes API calls

## Project Structure

```
cmd/
  kindplane/         # Main entry point
  gendocs/           # Documentation generator
  genschema/         # JSON schema generator
internal/
  cmd/               # Cobra commands
  config/            # Configuration loading/validation
  crossplane/        # Crossplane installation logic
  helm/              # Helm chart management
  kind/              # Kind cluster operations
  ui/                # TUI components and styles
  credentials/       # Cloud credential management
  diagnostics/       # Troubleshooting utilities
```

## UI/TUI Guidelines

- Use the `ui` package for consistent styling (see `internal/ui/styles.go`)
- Use lipgloss adaptive colors for light/dark terminal support
- Console output helpers: `ui.Success()`, `ui.Error()`, `ui.Warning()`, `ui.Info()`

## Documentation

```bash
# Generate CLI reference docs
task docs:generate

# Generate JSON schema for config
task schema:generate

# Serve docs locally
task docs:serve
```

### Keeping Documentation Updated

Documentation must be updated alongside code changes:

- **CLI commands**: When adding or modifying commands, run `task docs:generate` to regenerate CLI reference docs in `docs/cli-reference/`
- **Configuration options**: When changing config structs, run `task schema:generate` to update `kindplane.schema.json`
- **New features**: Update relevant docs in `docs/` directory (user guides, examples and commands)
- **Breaking changes**: Document migration steps and update the changelog
- **VHS recordings**: When modifying CLI output or adding commands, update VHS tape files in `vhs/` and regenerate GIFs with `task vhs:single TAPE=<name>`

## Cursor Rules Reference

This project includes cursor rules in `.cursor/rules/`:

- `go_development.mdc` - Go-specific guidelines
- `cursor_rules.mdc` - Rule authoring guidelines
- `self_improve.mdc` - Rule improvement triggers

When patterns repeat across 3+ files, consider adding a new rule.
