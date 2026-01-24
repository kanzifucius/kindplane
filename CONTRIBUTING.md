# Contributing

Thank you for your interest in contributing to kindplane!

## Getting Started

1. Fork the repository
2. Clone your fork:

    ```bash
    git clone https://github.com/<your-username>/kindplane.git
    cd kindplane
    ```

3. Add the upstream remote:

    ```bash
    git remote add upstream https://github.com/kanzifucius/kindplane.git
    ```

## Development Setup

### Prerequisites

- Go 1.23 or later
- Docker
- Task (optional, but recommended)

### Building

```bash
# Using Task
task build

# Using Go
go build -o bin/kindplane ./cmd/kindplane
```

### Running Tests

```bash
# Using Task
task test

# Using Go
go test ./...
```

### Linting

```bash
# Using Task
task lint

# Using Go
golangci-lint run
```

## Making Changes

### Create a Feature Branch

```bash
git checkout -b feature/my-feature
```

### Commit Messages

Use conventional commit format:

```
type(scope): description

[optional body]
```

Types:

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `refactor`: Code refactoring
- `test`: Adding tests
- `chore`: Maintenance

Examples:

```
feat(provider): add GCP provider support
fix(helm): handle chart installation timeout
docs(readme): update installation instructions
```

### Code Style

- Follow Go conventions
- Use `gofmt` for formatting
- Run linter before committing
- Add tests for new functionality

### Documentation

- Update docs for new features
- Add command examples
- Include configuration examples

## Pull Requests

### Before Submitting

1. Sync with upstream:

    ```bash
    git fetch upstream
    git rebase upstream/main
    ```

2. Run tests:

    ```bash
    task test
    ```

3. Run linter:

    ```bash
    task lint
    ```

4. Build successfully:

    ```bash
    task build
    ```

### Creating the PR

1. Push your branch:

    ```bash
    git push origin feature/my-feature
    ```

2. Open a Pull Request on GitHub

3. Fill in the PR template:
    - Describe the changes
    - Link related issues
    - Add screenshots if applicable

### Review Process

1. Maintainers will review your PR
2. Address any feedback
3. Once approved, your PR will be merged

## Issue Reporting

### Bug Reports

Include:

- kindplane version
- Go version
- Operating system
- Configuration file (redact secrets)
- Steps to reproduce
- Expected vs actual behaviour
- Error messages/logs

### Feature Requests

Include:

- Use case description
- Proposed solution
- Alternatives considered

## Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Help others learn and grow

## Questions?

- Open a GitHub Discussion
- Check existing issues
- Read the documentation

Thank you for contributing!
