# kdeps: AI agents as code.

Build autonomous AI agents in YAML — wire LLMs, APIs, databases, and Python scripts with no glue code.

> **Highly experimental.** APIs, YAML schemas, and CLI flags can change without notice. Do not use in production. [Report issues or give feedback](https://github.com/kdeps/kdeps/issues).

## Install

```bash
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh
```

## Your first agent in 60 seconds

```bash
kdeps new                    # scaffolds a project interactively
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
    model: llama3.2:1b
    prompt: "{{ get('message') }}"
  apiResponse:
    response: "{{ output('chat') }}"
```

That's it. kdeps handles the Ollama server, request routing, and output wiring automatically.

## Fetch, search, and store — no installs

Three capabilities are built into the binary — no `kdeps component install` needed:

```yaml
# Scrape a web page
run:
  scraper:
    url: "{{ get('url') }}"
    selector: "article"   # optional CSS selector

# Keyword search across local files
run:
  searchLocal:
    path: "/data/docs"
    query: "invoice"
    glob: "*.txt"         # optional filename filter
    limit: 10

# SQLite-backed keyword store (index/search/upsert/delete)
run:
  embedding:
    operation: "index"
    text: "{{ get('content') }}"
    collection: "docs"
    dbPath: "/data/store.db"

# Web search (DuckDuckGo by default, no API key needed)
run:
  searchWeb:
    query: "{{ get('query') }}"
    maxResults: 5
    # provider: brave  # optional: ddg (default) | brave | bing | tavily
    # apiKey: "..."    # required for non-DDG providers
```

Wire them together in a pipeline:

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
    model: llama3.2:1b
    prompt: "Summarize: {{ output('fetch').content }}"
  apiResponse:
    response: "{{ output('summarize') }}"
```

Need more capabilities? Install a component:

```bash
kdeps component install pdf
kdeps component install tts
kdeps component install browser
```

## Interactive chat with tools

```bash
kdeps run examples/llm-chat-tools/workflow.yaml --dev
```

Or add `--interactive` to any workflow to open a REPL alongside it:

```bash
kdeps run workflow.yaml --interactive
# /run <actionId>   — invoke a resource directly
# /list             — list resources and components
# /help             — show all commands
# /quit             — exit
```

## Key concepts

| Concept | What it does |
|---------|-------------|
| `requires:` | Declare execution order — dependencies run first |
| `get('key')` | Read from request body, query, headers, or memory |
| `output('id')` | Read the result of another resource |
| `run.component:` | Call an installed component with typed inputs |
| `run.chat:` | Call any LLM (Ollama, OpenAI, Anthropic, Groq, …) |
| `run.python:` | Run a Python script in an isolated `uv` environment |
| `run.exec:` | Run a shell command |
| `run.sql:` | Query a database (Postgres, MySQL, SQLite, Oracle) |
| `run.httpClient:` | Call any REST API |
| `loop:` | Schedule or repeat a resource (`every:`, `at:`) |
| `items:` | Iterate over arrays |
| `validations:` | Guard clauses and error handling |

Full expression reference: [kdeps.com/concepts/expressions](https://kdeps.com/concepts/expressions)

## Components

Components are self-contained capability packages. Install globally, call from anywhere.

```bash
kdeps component install <name>   # install from registry
kdeps component list             # list installed
kdeps component show <name>      # show README
kdeps component clone owner/repo # install from GitHub
```

Available: `scraper`, `search`, `embedding`, `memory`, `browser`, `tts`, `email`, `calendar`, `pdf`, `botreply`, `remoteagent`, `autopilot`

Components declare their own dependencies — kdeps auto-installs them on first use:

```yaml
# component.yaml
setup:
  pythonPackages: [requests, beautifulsoup4]
  osPackages: [wkhtmltopdf]
```

## Input sources

Declare how your agent receives input via `settings.input.sources`:

| Source | Use case |
|--------|----------|
| `api` | HTTP API (default) |
| `bot` | Telegram, Discord, Slack, WhatsApp |
| `file` | One-shot from `--file`, stdin, or env |
| `llm` | Interactive stdin REPL |
| `audio` / `video` / `telephony` | Hardware media capture |
| `component` | Invokable only from a parent workflow |

## Agencies

Compose multiple agents into a self-governing **Agency**:

```bash
kdeps bundle package my-agency/   # produces .kagency archive
kdeps run my-agency.kagency --dev
```

Agents in an agency communicate via `run.agent:` — no network calls, no ports.

## Model allowlist

Lock a workflow to specific models via `agentSettings.models`. Resources requesting any other model are automatically overridden to `models[0]` and a warning is logged:

```yaml
# workflow.yaml
settings:
  agentSettings:
    models: [llama3.3:latest]
```

## Build and deploy

```bash
kdeps bundle build          # Docker image
kdeps bundle export iso     # bootable edge ISO
kdeps bundle prepackage     # self-contained binary per arch
kdeps cloud push            # live-update a running container
```

## Global configuration

`~/.kdeps/config.yaml` holds your LLM credentials and global defaults. It is created on first run — edit it with:

```bash
kdeps edit
```

```yaml
# ~/.kdeps/config.yaml

llm:
  # Local inference via Ollama (no API key needed)
  # ollama_host: http://localhost:11434
  # model: llama3.2          # global default; overridden per resource

  # Online providers — set only the ones you use
  # openai_api_key: ""
  # anthropic_api_key: ""
  # google_api_key: ""
  # groq_api_key: ""
  # deepseek_api_key: ""
  # openrouter_api_key: ""
  # ... and more

# Global defaults applied to all workflows that don't override them
defaults:
  # timezone: UTC
  # python_version: "3.12"
  # offline_mode: false
```

All values are applied as environment variables at startup. Explicit env vars always take precedence over the config file.

---

[Documentation](https://kdeps.com) | [Visual Editor](https://kdeps.io) | Apache 2.0
