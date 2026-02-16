---
layout: home

hero:
  name: KDeps
  text: Workflow Orchestration Framework
  tagline: Build stateful REST APIs with YAML configuration - handle auth, data flow, storage, and validation without writing boilerplate code
  image:
    src: /logo.svg
    alt: KDeps Logo
  actions:
    - theme: brand
      text: Get Started
      link: /getting-started/installation
    - theme: alt
      text: View on GitHub
      link: https://github.com/kdeps/kdeps

features:
  - icon: 📝
    title: YAML-First Configuration
    details: Define workflows with simple, readable YAML. No complex programming required.
  - icon: ⚡
    title: Fast Local Development
    details: Sub-second startup time. Run locally during development, Docker only for deployment.
  - icon: 🔌
    title: Unified API
    details: Just get() and set() - access data from any source without memorizing 15+ functions.
  - icon: 🎨
    title: Mustache Expressions
    details: Simpler syntax for variables - 56% less typing. Mix with expr-lang for complex logic.
  - icon: 🤖
    title: LLM Integration
    details: Ollama for local models, or any OpenAI-compatible API endpoint.
  - icon: 🗄️
    title: Built-in SQL Support
    details: PostgreSQL, MySQL, SQLite, SQL Server, Oracle with connection pooling.
  - icon: 🐳
    title: Docker Ready
    details: Package everything into optimized Docker images with optional GPU support.
---

# Introduction

KDeps is a YAML-based workflow orchestration framework for building stateful REST APIs. Built on **~92,000 lines of Go code** with **70% test coverage**, it packages AI tasks, data processing, and API integrations into portable units, eliminating boilerplate code for common patterns like authentication, data flow, storage, and validation.

## Technical Highlights

**Architecture**: Clean architecture with 5 distinct layers (CLI → Executor → Parser → Domain → Infrastructure)

**Scale**: 218 source files, 26 CLI commands, 5 resource executor types, 14 working examples

**Testing**: 13 integration tests + 35 e2e scripts ensuring production readiness

**Multi-Target**: Native CLI, Docker containers, and WebAssembly for browser execution

## Key Highlights

### YAML-First Configuration
Build workflows using simple, self-contained YAML configuration blocks. No complex programming required - just define your resources and let KDeps handle the orchestration.

```yaml
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: my-agent
  version: "1.0.0"
  targetActionId: responseResource
settings:
  apiServerMode: true
  apiServer:
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]
```

### Fast Local Development
Run workflows instantly on your local machine with sub-second startup time. Docker is optional and only needed for deployment.

```bash
# Run locally (instant startup)
kdeps run workflow.yaml

# Hot reload for development
kdeps run workflow.yaml --dev
```

### Unified API
Access data from any source with just two functions: `get()` and `set()`. No more memorizing 15+ different function names.

<div v-pre>

```yaml
# All of these work with get():
query: get('q')                    # Query parameter
auth: get('Authorization')         # Header
data: get('llmResource')           # Resource output
user: get('user_name', 'session')  # Session storage
```

</div>

### Mustache Expressions

KDeps v2 supports both traditional expr-lang and simpler Mustache-style expressions. Choose what fits your needs!

<div v-pre>

```yaml
# Traditional expr-lang (full power)
prompt: "{{ get('q') }}"
time: "{{ info('current_time') }}"

# Mustache (simpler - 56% less typing!)
prompt: "{{q}}"
time: "{{current_time}}"

# Mix them naturally in the same workflow
message: "Hello {{name}}, your score is {{ get('points') * 2 }}"
```

</div>

**Key Benefits:**
- **56% less typing** for simple variables
- **No whitespace rules** - <code v-pre>{{var}}</code> = <code v-pre>{{ var }}</code>
- **Backward compatible** - all existing workflows work
- **Natural mixing** - simple and complex together

**When to use:**
- Mustache for simple variables: <code v-pre>{{name}}</code>, <code v-pre>{{email}}</code>
- expr-lang for functions and logic: <code v-pre>{{ get('x') }}</code>, <code v-pre>{{ a + b }}</code>

[Learn more →](/concepts/expressions)

### LLM Integration
Use Ollama for local model serving, or connect to any OpenAI-compatible API endpoint.

| Backend | Description |
|---------|-------------|
| Ollama | Local model serving (default) |
| OpenAI-compatible | Any API endpoint with OpenAI-compatible interface |

### Core Features
- **Session persistence** with SQLite or in-memory storage
- **Connection pooling** for databases
- **Retry logic** with exponential backoff
- **Response caching** with TTL
- **CORS configuration** for web applications
- **WebServer mode** for static files and reverse proxying

## Quick Start

```bash
# Install KDeps (Mac/Linux)
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh

# Or via Homebrew (Mac)
brew install kdeps/tap/kdeps

# Create a new agent interactively
kdeps new my-agent
```

## Example: Simple Chatbot

**workflow.yaml**
<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: chatbot
  version: "1.0.0"
  targetActionId: responseResource
settings:
  apiServerMode: true
  apiServer:
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]
  agentSettings:
    models:
      - llama3.2:1b
```

</div>

**resources/llm.yaml**
<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: llmResource
  name: LLM Chat
run:
  chat:
    model: llama3.2:1b
    prompt: "{{ get('q') }}"
    jsonResponse: true
    jsonResponseKeys:
      - answer
```

</div>

**resources/response.yaml**
```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: responseResource
  requires:
    - llmResource
run:
  apiResponse:
    success: true
    response:
      data: get('llmResource')
```

**Test it:**
```bash
kdeps run workflow.yaml
curl -X POST http://localhost:16395/api/v1/chat -d '{"q": "What is AI?"}'
```

## Architecture

KDeps implements clean architecture with five distinct layers:

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

### Resource Executors

Five built-in executor types handle different workload types:

| Executor | Implementation | Features |
|----------|----------------|----------|
| **LLM** | 8 files | Ollama, OpenAI-compatible, streaming, tools |
| **HTTP** | 2 files | REST APIs, auth, retries, caching |
| **SQL** | 4 files | 5 database drivers, connection pooling |
| **Python** | 3 files | uv integration (97% smaller images) |
| **Exec** | 3 files | Secure shell command execution |

### Design Patterns

- **Clean Architecture**: Zero external dependencies in domain layer
- **Graph-Based Orchestration**: Topological sorting with cycle detection
- **Dependency Injection**: Interface-based validators and executors
- **Registry Pattern**: Dynamic executor registration
- **Adapter Pattern**: Domain → executor config conversion

### Documentation

### Getting Started
- [Installation](getting-started/installation) - Install KDeps on your system
- [Quickstart](getting-started/quickstart) - Build your first workflow

### Configuration
- [Workflow](configuration/workflow) - Workflow configuration reference
- [Session & Storage](configuration/session) - Session persistence and storage
- [CORS](configuration/cors) - Cross-origin resource sharing
- [Advanced](configuration/advanced) - Imports, request object, agent settings

### Resources
- [Overview](resources/overview) - Resource types and common configuration
- [LLM (Chat)](resources/llm) - Language model integration
- [LLM Backends](resources/llm-backends) - Supported LLM backends
- [HTTP Client](resources/http-client) - External API calls
- [SQL](resources/sql) - Database queries
- [Python](resources/python) - Python script execution
- [Exec](resources/exec) - Shell command execution
- [API Response](resources/api-response) - Response formatting

### Concepts
- [Unified API](concepts/unified-api) - get(), set(), file(), info()
- [Expression Helpers](concepts/expression-helpers) - json(), safe(), debug(), default()
- [Expressions](concepts/expressions) - Expression syntax
- [Expression Functions Reference](concepts/expression-functions-reference) - Complete function reference
- [Advanced Expressions](concepts/advanced-expressions) - Advanced expression features
- [Request Object](concepts/request-object) - HTTP request data and file access
- [Input Object](concepts/input-object) - Property-based request body access
- [Tools](concepts/tools) - LLM function calling
- [Items Iteration](concepts/items) - Batch processing with item object
- [Validation](concepts/validation) - Input validation and control flow
- [Error Handling](concepts/error-handling) - onError with retries and fallbacks
- [Route Restrictions](concepts/route-restrictions) - HTTP method and route filtering

### Deployment
- [Docker](deployment/docker) - Build and deploy Docker images
- [WebServer Mode](deployment/webserver) - Serve frontends and proxy apps

### Tutorials
- [Building a Chatbot](tutorials/chatbot)
- [File Upload Processing](tutorials/file-upload)
- [Multi-Database Workflow](tutorials/multi-database)


## Why KDeps v2?

| Feature | v1 (PKL) | v2 (YAML) |
|---------|----------|-----------|
| Configuration | PKL (Apple's language) | Standard YAML |
| Functions | 15+ to learn | 2 (get, set) |
| Startup time | ~30 seconds | < 1 second |
| Docker | Required | Optional |
| Python env | Anaconda (~20GB) | uv (97% smaller) |
| Learning curve | 2-3 days | ~1 hour |

## Examples

Explore working examples:

- [Simple Chatbot](https://github.com/kdeps/kdeps/tree/main/examples/chatbot) - LLM chatbot
- [ChatGPT Clone](https://github.com/kdeps/kdeps/tree/main/examples/chatgpt-clone) - Full chat UI
- [File Upload](https://github.com/kdeps/kdeps/tree/main/examples/file-upload) - File processing
- [HTTP Advanced](https://github.com/kdeps/kdeps/tree/main/examples/http-advanced) - API integration
- [SQL Advanced](https://github.com/kdeps/kdeps/tree/main/examples/sql-advanced) - Multi-database
- [Batch Processing](https://github.com/kdeps/kdeps/tree/main/examples/batch-processing) - Items iteration
- [Tools](https://github.com/kdeps/kdeps/tree/main/examples/tools) - LLM function calling
- [Vision](https://github.com/kdeps/kdeps/tree/main/examples/vision) - Image processing

## Community

- **GitHub**: [github.com/kdeps/kdeps](https://github.com/kdeps/kdeps)
- **Issues**: [Report bugs and request features](https://github.com/kdeps/kdeps/issues)
- **Contributing**: [CONTRIBUTING.md](https://github.com/kdeps/kdeps/blob/main/CONTRIBUTING.md)
- **Examples**: [Browse example workflows](https://github.com/kdeps/kdeps/tree/main/examples)
