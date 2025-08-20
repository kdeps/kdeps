# KDeps Codebase Guide for AI Agents

## Commands
- **Build**: `make build` (builds to bin/kdeps) or `make dev-build` (Linux x86_64)
- **Test**: `make test` or `go test ./...` for all tests, `go test -run TestName ./path/to/package` for single test
- **Lint**: `make format` (runs go vet, gofumpt, golangci-lint) or `golangci-lint run ./...`
- **Coverage**: `make test-coverage` (requires 50%+ coverage to pass)
- **Docs**: `npm run docs:dev` (VitePress docs at docs/)

## Architecture
- **CLI Tool**: Built in Go, creates Dockerized full-stack AI applications using PKL configuration
- **Core Components**: 
  - `cmd/` - CLI commands (add, build, new, package, run)
  - `pkg/resolver/` - Dependency graph resolution and workflow execution
  - `pkg/docker/` - Docker container management and orchestration
  - `pkg/cfg/` - PKL configuration parsing and validation
  - `internal/` - Internal packages for core logic
- **Workflow System**: PKL configs define AI agents with resources (LLM, HTTP, Python, etc.)
- **No Database**: Uses SQLite via mattn/go-sqlite3 for local state

## Code Style & Conventions
- **Imports**: Standard lib first, third-party second, local last with blank lines between groups
- **Naming**: CamelCase exports, camelCase locals, short package names (cfg, utils, cmd)
- **Error Handling**: Explicit checking with `fmt.Errorf("context: %w", err)` wrapping
- **Dependency Injection**: Function variables for testing (newGraphResolverFn, etc.)
- **Logging**: Use `github.com/charmbracelet/log` with structured logging
- **Testing**: Use testify/assert, separate _test packages, NON_INTERACTIVE=1 for CI
- **No Globals**: golangci-lint enforces no global variables except function injection vars
