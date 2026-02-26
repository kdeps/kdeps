<p align="center"><img src="docs/v2/public/kdeps-logo-dark.png" alt="kdeps" width="300" /></p>

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
- ✅ **Mustache Templates** - Support for both Go templates and Mustache syntax for project scaffolding.
- ✅ **Media Input** - First-class `input:` block supporting API, Audio, Video, Telephony, and Bot sources (Discord, Slack, Telegram, WhatsApp) with optional transcription (online/offline) and wake-phrase activation.
- ✅ **Chat Bots** - Connect workflows to Discord, Slack, Telegram, and WhatsApp via persistent polling or serverless stateless mode. Access the inbound message with `input('message')` and reply automatically.
- ✅ **TTS Output** - Built-in Text-to-Speech resource with 5 online providers (OpenAI, Google, ElevenLabs, Azure, AWS Polly) and 4 offline engines (Piper, eSpeak, Festival, Coqui-TTS).
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
- `kdeps push` - Push workflow updates to running containers (live update)
- `kdeps export` - Export workflow configurations

**Cloud Commands** (for kdeps.io):
- `kdeps login/logout` - Authenticate with cloud
- `kdeps whoami/account` - Manage account
- `kdeps workflows/deployments` - Manage cloud resources

For complete command reference, see [CLI Documentation](docs/v2/getting-started/cli-reference.md).

## Live Workflow Updates (`kdeps push`)

Update a running kdeps container without rebuilding the image:

```bash
# Start container with management token
docker run -e KDEPS_MANAGEMENT_TOKEN=mysecret -p 16395:16395 myregistry/myagent:latest

# Push updated workflow directory
kdeps push --token mysecret ./my-agent http://localhost:16395

# Push packaged .kdeps archive
kdeps push --token mysecret myagent-2.0.0.kdeps http://localhost:16395

# Token from env (not stored in shell history)
export KDEPS_MANAGEMENT_TOKEN=mysecret
kdeps push ./my-agent http://localhost:16395
```

The management API (`/_kdeps/*`) is built into every kdeps server and supports:

| Endpoint | Auth | Description |
|----------|------|-------------|
| `GET /_kdeps/status` | — | Workflow name, version, resource count |
| `PUT /_kdeps/workflow` | ✓ | Push YAML → hot-reload |
| `PUT /_kdeps/package` | ✓ | Push `.kdeps` archive → extract → hot-reload |
| `POST /_kdeps/reload` | ✓ | Reload from current on-disk file |

Write endpoints require a bearer token set via `KDEPS_MANAGEMENT_TOKEN`. Unset → `503`, wrong token → `401`.
See [Management API docs](docs/v2/concepts/management-api.md) for full details.

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

## Mustache Expressions

KDeps v2 supports both expr-lang and Mustache-style variable interpolation:

```yaml
# expr-lang (functions and logic)
prompt: "{{ get('q') }}"
result: "{{ score > 80 ? 'Pass' : 'Fail' }}"

# Mustache (simple variables)
prompt: "{{q}}"
message: "Hello {{name}}, score: {{ get('points') * 2 }}"
```

Use Mustache for simple variable access; use expr-lang for function calls, arithmetic, and conditionals. Both styles can be mixed freely in the same workflow. See [Expressions guide](docs/v2/concepts/expressions.md).

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

### Inline Resources
Execute multiple resources before or after the main resource.

```yaml
run:
  # Run before main resource
  before:
    - httpClient:
        method: GET
        url: "https://api.example.com/config"
    - exec:
        command: "echo 'Preparing...'"
  
  # Main resource
  chat:
    model: llama3.2:1b
    prompt: "{{ get('prompt') }}"
  
  # Run after main resource
  after:
    - sql:
        connection: "sqlite3://./db.sqlite"
        query: "INSERT INTO logs VALUES (?)"
    - python:
        script: "print('Done')"
```

### Bot / Chat Platform

Connect a workflow to Discord, Slack, Telegram, or WhatsApp. The inbound message is available via `input('message')`, `input('chatId')`, `input('userId')`, and `input('platform')`.

```yaml
settings:
  input:
    sources: [bot]
    bot:
      executionType: polling   # persistent long-running connection
      telegram:
        botToken: "{{ env('TELEGRAM_BOT_TOKEN') }}"

resources:
  - metadata:
      actionId: reply
    run:
      chat:
        model: llama3.2:1b
        messages:
          - role: user
            content: "{{ input('message') }}"
```

**Stateless mode** — read one message from stdin, run once, write reply to stdout:

```yaml
settings:
  input:
    sources: [bot]
    bot:
      executionType: stateless
```

```bash
echo '{"message":"What is 2+2?","platform":"telegram"}' | kdeps run workflow.yaml
```

## Examples

Explore working examples in the [examples/](examples/) directory:

- **[chatbot](examples/chatbot/)** - Simple LLM chatbot via HTTP API
- **[chatgpt-clone](examples/chatgpt-clone/)** - Full-featured chat interface
- **[telegram-bot](examples/telegram-bot/)** - Telegram bot with LLM replies (polling)
- **[stateless-bot](examples/stateless-bot/)** - One-shot bot execution from stdin
- **[inline-resources](examples/inline-resources/)** - Before/after resource execution
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

KDeps uses clean architecture (~92,000 lines of Go, 70% test coverage) with five layers:

```
CLI (cmd/)  →  Execution Engine (pkg/executor/)  →  Parser/Validator
           →  Domain Models (pkg/domain/)         →  Infrastructure (pkg/infra/)
```

**Resource executors**: LLM (Ollama/OpenAI-compatible), HTTP client, SQL (5 drivers), Python (uv), shell exec, API response, and TTS.

**Project structure:**
```
kdeps/
├── cmd/          # CLI commands
├── pkg/
│   ├── domain/   # Core models (no external deps)
│   ├── executor/ # Resource execution engine
│   ├── parser/   # YAML and expression parsing
│   ├── validator/# Configuration validation
│   └── infra/    # Docker, HTTP, storage, cloud
├── examples/     # Example workflows
├── tests/        # Integration and e2e tests
└── docs/         # VitePress documentation
```

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

## Community & Support

- **Documentation**: [kdeps.io](https://kdeps.io) (coming soon)
- **GitHub Issues**: [Report bugs or request features](https://github.com/kdeps/kdeps/issues)
- **Examples**: [Browse example workflows](https://github.com/kdeps/kdeps/tree/main/examples)
- **Contributing**: See [CONTRIBUTING.md](CONTRIBUTING.md)

## License

Apache 2.0 - See [LICENSE](LICENSE) for details.