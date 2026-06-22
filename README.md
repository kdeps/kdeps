# kdeps

[![NVIDIA Inception](https://img.shields.io/badge/NVIDIA-Inception-76B900?logo=nvidia&logoColor=white)](https://www.nvidia.com/en-us/startups/)
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

Run AI workflows locally. Or deploy them anywhere. Proud member of the [NVIDIA Inception](https://www.nvidia.com/en-us/startups/) program for AI startups.

## Run in 30 seconds

```bash
# Install
brew install kdeps/tap/kdeps
# or
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh

# Run - you're in an AI REPL immediately
kdeps
```

No API key needed if you have [Ollama](https://ollama.com) or [llamafile](https://github.com/Mozilla-Ocho/llamafile) installed. kdeps auto-detects local models and downloads them on first use.

```bash
kdeps --model llama3.2              # use any Ollama model
kdeps --model llama3.2:1b-q4        # GGUF quantization - auto-downloaded from HuggingFace
kdeps --model /path/to/model.gguf   # point directly at a local GGUF file
kdeps ./my-agent/                   # load your workflow as tools for the agent
```

**Local models** - three options, zero cloud dependency:
- **Ollama** (`backend: ollama`) - managed model server, `ollama pull llama3.2` then `kdeps`
- **llamafile** (`backend: file`) - model + server as a single binary, runs on any OS, no install
- **GGUF** (`backend: gguf`) - raw GGUF files, point `llm.model_path` at a local file or let kdeps auto-download from HuggingFace using aria2c

Slash commands inside the REPL: `/model` switches models (opens a TUI picker if no argument), `/clear` resets context, `/help` shows all commands, `/exit` quits. Sessions persist under `~/.kdeps/sessions/` and resume with `--resume <session-id>`.

## Build with AI assistance

Use Claude Code, Cursor, or any coding agent to scaffold kdeps workflows:

```bash
npx skills add https://github.com/kdeps/skill --skill kdeps
```

Then ask your agent: *"create a kdeps workflow that summarizes a URL and returns JSON"* - it knows the full schema, resource types, and packaging format. Works with the global flag (`-g`) to install once for all projects.

Docs: [kdeps.com/getting-started/agent-skills](https://kdeps.com/getting-started/agent-skills)

## Book

[<img src="https://d2sofvawe08yqg.cloudfront.net/kdeps/s_hero?1779817160" alt="AI Appliances book cover" width="140" align="right" style="margin-left:16px">](https://leanpub.com/kdeps)

**[AI Appliances - Build & Deploy Autonomous AI Agents and Agencies in YAML](https://leanpub.com/kdeps)**
Free. PDF, EPUB, and web.

Hands-on guide covering deterministic pipelines, multi-agent orchestration, error handling, and vendor-agnostic deployment - the production challenges most AI frameworks leave to you.

<br clear="right">

## Build your own workflow

A workflow is a DAG of resources. Each step declares what it needs via `requires:` and runs in the correct order automatically.

```bash
kdeps init my-agent         # scaffold a new workflow directory
cd my-agent
kdeps run workflow.yaml     # run a single workflow file
kdeps run .                 # run from directory (finds workflow.yaml automatically)
kdeps run . --dev           # hot reload on file change
```

See the full YAML example under [Workflow mode](#workflow-mode) below, or scaffold one with the [kdeps skill](#build-with-ai-assistance).

## Distribute your agents

Workflows, components, and agencies compile to portable package files:

```bash
kdeps bundle package my-agent/        # creates my-agent-1.0.0.kdeps
kdeps bundle package my-component/    # creates my-component-1.0.0.komponent
kdeps bundle package my-agency/       # creates my-agency-1.0.0.kagency
```

Recipients run or install them directly - no source needed:

```bash
# Run a package file directly (no install step)
kdeps run my-agent-1.0.0.kdeps
kdeps run my-agency-1.0.0.kagency

# Or install system-wide and run by name
kdeps registry install my-agent.kdeps       # installs to ~/.kdeps/agents/my-agent/
kdeps registry install my-comp.komponent    # installs to ~/.kdeps/components/my-comp/
kdeps exec my-agent                         # run installed agent by name
kdeps registry install my-agent            # install by name from kdeps.io
```

Publish to [kdeps.io](https://kdeps.io) for one-line install by the community:

```bash
kdeps registry verify .
kdeps registry submit --tag v1.0.0
```

## Deploy anywhere

```bash
kdeps bundle build          # Docker image
kdeps bundle export iso     # bootable edge ISO
kdeps bundle prepackage     # self-contained binary per arch
kdeps export k8s            # Kubernetes manifests
```

Full deployment guide: [kdeps.com/guides/deployment-guide](https://kdeps.com/guides/deployment-guide)

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
    timezone: Etc/UTC
```

```yaml
# resources/fetch.yaml
actionId: fetch
name: Fetch Page
httpClient:
  method: GET
  url: "{{ get('url') }}"
  timeout: 10s
```

```yaml
# resources/respond.yaml
actionId: respond
name: Summarize and Respond
requires: [fetch]
chat:
  model: llama3.2:1b  # local llamafile, auto-downloaded on first run - no LLM server needed
  prompt: "Summarize this page: {{ output('fetch').body }}"
apiResponse:
  response: "{{ output('respond').message.content }}"  # the model's reply text
```

```bash
kdeps run workflow.yaml          # local, instant startup
kdeps run workflow.yaml --dev    # hot reload
```

**Resource types:** `chat`, `httpClient`, `python`, `exec`, `sql`, `email`, `scraper`, `browser`, `embedding`, `searchLocal`, `searchWeb`, `agent`, `component`, `file`, `git`, `codeIntelligence`, `loader`, `vectorStore`, `transcribe`

**LLM providers:** OpenAI, Anthropic, Google (Gemini / Vertex AI), DeepSeek, Groq, xAI, Mistral, Cohere, Together, Perplexity, OpenRouter, Bedrock (AWS), WatsonX (IBM), Cloudflare, HuggingFace, Maritaca, Ernie — plus Ollama, llamafile, and GGUF for local inference

**Advanced LLM features:** chain-of-thought injection, semantic few-shot selection (embedding-based), Anthropic prompt caching + 128K extended output, Google AI cached content + Vertex AI, Ollama extended thinking, per-provider sampling controls (`candidateCount`, `minLength`, `maxLength`)

**Embedding backends:** OpenAI, Google, HuggingFace, Jina, VoyageAI, Bedrock, Cybertron (local), Ollama

**Vector store providers:** Qdrant, Chroma, Pinecone, Weaviate, OpenSearch, pgvector, MongoDB, Redis, Azure AI Search, MariaDB, Dolt, Bedrock Knowledge Bases

**Download acceleration:** aria2c with resume support, configurable via `llm.aria2c_flags` in config.yaml — falls back to built-in HTTP downloader if aria2c is not installed

**Expressions:** `get('key')` reads request input, `output('actionId')` reads a prior step's result, `set('key', val)` stores state. All expressions are safe inside `{{ }}` — Jinja2 control flow (`{% if %}`, `{% for %}`) is also supported.

### Agent mode

Autonomous LLM loop. Each workflow is registered as a callable tool, named after its `metadata.name` -- the LLM decides which tools to call, in what order, to complete the task. Calling a tool runs that workflow's full pipeline, so every `requires:` dependency resolves correctly. Agencies and installed components become tools too; individual resources are never exposed directly.

```
stdin prompt
      |
      v
+---------------------+
|  LLM                |  plans steps, picks tools
+---------------------+
      |
      +-- call tool: summarizer    -->  runs that workflow's full DAG
      |
      +-- call tool: research-bot  -->  runs another workflow
      |
      +-- call tool: scraper       -->  runs an installed component
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
kdeps                              # model-only REPL, no workflows
kdeps ./my-agent/                  # one workflow = one tool
kdeps ./agents/                    # folder = every workflow inside becomes a tool
kdeps ./my-agent/ --model llama3.2 --system "You are a DevOps assistant."
kdeps --skill ~/.kdeps/skills/     # load skill files into the agent
kdeps --resume <session-id>        # continue a previous conversation
```

The agent reads from stdin (REPL with slash commands: `/help`, `/clear`, `/model`, `/skills`, `/history`, `/exit`) and runs until you exit. Sessions are persisted as JSONL under `~/.kdeps/sessions/` and can be resumed with `--resume`. Workflows, agencies, and installed components are available as tools without any extra wiring.

`/model` with no arguments opens an interactive TUI model picker with search, type-to-filter, and visual tags for local vs cloud models. `/model <name>` switches models and auto-starts local servers for llamafile, GGUF, and Ollama models. Model downloads use aria2c for fast parallel downloads with resume support. Local model servers are automatically cleaned up on exit.

```bash
kdeps llamafile list               # list all LF + GGUF + Ollama models
kdeps llamafile update             # refresh from HuggingFace
```

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
# resources/draft.yaml
actionId: draft
name: Draft Post
agent:
  name: content-writer        # runs agents/content-writer/workflow.yaml
  params:
    topic: "{{ get('topic') }}"  # passed as get('topic') inside that agent
```

```yaml
# resources/publish.yaml
actionId: publish
name: Publish Post
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

## Registry

```bash
kdeps registry search <query>
kdeps registry install <package>
kdeps registry submit --tag v1.0.0   # generate formula for kdeps.io PR
```

## Agent skill

A [coding-agent skill](https://github.com/kdeps/skill) teaches Claude Code, Cursor,
Grok, and other agents how to scaffold kdeps workflows, components, and agencies —
including `kdeps.pkg.yaml` for [kdeps.io](https://kdeps.io) distribution.

```bash
git clone https://github.com/kdeps/skill ~/.claude/skills/kdeps
```

Docs: [kdeps.com/getting-started/agent-skills](https://kdeps.com/getting-started/agent-skills)

## Global config

```bash
kdeps edit    # opens ~/.kdeps/config.yaml
kdeps doctor  # check config, LLM backend, Python, installed agents
```

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: file             # default: local llamafile, no server install. Also: gguf, ollama, openai, anthropic, groq, xai, openrouter, ...
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
  webServer:                         # optional; same rateLimit/maxBodyBytes/maxConcurrent fields
    rateLimit:
      requestsPerMinute: 120
  certFile: /path/to/cert.pem        # TLS -- omit for plain HTTP
  keyFile: /path/to/key.pem
```

## Logging

Structured JSON via `log/slog`. Set `KDEPS_LOG_FORMAT=json` for production output. Default level: WARN. Flags: `--verbose` (INFO), `--debug` (DEBUG).

---

[Documentation](https://kdeps.com) | [Registry](https://kdeps.io) | Apache 2.0
