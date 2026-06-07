# kdeps

[![Build and Test](https://github.com/kdeps/kdeps/actions/workflows/build-test.yml/badge.svg?branch=main)](https://github.com/kdeps/kdeps/actions/workflows/build-test.yml)
[![Coverage](https://codecov.io/gh/kdeps/kdeps/branch/main/graph/badge.svg)](https://codecov.io/gh/kdeps/kdeps)
[![Release](https://img.shields.io/github/v/tag/kdeps/kdeps?sort=semver&label=release)](https://github.com/kdeps/kdeps/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/kdeps/kdeps)](https://go.dev/)
[![License](https://img.shields.io/github/license/kdeps/kdeps)](https://github.com/kdeps/kdeps/blob/main/LICENSE)
[![CodeQL](https://github.com/kdeps/kdeps/actions/workflows/codeql.yml/badge.svg?branch=main)](https://github.com/kdeps/kdeps/actions/workflows/codeql.yml)
[![Docs](https://github.com/kdeps/kdeps/actions/workflows/docs.yml/badge.svg?branch=main)](https://kdeps.com)
[![Documentation](https://img.shields.io/badge/docs-kdeps.com-00E5FF)](https://kdeps.com)
[![Registry](https://img.shields.io/badge/registry-kdeps.io-00E5FF)](https://kdeps.io)
[![GitHub stars](https://img.shields.io/github/stars/kdeps/kdeps)](https://github.com/kdeps/kdeps/stargazers)

Build and deploy AI agents in YAML. Two modes: **workflow** (DAG pipelines), **agent** (autonomous LLM loop).

> **Highly experimental.** APIs, schemas, and CLI flags change without notice. Not for production. [Report issues](https://github.com/kdeps/kdeps/issues).

## Book

[<img src="https://d2sofvawe08yqg.cloudfront.net/kdeps/s_hero?1779817160" alt="AI Appliances book cover" width="140" align="right" style="margin-left:16px">](https://leanpub.com/kdeps)

**[AI Appliances - Build & Deploy Autonomous AI Agents and Agencies in YAML](https://leanpub.com/kdeps)**
Free. PDF, EPUB, and web.

Hands-on guide covering deterministic pipelines, multi-agent orchestration, error handling, and vendor-agnostic deployment - the production challenges most AI frameworks leave to you.

<br clear="right">

## Install

```bash
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh
```

## Modes

### Workflow mode

DAG-deterministic request/response pipelines. Each resource declares its dependencies via `requires:` and runs in order. Supports API server, web server, file input, and bot input.

```
POST /summarize  {"url": "..."}
        |
        v
+---------------------+
|  fetch              |  httpClient -- fetches the URL
+---------------------+
        |
        v
+---------------------+
|  respond            |  chat -- summarizes the fetched body
+---------------------+
        |
        v
   apiResponse        <- output('respond') becomes the HTTP response body
```

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

**Resource types:** `chat`, `httpClient`, `python`, `exec`, `sql`, `email`, `scraper`, `browser`, `embedding`, `searchLocal`, `searchWeb`, `agent`, `component`

**Expressions:** `get('key')` reads request input, `output('actionId')` reads a prior step's result, `set('key', val)` stores state. All expressions are safe inside `{{ }}` — Jinja2 control flow (`{% if %}`, `{% for %}`) is also supported.

### Agent mode

Autonomous LLM loop. Every resource in the workflow is auto-registered as a callable tool -- the LLM decides which tools to call, in what order, to complete the task.

```
stdin prompt
      |
      v
+---------------------+
|  LLM                |  plans steps, picks tools
+---------------------+
      |
      +-- call tool: httpClient  -->  fetch URL
      |
      +-- call tool: python      -->  process data
      |
      +-- call tool: sql         -->  query database
      |
      v
+---------------------+
|  LLM (again)        |  synthesizes results into final answer
+---------------------+
      |
      v
   stdout response
```

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

An agency is a collection of agents that work together. Each agent is its own `workflow.yaml` with its own resources, model, and logic. You wire them together using the `agent:` resource type, which runs another agent's full workflow and returns its output — like calling a function, but the function is an entire AI pipeline.

```
POST /run-marketing-pipeline
        │
        ▼
┌─────────────────────┐
│   content-writer    │  ← its own workflow.yaml, writes the blog post
└────────┬────────────┘
         │ output passed as params
         ▼
┌─────────────────────┐
│   cms-publisher     │  ← its own workflow.yaml, publishes to CMS
└─────────────────────┘
         │
         ▼
      response
```

The orchestrating workflow calls each agent in order using `agent:`:

```yaml
# resources/pipeline.yaml

actionId: draft
agent:
  name: content-writer        # runs agents/content-writer/workflow.yaml
  params:
    topic: "{{ get('topic') }}"  # passed as get('topic') inside that agent

---
actionId: publish
requires: [draft]
agent:
  name: cms-publisher         # runs agents/cms-publisher/workflow.yaml
  params:
    content: "{{ output('draft') }}"  # previous agent's output forwarded
apiResponse:
  response: "{{ output('publish') }}"
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
  openai_api_key: sk-...    # only needed for the relevant backend

defaults:
  timezone: UTC
  python_version: "3.12"

resource_defaults:          # applied to every resource of that type
  chat:
    timeout: 60s            # hard stop per LLM call
    context_length: 4096
  http:
    timeout: 30s
```

Per-agent config overrides: add an `agents:` block keyed by the workflow name to override globals for that agent only:

```yaml
agents:
  my-agent:          # matches metadata.name in workflow.yaml
    llm:
      backend: openai
      openai_api_key: sk-...
```

Config is validated on load. Warnings go to stderr for unknown keys, missing API keys, invalid durations, and agent profiles that don't match any installed workflow.

## Security

When `apiServer` is configured, authentication is required. Set the token via `KDEPS_API_AUTH_TOKEN` or `api_auth_token` in `~/.kdeps/config.yaml` (never in `workflow.yaml`). Clients send `Authorization: Bearer <token>` or `X-Api-Key: <token>`. `/health` is exempt. `/_kdeps/*` management routes use `KDEPS_MANAGEMENT_TOKEN`.

```bash
export KDEPS_API_AUTH_TOKEN=your-secret-token
kdeps run workflow.yaml
```

```yaml
settings:
  apiServer:
    rateLimit:
      requestsPerMinute: 60          # sustained per-IP rate; excess gets 429
      burst: 10                      # burst allowance above the sustained rate
    maxBodyBytes: 1048576            # 1 MB request body cap; 413 if exceeded
    trustedProxies:                  # honor X-Forwarded-For only from these peers
      - "10.0.0.0/8"
    cors:
      allowOrigins:
        - https://myapp.com
  certFile: /path/to/cert.pem        # TLS -- omit for plain HTTP
  keyFile: /path/to/key.pem
```

## Logging

Structured JSON via `log/slog`. Set `KDEPS_LOG_FORMAT=json` for production output. Default level: WARN. Flags: `--verbose` (INFO), `--debug` (DEBUG).

---

[Documentation](https://kdeps.com) | [Registry](https://kdeps.io) | Apache 2.0
