# kdeps: AI agents in YAML.

Orchestrate LLMs, databases, and APIs without glue or legacy code.

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
- [`validate:`](https://kdeps.com/concepts/validation) – Logic & control flow (if/else)
- [`env('KEY')`](https://kdeps.com/concepts/expression-functions-reference#get-key-typehint) – Access environment variables
- [`session()`](https://kdeps.com/concepts/expression-functions-reference#session) – Access persistent session data
- [`file('*')`](https://kdeps.com/concepts/expression-functions-reference#file-pattern-selector) – Access uploaded or local files
- [`input('m')`](https://kdeps.com/concepts/expression-functions-reference#input-name-type) – Access bot/hardware input data
- [`info('dt')`](https://kdeps.com/concepts/expression-functions-reference#info-field) – Access system metadata

### 🤖 [Resources (Executors)](https://kdeps.com/resources/overview)
- [`chat:`](https://kdeps.com/resources/llm) – LLM (Ollama, OpenAI, Anthropic, Groq, etc.)
- [`httpClient:`](https://kdeps.com/resources/http-client) – REST APIs (GET, POST, etc.)
- [`sql:`](https://kdeps.com/resources/sql) – Databases (Postgres, MySQL, SQLite, Oracle)
- [`python:`](https://kdeps.com/resources/python) – Scripts via isolated `uv` environments
- [`scraper:`](https://kdeps.com/resources/scraper) – Text extraction from 15+ file types
- [`embedding:`](https://kdeps.com/resources/embedding) – Vector RAG (index or search)
- [`tts:`](https://kdeps.com/resources/tts) – Text-to-speech (offline or cloud)
- [`exec:`](https://kdeps.com/resources/exec) – Shell commands and automation
- [`botReply:`](https://kdeps.com/concepts/input-sources#chat-bot-platforms) – Send messages to Discord/Slack/Telegram
- [`apiResponse:`](https://kdeps.com/resources/api-response) – Return data to the HTTP caller

## CLI Cheatsheet
- `kdeps run` – Execute workflows with hot reload
- `kdeps new` – Create projects via interactive wizard
- `kdeps validate` – Check YAML syntax and logic
- `kdeps build` – Create Docker images from workflows
- `kdeps push` – Live-update running containers
- `kdeps export iso` – Generate bootable edge ISOs

[Documentation](https://kdeps.com) | [Visual Editor](https://kdeps.io) | Apache 2.0
