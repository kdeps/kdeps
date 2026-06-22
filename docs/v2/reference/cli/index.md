# CLI Reference

The `kdeps` CLI creates, runs, tests, packages, and deploys agents. All commands follow `kdeps [command] [options]`.

## Global Flags

All commands support these global flags:

| Flag | Description |
|---|---|
| `--verbose` | Enable INFO-level log output (default: WARN) |
| `--debug` | Enable DEBUG-level log output with source locations |
| `--instrument` | Enable call-chain instrumentation tracing |

### Structured Logging

kdeps uses structured JSON logging via Go's `log/slog`. Warnings and errors go to stderr in human-readable format by default.

| Flag / Env | Level | Output |
|---|---|---|
| (none) | WARN | Warnings and errors only |
| `--verbose` | INFO | Informational messages + above |
| `--debug` or `KDEPS_DEBUG=true` | DEBUG | Debug details + above |
| `KDEPS_LOG_FORMAT=json` | (any) | Structured JSON output on stderr |

**JSON format:**

```bash
export KDEPS_LOG_FORMAT=json
kdeps run workflow.yaml
# Output: {"time":"...","level":"WARN","msg":"experimental software",...}
```

## Commands

| Command | Page | Description |
|---|---|---|
| `kdeps run` | [Dev Commands](/reference/cli/dev#kdeps-run) | Run a workflow locally |
| `kdeps [path]` | [Dev Commands](/reference/cli/dev#kdeps-path-agent-repl) | Agent mode REPL |
| `kdeps validate` | [Dev Commands](/reference/cli/dev#kdeps-validate) | Validate workflow config |
| `kdeps new` | [Dev Commands](/reference/cli/dev#kdeps-new) | Scaffold a new agent |
| `kdeps edit` | [Dev Commands](/reference/cli/dev#kdeps-edit) | Edit global config |
| `kdeps doctor` | [Dev Commands](/reference/cli/dev#kdeps-doctor) | System health checks |
| `kdeps chat` | [Dev Commands](/reference/cli/dev#kdeps-chat) | Interactive workflow generator |
| `kdeps llamafile` | [Dev Commands](/reference/cli/dev#kdeps-llamafile) | Llamafile model registry (list, update) |
| `kdeps registry` | [Registry Commands](/reference/cli/registry) | Search, install, publish packages |
| `kdeps bundle package` | [Packaging Commands](/reference/cli/packaging#kdeps-bundle-package) | Package for distribution |
| `kdeps bundle build` | [Packaging Commands](/reference/cli/packaging#kdeps-bundle-build) | Build Docker image |
| `kdeps export iso` | [Packaging Commands](/reference/cli/packaging#kdeps-export-iso) | Export bootable image |
| `kdeps export k8s` | [Packaging Commands](/reference/cli/packaging#kdeps-export-k8s) | Generate Kubernetes manifests |

## Command Workflow

### Typical Development Flow

```bash
# 1. Create new agent
kdeps new my-agent

# 2. Validate configuration
cd my-agent
kdeps validate workflow.yaml

# 3. Run locally with hot reload
kdeps run workflow.yaml --dev

# 4. Package for deployment
kdeps bundle package . --output dist/

# 5. Build Docker image (optional)
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
docker push registry.com/my-agent:v1.0.0
```

### Kubernetes Deployment Flow

```bash
# 1. Build Docker image
kdeps bundle build . --tag registry.com/my-agent:v1.0.0
docker push registry.com/my-agent:v1.0.0

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
|---|---|
| `0` | Success |
| `1` | General error |
| `2` | Validation error |
| `3` | Execution error |

## Environment Variables

| Variable | Description |
|---|---|
| `KDEPS_DEBUG` | Set `true` to enable debug logging (equivalent to `--debug`) |
| `KDEPS_LOG_FORMAT` | Set `json` for structured JSON log output |
| `KDEPS_EDITOR` | Editor for `kdeps edit` (overrides `VISUAL`/`EDITOR`) |
| `KDEPS_PYTHON_VERSION` | Global Python version (e.g., `3.12`) |
| `KDEPS_OFFLINE_MODE` | Set `true` to block all external LLM calls |
| `OLLAMA_HOST` | Ollama server URL |
| `TZ` | Timezone applied to all workflow runs |
| `KDEPS_CHAT_TIMEOUT` | Default timeout for LLM chat resources |
| `KDEPS_HTTP_TIMEOUT` | Default timeout for HTTP client resources |
| `KDEPS_PYTHON_TIMEOUT` | Default timeout for Python resources |
| `KDEPS_EXEC_TIMEOUT` | Default timeout for exec resources |
| `KDEPS_SQL_TIMEOUT` | Default timeout for SQL resources |
| `KDEPS_ON_ERROR_ACTION` | Default error action: `fail`, `continue`, `retry` |
| `KDEPS_ON_ERROR_MAX_RETRIES` | Default max retries for `retry` action |
| `KDEPS_ON_ERROR_RETRY_DELAY` | Default delay between retries |

## Tips

- Use `--dev` flag for hot reload during development
- Use `--debug` flag to troubleshoot issues
- Validate frequently with `kdeps validate`
- Always validate before packaging: `kdeps validate workflow.yaml`
- Use `--gpu cuda` for GPU workloads
- Tag images with version numbers: `--tag my-agent:v1.0.0`

## See Also

- [Dev Commands](/reference/cli/dev) -- run, serve, validate, new, edit, doctor, chat
- [Registry Commands](/reference/cli/registry) -- search, install, publish
- [Packaging Commands](/reference/cli/packaging) -- bundle, export, build
- [Installation](/getting-started/installation)
