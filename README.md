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

## Architecture

KDeps follows a clean architecture to ensure separation of concerns and maintainability.

```
/
├── cmd/                     # CLI commands
│   ├── root.go              # Root command
│   ├── run.go               # Run workflow locally
│   ├── new.go               # Interactive wizard
│   ├── validate.go          # Validate YAML
│   └── build.go             # Build Docker image
│
├── pkg/                     # Core packages
│   ├── domain/              # Domain models (no dependencies)
│   ├── parser/              # YAML & expression parsing
│   ├── validator/           # Schema & business validation
│   ├── executor/            # Resource execution engine
│   └── infra/               # External integrations (Docker, FS)
│
├── examples/                # Example workflows
└── main.go                  # Entry point
```

## Why KDeps?

- **Simplified Development**: Configure workflows in YAML instead of writing boilerplate code.
- **Portability**: Package everything (code, dependencies, config) into a single deployable unit.
- **Flexibility**: Run locally during development, deploy to containers for production.
- **Privacy**: Keep sensitive data on your own infrastructure when needed.
- **Control**: Avoid vendor lock-in with containerized, reproducible deployments.

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

```bash
# Run all tests (unit + integration + e2e)
make test

# Run only unit tests
make test-unit

# Run integration tests
make test-integration

# Run e2e tests
make test-e2e

# Run linter
make lint

# Format code
make fmt
```

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