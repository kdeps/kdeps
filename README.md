# kdeps: AI agents as code.

AI agents in YAML. Orchestrate LLMs, databases, and APIs without glue or legacy code.

## Why kdeps? Ordered, Repeatable Systems (ORS)

AI agents fail in production for one reason: **inconsistency**. Same task, different results. No audit trail. No way to debug or reproduce.

kdeps enforces **Ordered, Repeatable Systems** by design:

- **Ordered** — Declarative YAML defines every execution step. No hidden logic, no surprise tool calls.
- **Repeatable** — Same inputs produce same outputs. Deterministic pipelines, version-controlled agent definitions.
- **Systems** — LLMs, databases, and APIs unified in one spec. No glue code, no legacy bridges.

## 1. Install
```bash
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh
```

## 2. Multi-Model RAG Example
Chain scrapers, vector search, and multiple LLMs in one file.

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: analyze
run:
  before:
    - scraper: { url: "{{ get('url') }}" } # Extract text from web/PDF
    - embedding:
        operation: search
        input: "{{ get('scraper') }}"
        collection: docs
  chat:
    model: gpt-4o
    prompt: |
      Context: {{ get('embedding').results }}
      Data: {{ get('scraper') }}
      Question: {{ get('q') }}
  apiResponse:
    success: true
    response: { data: "{{ get('chat') }}" }
```

## Syntax Cheatsheet

### ⚡ [Syntax & Logic](https://kdeps.com/concepts/expressions)
- [`get('q')`](https://kdeps.com/concepts/unified-api) – Get data (body, query, header, output)
- [`set('k', v)`](https://kdeps.com/concepts/expression-functions-reference#set-key-value-storage) – Store in memory or session
- [`items:`](https://kdeps.com/concepts/items) – Iterate over arrays/collections
- [`loop:`](https://kdeps.com/concepts/loop) – Conditional while-loop, repeated tasks (`every:`), and scheduled fire times (`at:`)
- [`validations:`](https://kdeps.com/concepts/validation) – Validation, filtering & control flow
- [`env('KEY')`](https://kdeps.com/concepts/expression-functions-reference#get-key-typehint) – Access environment variables
- [`session()`](https://kdeps.com/concepts/expression-functions-reference#session) – Access persistent session data
- [`file('*')`](https://kdeps.com/concepts/expression-functions-reference#file-pattern-selector) – Access uploaded or local files
- [`input('m')`](https://kdeps.com/concepts/expression-functions-reference#input-name-type) – Access bot/hardware input data
- [`info('dt')`](https://kdeps.com/concepts/expression-functions-reference#info-field) – Access system metadata

### 🤖 [Resources (Executors)](https://kdeps.com/resources/overview)
- [`chat:`](https://kdeps.com/resources/llm) – LLM (Ollama, OpenAI, Anthropic, Groq, etc.) · `streaming: true` for Ollama NDJSON streaming · `tools:` with `mcp:` for [MCP tool servers](https://kdeps.com/concepts/tools#mcp-tools)
- [`httpClient:`](https://kdeps.com/resources/http-client) – REST APIs (GET, POST, etc.)
- [`sql:`](https://kdeps.com/resources/sql) – Databases (Postgres, MySQL, SQLite, Oracle)
- [`python:`](https://kdeps.com/resources/python) – Scripts via isolated `uv` environments
- [`scraper:`](https://kdeps.com/resources/scraper) – Text extraction from 15+ file types
- [`embedding:`](https://kdeps.com/resources/embedding) – Vector RAG (index or search)
- [`tts:`](https://kdeps.com/resources/tts) – Text-to-speech (offline or cloud)
- [`pdf:`](https://kdeps.com/resources/pdf) – PDF generation from HTML or Markdown
- [`exec:`](https://kdeps.com/resources/exec) – Shell commands and automation
- [`email:`](https://kdeps.com/resources/email) – Send email (SMTP) and read/search/modify messages (IMAP)
- [`calendar:`](https://kdeps.com/resources/calendar) – Read and write local ICS (iCalendar) files
- [`search:`](https://kdeps.com/resources/search) – Web search (Brave, SerpAPI, DuckDuckGo, Tavily) or local file discovery
- [`botReply:`](https://kdeps.com/concepts/input-sources#chat-bot-platforms) – Send messages to Discord/Slack/Telegram
- [`agent:`](https://kdeps.com/concepts/agency) – Delegate work to another agent in an agency
- [`apiResponse:`](https://kdeps.com/resources/api-response) – Return data to the HTTP caller

### 🏢 [Agency & Multi-Agent Orchestration](https://kdeps.com/concepts/agency)
- **`agency.yaml`** – Bundle multiple agents under one manifest with a `targetAgentId` entry point
- **Auto-discovery** – `agents/**/workflow.*` dirs and `agents/*.kdeps` archives discovered automatically
- **`.kagency` archives** – Pack the full agency into one portable file: `kdeps package my-agency/`
- **Docker / ISO / binary** – `kdeps build`, `kdeps export iso`, and `kdeps prepackage` all accept agencies

## CLI Cheatsheet
- `kdeps run` – Execute workflows or agencies with hot reload
- `kdeps new` – Create projects via interactive wizard
- `kdeps validate` – Check YAML syntax and logic
- `kdeps package` – Pack a workflow (`.kdeps`) or agency (`.kagency`) into a portable archive
- `kdeps build` – Create Docker images from workflows or agencies
- `kdeps push` – Live-update running containers
- `kdeps export iso` – Generate bootable edge ISOs from workflows or agencies
- `kdeps prepackage` – Bundle a `.kdeps`/`.kagency` file into self-contained executables per arch

[Documentation](https://kdeps.com) | [Visual Editor](https://kdeps.io) | Apache 2.0
