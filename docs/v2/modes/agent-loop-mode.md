# Agent Loop Mode

Agent loop mode starts an interactive LLM REPL where whole workflows and components are registered as callable tools. The LLM decides which tool to invoke based on the user's prompt. Workflow tools run the full pipeline atomically so all `requires:` dependencies resolve correctly.

Running `kdeps` with no arguments starts a model-only REPL with no workflow tools. Pass a path to load workflows/agencies as tools.

## Starting the agent loop

```bash
kdeps                              # model-only REPL (no tools)
kdeps ./my-agent/                  # one workflow = one tool
kdeps ./agents/                    # folder = every workflow inside becomes a tool
kdeps ./my-agent/ --model llama3.2 --system "You are a DevOps assistant."
kdeps --skill ~/.kdeps/skills/     # load skill files
kdeps --resume <session-id>        # continue a saved session
```

## REPL slash commands

Inside the REPL, type `/help` for the full list:

| Command | Description |
|---------|-------------|
| `/help` | Show available commands |
| `/clear` | Summarize and clear the current conversation |
| `/model [name]` | Show or switch LLM model mid-session (tab-complete shows up to 10 suggestions) |
| `/model default [name]` | Show or set the default startup model, persisted to `~/.kdeps/agent-loop-settings.yaml` |
| `/model list` | List all available models with provider status |
| `/model ps` | List running local model servers (llamafile/gguf) with PID, port, and health |
| `/model ps kill <model>` | Kill a running local model server and clean up its port file |
| `/model ps switch <model>` | Switch the active model to a running local server |
| `/model hff search <query>` | Search HuggingFace for GGUF repos (sorted by downloads) |
| `/model hff info <repo>` | List GGUF files and sizes available in a HuggingFace repo |
| `/model hff download <repo> [file]` | Download a GGUF from HuggingFace; auto-registers an alias for `/model` |
| `/skills` | List loaded skills |
| `/prompts` | List loaded prompt templates |
| `/<skill-name> [prompt]` | Invoke a skill or prompt template directly |
| `/compact` | Summarize history to free context |
| `/history` | Show conversation history |
| `/thinking [off\|low\|medium\|high\|auto]` | Enable extended reasoning (Claude only; warns if current model does not support it) |
| `/session list\|save\|load\|delete\|checkpoint\|goto\|branches\|import` | Manage saved sessions and navigate branching history |
| `/editor` | Open current input in `$EDITOR` (ctrl+g) |
| `/copy` | Copy last assistant response to clipboard |
| `/reload` | Reload skills and prompt templates from disk |
| `/settings` | Open the tool/skill selector |
| `/exit` | Exit the REPL |
| `! <cmd>` | Run a shell command; result is added to LLM context |
| `!! <cmd>` | Run a shell command without adding it to LLM context |

## Local model management

### Switching models

`/model <name>` switches models mid-session. For local backends (`file`, `gguf`), the REPL downloads and starts the server if it isn't already running, then shows a progress display until the completions endpoint is accepting requests — the first prompt after the switch never gets a "network error" while weights load.

```
/model qwen3.5-4b                     # switch to a known alias
/model default qwen3.5-4b             # save as default startup model
/model default                        # show the current default
```

The default model is persisted to `~/.kdeps/agent-loop-settings.yaml` and loaded automatically at startup when `--model` is not passed.

### Searching and downloading from HuggingFace

`/model hff` lets you discover and download GGUF models directly from within the REPL. Set `HF_TOKEN` in your environment to authenticate (required for gated models; increases rate limits for all requests).

```bash
# Search for GGUF repos by keyword (sorted by downloads)
/model hff search qwen3

# List GGUF files and sizes inside a repo
/model hff info unsloth/Qwen2.5-VL-7B-Instruct-GGUF

# Download a specific file — registers it as an alias in ~/.kdeps/gguf_versions.yaml
/model hff download unsloth/Qwen2.5-VL-7B-Instruct-GGUF Qwen2.5-VL-7B-Instruct-Q4_K_M.gguf

# Switch to it immediately after download
/model Qwen2.5-VL-7B-Instruct-Q4_K_M
```

`/model hff download <repo>` without a filename shows the available files (same as `/model hff info`). Downloaded files go to `~/.kdeps/models/` and the alias is the filename without the `.gguf` extension.

### Managing running servers

`/model ps` shows all llamafile and llama-server processes started in the current session:

```
PID      PORT   BACKEND      MODEL                                STATUS
12345    8080   gguf         Qwen2.5-VL-7B-Instruct-Q4_K_M       healthy
12346    8081   file         phi4                                  loading
```

```
/model ps kill phi4           # send SIGKILL, remove port file
/model ps switch phi4         # set active model to an already-running server
```

## Built-in tools

The agent has access to a set of built-in tools that the LLM can call without any YAML configuration. Tools that require credentials are only registered when the relevant environment variable is set.

### Shell execution

`bash_exec` runs any shell command and streams output to the terminal. Two keyboard shortcuts change its behavior mid-run:

| Key | Effect |
|-----|--------|
| `Ctrl+C` | Kill the process. Partial output is returned to the LLM as a success result so it can decide what to do next. |
| `Ctrl+Z` | Detach the process as a background job. `bash_exec` immediately returns `{"status":"backgrounded","job_id":N}` to the LLM. |

`Ctrl+Z` at the REPL prompt (no tool running) suspends kdeps normally (`fg` to resume).

Background jobs are managed with two companion tools:

| Tool | Description |
|------|-------------|
| `bash_job_list` | Show all background jobs with status (`running`/`done`/`failed`), elapsed time, and command |
| `bash_job_wait` | Block until a job completes and return its full output. Pass `job_id` from the backgrounded result. |

Set `KDEPS_ALLOW_BASH=false` to disable all three `bash_*` tools.

### File operations

Always available. No environment variables required.

| Tool | Description |
|------|-------------|
| `read_file` | Read file contents |
| `write_file` | Write or overwrite a file |
| `edit_file` | Apply a unified diff to a file |
| `list_files` | List directory contents |

### Web and search

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `web_search` | (none -- uses DuckDuckGo) | Search the web |
| `wikipedia` | (none) | Fetch a Wikipedia article |
| `web_scraper` | (none) | Fetch and extract text from any URL |
| `serpapi_search` | `SERPAPI_API_KEY` | Google search via SerpAPI |
| `exa_search` | `EXA_API_KEY` or `METAPHOR_API_KEY` | Neural search via Exa |
| `perplexity_search` | `PERPLEXITY_API_KEY` | Search via Perplexity |

### Computation

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `calculator` | (none) | Evaluate math expressions |
| `wolfram_alpha` | `WOLFRAM_APP_ID` | Wolfram Alpha queries |

### Data and SQL

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `sql_list_tables` | `KDEPS_SQL_DB_PATH` or connection config | List tables in a database |
| `sql_describe_table` | same | Describe a table's columns and types |
| `sql_query` | same | Execute a SELECT query |

### Embeddings and reranking

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `retrieve_context` | (none) | Semantic search over a local vector store |
| `cohere_rerank` | `COHERE_API_KEY` | Rerank results using Cohere |
| `voyageai_rerank` | `VOYAGEAI_API_KEY` | Rerank results using VoyageAI |
| `jina_rerank` | `JINA_API_KEY` | Rerank results using Jina |

### Actions and integrations

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `zapier_list_actions` | `ZAPIER_NLA_API_KEY` | List available Zapier NLA actions |
| `zapier_run_action` | `ZAPIER_NLA_API_KEY` | Execute a Zapier NLA action |
| `google_cache_create` | (Google credentials) | Create a Google AI cached content object |
| `google_cache_list` | (Google credentials) | List Google AI cached content objects |
| `google_cache_delete` | (Google credentials) | Delete a Google AI cached content object |

### Resource-backed tools

These always-on tools invoke the corresponding kdeps executor directly:

| Tool | Description |
|------|-------------|
| `http_request` | Make an HTTP request (GET/POST/PUT/DELETE/PATCH) |
| `search_local` | Search the local document index |
| `transcribe_audio` | Transcribe an audio file |
| `load_document` | Load and extract text from a document |

## Multimodal input

Attach images and other binary files to your prompt using `@`:

```bash
# Attach a local image
describe @photo.png what is in this image?

# Attach multiple images
compare @before.jpg @after.jpg what changed?

# Attach a remote image URL
analyze @https://example.com/chart.png what trend does this show?

# Embed a text file inline (text files expand inline, not as attachments)
review @notes.txt and summarize the key points
```

- Image/binary refs (`.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`, `.bmp`, `.tiff`, `.pdf`, `.mp3`, `.mp4`, `.wav`) are sent as multimodal content to the LLM
- Text file refs are expanded inline in the prompt
- Unresolvable refs (file not found, access denied) are left unchanged in the text

## Skills

Skills are markdown files with optional YAML frontmatter that teach the agent how to behave in specific contexts. Place them in `~/.kdeps/skills/` or pass `--skill <path>` at startup.

```markdown
---
name: code-review
description: Guidelines for reviewing Go code
---

Always check for error handling. Prefer early returns over nested conditions.
```

Skills are discovered from:
- `~/.kdeps/skills/` (global)
- `./.kdeps/skills/` (project-local)
- Paths passed with `--skill` (explicit, repeatable)

Invoke a skill from the REPL with `/<skill-name>` or `/<skill-name> extra context here`.

## Prompt templates

Prompt templates are reusable named prompts loaded from `.md` files. They work exactly like skills: invoke them by name from the REPL.

```markdown
---
name: review-pr
description: Review a GitHub pull request
argument-hint: <PR number or URL>
---

Review the pull request at $1. Check for: correctness, test coverage, and breaking changes.
```

Place templates in `~/.kdeps/prompts/` or `./.kdeps/prompts/`. Templates use the same placeholder syntax as skills: `$1`, `$2`, `$@`, `${1:-default}`.

```bash
/review-pr 1234
/summarize this document for a technical audience
```

## Instructions

The agent automatically discovers instruction files by walking up the directory tree from CWD:

- `CLAUDE.md`, `CLAUDE.local.md` at any ancestor directory
- `.kdeps/CLAUDE.md`, `.kdeps/instructions.md` at any ancestor directory

Duplicate content (by hash) is deduplicated. Total injected context is capped at ~12 KB. Instructions are injected into the system prompt at startup.

## Session persistence

Every conversation is saved as a JSONL file under `~/.kdeps/sessions/`. To resume a previous session:

```bash
kdeps --resume <session-id>
```

Session IDs are shown at the start of each run.

### Session commands

```
/session list                  # list all saved sessions
/session save [name]           # save current session
/session load <id>             # restore a saved session
/session delete <id>           # delete a saved session
/session checkpoint            # print the current entry ID (for /session goto)
/session goto <entry-id>       # restore session to the turn at that entry ID
/session branches              # list stashed (pruned) turns from prior /session goto calls
/session import <path>         # load a JSONL session file exported from another run
```

`/session goto` is non-destructive: the pruned tail is stashed. Use `/session branches` to see stashed entry IDs, then `/session goto <id>` again to navigate back.

### Auto-retry

Transient LLM errors (HTTP 429, 5xx, network timeouts) are automatically retried up to 3 times with exponential backoff (2s, 4s, 8s). Context-overflow and authentication errors are not retried.

## Single workflow vs folder

```bash
kdeps ./my-agent/     # One workflow = one tool (named after metadata.name)
kdeps ./agents/       # Folder = every workflow and agency inside becomes a separate tool
```

When you point to a folder, kdeps discovers every workflow and agency file inside it (recursively). Each becomes a separate tool. The tool name is `metadata.name` from the workflow's manifest -- not the filename.

## Concrete example

Given this workflow:

```yaml
# my-agent/workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: my-agent          # this becomes the tool name the LLM sees
  version: "1.0.0"
  description: "Answers questions about our product"
  targetActionId: response

settings:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]
```

Running:

```bash
kdeps ./my-agent/
```

The LLM receives one tool named `my-agent`. When it calls that tool, kdeps runs the full workflow DAG -- every resource in dependency order -- and returns `apiResponse.response` to the LLM.

## How it works

```d2
direction: down

A: user prompt {shape: oval}
B: "LLM receives prompt\ntool registry: one tool per workflow, one per agency, one per component"
C: tool type? {shape: diamond}
D: "kdeps runs full workflow pipeline\nall requires: deps resolve in order"
E: "kdeps runs agency entry-point pipeline\ninternal agents resolve via agent: resource type"
H: "kdeps runs component in isolation\ninputs map to component interface fields"
F: more tools needed? {shape: diamond}
G: final answer {shape: oval}

A -> B
B -> C: LLM picks a tool
C -> D: workflow
C -> E: agency
C -> H: component
D -> F: apiResponse returned to LLM
E -> F: result returned to LLM
H -> F: result returned to LLM
F -> C: yes
F -> G: no
```

## Tool registration

| Target | Tools registered |
|--------|-----------------|
| No path (model-only) | None -- pure LLM conversation |
| Single workflow file/dir | One tool (`metadata.name`) + one tool per component |
| Single agency file | One tool (`agency metadata.name`) |
| Folder | One tool per workflow/agency found recursively + component tools |

## Command

```bash
kdeps [path] [flags]
```

`[path]` is optional. When provided it must be a workflow/agency file or directory. The tool name comes from `metadata.name` -- not the filename.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--model` | `KDEPS_AGENT_MODEL` or `llama3.2` | LLM model name |
| `--backend` | `KDEPS_AGENT_BACKEND` or `file` | LLM backend (`file`, `gguf`, `ollama`, `openai`, ...) |
| `--base-url` | `KDEPS_AGENT_BASE_URL` | LLM API base URL |
| `--system` | (none) | System prompt injected at conversation start |
| `--skill` | (none) | Path to a skill file or directory (repeatable) |
| `--prompt` | (none) | Path to a prompt templates directory (repeatable) |
| `--resume` | (none) | Session ID to resume a previous conversation |
| `--debug` | false | Enable debug logging |

### Environment variables

```bash
KDEPS_AGENT_MODEL=llama3.2
KDEPS_AGENT_BACKEND=file              # default: local llamafile
# KDEPS_AGENT_BACKEND=gguf           # llama.cpp via llama-server
# KDEPS_AGENT_BACKEND=ollama         # requires ollama server
# KDEPS_AGENT_BASE_URL=http://localhost:11434
```

## Examples

```bash
# Pure LLM REPL, no workflows
kdeps

# Single workflow -- one tool
kdeps ./my-agent/

# All workflows in a folder
kdeps ./agents/

# Specify model and system prompt
kdeps ./agents/ --model mistral --system "You are a data analyst."

# GGUF backend with local model file
kdeps --backend gguf --model qwen3.5-4b

# OpenAI backend
KDEPS_AGENT_BACKEND=openai kdeps ./agents/ --model gpt-4o

# Load a skill directory
kdeps --skill ~/.kdeps/skills/

# Resume a previous session
kdeps --resume abc123def456
```

## Differences from workflow mode

| | Workflow mode (`kdeps run`) | Agent loop mode (`kdeps [path]`) |
|--|-----------------------------|---------------------------------|
| Execution | DAG, deterministic | LLM loop, tool-driven |
| Entry point | `metadata.targetActionId` | User prompt |
| Unit of work | Individual resources | Whole workflows |
| Tools exposed | N/A | One per workflow + one per component |
| Input | Single workflow path | Optional file or folder |
| Session memory | None | Multi-turn, persistent JSONL |

## See Also

- [Workflow Mode](workflow-mode) - Deterministic DAG pipelines
- [LLM Provider Reference](/reference/llm-providers) - Backend config and model names
- [Agencies](/concepts/agency) - Multi-agent orchestration
