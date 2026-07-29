# Chapter 5: Agent Mode

Agent mode flips the control model. In workflow mode, you define a fixed execution path and the framework follows it. In agent mode, you define tools (each tool is a complete workflow), and the LLM decides which tools to call, in what order, based on what the user asks.

This is the mode that makes kdeps an **autonomous agent platform**. The LLM does not call individual resources — it calls whole workflows. Each workflow call runs the full DAG deterministically. The non-determinism lives at the orchestration layer (which tool to call), not inside the tools themselves.

## Starting Agent Mode

```bash
# Model-only REPL, no workflow tools
$ kdeps

# Load a single workflow as one tool
$ kdeps ./my-agent/

# Load a directory — every workflow inside becomes a separate tool
$ kdeps ./agents/

# Load skills from a directory
$ kdeps --skill ~/.kdeps/skills/

# Resume a previous session
$ kdeps --resume <session-id>
```

When you run `kdeps`, kdeps:

1. Discovers all `workflow.yaml` files in the target path
2. Registers each as a callable tool using `metadata.name` as the tool name and `metadata.description` as the tool description
3. Starts the LLM loop
4. Waits for user input

The LLM REPL starts:

```
kdeps agent> How can I help you?
> 
```

Type a prompt. The LLM decides which tools to call. Tool calls trigger the full workflow DAG. Results come back to the LLM. The LLM synthesizes a response.

## What the LLM Sees

When the LLM is presented with a set of tools, it sees tool names, descriptions, and input schemas — not individual resources.

Given a workflow:

```yaml
# workflow.yaml
metadata:
  name: web-researcher
  description: "Fetches a URL and answers questions about its content"
```

The LLM sees:

```
Tool: web-researcher
Description: Fetches a URL and answers questions about its content
Input: { "input": string }
```

The LLM never knows about `resources/scraper.yaml` or `resources/llm.yaml` inside the workflow. It only knows there is a tool called `web-researcher` that takes an input string. The entire internal pipeline is an implementation detail.

This is why the `description` field in `workflow.yaml` matters in agent mode: it is what the LLM reads to decide whether to call this tool. Make it specific.

## Single Workflow vs. Folder Mode

**Single workflow:**

```bash
$ kdeps ./my-agent/
```

One tool is registered: `my-agent` (from `metadata.name` in `workflow.yaml`). The LLM has one thing it can call.

**Folder mode:**

```
agents/
  research/
    workflow.yaml    # metadata.name: research-agent
  writer/
    workflow.yaml    # metadata.name: writer-agent
  summarizer/
    workflow.yaml    # metadata.name: summarizer-agent
```

```bash
$ kdeps ./agents/
```

Three tools are registered: `research-agent`, `writer-agent`, `summarizer-agent`. The LLM can call any of them, in any order, any number of times.

Folder mode is how you build autonomous multi-step systems. You give the LLM a set of specialized tools and a task, and it composes them to solve the task.

## Tool Inputs and Outputs

When the LLM calls a tool, it passes an `input` field. Inside the workflow, this value is available via `get('input')`.

```yaml
# resources/process.yaml
actionId: process
chat:
  model: llama3.2:1b
  prompt: "Process this: &#123;&#123; get('input') &#125;&#125;"
```

When the workflow completes, the value of `apiResponse.response` is returned to the LLM as the tool result. This is what the LLM reads before deciding what to do next.

```yaml
# resources/respond.yaml
actionId: respond
requires: [process]
apiResponse:
  success: true
  response:
    result: get('process')
    summary: get('summary')
```

The LLM receives `{ "result": "...", "summary": "..." }` as the tool result.

## A Practical Example: Research Assistant

Suppose you have three workflows:

```
agents/
  search/workflow.yaml      # metadata.name: web-search
  scraper/workflow.yaml     # metadata.name: page-reader
  writer/workflow.yaml      # metadata.name: report-writer
```

```yaml
# agents/search/workflow.yaml
metadata:
  name: web-search
  description: "Searches the web for recent information on a topic. Returns a list of URLs and snippets."
```

```yaml
# agents/scraper/workflow.yaml
metadata:
  name: page-reader
  description: "Fetches the full text content of a given URL. Use this after web-search to read specific pages."
```

```yaml
# agents/writer/workflow.yaml
metadata:
  name: report-writer
  description: "Takes a collection of research notes and writes a structured report."
```

```bash
$ kdeps ./agents/
kdeps agent> You have access to: web-search, page-reader, report-writer

> Write a report on recent advances in battery technology.
```

The LLM might:
1. Call `web-search` with "recent advances in battery technology 2024"
2. Call `page-reader` on the top 3 URLs from the search results
3. Call `report-writer` with the scraped content
4. Return the finished report

You did not write any orchestration code. You wrote three independent workflows with clear descriptions, and the LLM composed them.

## Tool Names Matter

The LLM uses `metadata.name` as the tool identifier and `metadata.description` to decide when to call it. Treat these like an API contract:

- **Names** should be lowercase, hyphenated, descriptive: `web-search`, `database-lookup`, `email-sender`
- **Descriptions** should be precise about what the tool does and when to use it: "Searches recent news articles (last 7 days). Returns title, URL, and snippet for up to 5 results."
- **Avoid overlapping descriptions.** If two tools sound like they do the same thing, the LLM will pick arbitrarily. Make each tool's purpose unambiguous.

## Built-In Agent Tools

Beyond the workflow tools you register, the agent loop provides a set of built-in tools that the LLM can call without any workflow being defined. These run directly in the agent loop and are always available.

**Web and search:**
- `web_search` -- searches the web via DuckDuckGo (no key)
- `wikipedia` -- looks up a Wikipedia article
- `web_scraper` -- fetches and extracts readable text from a URL
- `http_request` -- makes a raw HTTP request (GET/POST/PUT/DELETE/PATCH) to any API
- `serpapi_search` -- Google results via SerpAPI (requires `SERPAPI_API_KEY`)
- `exa_search` -- neural web search via Exa (requires `EXA_API_KEY` or `METAPHOR_API_KEY`)
- `perplexity_search` -- cited, up-to-date answers via Perplexity (requires `PERPLEXITY_API_KEY`)

**Files and code:**
- `read_file`, `write_file`, `edit_file`, `list_files` -- read, create, string-edit, and list local files
- `search_local` -- ripgrep search across local files (path plus query, optional glob)
- `code_search`, `code_definition`, `code_references`, `code_symbols`, `code_hover`, `code_diagnostics` -- LSP-powered code intelligence

Query tools (`web_search`, `web_scraper`, `wikipedia`, `serpapi_search`, `exa_search`, `perplexity_search`, `wolfram_alpha`) cache successful results for the process lifetime: repeating a query or URL returns instantly instead of refetching. Failed and empty lookups are not cached and are retried on the next call.

**Live tool monitor.** While a tool runs, the REPL shows a status line refreshed every second - spinner, tool name, elapsed time, and what the tool is acting on - replaced by the `... done (elapsed)` summary on completion. Every tool gets a meaningful line, not just `bash_exec`: it is seeded from the tool's arguments (the URL for `web_scraper`/`http_request`, the query for `web_search`/`sql_query`, the path for `search_local`), and streaming tools like `bash_exec` then show their latest output line as it flows. Hung tools are detected by silence: after 2 minutes without output the line warns, and after the stall timeout (default 10m, `/model tool set stall-timeout <dur>`, 0 disables) the tool is killed and the model gets an error explaining the hang so it can retry differently or background the command.

**Colored file diffs.** When the agent runs `write_file` or `edit_file`, the REPL prints a colored diff of what changed under the tool call - removed lines in red, added lines in green, plus a little context - so every edit is visible at a glance. Large diffs (like writing a whole new file) are capped. The diff is a terminal-only display; the model gets a concise result string, not the ANSI-colored text.

When a tool stalls, the default is to **auto-increase** the stall timeout by the increment (default 5m) and announce it, so a long silent-but-alive command keeps running without a prompt. `/model tool set autokill on` switches to **killing** a stalled tool at the timeout instead; autokill and auto-increase are mutually exclusive (enabling one disables the other), and both are shown by `/model tool`. `AutoToolAllocation` (tool budget) and `AutoStallAllocation` (stall time) are independent mechanisms, both on by default.

The agent loop also tracks a tool budget (`MaxToolRounds`) that limits how many tool calls the agent can make per turn. When the budget is nearly exhausted, the REPL presents: `(i)ncrease` the budget (adds 100 rounds), `(c)hange` to a specific number (`0` = unlimited), or `(g)nore` to continue. When `AutoToolAllocation` is enabled, the budget increases automatically.

**Git commit attribution.** Commits the agent creates end with a co-author trailer that names the model that wrote them, e.g. `Co-Authored-By: kdeps (deepseek/deepseek-reasoner) <noreply@kdeps.com>`. Local llamafile/GGUF models keep their HuggingFace namespace and append the runtime (`kdeps (hfuser/model gguf) <noreply@kdeps.com>`); with no model configured it falls back to `Co-Authored-By: kdeps <noreply@kdeps.com>`.

**Permission modes.** `KDEPS_PERMISSION_MODE` restricts which tools the agent may call: `read-only` (reads, searches, lookups only), `workspace-write` (adds file writes and `bash_exec`), or `danger-full-access` (no restrictions - the default). Blocked calls return a `permission denied` error to the model. Unknown tools, including workflow tools, require `workspace-write`.

**Lean mode.** `KDEPS_LEAN_MODE=true` further restricts the tool surface for CI/automation. No `bash_exec`, `web_search`, `web_scraper`, `wikipedia`, `http_request`, or any external API tools. Only file operations, code intelligence, document loading, calculator, embeddings, and transcription remain.

**Agent presets.** `KDEPS_AGENT_PRESET={audit|explain|implement}` combines lean mode with a permission mode in one flag:

| Preset | Permission mode | Tool set |
|--------|----------------|----------|
| `audit` | ReadOnly | Lean (no bash, no network) |
| `explain` | ReadOnly | Lean (no bash, no network) |
| `implement` | WorkspaceWrite | Lean + file writes |

## Agent Registries (In-Memory)

The agent loop maintains three concurrency-safe in-memory registries for lifecycle management:

### TaskRegistry

Tracks every task created by the agent loop. Each task has a unique ID (`task-N`), a status lifecycle (`created` -> `running` -> `completed`/`failed`/`stopped`), a description and prompt, and an append-only output and message transcript. Tasks can be assigned to a team and carry a heartbeat for stall detection.

Key methods: `Create`, `Get`, `List`, `ListByStatus`, `SetStatus`, `Stop`, `AppendOutput`, `AppendMessage`, `AssignTeam`, `UpdateHeartbeat`, `StalledTasks(stalledAfter)`, `Delete`.

`StalledTasks` returns running tasks whose heartbeat has expired -- the agent loop calls this periodically to detect and restart stuck work.

### TeamRegistry

Groups tasks for multi-agent coordination. Each team has a name, a list of task IDs, and a status (`created` -> `running` -> `completed` -> `deleted`). Use `AddTask` to assign a task to a team and `SetStatus` to manage team lifecycle.

### CronRegistry

Schedules recurring task creation from the `kdeps serve` process. Each cron job stores a cron expression, prompt and description templates, and tracks `LastRun`/`NextRun` times. **Cron jobs fire automatically** -- `kdeps serve` runs a background goroutine that polls `Tick()` every 60 seconds and creates tasks for any due jobs.

```bash
# In the REPL, the agent creates a cron job:
cron_create name=daily-cleanup expression="0 6 * * *" prompt="Clean up stale cache" desc="daily cleanup"

# The background goroutine fires it automatically at 06:00.
# Each firing creates a new task in GlobalTaskRegistry.
```

No manual goroutine setup needed. Start `kdeps serve path/to/agent/` and cron runs in the background. Use the CLI tools to manage jobs:

| Tool | Description |
|------|-------------|
| `cron_create` | Create a new cron job |
| `cron_list` | List all jobs with status and last/next run times |
| `cron_pause` / `cron_resume` | Pause or resume a job |
| `cron_delete` | Delete a job |

## Approval Tokens

When a tool call is denied by the permission mode, the agent can request a one-time exception via an `ApprovalToken`. Tokens let you grant time-limited, scoped overrides for specific tool+action combinations without relaxing the overall permission mode.

### How it works in practice

1. Run with `KDEPS_PERMISSION_MODE=read-only`
2. The agent attempts a write operation (e.g. `bash_exec rm -rf /tmp/cache`)
3. `PermissionEnforcer` blocks the call
4. The agent calls `approval_request(tool=bash_exec, action="rm -rf /tmp/cache")` -- creates a `pending` token
5. The agent calls `approval_list` to show you the pending token:
   ```
   Pending approval:
     apt-1: tool=bash_exec action="rm -rf /tmp/cache" status=pending
   ```
6. You run `/run approval_grant token_id=apt-1`
7. The agent retries the tool call -- `BeforeToolCall` finds the granted token via `FindMatchingGranted`, consumes it (one-time use, TTL checked), and lets the call proceed

### CLI tools

| Tool | Description |
|------|-------------|
| `approval_request` | Create a pending token for a tool+action scope |
| `approval_grant` | Grant a pending token |
| `approval_list` | List all tokens with status |
| `approval_revoke` | Revoke a granted or pending token |

### Lifecycle

- **Pending** -- created when a tool call is denied, waiting for your approval
- **Granted** -- you approved via `approval_grant`
- **Consumed** -- used for one tool call and spent
- **Expired** -- TTL elapsed without being consumed (default 5 minutes)
- **Revoked** -- manually revoked via `approval_revoke`

Scope matching supports wildcards -- an empty `Action` matches any action. `FindMatchingGranted(toolName, action, now)` is called automatically in the `BeforeToolCall` hook -- you never call it directly.

Tokens are stored in `GlobalApprovalTokenRegistry` -- a concurrency-safe in-memory singleton shared across the agent loop.

**Math and data:**
- `calculator` -- evaluates a math expression
- `wolfram_alpha` -- factual computation, math, and unit conversions via Wolfram Alpha (requires `WOLFRAM_APP_ID`)

**Database:**
- `sql_query` -- run a `SELECT` against a SQLite database (non-`SELECT` statements are rejected)
- `sql_list_tables` -- list the tables in the database
- `sql_describe_table` -- return a table's column names and types

**Memory:**
- `memory_save` -- save a fact to persistent memory (key-value, survives restarts)
- `memory_search` -- search persistent memory entries by content (case-insensitive substring match)
- `memory_delete` -- delete a specific entry from persistent memory
- `memory_list` -- list all keys in persistent memory

Memory is automatic by default. Every tool call result (from write/exec/search tools) is saved as an entry. The agent can use `[MEMORY: key] value` markers in any response to persist facts explicitly. Action sentences and file references are extracted from each turn automatically.

Entries are auto-classified into types (`prompt`, `purpose`, `progress`, `tool_result`, `result`, `status`, etc.) and linked into a directed dependency graph inlined into the `<memory>` block on every LLM call — entries in causal order, each showing its `<- parent` edge, with the current unfinished task flagged `<== RESUME` alongside its relative age (e.g. `resume: result:build (2m ago)`) so a cold model can tell if the resume point is fresh or stale. Under a token budget the block keeps the active task chain and prompt-relevant entries first, so a large memory never drops what matters. Compaction summaries are auto-captured as `checkpoint:summary` entries (and the decisions/context they carry link back to that checkpoint) preserving the project snapshot.

Memory tools access the same persistent store that `set(..., 'memory')` writes to in workflow expressions. See Chapter 15 for the full persistent memory reference.

**System:**
- `bash_exec` -- runs a shell command and streams its output to the terminal
- `bash_job_list` -- lists background jobs started from `bash_exec`
- `bash_job_wait` -- blocks until a background job finishes and returns its full output

`bash_exec` has no fixed timeout; it runs until the command completes or you intervene, and two keystrokes change a running command mid-flight. **Ctrl+C** cancels it and hands the partial output back to the model as the tool result, so the agent can react to whatever the command managed to produce. **Ctrl+Z** detaches the command as a background job: `bash_exec` returns `{"status":"backgrounded","job_id":N}` immediately, the agent keeps working, and it later calls `bash_job_wait` with that `job_id` to collect the output once the job is done. `bash_job_list` shows every background job with its status (`running`/`done`/`failed`), elapsed time, and command. This is how a long build or test run coexists with an agent that should not sit blocked on it. `KDEPS_ALLOW_BASH=false` removes all three `bash_*` tools; `KDEPS_BASH_MODE=read-only` keeps them but blocks commands that would mutate state.

**Token savings with rtk.** [rtk](https://github.com/rtk-ai/rtk) is an optional CLI proxy that compresses command output before it reaches the model. When `rtk` is on the `PATH`, `bash_exec` rewrites each command through it -- `git status` runs as `rtk git status`, `go test ./...` as `rtk go test ./...` -- and the model sees the filtered result, often 60-90% fewer tokens for the same information. Nothing needs configuring: install rtk (`brew install rtk`) and the agent loop picks it up; leave it out and commands run exactly as written.

The integration stays out of the way. If rtk is missing, too old, wedged, or has no compression for a given command, the original command runs unchanged -- rtk can never block execution. kdeps still enforces its own permission checks, so rtk acts only as a compressor here, never a gate. And because kdeps verifies rtk by behavior rather than by name, an unrelated binary that merely happens to be called `rtk` on your `PATH` is ignored instead of corrupting commands. Set `KDEPS_RTK=off` (or rtk's own `RTK_DISABLED=1`) to disable it. Only agent-mode `bash_exec` uses rtk; workflow-mode `exec` resources always keep their raw output, because downstream pipeline steps parse it.

**Documents and retrieval:**
- `load_document` -- load a PDF, DOCX, EPUB, HTML, CSV, or text file (or a directory) as text, optionally chunked for RAG
- `embedding_vectorize` -- convert text to embeddings and index it in the local embedding database
- `embedding_search` -- semantic search over the local embedding database
- `retrieve_context` -- retrieve chunks from a remote RAG endpoint; registered only when `KDEPS_RAG_BASE_URL` is set

**Reranking:**
- `cohere_rerank` -- rerank documents by relevance (requires `COHERE_API_KEY`)
- `voyageai_rerank` -- rerank via VoyageAI (requires `VOYAGEAI_API_KEY`)
- `jina_rerank` -- rerank via Jina (requires `JINA_API_KEY`)

**Audio:**
- `transcribe_audio` -- Whisper speech-to-text (requires `OPENAI_API_KEY` or `GROQ_API_KEY`; the `local` backend needs no key)

**Integrations:**
- `zapier_list_actions`, `zapier_run_action` -- discover and run Zapier NLA actions (requires `ZAPIER_NLA_API_KEY`)

**Orchestration:** these tools drive the in-memory registries covered earlier in this chapter, so the agent can manage multi-step work across turns:
- `task_create`, `task_get`, `task_list`, `task_stop`, `task_complete`, `task_append_output`, `task_assign_team` -- track discrete work items (see Agent Registries above)
- `team_create`, `team_get`, `team_list`, `team_add_task`, `team_delete` -- group tasks for multi-agent coordination
- `cron_create`, `cron_list`, `cron_pause`, `cron_resume`, `cron_delete` -- schedule recurring tasks (see CronRegistry above)
- `approval_request`, `approval_grant`, `approval_list`, `approval_revoke` -- manage one-time permission exceptions (see Approval Tokens above)

**Google AI cache management** (requires `GOOGLE_API_KEY`):
- `google_cache_create` -- creates a Google AI `CachedContent` resource from text or files, returning the cache name for use in `googleCachedContent:` in chat resources
- `google_cache_delete` -- deletes a named Google AI `CachedContent` resource
- `google_cache_list` -- lists all `CachedContent` resources in your Google AI account

The `google_cache_*` tools are useful in agent sessions where you want to pre-cache a large document or instruction set and then reference it by name across many subsequent LLM calls. The workflow to use them is:

1. Ask the agent to create a cache from your document with `google_cache_create`
2. Note the returned cache name (e.g. `cachedContents/doc-cache-abc123`)
3. Reference the name in any `chat:` resource with `googleCachedContent: "cachedContents/doc-cache-abc123"`

## REPL Slash Commands

Inside the REPL, type `/help` for the full list:

| Command | Description |
|---------|-------------|
| `/help` | Show available commands |
| `/clear` | Clear the current conversation |
| `/model <name>` | Switch model mid-session (no args = TUI picker) |
| `/model default <name>` | Persist startup model to `~/.kdeps/agent-loop-settings.yaml` |
| `/model list` | List all available models with type tags |
| `/model ps` | List running local model servers (llamafile/gguf) |
| `/model ps kill <model>` | Kill a running local model server |
| `/model ps switch <model>` | Switch to an already-running server |
| `/model hff search <query>` | Search HuggingFace for GGUF models |
| `/model hff info <repo>` | List GGUF files and sizes in a HuggingFace repo |
| `/model hff download <repo> [file]` | Download a GGUF file and register it locally |
| `/model tool [list]` | Show agent loop settings (tool rounds, retries, compaction, history caps, stall timeout, auto-allocation) |
| `/model tool set <setting> <value>` | Change a setting for this session (e.g. `rounds 80`, `retry-delay 5s`, `stall-timeout 5m`) |
| `/skills` | List loaded skills |
| `/<skill-name>` | Invoke a skill directly |
| `/compact` | Summarize history to free context |
| `/history` | Show conversation history |
| `/session list\|save\|load\|delete` | Manage saved sessions |
| `/exit` | Exit the REPL |

The REPL includes a TUI model picker (arrow keys, type-to-filter, visual tags for local vs cloud). `/model` switches models and auto-starts local servers for llamafile, GGUF, and Ollama models. Model downloads use aria2c for fast parallel downloads with resume; Ctrl+C cancels an in-flight download or model start immediately. Local servers are cleaned up on exit.

### Response Rendering and Pasting

The REPL renders the model's markdown responses in color — headings, bold, lists, tables, and syntax-highlighted code blocks. It auto-detects the terminal's color depth (truecolor, 256-color, or none) and downsamples the palette to match, so colors render correctly on terminals without 24-bit color (such as macOS Terminal.app) instead of collapsing to gray; piped output stays uncolored. When extended reasoning is on (`/thinking`), the streamed reasoning is rendered as live markdown that updates in place, shown in muted gray under a `* thinking` header and behind a dim left gutter (`│`) so the whole block reads as a distinct aside from the final answer. Inline code renders styled (by color, not literal backticks) in both the reasoning and the response.

Pasting a block of text collapses it to a single `▧` marker on the input line (the full content is held off-screen so a large paste never redraws the terminal), and the REPL submits it as **one prompt** — press Enter once and the marker expands to the full pasted text with embedded newlines preserved. Because the paste is a single character, you can edit around it: use the arrow keys or `Ctrl+A`/`Ctrl+E` to move before or after the `▧` and type there (for example, paste a stack trace and type `why does this happen: ` in front of it). This uses the terminal's bracketed-paste mode, so it works in any modern terminal, tmux, and screen.

### Setting a Default Model

To avoid passing `--model` on every invocation, use `/model default`:

```
> /model default Qwen2.5-VL-7B-Instruct-Q4_K_M
Default model saved. Next time you run kdeps it will start with this model.
```

The setting is written to `~/.kdeps/agent-loop-settings.yaml` and loaded automatically at startup. You can still override it at runtime with `--model`.

### Managing Local Model Servers

When you load a llamafile or GGUF model, kdeps starts a background `llama-server` process. The `/model ps` command shows all servers started by the current session:

```
> /model ps
PID      PORT   BACKEND      MODEL                                STATUS
--------------------------------------------------------------------
84321    8080   gguf         Qwen2.5-VL-7B-Instruct-Q4_K_M       healthy
84105    8081   llamafile    phi4                                  loading
```

To kill a server that is no longer needed:

```
> /model ps kill phi4
Killed server for: phi4
```

To switch to a server that is already running without re-downloading or restarting it:

```
> /model ps switch Qwen2.5-VL-7B-Instruct-Q4_K_M
```

After `/model <local-gguf>`, the REPL blocks with a "Loading model..." progress indicator until the completions endpoint confirms the model weights are fully loaded. Large models (7B and above) can take several minutes on first load. This prevents the network error that would otherwise appear if you sent a prompt before the model was ready.

### Discovering and Downloading Models from HuggingFace

The `/model hff` commands let you find and install GGUF models without leaving the REPL. All three sub-commands use the `HF_TOKEN` environment variable for authentication when it is set, which is required for gated models.

**Search for models:**

```
> /model hff search qwen3
Searching HuggingFace for GGUF: qwen3...
REPO                                               DOWNLOADS   LIKES
------------------------------------------------------------------------
unsloth/Qwen3-VL-2B-Instruct-GGUF                    482310     291
bartowski/Qwen3-8B-Instruct-GGUF                      310045     184
...
```

**Inspect a repo:**

```
> /model hff info unsloth/Qwen3-VL-2B-Instruct-GGUF
GGUF files in unsloth/Qwen3-VL-2B-Instruct-GGUF:
------------------------------------------------------------------------
FILE                                                       SIZE
------------------------------------------------------------------------
Qwen3-VL-2B-Instruct-Q4_K_M.gguf                          1.7GB
Qwen3-VL-2B-Instruct-Q8_0.gguf                            3.0GB
...
```

**Download a file:**

```
> /model hff download unsloth/Qwen3-VL-2B-Instruct-GGUF Qwen3-VL-2B-Instruct-Q4_K_M.gguf
Downloading unsloth/Qwen3-VL-2B-Instruct-GGUF/Qwen3-VL-2B-Instruct-Q4_K_M.gguf...
Downloaded: ~/.kdeps/models/Qwen3-VL-2B-Instruct-Q4_K_M.gguf
Registered as: Qwen3-VL-2B-Instruct-Q4_K_M
Use /model Qwen3-VL-2B-Instruct-Q4_K_M to switch to it.
```

The downloaded file is saved to `~/.kdeps/models/` and the alias is added to `~/.kdeps/gguf_versions.yaml`. From that point on, `Qwen3-VL-2B-Instruct-Q4_K_M` is a known model you can use with `/model` or `--model`.

A complete first-use workflow looks like this:

1. `/model hff search llama3` -- find repos
2. `/model hff info <repo>` -- check available quantizations and file sizes
3. `/model hff download <repo> <file>` -- download the quantization that fits your hardware
4. `/model <alias>` -- switch to it immediately

## Skills

Skills are markdown files that teach the agent how to behave in specific contexts:

```markdown
---
name: code-review
description: Guidelines for reviewing Go code
---

Always check for error handling. Prefer early returns over nested conditions.
```

Load them at startup:

```bash
$ kdeps --skill ~/.kdeps/skills/     # directory of .md files
$ kdeps --skill ./my-skill.md        # single file
```

Or place them in `~/.kdeps/skills/` (global) or `./.kdeps/skills/` (project-local) and they are discovered automatically. Invoke from the REPL with `/<skill-name>`.

## Session Persistence

Every conversation is saved as a JSONL file under `~/.kdeps/sessions/`. Resume a previous session:

```bash
$ kdeps --resume abc123def456
```

## Mixing Modes: The Two-Layer Architecture

The most effective production pattern uses both modes:

```
User prompt
    │
    ▼
Agent mode (kdeps [path])
  LLM decides which workflow tools to call
    │
    ▼
Workflow tool call
  kdeps run (DAG executes deterministically)
    │
    ▼
apiResponse.response returned to LLM
    │
    ▼
LLM synthesizes and returns answer
```

The agent layer handles the non-determinism: choosing which specialized workflow to call for a given task. The workflow layer handles deterministic execution: validating inputs, calling external services in order, returning structured output.

This separation makes each layer independently testable. You can test workflows with curl (no LLM involved). You can test the agent loop with scripted conversations.

## Limitations and Trade-offs

Agent mode's flexibility comes with costs:

**Non-determinism.** The LLM may call tools in different orders for the same prompt. This is fine for assistants; it is not acceptable for compliance workflows. Use workflow mode for anything that must be auditable.

**Latency.** Each LLM decision point adds a round-trip. A five-tool orchestration might involve five separate LLM calls plus five workflow executions. Design tools to be coarse-grained (one tool = one meaningful unit of work) to minimize round-trips.

**Cost.** Every LLM call costs tokens. In workflow mode, you control exactly how many LLM calls happen. In agent mode, the LLM might call tools more times than necessary.

**Debugging.** When an agent produces wrong results, the failure might be in the tool's pipeline, in the LLM's decision to call the wrong tool, or in the LLM's synthesis of tool results. These are different problems requiring different debugging approaches.

For most systems, agent mode is most useful at the edges of your architecture — as the interface between users and a well-defined set of deterministic tools — rather than as the core execution engine for business logic.

## Next Steps

The remainder of Part II focuses on what goes inside the tools: every resource type, how the expression engine works, and how to configure LLM backends. With that foundation, building useful tools for agent-mode composition becomes straightforward.

X> ## Exercise
X>
X> Build a two-tool agent that answers questions about either weather or math, and lets the LLM decide which tool to invoke.
X>
X> 1. Create `agents/weather/` with a minimal workflow that accepts a `location` param and returns a mock weather summary (you can hardcode it with an `exec:` resource — the goal is tool wiring, not a real weather API).
X> 2. Create `agents/math/` with a workflow that accepts an `expression` param, evaluates it via `python:`, and returns the result.
X> 3. Start agent mode: `kdeps ./agents/`
X> 4. Send prompts that require each tool:
X>    - `"What is the weather in Amsterdam?"`
X>    - `"What is 17 multiplied by 43?"`
X> 5. Confirm from the logs which tool was invoked for each prompt.
X>
X> Write a `metadata.description` for each workflow that clearly describes what it does and what input format it expects. Observe how changing the description affects which tool the LLM selects for ambiguous prompts like `"Calculate the temperature conversion from 72°F to Celsius"`.
X>
X> **Stretch goal:** Add a third tool `lookup-agent` that searches a local SQLite database. Test a prompt that requires combining results from two tools.
