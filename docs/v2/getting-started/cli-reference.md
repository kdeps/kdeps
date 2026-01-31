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
| `--verbose` | Enable verbose output |
| `--debug` | Enable debug logging |

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
kdeps run workflow.yaml --port 8080
```

**Features:**
- Instant startup (< 1 second)
- Hot reload in dev mode
- Easy debugging
- No Docker overhead

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
| `--no-prompt` | Skip interactive prompts | `false` |
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

# Non-interactive mode
kdeps new my-agent --template api-service --no-prompt

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

### `kdeps scaffold`

Add resource files to an existing agent.

**Usage:**
```bash
kdeps scaffold [resource-names...] [flags]
```

**Arguments:**
- `resource-names...` - One or more resource types to add

**Available Resources:**
- `http-client` - HTTP client for API calls
- `llm` - Large Language Model interaction
- `sql` - SQL database queries
- `python` - Python script execution
- `exec` - Shell command execution
- `response` - API response handling

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--dir` | Target directory | `.` (current) |
| `--force` | Overwrite existing files | `false` |

**Examples:**
```bash
# Add single resource
kdeps scaffold llm

# Add multiple resources
kdeps scaffold http-client llm response

# Add to specific directory
kdeps scaffold llm --dir my-agent/

# Overwrite existing files
kdeps scaffold sql --force
```

**What it does:**
- Creates resource YAML files in `resources/` directory
- Auto-updates `workflow.yaml` with new resources
- Generates resource templates with examples
- Preserves existing resources

---

### `kdeps package`

Package workflow into `.kdeps` file for Docker build.

**Usage:**
```bash
kdeps package [workflow-directory] [flags]
```

**Arguments:**
- `workflow-directory` - Directory containing `workflow.yaml`

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--output, -o` | Output directory | `.` (current) |
| `--name` | Package name | From workflow metadata |

**What it packages:**
- ✅ `workflow.yaml`
- ✅ All resource files
- ✅ Python requirements (`requirements.txt`)
- ✅ Data files
- ✅ Scripts

**Examples:**
```bash
# Package workflow
kdeps package my-agent/

# Specify output path
kdeps package my-agent/ --output dist/

# Create with custom name
kdeps package my-agent/ --name custom-agent
```

**Output:**
Creates `{name}-{version}.kdeps` file containing all workflow files in a compressed archive.

---

### `kdeps build`

Build Docker image from workflow (optional, for deployment).

**Usage:**
```bash
kdeps build [path] [flags]
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
kdeps build examples/chatbot

# Build from workflow file
kdeps build examples/chatbot/workflow.yaml

# Build with GPU support (NVIDIA CUDA on Ubuntu)
kdeps build examples/chatbot --gpu cuda

# Build with AMD ROCm GPU support
kdeps build examples/chatbot --gpu rocm

# Build with custom tag
kdeps build examples/chatbot --tag my-agent:v1.0.0

# Build from package
kdeps build myapp-1.0.0.kdeps

# Build and push to registry
kdeps build examples/chatbot --tag registry.com/my-agent:v1.0.0 --push
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

## Command Workflow

### Typical Development Flow

```bash
# 1. Create new agent
kdeps new my-agent

# 2. Add resources
cd my-agent
kdeps scaffold llm http-client

# 3. Validate configuration
kdeps validate workflow.yaml

# 4. Run locally with hot reload
kdeps run workflow.yaml --dev

# 5. Test and iterate
# (Edit files, server auto-reloads)

# 6. Package for deployment
kdeps package . --output dist/

# 7. Build Docker image (optional)
kdeps build dist/my-agent-1.0.0.kdeps --tag my-agent:latest
```

### Production Deployment Flow

```bash
# 1. Validate before packaging
kdeps validate workflow.yaml

# 2. Package workflow
kdeps package . --output dist/

# 3. Build Docker image
kdeps build dist/my-agent-1.0.0.kdeps \
  --tag registry.com/my-agent:v1.0.0 \
  --gpu cuda

# 4. Push to registry
kdeps build dist/my-agent-1.0.0.kdeps \
  --tag registry.com/my-agent:v1.0.0 \
  --push
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
| `KDEPS_CONFIG_PATH` | Path to config file (if supported) |

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

- [Installation](installation) - Install KDeps CLI
- [Quickstart](quickstart) - Build your first agent
- [Workflow Configuration](../configuration/workflow) - Configure workflows
- [Docker Deployment](../deployment/docker) - Production deployment
