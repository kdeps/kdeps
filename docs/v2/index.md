---
layout: home

hero:
  name: KDeps
  text: Edge AI Workflow Framework
  tagline: Build AI agents for edge devices and APIs with YAML — audio, video, and telephony input, offline LLMs, wake-phrase activation, and speech output. No cloud required.
  image:
    src: /logo.svg
    alt: KDeps Logo
  actions:
    - theme: brand
      text: Get Started
      link: /getting-started/installation
    - theme: alt
      text: View on GitHub
      link: https://github.com/kdeps/kdeps

features:
  - icon: 🎙️
    title: Multi-Source I/O
    details: Accept input from audio hardware, cameras, telephony, and HTTP APIs — simultaneously. Transcribe with local Whisper or cloud STT.
  - icon: 🔇
    title: Offline-First
    details: Run entirely on-device — local LLMs via Ollama, offline STT (Whisper, Vosk), offline TTS (Piper, eSpeak). No network required.
  - icon: 🗣️
    title: Wake-Phrase Activation
    details: Always-on listening loop. Workflow triggers only when the wake phrase is detected — like "hey kdeps".
  - icon: 📝
    title: YAML-First Configuration
    details: Define workflows with simple, readable YAML. No complex programming required.
  - icon: 🤖
    title: LLM Integration
    details: Ollama for local models, or any OpenAI-compatible API endpoint. Vision, tools, and streaming supported.
  - icon: 🗄️
    title: Built-in SQL Support
    details: PostgreSQL, MySQL, SQLite, SQL Server, Oracle with connection pooling.
  - icon: 🐳
    title: Docker Ready
    details: Package everything into optimized Docker images. Runs on Raspberry Pi, Jetson, and x86 edge hardware.
---

# Introduction

KDeps is a YAML-based workflow framework for building AI agents on edge devices and API backends. It combines multi-source hardware I/O (audio, video, telephony), offline-capable LLMs, speech recognition, wake-phrase activation, and text-to-speech into portable, self-contained units that run anywhere — from Raspberry Pi to cloud servers.

## Key Highlights

### Multi-Source I/O for Edge Devices

KDeps accepts input from hardware devices and HTTP APIs — simultaneously. Configure audio, video, telephony, and API sources in one `workflow.yaml`:

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

| Source | Hardware |
|--------|----------|
| `audio` | ALSA microphone, line-in, USB audio |
| `video` | V4L2 camera, USB webcam, CSI camera |
| `telephony` | SIP/ATA adapter, Twilio |
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
- [API Response](resources/api-response)

### Concepts
- [Input Sources](concepts/input-sources) · [Unified API](concepts/unified-api)
- [Expressions](concepts/expressions) · [Expression Functions Reference](concepts/expression-functions-reference)
- [Request Object](concepts/request-object) · [Input Object](concepts/input-object)
- [Tools](concepts/tools) · [Items Iteration](concepts/items)
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

## Examples

Explore working examples:

**Edge AI / Voice:**
- [Voice Assistant](https://github.com/kdeps/kdeps/tree/main/examples/voice-assistant) - Offline wake-phrase + LLM + TTS on edge hardware
- [Vision Surveillance](https://github.com/kdeps/kdeps/tree/main/examples/vision-surveillance) - Camera capture + vision LLM analysis

**API Backends:**
- [Simple Chatbot](https://github.com/kdeps/kdeps/tree/main/examples/chatbot) - LLM chatbot
- [ChatGPT Clone](https://github.com/kdeps/kdeps/tree/main/examples/chatgpt-clone) - Full chat UI
- [File Upload](https://github.com/kdeps/kdeps/tree/main/examples/file-upload) - File processing
- [HTTP Advanced](https://github.com/kdeps/kdeps/tree/main/examples/http-advanced) - API integration
- [SQL Advanced](https://github.com/kdeps/kdeps/tree/main/examples/sql-advanced) - Multi-database
- [Batch Processing](https://github.com/kdeps/kdeps/tree/main/examples/batch-processing) - Items iteration
- [Tools](https://github.com/kdeps/kdeps/tree/main/examples/tools) - LLM function calling
- [Vision](https://github.com/kdeps/kdeps/tree/main/examples/vision) - Image processing

## Community

- **GitHub**: [github.com/kdeps/kdeps](https://github.com/kdeps/kdeps)
- **Issues**: [Report bugs and request features](https://github.com/kdeps/kdeps/issues)
- **Contributing**: [CONTRIBUTING.md](https://github.com/kdeps/kdeps/blob/main/CONTRIBUTING.md)
- **Examples**: [Browse example workflows](https://github.com/kdeps/kdeps/tree/main/examples)
