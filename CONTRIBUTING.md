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
- **Go**: v1.22 or later.
- **uv**: Recommended for Python-related features.
- **Docker**: Optional, needed for containerization features.
- **golangci-lint**: For code linting.

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
We maintain high test coverage across unit, integration, and E2E tests.

- **Run all tests**: `make test`
- **Unit Tests**: `make test-unit` (tests in `pkg/` and `cmd/`)
- **Integration Tests**: `make test-integration` (tests in `tests/integration/`)
- **E2E Tests**: `make test-e2e` (tests in `tests/e2e/`)

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

## License
By contributing to KDeps, you agree that your contributions will be licensed under the project's [Apache 2.0 License](LICENSE).