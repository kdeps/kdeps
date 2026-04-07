# KDeps v2 Examples

This directory contains example workflows demonstrating KDeps v2 features.

## Examples

### 🤖 [Chatbot](./chatbot/)
Basic LLM chatbot with validation and error handling.
- YAML configuration
- HTTP API with CORS
- Preflight validation
- LLM integration with Ollama

**Run:**
```bash
cd chatbot && kdeps run workflow.yaml --dev
```

### 🌐 [HTTP Advanced](./http-advanced/)
Advanced HTTP client features with authentication, caching, and retries.
- Multiple authentication methods (Bearer, API Key, Basic, OAuth2)
- Intelligent retry logic with exponential backoff
- Response caching with TTL
- Proxy and TLS configuration
- Custom headers and timeout management

**Run:**
```bash
cd http-advanced && kdeps run workflow.yaml --dev
```

### 🐚 [Shell Exec](./shell-exec/)
Shell command execution with timeout and output capture.
- System command execution
- Output parsing and formatting
- Timeout handling
- Cross-platform shell support

**Run:**
```bash
cd shell-exec && kdeps run workflow.yaml --dev
```

### 🗄️ [SQL Advanced](./sql-advanced/)
Advanced SQL operations with batch processing and multiple result formats.
- Named database connections
- Batch operations (`paramsBatch`)
- CSV and JSON result formatting
- Transaction support
- Connection pooling

**Features:**
- PostgreSQL analytics queries (CSV output)
- MySQL inventory management
- Batch user status updates
- Multi-database transactions

**Run:**
```bash
cd sql-advanced && kdeps run workflow.yaml --dev
```

### 📤 [File Upload](./file-upload/)
File upload handling with multipart form-data support.
- Single and multiple file uploads
- File metadata access (count, names, types)
- File content, path, and MIME type access
- Integration with `get()` and `info()` functions

**Run:**
```bash
cd file-upload && kdeps run workflow.yaml --dev
```

### 👁️ [Vision](./vision/)
Vision model integration with image analysis.
- Image file upload via multipart form-data
- Vision model integration (moondream, llava)
- Image analysis with natural language queries
- JSON structured responses

**Prerequisites**: Install vision model: `ollama pull moondream:1.8b`

**Run:**
```bash
cd vision && kdeps run workflow.yaml --dev
```

### 🎙️ [Voice Assistant](./voice-assistant/)
Fully offline voice assistant with wake phrase detection, LLM response, and TTS output.
- Continuous microphone capture
- Wake phrase detection ("hey kdeps") with `faster-whisper`
- Offline speech-to-text transcription
- LLM response via Ollama (llama3.2:1b)
- Text-to-speech output with Piper TTS
- No cloud required — runs fully air-gapped

**Prerequisites**: `pip install faster-whisper piper-tts` and `ollama pull llama3.2:1b`

**Run:**
```bash
cd voice-assistant && kdeps run workflow.yaml
```

### 📞 [Telephony Bot](./telephony-bot/)
AI-powered phone call handler using Twilio, Deepgram STT, and OpenAI TTS.
- Twilio webhook integration for inbound calls
- Real-time speech transcription with Deepgram
- LLM response generation
- Cloud text-to-speech with OpenAI TTS

**Prerequisites**: Twilio account, Deepgram API key, OpenAI API key

**Run:**
```bash
cd telephony-bot && kdeps run workflow.yaml --dev
```

### 📹 [Video Analysis](./video-analysis/)
Continuous camera surveillance with AI-powered frame analysis.
- V4L2 / AVFoundation / DirectShow camera capture
- Vision LLM analysis with `llava:7b`
- Structured JSON output (people, vehicles, activity, alerts)
- Activity log written to disk

**Prerequisites**: `ollama pull llava:7b` and `ffmpeg`

**Run:**
```bash
cd video-analysis && kdeps run workflow.yaml
```

### 🔧 [Tools](./tools/)
LLM tool calling / function calling with automatic tool execution.
- Tool/function definitions in ChatConfig
- Tool parameter schemas (OpenAI Functions format)
- Automatic tool execution when LLM calls tools
- Tool results fed back to LLM for final response
- Multi-turn conversations with tools
- Multiple tools per request

**Features:**
- ✅ Tool definitions work
- ✅ Automatic tool execution
- ✅ Tool resources (Python, SQL, HTTP, Exec, APIResponse)
- ✅ Tool arguments automatically stored in memory

**Run:**
```bash
cd tools && kdeps run workflow.yaml --dev
```

### 🔐 [Session Auth](./session-auth/)
Session management with persistent storage.
- SQLite-backed persistent session storage
- Session TTL configuration
- Session data storage and retrieval
- Preflight checks using session data

**Run:**
```bash
cd session-auth && kdeps run workflow.yaml --dev
```

### 🌍 [Auto-Env](./auto-env/)
Demonstrates automatic environment variable scoping per component, auto-scaffolded `.env` templates, and `kdeps component update`.
- `TRANSLATOR_OPENAI_API_KEY` overrides `OPENAI_API_KEY` inside `translator` only
- On first run, kdeps auto-creates `.env` template with all `env()` vars listed blank
- On first run, kdeps auto-creates `README.md` from component metadata
- `kdeps component update` merges new vars into an existing `.env`
- Summarizer falls back to extractive summary when no API key is set

**Run:**
```bash
export OPENAI_API_KEY=sk-...          # global fallback
export TRANSLATOR_OPENAI_API_KEY=sk-... # translator only (optional)
cd auto-env && kdeps run workflow.yaml
```

### 🔌 [Component Input Source](./component-input-source/)
A sub-workflow designed to be called exclusively via `run.component`. Declares `sources: [component]` so no HTTP server or listener is started.
- `sources: [component]` - no listener started, driven by parent
- Reusable text-transformation sub-module
- Access via `run.component` from any parent workflow
- Demonstrates `component.description` field

**Invoke from a parent:**
```bash
# This workflow has no standalone entry-point.
# Call it from a parent workflow resource:
#   run:
#     component:
#       name: component-input-source
#       with:
#         text: "hello world"
#         style: title
kdeps validate examples/component-input-source
```

### 🔁 [Component Setup/Teardown](./component-setup-teardown/)
Demonstrates automatic dependency installation and per-run cleanup via `setup` and `teardown` lifecycle hooks in a custom local component.
- `setup.pythonPackages` — installed once via `uv pip`
- `setup.osPackages` — installed once via `apt-get` / `apk` / `brew`
- `setup.commands` — run once after packages are ready
- `teardown.commands` — run after every invocation

**Run:**
```bash
cd component-setup-teardown && kdeps run workflow.yaml
```

### 📄 [File Processor](./file-processor/)
Single-shot document summarizer using the `file` input source. Reads content from `--file`, stdin, or `KDEPS_FILE_PATH` and returns a structured AI analysis.
- `sources: [file]` — runs once and exits
- Input via `--file`, stdin pipe, or `KDEPS_FILE_PATH`
- `input('fileContent')` / `input('filePath')` in resources
- Structured JSON output with title, summary, key points

**Run:**
```bash
cd file-processor
echo "The Eiffel Tower was built in 1889 for the World's Fair." | kdeps run workflow.yaml
# or
kdeps run workflow.yaml --file /path/to/document.txt
```

### 🎛️ [Input Component](./input-component/)
Shows the built-in `input` component — pre-installed, no `kdeps component install` needed. Collects named input slots and returns structured JSON.
- Uses the built-in `input` component with `query`, `text`, and `key` slots
- 14 named slots: `query`, `prompt`, `text`, `data`, `key`, `value`, `a`–`h`
- Passes collected inputs to an LLM for Q&A
- Access component output with `output('actionId')`

**Run:**
```bash
cd input-component && kdeps run workflow.yaml
```

## Quick Start

1. Choose an example directory
2. Update database connections if needed
3. Run: `kdeps run workflow.yaml --dev`
4. Test endpoints with curl or your browser

## Learning Path

1. **Start with Chatbot** - Learn basic YAML, HTTP API, and LLM integration
2. **Advanced SQL** - Master database operations, batch processing, and result formatting
3. **Build Your Own** - Combine features from examples

## Database Setup

Most examples require databases. See each example's README for setup instructions.

For quick testing, you can use SQLite (no setup required) by changing connection strings to:
```yaml
connection: "sqlite:///data.db"
```

## Development Mode

Use `--dev` flag for hot reload during development:
```bash
kdeps run workflow.yaml --dev
```

This automatically restarts the server when you modify workflow files or resources.
