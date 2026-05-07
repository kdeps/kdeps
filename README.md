# kdeps

**Straightforward LLM dependency orchestration for multi-agent workflows.** Compose chat, code, and data into declarative pipelines in YAML. Export AI workflows as a single binary, ISO, Docker, or Kubernetes pods. Use Ollama, llamafile, or any cloud AI provider.

> **Highly experimental.** APIs, YAML schemas, and CLI flags can change without notice. Do not use in production. [Report issues or give feedback](https://github.com/kdeps/kdeps/issues).

## Why kdeps?

Chat AIs and their MCP extensions are tools you operate. kdeps is for building **deployable AI workflows** — pipelines that chain LLM calls with code execution, data lookups, and API requests, then export as a binary, Docker, ISO, or Kubernetes pod.

| | Chat AI + MCP | kdeps |
|---|---|---|
| **Deployed as** | A chat session | Binary, Docker, ISO, Kubernetes |
| **Logic lives in** | Prompts | YAML — versioned, reviewed, tested |
| **Orchestration** | Model-driven | Explicit dependency pipelines |
| **Ships to production** | No | Yes |

## Install

```bash
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh
```

## Quick start

```bash
kdeps new                    # scaffold a project
kdeps run workflow.yaml --dev  # hot-reload, no Docker needed
```

A minimal agent that answers questions via an LLM:

```yaml
# resources/chat.yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: chat
run:
  chat:
    prompt: "{{ get('message') }}"
  apiResponse:
    response: "{{ output('chat') }}"
```

Wire resources into pipelines — outputs flow between steps via `requires:`:

```yaml
metadata:
  actionId: fetch
run:
  scraper:
    url: "{{ get('url') }}"

---
metadata:
  actionId: summarize
  requires: [fetch]
run:
  chat:
    prompt: "Summarize: {{ output('fetch').content }}"
  apiResponse:
    response: "{{ output('summarize') }}"
```

## Build and deploy

```bash
kdeps bundle build          # Docker image
kdeps export k8s            # Kubernetes manifests
kdeps bundle export iso     # bootable edge ISO
kdeps bundle prepackage     # self-contained binary per arch
```

## Global config

```bash
kdeps edit    # opens ~/.kdeps/config.yaml
kdeps doctor  # check system health (config, Ollama, Python, agents)
```

Config is validated on load. Warnings are printed to stderr for:
- Typos in API key / field names
- Backend set without a corresponding API key
- Invalid routing strategy values
- Malformed duration strings
- Agent profiles not matching any installed workflow
- Empty agent profiles

Warnings and errors use structured JSON logging (via `log/slog`). Set `KDEPS_LOG_FORMAT=json` for production JSON output. Log level defaults to WARN; use `--verbose` for INFO or `--debug` for DEBUG.

```yaml
llm:
  backend: ollama           # ollama, openai, anthropic, groq, ...
  # openai_api_key: sk-...
  # anthropic_api_key: sk-ant-...

defaults:
  timezone: UTC
  python_version: "3.12"

resource_defaults:
  chat:
    timeout: "60s"
    context_length: 4096
  http:
    timeout: "30s"
```

---

[Documentation](https://kdeps.com) | [Registry](https://kdeps.io) | Apache 2.0
