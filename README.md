![version](https://img.shields.io/github/v/tag/kdeps/kdeps?style=flat-square&label=version)
![license](https://img.shields.io/github/license/kdeps/kdeps?style=flat-square)
![build](https://img.shields.io/github/actions/workflow/status/kdeps/kdeps/build-test.yml?branch=main&style=flat-square)
[![Go Report Card](https://goreportcard.com/badge/github.com/kdeps/kdeps)](https://goreportcard.com/report/github.com/kdeps/kdeps)
[![tests](https://img.shields.io/endpoint?style=flat-square&url=https://gist.githubusercontent.com/jjuliano/ce695f832cd51d014ae6d37353311c59/raw/kdeps-go-tests.json)](https://github.com/kdeps/kdeps/actions/workflows/build-test.yml)
[![coverage](https://img.shields.io/endpoint?style=flat-square&url=https://gist.githubusercontent.com/jjuliano/ce695f832cd51d014ae6d37353311c59/raw/kdeps-go-coverage.json)](https://github.com/kdeps/kdeps/actions/workflows/build-test.yml)

# kdeps - Workflow Orchestration for Stateful APIs

KDeps is a YAML-based workflow orchestration framework that packages AI tasks, data processing, and API integrations into portable, containerized units. It simplifies building stateful REST APIs by handling common patterns like authentication, data flow, storage, and validation through configuration instead of code.

> **Note:** Prior to v0.6.12, this project used PKL syntax. This new release uses YAML. If you prefer the old PKL syntax, please use version v0.6.12.

## Key Features

This v2 release is a complete rewrite focusing on developer experience and performance:

- ✅ **YAML Configuration** - Familiar syntax, no new language to learn (replaced PKL).
- ✅ **Unified API** - Smart `get()` and `set()` functions that auto-detect data sources, replacing 15+ specific functions.
- ✅ **Fast Startup** - Runs locally with < 1 sec startup time. Docker is optional for deployment.
- ✅ **uv for Python** - Integrated `uv` support for 97% smaller images and 11x faster builds.
- ✅ **SQL Integration** - Native support for PostgreSQL, MySQL, SQLite, SQL Server, and Oracle with connection pooling.
- ✅ **Interactive Wizard** - Create new agents easily with `kdeps new` (no YAML knowledge needed initially).
- ✅ **Hot Reload** - Auto-reload workflows on file changes in dev mode.
- **Graph-Based Engine** - Automatically handles execution order and data flow between resources.

## Quick Start

### Installation

```bash
# Install via script (Mac/Linux)
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh

# Or via Homebrew (Mac)
brew install kdeps/tap/kdeps

# Or build from source
go install github.com/kdeps/kdeps/v2@latest
```

### Usage

```bash
# Create a new agent interactively
kdeps new my-agent

# Run an existing workflow locally
kdeps run workflow.yaml

# Run with hot reload (dev mode)
kdeps run workflow.yaml --dev

# Validate workflow syntax
kdeps validate workflow.yaml

# Package for Docker (optional deployment)
kdeps package workflow.yaml
kdeps build my-agent-1.0.0.kdeps
```

### CLI Commands

KDeps provides 26 commands organized into categories:

**Development Commands**:
- `kdeps run` - Execute workflows locally with instant startup
- `kdeps validate` - Validate YAML configuration
- `kdeps new` - Interactive wizard for new projects
- `kdeps scaffold` - Add resources to existing projects

**Deployment Commands**:
- `kdeps package` - Package workflow into .kdeps archive
- `kdeps build` - Build Docker images from workflows
- `kdeps export` - Export workflow configurations

**Cloud Commands** (for kdeps.io):
- `kdeps login/logout` - Authenticate with cloud
- `kdeps whoami/account` - Manage account
- `kdeps workflows/deployments` - Manage cloud resources

For complete command reference, see [CLI Documentation](docs/v2/getting-started/cli-reference.md).

## Unified API

The core of KDeps v2 is the Unified API, which simplifies data access. The `get()` function automatically detects where your data is coming from based on a priority chain:

### Priority Chain
1. **Items** (Iteration context)
2. **Memory** (In-memory storage)
3. **Session** (Persistent session storage)
4. **Outputs** (Resource execution results)
5. **Query** (URL query parameters)
6. **Body** (Request body data)
7. **Headers** (HTTP headers)
8. **Files** (Uploaded files & specific patterns)
9. **Metadata** (System info)

### Examples

```yaml
# Get query parameter (auto-detected)
query: get('q')

# Get header (auto-detected)
auth: get('Authorization')

# Get resource output (auto-detected)
data: get('httpResource')

# String interpolation in any field
prompt: "Who is {{ get('q') }}?"

# Store in memory
expr:
  - set('last_query', get('q'))

# Store in session (persists across requests)
expr:
  - set('user_id', get('id'), 'session')
```

## Resource Examples

### LLM Resource
Connect to local Ollama models or any OpenAI-compatible API.

```yaml
run:
  chat:
    model: llama3.2:1b
    prompt: "{{ get('q') }}"
    jsonResponse: true
    jsonResponseKeys:
      - answer
      - confidence
```

### HTTP Client
Make external API calls.

```yaml
run:
  httpClient:
    method: GET
    url: "https://api.example.com/data/{{ get('id') }}"
    headers:
      Authorization: "Bearer {{ get('token') }}"
```

### SQL Database
Execute queries with connection pooling.

```yaml
run:
  sql:
    connectionName: analytics
    query: "SELECT * FROM users WHERE id = $1"
    params:
      - get('user_id')
    format: json
```

### Python Script
Run Python code in an isolated environment using `uv`.

```yaml
run:
  python:
    script: |
      import pandas as pd
      data = {{ get('httpResource') }}
      df = pd.DataFrame(data)
      print(df.describe().to_json())
```

### Shell Execution
Execute shell commands securely.

```yaml
run:
  exec:
    command: "ls"
    args:
      - "-la"
      - "{{ get('directory') }}"
```

## Examples

Explore working examples in the [examples/](examples/) directory:

- **[chatbot](examples/chatbot/)** - Simple LLM chatbot
- **[chatgpt-clone](examples/chatgpt-clone/)** - Full-featured chat interface
- **[file-upload](examples/file-upload/)** - File processing workflow
- **[http-advanced](examples/http-advanced/)** - Complex HTTP integrations
- **[sql-advanced](examples/sql-advanced/)** - Multi-database queries
- **[batch-processing](examples/batch-processing/)** - Items iteration
- **[session-auth](examples/session-auth/)** - Session management
- **[tools](examples/tools/)** - LLM function calling
- **[vision](examples/vision/)** - Image processing with LLMs

## Troubleshooting

### Common Issues

**Binary not found after installation**
```bash
# Ensure $HOME/go/bin is in your PATH
export PATH=$PATH:$HOME/go/bin
```

**Ollama connection refused**
```bash
# Start Ollama service
ollama serve

# Or specify custom URL in workflow
settings:
  agentSettings:
    ollamaUrl: "http://localhost:11434"
```

**Docker build fails**
```bash
# Ensure Docker daemon is running
docker info

# Check Docker permissions
sudo usermod -aG docker $USER
```

**Tests hanging**
```bash
# Run with short flag to skip Docker tests
go test -short ./...
```

## Technical Overview

### Core Components

**Execution Engine** - The heart of KDeps orchestrates workflow execution:
- **Graph**: Topological sorting for dependency resolution
- **Engine**: Resource orchestration with retry logic (1,800+ lines)
- **Context**: State management during execution
- **Registry**: Dynamic executor registration

**Resource Executors** - Five built-in executor types:

| Executor | Files | Features |
|----------|-------|----------|
| **LLM** | 8 files | Ollama, OpenAI-compatible APIs, streaming, function calling |
| **HTTP** | 2 files | External API calls, auth, caching, retries |
| **SQL** | 4 files | PostgreSQL, MySQL, SQLite, MSSQL, Oracle with pooling |
| **Python** | 3 files | Script execution with uv (97% smaller than Anaconda) |
| **Exec** | 3 files | Secure shell command execution |

**Expression Language** - Template engine with `{{ }}` syntax:
```yaml
# Variable access & interpolation
prompt: "Hello {{ get('name') }}"

# Arithmetic operations  
result: {{ 10 + 5 * 2 }}

# Conditional logic
skipCondition:
  - "get('status') == 'disabled'"

# Built-in functions
value: get('key')              # Auto-detect source
data: set('key', 'value')      # Store in memory
user: get('id', 'session')     # Session storage
```

**Multi-Target Support**:
- **Native Go**: CLI and server execution
- **Docker**: Containerized deployments with optimized images
- **WASM**: Browser-side execution (files with `_wasm.go` suffix)

### Key Technologies

**Core Dependencies**:
- `cobra` - CLI framework
- `expr` - Expression evaluation engine
- `yaml.v3` - YAML parsing
- `gojsonschema` - JSON validation

**Database Drivers**:
- PostgreSQL (`lib/pq`)
- MySQL (`go-sql-driver/mysql`)
- SQLite (`mattn/go-sqlite3`)
- SQL Server (`go-mssqldb`)
- Oracle (`go-ora`)

**Infrastructure**:
- `docker/docker` - Docker client API
- `fsnotify` - File watching for hot reload
- `websocket` - WebSocket support

## Architecture

KDeps follows a **clean architecture** pattern with clear separation of concerns across ~92,000 lines of well-structured Go code (218 source files, 70% test coverage).

### Layered Architecture

```
┌─────────────────────────────────────────────────────┐
│                  CLI Layer (cmd/)                    │
│  26 commands: run, build, validate, package, new... │
└─────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────┐
│            Execution Engine (pkg/executor/)          │
│    Graph → Engine → Context → Resource Executors    │
└─────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────┐
│       Parser & Validator (pkg/parser, validator)    │
│       YAML parsing, expression evaluation           │
└─────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────┐
│            Domain Models (pkg/domain/)               │
│      Workflow, Resource, RunConfig, Settings         │
└─────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────┐
│           Infrastructure (pkg/infra/)                │
│  Docker, HTTP, Storage, Python, Cloud, ISO, WASM    │
└─────────────────────────────────────────────────────┘
```

### Project Structure

```
kdeps/
├── cmd/                    # CLI commands (run, build, validate, etc.)
│   ├── run.go             # Execute workflows locally
│   ├── build.go           # Docker image builder
│   ├── package.go         # Workflow packager
│   ├── validate.go        # Configuration validator
│   ├── new.go             # Interactive project wizard
│   └── scaffold.go        # Add resources to projects
├── pkg/
│   ├── domain/            # Core domain models (no external deps)
│   ├── executor/          # Execution engine
│   │   ├── engine.go      # Orchestration engine (1,800+ lines)
│   │   ├── graph.go       # Dependency resolution
│   │   ├── llm/           # LLM executor (Ollama, OpenAI-compatible)
│   │   ├── http/          # HTTP client executor
│   │   ├── sql/           # Database executor (5 drivers)
│   │   ├── python/        # Python script executor (uv)
│   │   └── exec/          # Shell command executor
│   ├── parser/            # YAML and expression parsing
│   │   ├── yaml/          # YAML parser with .kdeps support
│   │   └── expression/    # Template engine ({{ }} syntax)
│   ├── validator/         # Schema & business validation
│   └── infra/             # External integrations
│       ├── docker/        # Docker client & builder (2,800+ lines)
│       ├── http/          # HTTP/WebSocket server (7 files)
│       ├── storage/       # Session & memory storage
│       ├── python/        # uv package management
│       ├── cloud/         # Cloud deployment client
│       ├── iso/           # Bootable ISO generation
│       └── wasm/          # WebAssembly bundler
├── examples/              # 14 working example workflows
├── tests/
│   ├── integration/       # 13 integration test files
│   └── e2e/               # 35 end-to-end test scripts
└── docs/                  # VitePress documentation
```

### Design Patterns

- **Clean Architecture**: Domain layer has zero external dependencies
- **Dependency Injection**: Interfaces for validators and executors
- **Registry Pattern**: Dynamic resource executor registration
- **Adapter Pattern**: Domain config → executor-specific format conversion
- **Graph-Based Execution**: Topological sort for dependency resolution with cycle detection

## Why KDeps?

**Production-Ready Framework**:
- ✅ **Mature Codebase**: ~92,000 lines of well-tested Go code
- ✅ **High Test Coverage**: 70% overall with integration and e2e tests
- ✅ **Clean Architecture**: Domain-driven design with zero external dependencies in core
- ✅ **Extensible**: Registry pattern for easy addition of new executors
- ✅ **Battle-Tested**: 14 working examples covering real-world use cases

**Key Advantages**:
- **Simplified Development**: Configure workflows in YAML instead of writing boilerplate code
- **Portability**: Package everything (code, dependencies, config) into a single deployable unit
- **Flexibility**: Run locally during development, deploy to containers for production
- **Graph-Based Orchestration**: Automatic dependency resolution with topological sorting
- **Error Resilience**: Built-in retry logic with exponential backoff
- **Privacy**: Keep sensitive data on your own infrastructure when needed
- **Control**: Avoid vendor lock-in with containerized, reproducible deployments

## About the Name

> "KDeps, short for 'knowledge dependencies,' is inspired by the principle that knowledge—whether from AI, machines, or humans—can be represented, organized, orchestrated, and interacted with through graph-based systems. The name grew out of my work on Kartographer, a lightweight graph library for organizing and interacting with information." — Joel Bryan Juliano, KDeps creator

## Development

### Building from Source

```bash
# Clone the repository
git clone https://github.com/kdeps/kdeps.git
cd kdeps

# Install dependencies
go mod download

# Build the binary
make build
# or
go build -o kdeps main.go
```

### Running Tests

KDeps maintains **~70% test coverage** across:
- **218 source files** (~92,000 lines of Go code)
- **13 integration tests** in `tests/integration/`
- **35 e2e shell scripts** in `tests/e2e/`

```bash
# Run all tests (unit + integration + e2e)
make test

# Run only unit tests (with -short flag for Docker tests)
make test-unit

# Run integration tests
make test-integration

# Run e2e tests
make test-e2e

# Run linter (golangci-lint v2)
make lint

# Format code
make fmt
```

**Note**: Some tests require Docker daemon. Use `-short` flag to skip: `go test -short ./...`

### Adding New Resource Executors

KDeps is designed for extensibility. To add a new resource type:

1. **Create executor** in `pkg/executor/<name>/`
2. **Implement interface** with `Execute(config, context)` method
3. **Add adapter** to convert domain config to executor format
4. **Register** in executor registry
5. **Add tests** following existing patterns

See `pkg/executor/llm/` or `pkg/executor/http/` for reference implementations.

### Project Structure

```
kdeps/
├── cmd/                    # CLI commands (run, build, validate, etc.)
├── pkg/
│   ├── domain/            # Core domain models
│   ├── executor/          # Resource execution engine
│   ├── parser/            # YAML and expression parsing
│   ├── validator/         # Configuration validation
│   └── infra/             # External integrations
├── examples/              # Example workflows
├── tests/
│   ├── integration/       # Integration tests
│   └── e2e/               # End-to-end tests
└── docs/                  # Documentation
```

## Community & Support

- **Documentation**: [kdeps.io](https://kdeps.io) (coming soon)
- **GitHub Issues**: [Report bugs or request features](https://github.com/kdeps/kdeps/issues)
- **Examples**: [Browse example workflows](https://github.com/kdeps/kdeps/tree/main/examples)
- **Contributing**: See [CONTRIBUTING.md](CONTRIBUTING.md)

## License

Apache 2.0 - See [LICENSE](LICENSE) for details.