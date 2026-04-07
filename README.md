# kdeps: AI agents as code.

AI agents in YAML. Build autonomous AI Agents and self-governing AI Agencies that orchestrate LLMs, databases, and APIs without glue or legacy code.

> **Highly experimental.** kdeps is under active development - APIs, YAML schemas, CLI flags, and behaviour can change without notice at any time. Do not use in production. [Report issues or give feedback](https://github.com/kdeps/kdeps/issues).

## Why kdeps

AI agents fail in production for one reason: **inconsistency**. Same task, different results. No audit trail. No way to debug or reproduce.

kdeps enforces deterministic AI agents, as **Ordered, Repeatable Systems (ORS)** by design — reproducible and repeatable by construction, so you can build truly **autonomous AI Agents** and compose them into full-scale **autonomous AI Agencies**:

- **Ordered** — Declarative YAML defines every execution step in graph order. No hidden logic, no surprise tool calls.
- **Repeatable** — Same inputs produce same outputs. Deterministic pipelines, version-controlled agent definitions. Output determinism depends on the underlying models, settings, and external APIs.
- **Systems** — LLMs, databases, and APIs unified in one spec. No glue code, no legacy bridges. Compose multiple agents into self-governing **autonomous AI Agencies** that operate without human-in-the-loop intervention.

## 1. Install
```bash
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh
```

## 2. Multi-Model RAG Example
Call the scraper and search components, then pass results to an LLM.

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: fetch-context
run:
  component:
    name: scraper
    with:
      url: "{{ get('url') }}"
      selector: ".article"

---
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: analyze
  requires: [fetch-context]
run:
  chat:
    model: gpt-4o
    prompt: |
      Context: {{ output('fetch-context') }}
      Question: {{ get('q') }}
  apiResponse:
    success: true
    response: { data: "{{ output('analyze') }}" }
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

### 🤖 Core Executors (built-in)

These are the 5 built-in resource types always available in every workflow:

| Executor | Description | Docs |
|---|---|---|
| [`chat:`](https://kdeps.com/resources/llm) | LLM (Ollama, OpenAI, Anthropic, Groq, etc.) | streaming, MCP tools |
| [`httpClient:`](https://kdeps.com/resources/http-client) | REST APIs (GET, POST, PUT, DELETE, …) | auth, retries |
| [`sql:`](https://kdeps.com/resources/sql) | Databases (Postgres, MySQL, SQLite, Oracle) | prepared statements |
| [`python:`](https://kdeps.com/resources/python) | Python scripts via isolated `uv` environments | pip packages |
| [`exec:`](https://kdeps.com/resources/exec) | Shell commands and system automation | env, timeout |
| [`agent:`](https://kdeps.com/concepts/agency) | Delegate to a sibling agent (agency mode) | |
| [`apiResponse:`](https://kdeps.com/resources/api-response) | Return data to the HTTP caller | streaming |

### 🧩 Components (installable)

Additional capabilities are provided as installable `.komponent` components. Install them once and call them from any workflow with a clean typed config:

```bash
kdeps component install scraper    # web/PDF text extraction
kdeps component install search     # web search (Tavily)
kdeps component install embedding  # vector embeddings (OpenAI)
kdeps component install tts        # text-to-speech (espeak / OpenAI)
kdeps component install email      # send email via SMTP
kdeps component install calendar   # generate .ics calendar files
kdeps component install pdf        # generate PDFs from HTML
kdeps component install memory     # key-value store via SQLite
kdeps component install browser    # browser automation (Playwright)
kdeps component install botreply   # Telegram bot replies
kdeps component install remoteagent # call a remote kdeps agent
kdeps component install autopilot  # goal-directed LLM planning
```

**Use a component in any resource:**

```yaml
run:
  component:
    name: scraper
    with:
      url: "https://example.com"
      selector: ".article"
```

The `with:` map is validated against the component's declared `interface.inputs`. Missing required inputs return an error; optional inputs use their declared defaults. Results are stored under `output('<actionId>')`.

**Call the same component twice with different inputs** — inputs are scoped to the calling resource's `actionId` so there's no collision:

```yaml
- actionId: fetch-home
  run:
    component: { name: scraper, with: { url: "https://example.com" } }

- actionId: fetch-docs
  run:
    component: { name: scraper, with: { url: "https://example.com/docs" } }
```

See the [Components guide](https://kdeps.com/concepts/components) for full documentation.

### 🏢 [Autonomous AI Agencies](https://kdeps.com/concepts/agency)
Compose multiple independent AI Agents into a single **autonomous AI Agency** — a self-governing system where agents delegate tasks, coordinate workflows, and respond without human-in-the-loop intervention.
- **`agency.yaml`** – Bundle multiple agents under one manifest with a `targetAgentId` entry point
- **Auto-discovery** – `agents/**/workflow.*` dirs and `agents/*.kdeps` archives discovered automatically
- **`.kagency` archives** – Pack the full agency into one portable file: `kdeps package my-agency/`
- **Docker / ISO / binary** – `kdeps build`, `kdeps export iso`, and `kdeps prepackage` all accept agencies

## CLI Cheatsheet
- `kdeps run` – Execute workflows or agencies with hot reload
- `kdeps run --file <path>` – Execute a workflow once with a file as input (file input source)
- `kdeps run --events` – Emit a structured NDJSON [event stream](https://kdeps.com/concepts/events) to stderr for every lifecycle transition
- `kdeps new` – Create projects via interactive wizard
- `kdeps validate` – Check YAML syntax and logic
- `kdeps package` – Pack a workflow (`.kdeps`), agency (`.kagency`), or component (`.komponent`)
- `kdeps build` – Create Docker images from workflows or agencies
- `kdeps push` – Live-update running containers
- `kdeps export iso` – Generate bootable edge ISOs from workflows or agencies
- `kdeps prepackage` – Bundle a `.kdeps`/`.kagency` file into self-contained executables per arch

### 🧩 Component CLI
- `kdeps component install <name>` – Install a component from the registry to `~/.kdeps/components/`
- `kdeps component list` – List installed components (internal, global, local)
- `kdeps component remove <name>` – Remove an installed component
- `kdeps component show <name>` – Show README for a component
- `kdeps info <ref>` – Show README for a local component, agent, agency, or remote GitHub repo (`owner/repo` or `owner/repo:subdir`)

### 🌐 Federation (UAF)
- `kdeps federation keygen --org <name>` – Generate Ed25519 keypair for signing
- `kdeps federation register` – Register an agent in a UAF registry
- `kdeps federation trust add` – Add a registry trust anchor (public key)
- `kdeps federation trust list` – List all configured trust anchors
- `kdeps federation mesh list` – List remote agents used in the current project
- `kdeps federation mesh publish` – Preview the registration manifest (dry-run)
- `kdeps federation receipt verify` – Verify a signed receipt from a remote agent
- `kdeps federation key-rotate --org <name>` – Rotate keypair (dual-key transition period)

[Documentation](https://kdeps.com) | [Visual Editor](https://kdeps.io) | Apache 2.0

