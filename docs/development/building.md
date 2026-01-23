# Building kindplane

This guide covers building kindplane from source.

## Prerequisites

- Go 1.23 or later
- Git
- Docker (for testing)

## Quick Build

```bash
# Clone the repository
git clone https://github.com/kanzifucius/kindplane.git
cd kindplane

# Build
go build -o bin/kindplane ./cmd/kindplane
```

## Using Task

[Task](https://taskfile.dev/) is the recommended way to build:

### Install Task

```bash
# macOS
brew install go-task

# Linux
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /usr/local/bin
```

### Available Tasks

```bash
# List all tasks
task --list

# Build binary
task build

# Build for all platforms
task build-all

# Run tests
task test

# Run linter
task lint

# Build and run
task dev -- up
```

## Build Options

### Debug Build

```bash
go build -gcflags="all=-N -l" -o bin/kindplane ./cmd/kindplane
```

### Release Build

```bash
CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/kindplane ./cmd/kindplane
```

### Cross-Compilation

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o bin/kindplane-linux-amd64 ./cmd/kindplane

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o bin/kindplane-linux-arm64 ./cmd/kindplane

# macOS AMD64
GOOS=darwin GOARCH=amd64 go build -o bin/kindplane-darwin-amd64 ./cmd/kindplane

# macOS ARM64
GOOS=darwin GOARCH=arm64 go build -o bin/kindplane-darwin-arm64 ./cmd/kindplane

# Windows
GOOS=windows GOARCH=amd64 go build -o bin/kindplane-windows-amd64.exe ./cmd/kindplane
```

## Testing

### Run All Tests

```bash
go test ./...
```

### Run with Verbose Output

```bash
go test -v ./...
```

### Run Specific Package

```bash
go test ./internal/config/...
```

### Run with Coverage

```bash
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Linting

### Install golangci-lint

```bash
# macOS
brew install golangci-lint

# Linux
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

### Run Linter

```bash
golangci-lint run
```

### Auto-fix Issues

```bash
golangci-lint run --fix
```

## Project Structure

```
kindplane/
├── cmd/kindplane/          # Main entry point
│   └── main.go
├── internal/
│   ├── cmd/                # CLI commands (Cobra)
│   │   ├── root.go
│   │   ├── init.go
│   │   ├── up.go
│   │   ├── down.go
│   │   ├── status.go
│   │   ├── dump.go
│   │   ├── validate.go
│   │   ├── provider/
│   │   ├── chart/
│   │   └── credentials/
│   ├── config/             # Configuration loading
│   ├── kind/               # Kind cluster management
│   ├── crossplane/         # Crossplane installation
│   ├── helm/               # Helm operations
│   ├── eso/                # External Secrets Operator
│   ├── credentials/        # Credential management
│   ├── diagnostics/        # Failure diagnostics
│   ├── dump/               # Resource export
│   ├── git/                # Git operations
│   └── ui/                 # Terminal UI (lipgloss)
├── go.mod
├── go.sum
├── Taskfile.yaml
└── README.md
```

## Dependencies

Key dependencies:

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/charmbracelet/lipgloss` | Terminal styling |
| `helm.sh/helm/v3` | Helm operations |
| `sigs.k8s.io/kind` | Kind cluster management |
| `k8s.io/client-go` | Kubernetes client |

### Update Dependencies

```bash
go mod tidy
go mod download
```

## Release Process

Releases are automated via GitHub Actions:

1. Tag a release:

    ```bash
    git tag -a v1.0.0 -m "Release v1.0.0"
    git push origin v1.0.0
    ```

2. GitHub Actions will:
    - Build binaries for all platforms
    - Create GitHub release
    - Upload binaries
    - Update install script

## Development Tips

### Quick Development Cycle

```bash
# Build and run in one command
task dev -- up

# Or manually
go run ./cmd/kindplane up
```

### Debugging

Use VS Code with the Go extension:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug kindplane up",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/kindplane",
      "args": ["up"]
    }
  ]
}
```

### Adding a New Command

1. Create file in `internal/cmd/`
2. Define command with Cobra
3. Add to root command in `root.go`
4. Add tests
5. Add documentation
