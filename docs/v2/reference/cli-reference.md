# CLI Reference

Complete reference for all KDeps command-line interface commands.

## Overview

KDeps provides a simple CLI for creating, running, and deploying AI agents:

```bash
kdeps [command] [options]
```

## Global Flags

All commands support these global flags:

| Flag | Description |
|------|-------------|
| `--verbose` | Enable INFO-level log output (default: WARN) |
| `--debug` | Enable DEBUG-level log output with source locations |
| `--instrument` | Enable call-chain instrumentation tracing |

### Structured Logging

kdeps uses structured JSON logging via Go's `log/slog`. Warnings and errors are written to stderr in a human-readable format by default.

**Log levels:**
| Flag / Env | Level | Output |
|---|---|---|
| (none) | WARN | Warnings and errors only |
| `--verbose` | INFO | Informational messages + above |
| `--debug` or `KDEPS_DEBUG=true` | DEBUG | Debug details + above |

**JSON format for production:**

```bash
# Set env var for JSON output
export KDEPS_LOG_FORMAT=json
kdeps run workflow.yaml
# Output: {"time":"...","level":"WARN","msg":"experimental software",...}
```

```bash
# Enable debug with JSON
KDEPS_LOG_FORMAT=json kdeps run workflow.yaml --debug
```
- Diagnosing performance bottlenecks

Use `--debug` in combination with `--verbose` for maximum detail.

## Commands

### `kdeps run`

Run a workflow locally (default execution mode).

**Usage:**
```bash
kdeps run [workflow.yaml | package.kdeps] [flags]
```

**Arguments:**
- `workflow.yaml` - Path to workflow file or directory containing `workflow.yaml`
- `package.kdeps` - Path to packaged workflow file

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--dev` | Enable hot reload mode | `false` |
| `--port` | API server port number | From workflow config |
| `--debug` | Enable debug logging | `false` |
| `--interactive` | Open an interactive LLM REPL (stdin/stdout) alongside the running workflow. Supports `/run`, `/list`, `/help`, `/quit` slash commands | `false` |
| `--self-test` | Run `tests:` block after server starts, keep running | `false` |
| `--self-test-only` | Run `tests:` block then exit (non-zero on failure) | `false` |
| `--write-tests` | Generate tests from resources and write to workflow file, then exit | `false` |

**Examples:**
```bash
# Run workflow from directory
kdeps run workflow.yaml

# Run workflow from .kdeps package
kdeps run myapp.kdeps

# Run with hot reload (auto-restart on file changes)
kdeps run workflow.yaml --dev

# Run with debug logging
kdeps run workflow.yaml --debug

# Run on specific port
kdeps run workflow.yaml --port 16395

# Auto-generate tests and write them into workflow.yaml, then exit
kdeps run workflow.yaml --write-tests

# Start server and run tests once (keep server running afterwards)
kdeps run workflow.yaml --self-test

# CI/CD: start server, run tests, exit with non-zero status on failure
kdeps run workflow.yaml --self-test-only

# Start interactive LLM REPL alongside normal workflow execution
kdeps run workflow.yaml --interactive
```

**Features:**
- Instant startup (< 1 second)
- Hot reload in dev mode
- Easy debugging
- No Docker overhead
- Built-in self-test runner with auto-generation

**Self-test workflow:**

```bash
# Step 1: scaffold tests from your workflow resources
kdeps run workflow.yaml --write-tests
# -> Appends a tests: block to workflow.yaml

# Step 2: review and edit workflow.yaml tests: section

# Step 3: run them in CI
kdeps run workflow.yaml --self-test-only
echo "Exit code: $?"
```

When no explicit `tests:` block is present, `--self-test` and `--self-test-only` auto-generate smoke tests from the workflow routes and resources at runtime (nothing is written to disk).

---

### `kdeps validate`

Validate workflow configuration against schema and business rules.

**Usage:**
```bash
kdeps validate [workflow.yaml | directory] [flags]
```

**Arguments:**
- `workflow.yaml` - Path to workflow file
- `directory` - Directory containing workflow files

**What it validates:**
- ✅ YAML syntax
- ✅ Schema compliance (JSON Schema)
- ✅ Resource dependencies
- ✅ Expression syntax
- ✅ Circular dependency detection
- ✅ Business rules
- ✅ Static analysis (unreachable resources, bad expression refs, missing component inputs)

**Examples:**
```bash
# Validate workflow file
kdeps validate workflow.yaml

# Validate with verbose output
kdeps validate workflow.yaml --verbose

# Validate all YAML files in directory
kdeps validate .

# Validate packaged workflow
kdeps validate myapp.kdeps
```

**Output:**
```
Validating: workflow.yaml

✓ YAML syntax valid
✓ Schema validation passed
✓ Resource dependencies resolved
✓ No circular dependencies
✓ Expression syntax valid

Workflow validated successfully
```

---

### `kdeps new`

Create a new AI agent with interactive prompts.

**Usage:**
```bash
kdeps new [agent-name] [flags]
```

**Arguments:**
- `agent-name` - Name of the agent (alphanumeric, hyphens allowed)

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--template, -t` | Agent template to use | Interactive selection |
| `--force` | Overwrite existing directory | `false` |

**Available Templates:**
- `api-service` - HTTP client + LLM + response handler
- `sql-agent` - SQL resource with query validation
- `file-processor` - File upload + Python + LLM
- `cli-tool` - Exec resource with input/output

**Examples:**
```bash
# Interactive mode (recommended)
kdeps new my-agent

# Quick start with template
kdeps new my-agent --template api-service

# Overwrite existing directory
kdeps new my-agent --force
```

**Generated Structure:**
```
my-agent/
├── workflow.yaml
├── resources/
│   ├── http_client.yaml
│   ├── llm.yaml
│   └── response.yaml
├── .env.example
└── README.md
```

**Interactive Prompts:**
1. Select agent template
2. Choose required resources
3. Configure basic settings (port, models, etc.)
4. Generate project files

---

### `kdeps edit`

Open the global kdeps configuration file (`~/.kdeps/config.yaml`) in your editor. If the file doesn't exist, it is scaffolded first.

**Usage:**
```bash
kdeps edit [flags]
```

**Editor resolution order:**
1. `$KDEPS_EDITOR`
2. `$VISUAL`
3. `$EDITOR`
4. `vi` (fallback)

**Example:**
```bash
kdeps edit
KDEPS_EDITOR=code kdeps edit   # open in VS Code
```

**Configuration file structure:**
```yaml
# ~/.kdeps/config.yaml

# Global LLM configuration (fallback for agents without a profile)
llm:
  # ollama_host: http://localhost:11434
  # openai_api_key: ""
  # anthropic_api_key: ""
  # ... other provider keys

# Per-agent profiles — override global config per workflow/agent name
agents:
  my-agent:
    llm:
      backend: openai
      openai_api_key: sk-...
    defaults:
      timezone: America/New_York
  my-other-agent:
    llm:
      backend: anthropic
      anthropic_api_key: sk-ant-...

# Global defaults applied to all workflows
defaults:
  # timezone: UTC
  # python_version: "3.12"
  # offline_mode: false

resource_defaults:
  chat:
    timeout: "60s"
    context_length: 4096
  http:
    timeout: "30s"
  python:
    timeout: "60s"
  # ... and more
```

All values are exported as environment variables before workflow execution. Explicit environment variables always take precedence over config file values.

---

### `kdeps doctor`

Run system health checks to diagnose common configuration and environment issues.

**Usage:**

```bash
kdeps doctor [flags]
```

**What it checks:**

| Check | Description |
|---|---|
| Config file | Existence of `~/.kdeps/config.yaml` |
| Config validation | Typos in API key names, missing keys, bad values |
| Ollama | TCP connectivity to the Ollama server |
| Python | `python3` availability in PATH |
| Backend/API key | Cloud backend configured without its API key |
| Agents | Installed agent count |
| Env vars | Critical environment variables set |

**Examples:**

```bash
kdeps doctor
# Output:
# kdeps doctor
# =============
#
#   [PASS] Config file: /home/user/.kdeps/config.yaml
#   [WARN] Config validation: 1 warning(s): unknown llm key "openai_apikey"
#   [PASS] Ollama: reachable at localhost:11434
#   [PASS] Python: python3 available
#   [PASS] Backend/API key: backend=ollama (no API key needed)
#   [PASS] Agents: 2 agent(s) installed
#   [PASS] Env vars: all critical vars set
#
# Overall: healthy
```

Exits with code 1 when any check has FAIL status.

---

### `kdeps chat`

Interactive AI assistant that generates and runs kdeps workflows from natural language.

**Usage:**
```bash
kdeps chat [flags]
```

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--model` | LLM model for workflow generation | From config |
| `--base-url` | LLM backend base URL | `http://localhost:11434` |
| `--api-key` | API key for online LLM providers | From env |
| `--session` | Resume a previous session by ID | New session |
| `--no-execute` | Generate workflow but do not allow `/run` | `false` |

**Slash commands inside the REPL:**
| Command | Description |
|---------|-------------|
| `/show` | Print the generated workflow YAML |
| `/run` | Execute the workflow with `kdeps run` |
| `/save [path]` | Save the workflow to directory |
| `/export` | Show Kubernetes manifests (`kdeps export k8s`) |
| `/reset` | Clear conversation and start fresh |
| `/quit` | Exit |

**Examples:**
```bash
# Start interactive assistant
kdeps chat

# Use a specific model
kdeps chat --model gpt-4o

# Generate but don't allow execution
echo "list files in /tmp" | kdeps chat --no-execute
```

---

### `kdeps registry`

Search, install, and manage packages from the kdeps registry.

**Usage:**
```bash
kdeps registry <subcommand> [flags]
```

**Subcommands:**

| Subcommand | Description |
|------------|-------------|
| `search` | Search for packages in the kdeps registry |
| `info` | Show metadata and README for a package, local component/agent, or GitHub repo |
| `install` | Install from the registry, a GitHub repo (`owner/repo`), or a local archive (`.kdeps` `.kagency` `.komponent`) |
| `uninstall` | Uninstall an agent or component installed from the registry |
| `update` | Update an installed agent or component to a newer version |
| `list` | List installed and local components |
| `submit` | Generate a registry formula YAML for submitting a package via GitHub PR |
| `verify` | Run LLM-agnostic verification on a package directory |

**Examples:**
```bash
kdeps registry search scraper
kdeps registry install scraper
kdeps registry install scraper@2.1.0
kdeps registry install jjuliano/kdeps-component-scraper
kdeps registry install ./scraper-1.0.0.komponent
kdeps registry list
kdeps registry info scraper
kdeps registry uninstall scraper
kdeps registry update scraper
kdeps registry submit --tag v1.2.0
kdeps registry verify .
```

**Publishing a package (GitHub-hosted):**

The kdeps registry is GitHub-hosted. Packages live in the author's own GitHub repo; the registry indexes a formula file per package.

```bash
# 1. Tag a release in your repo
git tag v1.2.0 && git push --tags

# 2. Generate the formula YAML (downloads the tarball and computes SHA256)
kdeps registry submit --tag v1.2.0

# 3. Open a PR to https://github.com/kdeps-io/registry
#    adding the printed formula as formulas/<your-package-name>.yaml
```

The formula file format:
```yaml
name: my-agent
version: 1.2.0
type: agent
github: owner/my-agent-repo
tarball: https://github.com/owner/my-agent-repo/archive/refs/tags/v1.2.0.tar.gz
sha256: <computed-by-kdeps-registry-submit>
description: ...
tags: [llm, chat]
license: Apache-2.0
```

`kdeps registry install` downloads from the GitHub tarball URL and verifies the SHA256 locally.

---

### `kdeps bundle package`

Package workflow or component into an archive for distribution.

**Usage:**
```bash
kdeps bundle package [directory] [flags]
```

**Arguments:**
- `directory` - Directory containing `workflow.yaml`, `agency.yaml`, or `component.yaml`

**Behavior:**
| Detected file | Output format | Archive extension |
|---------------|---------------|-------------------|
| `workflow.yaml` | Workflow package | `.kdeps` |
| `agency.yaml` | Agency package | `.kagency` |
| `component.yaml` | Component package | `.komponent` |

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--output, -o` | Output directory | `.` (current) |
| `--name` | Package name | From metadata (name-version) |

**What it packages:**
- ✅ Main manifest (`workflow.yaml`, `agency.yaml`, or `component.yaml`)
- ✅ All resource files (`resources/`)
- ✅ Python requirements (`requirements.txt`)
- ✅ Data files and scripts
- ✅ HTML/CSS/JS assets (for components)
- Respects `.kdepsignore` exclusions

**Examples:**
```bash
# Package a workflow (creates my-agent-1.0.0.kdeps)
kdeps bundle package my-agent/

# Package an agency (creates my-agency-1.0.0.kagency)
kdeps bundle package my-agency/

# Package a component (creates greeter-1.0.0.komponent)
kdeps bundle package my-component/

# Specify output path
kdeps bundle package my-agent/ --output dist/

# Create with custom name
kdeps bundle package my-agent/ --name custom-agent
```

**Output:**
Creates `{name}-{version}.{kdeps|kagency|komponent}` archive containing all files in a compressed tarball.

---

### `kdeps bundle build`

Build Docker image from workflow (optional, for deployment).

**Usage:**
```bash
kdeps bundle build [path] [flags]
```

**Arguments:**
- `path` - Directory, workflow file, or `.kdeps` package

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--gpu` | GPU support: `cuda` or `rocm` | None (CPU only) |
| `--os` | Base OS: `alpine` or `ubuntu` | Auto-detect |
| `--tag, -t` | Docker image tag | From workflow metadata |
| `--no-cache` | Build without cache | `false` |
| `--push` | Push to registry after build | `false` |

**Accepts:**
- Directory containing `workflow.yaml`
- Direct path to `workflow.yaml` file
- Package file (`.kdeps`)

**Examples:**
```bash
# Build from directory (CPU-only on Alpine)
kdeps bundle build examples/chatbot

# Build from workflow file
kdeps bundle build examples/chatbot/workflow.yaml

# Build with GPU support (NVIDIA CUDA on Ubuntu)
kdeps bundle build examples/chatbot --gpu cuda

# Build with AMD ROCm GPU support
kdeps bundle build examples/chatbot --gpu rocm

# Build with custom tag
kdeps bundle build examples/chatbot --tag my-agent:v1.0.0

# Build from package
kdeps bundle build myapp-1.0.0.kdeps

# Build and push to registry
kdeps bundle build examples/chatbot --tag registry.com/my-agent:v1.0.0 --push
```

**Features:**
- Multi-stage Docker build
- Optimized image size
- uv for Python (97% smaller than Anaconda)
- GPU support (CUDA/ROCm)
- Offline mode support
- Auto-detects base OS

**Note:** Docker is optional. KDeps runs locally by default. Use `build` only for deployment/distribution.

---

### `kdeps export iso`

Export a workflow as a bootable image (ISO, raw disk, or qcow2) using LinuxKit. See `kdeps export iso --help` for the full list of formats and flags.

---

### `kdeps export k8s`

Generate Kubernetes Deployment and Service manifests from a workflow.

**Usage:**
```bash
kdeps export k8s [path] [flags]
```

**Arguments:**
- `path` - Directory containing `workflow.yaml`, a `workflow.yaml` file, or a `.kdeps` package

**Flags:**
| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--image` | `-i` | Container image name | `{name}:{version}` |
| `--output` | `-o` | Output file path | stdout |
| `--replicas` | `-r` | Number of replicas (overrides workflow) | From workflow |

**Examples:**
```bash
# Print manifests to stdout
kdeps export k8s examples/chatbot

# Specify image and save to file
kdeps export k8s examples/chatbot \
  --image my-registry/chatbot:v1.0.0 \
  --output k8s.yaml

# Override replicas
kdeps export k8s examples/chatbot --replicas 5
```

Manifests are driven by `agentSettings` in `workflow.yaml`:
- `replicas` - number of pod replicas
- `resources` - CPU/memory limits and requests
- `env` - container environment variables
- `portNum` inside `apiServer:`/`webServer:` - exposed ports
- `installOllama: true` - adds Ollama backend port (11434)

See [Kubernetes Deployment](../deployment/kubernetes) for full details.

---

### `kdeps serve`

Run a workflow in agent mode - an interactive LLM loop where every resource,
component, and fformat tool is auto-registered as a callable tool.

**Usage:**

```bash
kdeps serve [workflow.yaml] [flags]
```

**Flags:**

| Flag | Default | Description |
|---|---|---|
| `--model` | `KDEPS_AGENT_MODEL` or `llama3.2` | LLM model name |
| `--backend` | `KDEPS_AGENT_BACKEND` or `ollama` | LLM backend |
| `--base-url` | `KDEPS_AGENT_BASE_URL` | LLM API base URL |
| `--system` | (none) | System prompt injected at conversation start |
| `--debug` | false | Enable debug logging |

**Examples:**

```bash
# Start agent REPL with a workflow
kdeps serve workflow.yaml

# Use a specific model
kdeps serve workflow.yaml --model mistral

# Provide a system prompt
kdeps serve workflow.yaml --system "You are a helpful assistant."
```

See [Agent Mode](/concepts/agent-mode) for full details.

---

## Command Workflow

### Typical Development Flow

```bash
# 1. Create new agent
kdeps new my-agent

# 2. Validate configuration
cd my-agent
kdeps validate workflow.yaml

# 3. Generate self-tests from your resources
kdeps run workflow.yaml --write-tests
# -> Appends tests: block to workflow.yaml; review and customise

# 4. Run locally with hot reload
kdeps run workflow.yaml --dev

# 5. Test and iterate
# (Edit files, server auto-reloads)

# 6. Run tests in CI
kdeps run workflow.yaml --self-test-only

# 7. Package for deployment
kdeps bundle package . --output dist/

# 8. Build Docker image (optional)
kdeps bundle build dist/my-agent-1.0.0.kdeps --tag my-agent:latest
```

### Production Deployment Flow

```bash
# 1. Validate before packaging
kdeps validate workflow.yaml

# 2. Package workflow
kdeps bundle package . --output dist/

# 3. Build Docker image
kdeps bundle build dist/my-agent-1.0.0.kdeps \
  --tag registry.com/my-agent:v1.0.0 \
  --gpu cuda

# 4. Push to registry
kdeps bundle build dist/my-agent-1.0.0.kdeps \
  --tag registry.com/my-agent:v1.0.0 \
  --push
```

### Kubernetes Deployment Flow

```bash
# 1. Build and push Docker image
kdeps bundle build . --tag registry.com/my-agent:v1.0.0 --push

# 2. Generate Kubernetes manifests
kdeps export k8s . \
  --image registry.com/my-agent:v1.0.0 \
  --output k8s.yaml

# 3. Deploy to cluster
kubectl apply -f k8s.yaml
kubectl rollout status deployment/my-agent
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | General error |
| `2` | Validation error |
| `3` | Execution error |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `KDEPS_LOG_LEVEL` | Log level: `debug`, `info`, `warn`, `error` |
| `KDEPS_EDITOR` | Editor for `kdeps edit` (overrides `VISUAL`/`EDITOR`) |
| `VISUAL` | Fallback editor for `kdeps edit` |
| `EDITOR` | Second fallback editor for `kdeps edit` (falls back to `vi`) |

| `KDEPS_PYTHON_VERSION` | Global Python version (e.g. `3.12`) |
| `KDEPS_OFFLINE_MODE` | Set `true` to block all external LLM calls |
| `OLLAMA_HOST` | Ollama server URL (e.g. `http://localhost:11434`) |
| `TZ` | Timezone applied to all workflow runs |
| `KDEPS_CHAT_TIMEOUT` | Default timeout for LLM chat resources |
| `KDEPS_CHAT_CONTEXT_LENGTH` | Default context window for LLM chat resources |
| `KDEPS_HTTP_TIMEOUT` | Default timeout for HTTP client resources |
| `KDEPS_PYTHON_TIMEOUT` | Default timeout for Python resources |
| `KDEPS_EXEC_TIMEOUT` | Default timeout for exec resources |
| `KDEPS_SQL_TIMEOUT` | Default timeout for SQL resources |
| `KDEPS_SQL_MAX_ROWS` | Default max rows for SQL query results |
| `KDEPS_ON_ERROR_ACTION` | Default error action: `fail`, `continue`, `retry` |
| `KDEPS_ON_ERROR_MAX_RETRIES` | Default max retries for `retry` action |
| `KDEPS_ON_ERROR_RETRY_DELAY` | Default delay between retries |

## Tips

### Development

- Use `--dev` flag for hot reload during development
- Use `--debug` flag to troubleshoot issues
- Validate frequently with `kdeps validate`

### Production

- Always validate before packaging: `kdeps validate workflow.yaml`
- Use `--gpu cuda` for GPU workloads
- Tag images with version numbers: `--tag my-agent:v1.0.0`

### Troubleshooting

- Check logs with `--debug` flag
- Validate configuration: `kdeps validate workflow.yaml`
- Test locally before building: `kdeps run workflow.yaml`

## Related Documentation

- [Installation](/getting-started/installation) - Install KDeps CLI
- [Quickstart](/getting-started/quickstart) - Build your first agent
- [Workflow Configuration](../configuration/workflow) - Configure workflows
- [Docker Deployment](../deployment/docker) - Production deployment
