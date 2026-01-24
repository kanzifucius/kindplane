# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
### Changed
### Deprecated
### Removed
### Fixed
### Security

---

## [0.2.1] - 2026-01-24

### Added
- Add CODEOWNERS file for code review assignments
- Add GIF demonstrations to documentation

### Changed
- Auto-detect latest version in install script instead of hardcoding

### Fixed
- Documentation enhancements with usage examples and GIF demos

---

## [0.2.0] - 2026-01-23

### Added
- Multi-platform support for different operating systems
- Version check and caching mechanism to notify users of updates
- New commands for managing Crossplane resources and cluster diagnostics
- Schema validation for configuration files
- Documentation badge to README
- GitHub Actions workflow for documentation deployment

### Changed
- Redirect all output to stderr in print functions for better logging
- Enhanced documentation and CLI reference generation
- Updated golangci-lint configuration and CI workflow

---

## [0.1.2] - 2026-01-23

### Added
- New UI components and improved command output styling
- Release-tag task for creating and pushing Git tags
- Quick install instructions to README

### Changed
- Updated dependencies and refactored command execution
- Updated install.sh to v0.1.0

### Fixed
- Tag existence check in Taskfile.yaml
- Updated Go version to 1.25 and fixed format string error

---

## [0.1.0] - 2026-01-23

### Added
- Initial release of kindplane CLI tool
- One-command bootstrap of Kind clusters with Crossplane
- Provider management commands (add, list, remove) for AWS, Azure, GCP, Kubernetes, and Helm providers
- External Secrets Operator (ESO) integration
- Helm chart support with configurable installation phases:
  - Pre-crossplane phase
  - Post-crossplane phase
  - Post-providers phase
  - Post-ESO phase
- GitOps-friendly resource export via `dump` command
- Rich terminal UI with lipgloss styling, colours, icons, and progress indicators
- Smart failure diagnostics with pod logs and conditions
- Credentials configuration for AWS, Azure, and Kubernetes
- Local container registry support for faster image iteration
- Trusted CA certificate support for private container registries
- Core commands:
  - `kindplane init` - Initialise configuration
  - `kindplane up` - Create and bootstrap cluster
  - `kindplane down` - Delete cluster
  - `kindplane status` - Show cluster status
  - `kindplane dump` - Export resources
  - `kindplane provider` - Manage providers
  - `kindplane chart` - Manage Helm charts
  - `kindplane credentials` - Configure credentials
- GitHub Actions workflows for CI/CD
- Installation script for easy setup

[Unreleased]: https://github.com/kanzifucius/kindplane/compare/v0.2.1...HEAD
[0.2.1]: https://github.com/kanzifucius/kindplane/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/kanzifucius/kindplane/compare/v0.1.2...v0.2.0
[0.1.2]: https://github.com/kanzifucius/kindplane/compare/v0.1.0...v0.1.2
[0.1.0]: https://github.com/kanzifucius/kindplane/releases/tag/v0.1.0
