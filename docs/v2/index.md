---
layout: home

hero:
  text: The Rails for AI Agents
  tagline: Define your agent in YAML. kdeps handles the wiring, the execution order, and the deployment. Ship to cloud, edge device, or bootable ISO with one command.
  actions:
    - theme: brand
      text: Get Started
      link: /getting-started/installation
    - theme: alt
      text: View on GitHub
      link: https://github.com/kdeps/kdeps

features:
  - icon: 📝
    title: YAML-First Configuration
    details: Define your entire agent in config — no glue code, no boilerplate. Chain LLMs, SQL, HTTP, Python, and shell into a graph that runs anywhere.
  - icon: 🤖
    title: 14+ LLM Providers
    details: Ollama for fully offline local models (llama3, mistral, phi), or any cloud provider — OpenAI, Anthropic, Google, Groq, Mistral, DeepSeek, and more.
  - icon: 🎙️
    title: Multi-Source I/O
    details: Accept input from audio hardware, cameras, telephony, HTTP APIs, and chat bots (Discord, Slack, Telegram, WhatsApp) — simultaneously.
  - icon: 🔇
    title: Offline-First
    details: Run entirely air-gapped — local LLMs via Ollama, offline STT (Whisper, Vosk), offline TTS (Piper, eSpeak), vector embeddings. No network required.
  - icon: 🗄️
    title: Built-in SQL & RAG
    details: PostgreSQL, MySQL, SQLite, SQL Server, Oracle with connection pooling. Built-in vector embeddings for RAG pipelines stored in SQLite.
  - icon: 🚀
    title: Deploy Anywhere
    details: Package as Docker image, push to kdeps.io cloud, or export to bootable ISO — the same agent runs on cloud infrastructure or a $50 edge device.
  - icon: 💬
    title: Chat Bot Platforms
    details: Connect to Discord, Slack, Telegram, or WhatsApp in persistent polling or stateless mode. Access the inbound message with input('message').
  - icon: 🔁
    title: Turing-Complete Loops
    details: while-loop construct with configurable maxIterations. Combine with get()/set() mutable state for accumulation, search, and retry patterns.
---

# Introduction

Rails made it possible to build a web app in a day instead of a month. KDeps does the same for AI agents.

You define what your agent does in YAML — which LLM, which database, which API, which inputs. KDeps figures out the execution order, handles the wiring between resources, and packages everything into a single deployable unit. Run it locally in under a second, push it to kdeps.io in one click, or export it as a bootable ISO that runs on a $50 device with no API costs forever.

No glue code. No vendor lock-in. No legacy code to maintain.

## Real-World Workflows

Each of these is a complete, deployable agent. 17–22 lines of YAML. No custom code.

Every agent shares the same entry point:

<div v-pre>

```yaml
# workflow.yaml (same for all agents)
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: my-agent
  version: "1.0.0"
  targetActionId: respond
settings:
  apiServerMode: true
  apiServer:
    portNum: 16395
    routes:
      - path: /api/v1/run
        methods: [POST]
```

</div>

### 1. Email — sort, draft, unsubscribe

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: respond
run:
  chat:
    model: gpt-4o-mini
    prompt: |
      Email: {{ get('email') }}
      Classify as urgent / normal / unsubscribe.
      If urgent: draft a reply.
      If unsubscribe: return the sender address.
    jsonResponse: true
    jsonResponseKeys: [label, draft, unsubscribe_from]
  apiResponse:
    success: true
    response:
      data: get('respond')
```

</div>

### 2. Meetings — agenda + action items

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: respond
run:
  chat:
    model: gpt-4o-mini
    prompt: |
      Meeting request: {{ get('request') }}
      Attendees: {{ get('attendees') }}
      Write a concise agenda and post-meeting action items.
    jsonResponse: true
    jsonResponseKeys: [agenda, action_items, suggested_time]
  apiResponse:
    success: true
    response:
      data: get('respond')
```

</div>

### 3. Late bills — cash-flow-aware due dates

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: respond
run:
  before:
    - httpClient:
        method: GET
        url: "{{ env('BANK_API_URL') }}/transactions"
        headers:
          Authorization: "Bearer {{ env('BANK_TOKEN') }}"
  chat:
    model: gpt-4o-mini
    prompt: |
      Transactions: {{ get('httpClient') }}
      Today: {{ info('current_date') }}
      List bills due in 7 days. Flag anything overdue.
    jsonResponse: true
    jsonResponseKeys: [due_soon, overdue, balance_after]
  apiResponse:
    success: true
    response:
      data: get('respond')
```

</div>

### 4. Subscription leaks — find what you forgot

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: respond
run:
  before:
    - httpClient:
        method: GET
        url: "{{ env('BANK_API_URL') }}/transactions?days=90"
        headers:
          Authorization: "Bearer {{ env('BANK_TOKEN') }}"
  chat:
    model: gpt-4o-mini
    prompt: |
      Transactions: {{ get('httpClient') }}
      Find all recurring charges. Flag any not used in 30+ days.
    jsonResponse: true
    jsonResponseKeys: [subscriptions, unused, monthly_total]
  apiResponse:
    success: true
    response:
      data: get('respond')
```

</div>

### 5. Finding your own files — RAG search

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: respond
run:
  before:
    - embedding:
        model: nomic-embed-text
        input: "{{ get('q') }}"
        collection: my-docs
        operation: search
        topK: 5
  chat:
    model: llama3.2
    prompt: |
      Context: {{ get('embedding').results }}
      Question: {{ get('q') }}
      Answer using only the context above.
  apiResponse:
    success: true
    response:
      data: get('respond')
```

</div>

### 6. Grocery waste — meal plan from what's expiring

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: respond
run:
  before:
    - httpClient:
        method: GET
        url: "{{ env('PANTRY_API') }}/inventory"
  chat:
    model: gpt-4o-mini
    prompt: |
      Pantry: {{ get('httpClient') }}
      Suggest 5 meals using items expiring soonest.
      List what needs reordering.
    jsonResponse: true
    jsonResponseKeys: [meals, reorder_list]
  apiResponse:
    success: true
    response:
      data: get('respond')
```

</div>

### 7. Travel planning — one pass

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: respond
run:
  before:
    - scraper:
        url: "https://www.kayak.com/flights/{{ get('from') }}-{{ get('to') }}/{{ get('date') }}"
  chat:
    model: gpt-4o-mini
    prompt: |
      Trip: {{ get('from') }} → {{ get('to') }} on {{ get('date') }}
      Data: {{ get('scraper') }}
      Best flight option, hotel, and 3-day itinerary.
    jsonResponse: true
    jsonResponseKeys: [flight, hotel, itinerary]
  apiResponse:
    success: true
    response:
      data: get('respond')
```

</div>

### 8. Admin overhead — generate invoice

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: respond
run:
  chat:
    model: gpt-4o-mini
    prompt: |
      Client: {{ get('client') }}
      Work done: {{ get('description') }}
      Hours: {{ get('hours') }} at {{ get('rate') }}/hr
      Generate a professional invoice.
    jsonResponse: true
    jsonResponseKeys: [invoice_number, line_items, subtotal, due_date]
  apiResponse:
    success: true
    response:
      data: get('respond')
```

</div>

## Key Highlights

### Multi-Source I/O

KDeps accepts input from hardware devices, HTTP APIs, and chat bot platforms — simultaneously. Configure audio, video, telephony, bot, and API sources in one `workflow.yaml`:

```yaml
settings:
  input:
    sources: [audio]          # audio | video | telephony | api
    audio:
      device: hw:0,0          # ALSA device (Linux), microphone name (macOS/Windows)
    activation:
      phrase: "hey kdeps"     # Wake phrase — workflow runs only when heard
      mode: offline
      offline:
        engine: faster-whisper
        model: small
    transcriber:
      mode: offline           # Fully local, no cloud required
      output: text
      offline:
        engine: faster-whisper
        model: small
```

| Source | Hardware / Platform |
|--------|---------------------|
| `audio` | ALSA microphone, line-in, USB audio |
| `video` | V4L2 camera, USB webcam, CSI camera |
| `telephony` | SIP/ATA adapter, Twilio |
| `bot` | Discord, Slack, Telegram, WhatsApp |
| `api` | HTTP REST (default) |

[Full Input Sources guide →](/concepts/input-sources)

### Offline-First AI Stack

Every AI component has an offline alternative — run completely air-gapped:

| Component | Offline Options | Cloud Options |
|-----------|----------------|---------------|
| LLM | Ollama (llama3, mistral, phi) | OpenAI, Anthropic, Google, Groq |
| STT | Whisper, Faster-Whisper, Vosk, Whisper.cpp | OpenAI Whisper API, Deepgram, Google STT |
| TTS | Piper, eSpeak-NG, Festival, Coqui TTS | OpenAI TTS, ElevenLabs, Azure TTS |
| Wake Phrase | Faster-Whisper, Vosk | Deepgram, AssemblyAI |

### YAML-First Configuration
Build workflows using simple, self-contained YAML configuration blocks. No complex programming required - just define your resources and let KDeps handle the orchestration.

```yaml
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: my-agent
  version: "1.0.0"
  targetActionId: responseResource
settings:
  apiServerMode: true
  apiServer:
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]
```

### Chat Bot Platforms

Connect a workflow to Discord, Slack, Telegram, or WhatsApp with a single config block. Access the inbound message via `input('message')` and reply automatically:

```yaml
settings:
  input:
    sources: [bot]
    bot:
      executionType: polling   # persistent long-running connection
      telegram:
        botToken: "{{ env('TELEGRAM_BOT_TOKEN') }}"
```

Run once from stdin in **stateless mode** — useful for serverless or piped execution:

```bash
echo '{"message":"What is 2+2?","platform":"telegram"}' | kdeps run workflow.yaml
```

### Deploy Anywhere — Including Bootable ISO

Package the same workflow as Docker image, push to kdeps.io cloud, or export to a bootable ISO that runs on bare-metal or a $50 edge device:

```bash
kdeps build my-agent-1.0.0.kdeps       # Docker image
kdeps export iso my-agent-1.0.0.kdeps  # Bootable ISO (EFI/raw/qcow2)
```

The bootable ISO bundles the agent, Ollama, and all dependencies — no API costs, no cloud required, runs air-gapped forever.

### Fast Local Development
Run workflows instantly on your local machine with sub-second startup time. Docker is optional and only needed for deployment.

```bash
# Run locally (instant startup)
kdeps run workflow.yaml

# Hot reload for development
kdeps run workflow.yaml --dev
```

### Unified API
Access data from any source with just two functions: `get()` and `set()`. No more memorizing 15+ different function names.

<div v-pre>

```yaml
# All of these work with get():
query: get('q')                    # Query parameter
auth: get('Authorization')         # Header
data: get('llmResource')           # Resource output
user: get('user_name', 'session')  # Session storage
```

</div>

### Mustache Expressions

KDeps v2 supports both expr-lang and Mustache-style variable interpolation:

<div v-pre>

```yaml
# expr-lang (functions and logic)
prompt: "{{ get('q') }}"
time:   "{{ info('current_time') }}"

# Mustache (simple variable access)
prompt: "{{q}}"
time:   "{{current_time}}"

# Mix in the same workflow
message: "Hello {{name}}, your score is {{ get('points') * 2 }}"
```

</div>

Use Mustache for simple variable access; use expr-lang for function calls, arithmetic, and conditionals. `{{var}}` and `{{ var }}` are identical.

[Learn more →](/concepts/expressions)

### LLM Integration

Use Ollama for local model serving or any OpenAI-compatible API. Vision, tools, and streaming are supported.

## Quick Start

```bash
# Install KDeps (Mac/Linux)
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh

# Or via Homebrew (Mac)
brew install kdeps/tap/kdeps

# Create a new agent interactively
kdeps new my-agent
```

## Example: Simple Chatbot

**workflow.yaml**
<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: chatbot
  version: "1.0.0"
  targetActionId: responseResource
settings:
  apiServerMode: true
  apiServer:
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]
  agentSettings:
    models:
      - llama3.2:1b
```

</div>

**resources/llm.yaml**
<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: llmResource
  name: LLM Chat
run:
  chat:
    model: llama3.2:1b
    prompt: "{{ get('q') }}"
    jsonResponse: true
    jsonResponseKeys:
      - answer
```

</div>

**resources/response.yaml**
```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: responseResource
  requires:
    - llmResource
run:
  apiResponse:
    success: true
    response:
      data: get('llmResource')
```

**Test it:**
```bash
kdeps run workflow.yaml
curl -X POST http://localhost:16395/api/v1/chat -d '{"q": "What is AI?"}'
```

## Documentation

### Getting Started
- [Installation](getting-started/installation)
- [Quickstart](getting-started/quickstart)
- [CLI Reference](getting-started/cli-reference)

### Configuration
- [Workflow](configuration/workflow)
- [Session & Storage](configuration/session)
- [CORS](configuration/cors)
- [Advanced](configuration/advanced)

### Resources
- [Overview](resources/overview)
- [LLM (Chat)](resources/llm) · [LLM Backends](resources/llm-backends)
- [TTS](resources/tts) · [HTTP Client](resources/http-client)
- [SQL](resources/sql) · [Python](resources/python) · [Exec](resources/exec)
- [Scraper](resources/scraper) · [API Response](resources/api-response)

### Concepts
- [Input Sources](concepts/input-sources) · [Unified API](concepts/unified-api)
- [Expressions](concepts/expressions) · [Expression Functions Reference](concepts/expression-functions-reference)
- [Request Object](concepts/request-object) · [Input Object](concepts/input-object)
- [Tools](concepts/tools) · [Items Iteration](concepts/items) · [Loop Iteration](concepts/loop)
- [Validation](concepts/validation) · [Error Handling](concepts/error-handling)
- [Inline Resources](concepts/inline-resources) · [Route Restrictions](concepts/route-restrictions)
- [Management API](concepts/management-api)

### Deployment
- [Docker](deployment/docker) · [WebServer Mode](deployment/webserver)

### Tutorials
- [Building a Chatbot](tutorials/chatbot) · [File Upload](tutorials/file-upload)
- [Multi-Database](tutorials/multi-database) · [Vision](tutorials/vision)


## Why KDeps v2?

| Feature | v1 (PKL) | v2 (YAML) |
|---------|----------|-----------|
| Configuration | PKL (Apple's language) | Standard YAML |
| Functions | 15+ to learn | 2 (get, set) |
| Startup time | ~30 seconds | < 1 second |
| Docker | Required | Optional |
| Python env | Anaconda (~20GB) | uv (97% smaller) |
| Learning curve | 2-3 days | ~1 hour |
| Chat bots | — | Discord, Slack, Telegram, WhatsApp |
| Vector RAG | — | Built-in embeddings (Ollama, OpenAI, Cohere) |
| Content scraper | — | 15 source types (PDF, web, DOCX, images…) |
| ISO export | — | Bootable ISO for bare-metal / edge |

## Examples

Explore working examples:

**Chat Bots:**
- [Telegram Bot](https://github.com/kdeps/kdeps/tree/main/examples/telegram-bot) - Telegram bot with LLM replies (polling)
- [Stateless Bot](https://github.com/kdeps/kdeps/tree/main/examples/stateless-bot) - One-shot bot execution from stdin
- [Telephony Bot](https://github.com/kdeps/kdeps/tree/main/examples/telephony-bot) - Voice call workflow via SIP/Twilio

**Edge AI / Voice:**
- [Voice Assistant](https://github.com/kdeps/kdeps/tree/main/examples/voice-assistant) - Offline wake-phrase + LLM + TTS on edge hardware
- [Video Analysis](https://github.com/kdeps/kdeps/tree/main/examples/video-analysis) - Camera capture + vision LLM analysis

**API Backends:**
- [Simple Chatbot](https://github.com/kdeps/kdeps/tree/main/examples/chatbot) - LLM chatbot
- [ChatGPT Clone](https://github.com/kdeps/kdeps/tree/main/examples/chatgpt-clone) - Full chat UI
- [File Upload](https://github.com/kdeps/kdeps/tree/main/examples/file-upload) - File processing
- [HTTP Advanced](https://github.com/kdeps/kdeps/tree/main/examples/http-advanced) - API integration
- [SQL Advanced](https://github.com/kdeps/kdeps/tree/main/examples/sql-advanced) - Multi-database
- [Batch Processing](https://github.com/kdeps/kdeps/tree/main/examples/batch-processing) - Items iteration
- [Control Flow](https://github.com/kdeps/kdeps/tree/main/examples/control-flow) - Conditionals and loop patterns
- [Tools](https://github.com/kdeps/kdeps/tree/main/examples/tools) - LLM function calling
- [Vision](https://github.com/kdeps/kdeps/tree/main/examples/vision) - Image processing

## Community

- **GitHub**: [github.com/kdeps/kdeps](https://github.com/kdeps/kdeps)
- **Issues**: [Report bugs and request features](https://github.com/kdeps/kdeps/issues)
- **Contributing**: [CONTRIBUTING.md](https://github.com/kdeps/kdeps/blob/main/CONTRIBUTING.md)
- **Examples**: [Browse example workflows](https://github.com/kdeps/kdeps/tree/main/examples)
