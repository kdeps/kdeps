# kdeps: The AI Appliance Builder.

**AI agents as code.** Declarative LLM orchestration in YAML for specialized and coordinated multi-agent systems. Compose multi-step workflows across models, APIs, and data sources with fully defined control flow.

> **Highly experimental.** APIs, YAML schemas, and CLI flags can change without notice. Do not use in production. [Report issues or give feedback](https://github.com/kdeps/kdeps/issues).

## Why an AI Appliance Builder?

Chat AIs (Claude, Gemini, ChatGPT) and their CLI/MCP extensions are tools you operate. You prompt them, they respond, the session ends. They are powerful, but they are not something you ship as a product.

kdeps is for building **AI appliances**. You define specialized agents, coordinate them into an Agency, and deploy the entire system as a self-contained unit (Docker, edge ISO, or binary). It exposes an HTTP API, runs on a schedule, or processes data streams — without a human in the loop.

| | Chat AI + MCP | kdeps (Appliance) |
|---|---|---|
| **Who drives it** | You | The system (autonomous/event-driven) |
| **Deployed as** | A chat session | Docker, Edge ISO, or Binary |
| **Logic lives in** | Prompts and MCP config | YAML code - versioned, reviewed, tested |
| **Orchestration** | Model-driven | Fully defined control flow |
| **Multi-Agent** | Sequential prompts | Coordinated, specialized Agencies |
| **Ships to production** | No | Yes |

### Strictness as a feature

While chat interfaces prioritize open-ended flexibility, kdeps prioritizes **reproducibility and safety**. Inputs are declared, outputs are typed, and control flow is explicit. If a resource fails or a model hallucinates outside of your defined schema, kdeps fails fast with a clear error rather than continuing with bad data.

**Built for:**
- Developers shipping AI features into products (APIs, bots, pipelines)
- Teams that need specialized multi-agent logic in version control
- Engineers deploying to edge, Docker, or air-gapped environments
- Systems requiring fully defined control flow across heterogeneous models and APIs

**Not for:**
- Interactive coding assistance - use Claude Code or Copilot
- One-off research or Q&A - use a chat interface
- No-code AI assistants - kdeps is infrastructure, not an end-user app

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
    prompt: "{{ get('message') }}"
  apiResponse:
    response: "{{ output('chat') }}"
```

That's it. kdeps handles the Ollama server, request routing, and output wiring automatically.

## Fetch, search, and store

Three native capabilities compiled into the binary:

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
    prompt: "Summarize: {{ output('fetch').content }}"
  apiResponse:
    response: "{{ output('summarize') }}"
```

Need more capabilities? Install a component:

```bash
kdeps registry install pdf
kdeps registry install tts
kdeps registry install browser
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

Or use the interactive assistant to generate and run workflows from natural language:

```bash
kdeps chat
# Describe what you want: "summarize the latest news about AI"
# kdeps chat generates the YAML, lets you inspect it (/show), 
# and run it (/run).
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
| `run.telephony:` | Programmable IVR: answer, say, ask, menu, dial, record, hangup |
| `loop:` | Schedule or repeat a resource (`every:`, `at:`) |
| `items:` | Iterate over arrays |
| `validations:` | Guard clauses and error handling |

Full expression reference: [kdeps.com/concepts/expressions](https://kdeps.com/concepts/expressions)

## Components

Components are self-contained capability packages. Install globally, call from anywhere.

```bash
kdeps registry install <name>              # install from registry
kdeps registry install owner/repo          # install from GitHub
kdeps registry install ./archive.komponent # install from local file
kdeps registry list                        # list installed
kdeps registry info <name>                 # show metadata and README
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

Lock a deployment to specific models via `llm.models` in `~/.kdeps/config.yaml`. Resources requesting any other model are automatically overridden to `models[0]` and a warning is logged:

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: ollama
  model: llama3.3:latest
  models: [llama3.3:latest]
```

## Build and deploy

```bash
kdeps bundle build          # Docker image
kdeps export k8s            # Kubernetes manifests (Deployment + Service)
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
  # backend: ollama
  # base_url: http://localhost:11434

  # Local inference via llamafile (self-contained binary, no install needed)
  # Use backend: file in a chat resource, set model to a .llamafile path or URL

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

# Per-resource global defaults — applied when a resource does not set the value
resource_defaults:
  chat:
    timeout: "60s"          # default LLM call timeout
    context_length: 4096    # default context window in tokens
  http:
    timeout: "30s"          # default HTTP request timeout
  python:
    timeout: "60s"          # default Python script timeout
  exec:
    timeout: "30s"          # default shell command timeout
  sql:
    timeout: "30s"          # default SQL query timeout
    max_rows: 0             # default row limit (0 = unlimited)
  onError:
    action: "fail"          # "fail" | "continue" | "retry"
    max_retries: 3          # retries when action is "retry"
    retry_delay: "1s"       # delay between retries
```

All values are applied as environment variables at startup. Explicit env vars always take precedence over the config file.

---

[Documentation](https://kdeps.com) | [Registry](https://kdeps.io) | Apache 2.0
