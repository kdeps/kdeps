# kdeps

Build and deploy AI agents in YAML. Two modes: **workflow** (DAG pipelines), **agent** (autonomous LLM loop).

> **Highly experimental.** APIs, schemas, and CLI flags change without notice. Not for production. [Report issues](https://github.com/kdeps/kdeps/issues).

## Install

```bash
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh
```

## Three modes

### Workflow mode

DAG-deterministic request/response pipelines. Resources declare dependencies via `requires:` and execute in order. Supports API server, web server, file input, and bot input.

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: summarizer
  version: "1.0.0"
  targetActionId: respond
settings:
  apiServer:
    portNum: 16395
    routes:
      - path: /summarize
        methods: [POST]
  agentSettings:
    installOllama: true
```

```yaml
# resources/fetch.yaml
actionId: fetch
httpClient:
  method: GET
  url: "{{ get('url') }}"
  timeout: 10s

---
actionId: respond
requires: [fetch]
chat:
  model: llama3.2:1b
  prompt: "Summarize this page: {{ output('fetch').body }}"
apiResponse:
  response: "{{ output('respond') }}"
```

```bash
kdeps run workflow.yaml          # local, instant startup
kdeps run workflow.yaml --dev    # hot reload
```

**Resource types:** `chat`, `httpClient`, `python`, `exec`, `sql`, `scraper`, `browser`, `embedding`, `searchLocal`, `searchWeb`, `agent`, `component`

**Expressions:** `get('key')` reads request input, `output('actionId')` reads a prior step's result, `set('key', val)` stores state. All expressions are safe inside `{{ }}` — Jinja2 control flow (`{% if %}`, `{% for %}`) is also supported.

### Agent mode

Autonomous LLM loop. Every resource in the workflow is auto-registered as a callable tool. The agent plans and executes multi-step tasks using the kdeps engine.

```bash
kdeps serve workflow.yaml
kdeps serve workflow.yaml --model llama3.2 --system "You are a DevOps assistant."
```

The agent reads from stdin and runs until you exit. All resource types (http, python, exec, sql, ...) are available as tools without any extra wiring.

```
KDEPS_AGENT_MODEL=claude-3-5-sonnet   # override model via env
KDEPS_AGENT_BACKEND=anthropic
```

## Agencies

Multiple agents collaborating. Each agent is a separate workflow; the `agent:` resource type calls another agent by name and passes parameters.

```yaml
# resources/pipeline.yaml
actionId: analyze
agent:
  name: code-reviewer
  params:
    code: "{{ get('source') }}"

---
actionId: report
requires: [analyze]
agent:
  name: report-writer
  params:
    findings: "{{ output('analyze') }}"
apiResponse:
  response: "{{ output('report') }}"
```

Run an agency:

```bash
kdeps run agency.yaml
```

## Build and deploy

```bash
kdeps bundle build          # Docker image
kdeps bundle export iso     # bootable edge ISO
kdeps bundle prepackage     # self-contained binary per arch
kdeps export k8s            # Kubernetes manifests
```

## Registry

```bash
kdeps registry search <query>
kdeps registry install <package>
kdeps registry publish
```

## Global config

```bash
kdeps edit    # opens ~/.kdeps/config.yaml
kdeps doctor  # check config, Ollama, Python, installed agents
```

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: ollama           # ollama, openai, anthropic, groq, ...
  openai_api_key: sk-...

defaults:
  timezone: UTC
  python_version: "3.12"

resource_defaults:
  chat:
    timeout: 60s
    context_length: 4096
  http:
    timeout: 30s
```

Per-agent config overrides: add a key under `agents.<workflow-name>` to override globals for that agent only.

Config is validated on load. Warnings go to stderr for unknown keys, missing API keys, invalid durations, and agent profiles that don't match any installed workflow.

## Security

Set in `workflow.yaml` under `settings.apiServer`:

- `auth.token` - Bearer or `X-Api-Key` header required on every request
- `rateLimit.requestsPerMinute` / `rateLimit.burst` - per-IP throttling
- `maxBodyBytes` - request body size cap
- `cors.allowOrigins` - CORS origins (presence of `cors:` block enables CORS)
- `settings.certFile` / `settings.keyFile` - TLS

## Logging

Structured JSON via `log/slog`. Set `KDEPS_LOG_FORMAT=json` for production output. Default level: WARN. Flags: `--verbose` (INFO), `--debug` (DEBUG).

---

[Documentation](https://kdeps.com) | [Registry](https://kdeps.io) | Apache 2.0
