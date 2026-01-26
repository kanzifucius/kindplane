---
description: Go development guidelines including formatting, linting, testing, and code patterns for the kindplane project.
applyTo: "**/*.go"
---

- **Code Formatting:**
  - Always run `gofmt` and `goimports` before committing
  - Use `task fmt` to format all Go files
  - Imports should be grouped: stdlib, external, internal (github.com/kanzi/kindplane)
  - The project uses [.golangci.yml](.golangci.yml) for linting configuration

- **Linting:**
  - Run `task lint` before committing to catch issues early
  - Run `task lint:fix` to auto-fix linting issues where possible
  - Run `task check` for full quality assurance (fmt, lint, test)
  - Key linters enabled: errcheck, govet, staticcheck, gosec, errorlint

- **Testing:**
  - Run `task test` to execute all tests with race detection
  - Run `task test:coverage` for coverage reports
  - Run `task test:short` for quick feedback during development
  - Use table-driven tests with descriptive test case names
  - Skip flaky/E2E tests with environment variable checks:
    ```go
    if _, ok := os.LookupEnv("KIND_E2E"); !ok {
        t.Skip("skipping Kind E2E tests")
    }
    ```

- **Error Handling:**
  - Always wrap errors with context using `fmt.Errorf("context: %w", err)`
  - Check error returns immediately after function calls
  - Use typed errors or sentinel values for specific error conditions
  - Handle context cancellation explicitly in long-running operations

- **Concurrency Patterns:**
  - Use `context.Context` as the first parameter for cancellable operations
  - Set channels to `nil` after closing to prevent busy-loops in select statements
  - Use `k8s.io/apimachinery/pkg/util/wait` for retry/backoff patterns
  - Avoid spawning goroutines in loops without proper lifecycle management

- **Kubernetes Client Patterns:**
  - Use retry/backoff when calling APIs that may still be bootstrapping
  - Prefer typed clients over dynamic clients where possible
  - Always pass context to Kubernetes API calls

- **Project Commands:**
  - `task build` - Build the binary
  - `task dev:run -- <command>` - Build and run any kindplane command
  - `task fmt` - Format code with gofmt and goimports
  - `task lint` - Run golangci-lint
  - `task lint:fix` - Auto-fix linting issues
  - `task test` - Run tests with race detection
  - `task check` - Run fmt, lint, and test together
  - `task tools:install` - Install required development tools

- **Dependencies:**
  - Run `task build:tidy` to tidy and verify go.mod
  - Don't invent dependency versions; use `go get` to add latest versions
  - Keep dependencies minimal and prefer stdlib where possible
