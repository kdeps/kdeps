# Contributing to kdeps

Thank you for considering a contribution to kdeps.

kdeps is an AI appliance builder: you define what an agent does in YAML and run
it locally or deploy it anywhere. Contributions that make AI workflow
orchestration simpler, faster, or more reliable are welcome.

By contributing, you agree that your contributions are licensed under the
project's [Apache 2.0 License](LICENSE), and you agree to follow the
[code of conduct](CODE_OF_CONDUCT.md).

## Ways to contribute

- **Report a bug**: open an issue with a clear description, reproduction steps,
  and expected versus actual behavior.
- **Suggest a feature**: start a thread in
  [Discussions](https://github.com/kdeps/kdeps/discussions).
- **Improve the docs**: edit the pages under `docs/v2/`. Follow the
  [documentation style guide](docs/v2/STYLE-GUIDE.md) and
  [docs contributing guide](docs/v2/CONTRIBUTING-DOCS.md).
- **Write code**: submit a pull request for a bug fix or a feature.

## Before you start

Install:

- **Go** 1.24 or later.
- **golangci-lint v2**:
  `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`.
- **uv** (recommended) for Python-backed resources.
- **Docker** (optional) for container image builds.
- **A C compiler** on Windows: `go-sqlite3` needs cgo. Without `gcc` on `PATH`,
  `CGO_ENABLED` auto-disables and sqlite-backed tests fail at runtime.

Then:

```bash
git clone https://github.com/kdeps/kdeps.git
cd kdeps
make deps
```

## Development workflow

### Build

```bash
make build      # produces ./kdeps in the repo root
```

Use `./kdeps` for local verification. Do not rely on a globally installed
binary - it is not updated by `make build`.

### Test

```bash
make test              # fmt, lint, build, unit tests
make test-integration  # tests in tests/integration/
make test-e2e          # tests in tests/e2e/
```

Some tests need a running Docker daemon and are skipped with `-short`.

### Lint

```bash
make fmt
make lint
```

Fix every lint issue before opening a pull request, including pre-existing ones
in files you touched.

## Pull request guidelines

1. **Discuss large changes first** in an issue or discussion.
2. **Branch** from `main` (for example `feat/add-x` or `fix/y`).
3. **Add tests** for every code change: unit, and at least one integration and
   one end-to-end test for a new CLI flag, resource, or scanner.
4. **Update the docs** in the same pull request: `README.md`, every affected
   page under `docs/v2/`, and the matching chapters under `book/`. Stale docs
   are a bug.
5. **Validate the docs**: `cd docs/v2 && npm run build`.
6. **Keep commits focused** with clear messages.
7. **Explain the why** in the pull request description, not just the what.

## Project layout

| Path | Contents |
| :--- | :--- |
| `cmd/` | CLI commands (Cobra) |
| `pkg/domain/` | Core data models and interfaces |
| `pkg/parser/` | YAML and expression parsing |
| `pkg/executor/` | Execution engine and resource executors |
| `pkg/agent/` | Agent loop mode |
| `pkg/infra/` | External integrations (Docker, storage) |
| `docs/v2/` | VitePress documentation source |

## Releases

- **Versioned releases** are triggered by pushing a `v*` tag. The release
  workflow builds and publishes binaries for all supported platforms.
- **Nightly releases** run daily at 02:00 UTC: update Go modules, validate,
  commit `go.mod` and `go.sum`, and publish a `nightly-YYYYMMDD-HHMM` release.

## See also

- [Code of conduct](CODE_OF_CONDUCT.md)
- [Security policy](SECURITY.md)
- [Support](SUPPORT.md)
- [Governance](GOVERNANCE.md)
