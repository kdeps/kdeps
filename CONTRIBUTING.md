# Contributing to KDeps

First off, thank you for considering contributing to KDeps!

KDeps v2 is a complete rewrite focusing on a "local-first" execution model, YAML configuration, and a Unified API. We welcome contributions that help make AI workflow orchestration simpler and more efficient.

## Ways to Contribute
- **Reporting Bugs**: Open an issue with a clear description, reproduction steps, and expected vs. actual behavior.
- **Suggesting Features**: We love new ideas! Open an issue to discuss your proposal.
- **Documentation**: Help us improve the `docs/v2` folder. Documentation is as important as code.
- **Code**: Submit Pull Requests for bug fixes or new features.

## Getting Started

### Prerequisites
- **Go**: v1.24 or later.
- **uv**: Recommended for Python-related features.
- **Docker**: Optional, needed for containerization features.
- **golangci-lint v2**: For code linting (`go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`).

### Environment Setup
1. Clone the repository:
   ```bash
   git clone https://github.com/kdeps/kdeps.git
   cd kdeps
   ```
2. Install dependencies:
   ```bash
   make deps
   ```

## Development Workflow

### Building
To build the `kdeps` binary locally:
```bash
make build
```
The binary will be created in the root directory as `./kdeps`.

### Testing
We maintain ~70% test coverage across unit, integration, and E2E tests.

- **Run all tests**: `make test` (runs fmt, lint, build, and unit tests)
- **Unit Tests**: `make test-unit` (tests in `pkg/` and `cmd/`)
- **Integration Tests**: `make test-integration` (tests in `tests/integration/`)
- **E2E Tests**: `make test-e2e` (tests in `tests/e2e/`)

**Note**: Some tests require Docker daemon to be running. Tests that interact with Docker are automatically skipped when using the `-short` flag.

### Linting and Formatting
Before submitting a PR, ensure your code is formatted and linted:
```bash
make fmt
make lint
```

## Pull Request Guidelines
1. **Discuss First**: For large changes, please open an issue first to discuss the approach.
2. **Feature Branch**: Create a new branch for your changes (e.g., `feat/add-new-resource`).
3. **Tests Required**: Every code change should include corresponding tests.
4. **Documentation**: Update relevant files in `docs/v2` if you change or add configuration options.
5. **Clean History**: Keep your commits focused and provide clear commit messages.
6. **PR Description**: Describe *why* the change is needed and *what* it accomplishes.

## Project Architecture
- `cmd/`: CLI command implementations (using Cobra).
- `pkg/domain/`: Core data models and interfaces (no external dependencies).
- `pkg/parser/`: YAML and Expression parsing logic.
- `pkg/executor/`: The execution engine and individual resource executors (LLM, SQL, etc.).
- `pkg/infra/`: External integrations like Docker and Storage.
- `docs/v2/`: VitePress documentation source.

## Release Process

### Regular Releases
Regular versioned releases are triggered when a tag matching `v*` is pushed to the repository. The release workflow automatically builds binaries for all supported platforms and publishes them.

### Nightly Releases
KDeps has an automated nightly release process that:
- Runs daily at 2 AM UTC
- Updates all Go modules to their latest versions
- Validates the updates with linting, building, and testing
- Commits the updated `go.mod` and `go.sum` to the main branch
- Creates a nightly release with the tag format `nightly-YYYYMMDD-HHMM`
- Publishes binaries with the latest dependencies
- **Release status**: Marked as "latest" when all validation checks pass, or as "prerelease" if any linting, build, or test failures occur

The nightly workflow can also be manually triggered via the GitHub Actions UI if needed. If no module updates are available, the workflow will skip the release process.

## License
By contributing to KDeps, you agree that your contributions will be licensed under the project's [Apache 2.0 License](LICENSE).