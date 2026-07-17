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
| `/model tool [list]` | Show agent loop settings: tool rounds, retries, retry delay, compaction, history caps, stall timeout, auto-allocation |
| `/model tool set <setting> <value>` | Change a setting for this session, e.g. `set rounds 80` (`0` = unlimited), `set compact-threshold 40k`, `set retry-delay 5s`, `set stall-timeout 5m` |
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
| `/context` | Show current context window size |
| `/context <size>` | Set context window size (e.g. `32768` or `32k`); restarts local model servers with the new `--ctx-size` |
| `/settings` | Open the tool/skill selector |
| `/exit` | Exit the REPL |
| `! <cmd>` | Run a shell command; the output becomes an agent turn - the model responds and can act on it (e.g. `!make lint` -> the model fixes the findings) |
| `!! <cmd>` | Run a shell command silently - no LLM turn, nothing added to context |

## Local model management

### Switching models

`/model <name>` switches models mid-session. For local backends (`file`, `gguf`), the REPL downloads and starts the server if it isn't already running, then shows a progress display until the completions endpoint is accepting requests — the first prompt after the switch never gets a "network error" while weights load.

```
/model qwen3.5-4b                     # switch to a known alias
/model default qwen3.5-4b             # save as default startup model
/model default                        # show the current default
```

The default model is persisted to `~/.kdeps/agent-loop-settings.yaml` and loaded automatically at startup when `--model` is not passed.

### Registering a model by URL

`/model <url>` registers a custom model and switches to it. The URL kind is detected automatically:

```bash
# Direct GGUF or llamafile file - downloaded immediately, then served locally
/model https://huggingface.co/user/repo/resolve/main/Qwen2.5-7B-Q4_K_M.gguf
/model https://example.com/rocket-3b.Q4_K_M.llamafile

# Any other http(s) URL is treated as an OpenAI-compatible endpoint
/model http://localhost:1234/v1          # LM Studio / llama.cpp server
/model https://api.together.xyz/v1       # a hosted compat provider
```

Each registered model gets a memorable, kind-prefixed ID so it's easy to recall and retype next time:

- `.gguf` URL -> `gguf-<filename>` (e.g. `gguf-Qwen2.5-7B-Q4_K_M`)
- `.llamafile` URL -> `llamafile-<filename>`
- OpenAI-compatible endpoint -> `api-<host>` (e.g. `api-localhost-1234`)

A collision with an existing model gets the next free `-2`, `-3`, ... suffix, so re-registering never overwrites.

Registered models persist and keep appearing in `/model` and `/model <tab>`:

- `.gguf` / `.llamafile` URLs are added to `~/.kdeps/gguf_versions.yaml` / `llamafile_versions.yaml` (downloaded on registration).
- OpenAI-compatible endpoints are saved to `~/.kdeps/agent-loop-settings.yaml`. No API key is stored; if the endpoint needs one, set `OPENAI_API_KEY` (or `KDEPS_CUSTOM_API_KEY`) in your environment.

### Favorite models

Star models you use often so they lead the `/model` and `/model <tab>` lists and persist across sessions:

```bash
/model favorite gpt-4o          # star it (also: /model fav, /model star)
/model favorite gguf-my-model
/model unfavorite gpt-4o        # remove the star (also: /model unfav)
```

Favorites are saved to `~/.kdeps/agent-loop-settings.yaml`, shown first (marked `★`) with no text typed, and remain selectable even if the model is a cloud model or a not-yet-downloaded alias.

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

## Context window size

`/context` shows or changes the context window size for the current model. The effect depends on the backend:

| Backend | Effect |
|---------|--------|
| `file` (llamafile) | Kills the running server and restarts it with `--ctx-size <n>` |
| `gguf` (llama-server) | Kills the running server and restarts it with `--ctx-size <n>` |
| `ollama` | Sets `num_ctx` on the next request - no restart needed |
| Cloud (openai, anthropic, etc.) | No effect - context size is managed server-side |

```
/context              # show current size (e.g. "Context window: 4096 tokens")
/context 32768        # set to 32K
/context 128k         # shorthand - equivalent to 131072
```

You can also set the default at startup with the `KDEPS_GGUF_CTX_SIZE` (gguf) or `KDEPS_LLAMAFILE_CTX_SIZE` (file) environment variables.

In resource YAML, set `contextSize:` on any `chat:` block to override per-call:

```yaml
resources:
  - action: analyze
    chat:
      model: llama3.2
      contextSize: 32768   # restarts the server with this size if backend is file/gguf
      prompt: |
        Summarize: $request.body.text
```

For Ollama only, `ollamaNumCtx:` is also accepted and takes precedence over `contextSize:`.

## Turn-complete alert

When a turn takes a while (a long research loop, a slow local model), the REPL rings the terminal and posts a desktop notification once the response is ready, so you can step away and come back when it beeps:

- The terminal **bell** marks the tab/window as having activity in most terminals, tmux, and screen.
- An **OSC 9** desktop notification (`kdeps: response ready`) appears in terminals that support it (iTerm2, WezTerm, kitty); it is silently ignored elsewhere.

Only turns longer than a threshold alert, so quick replies stay quiet.

| Env var | Effect |
|---------|--------|
| `KDEPS_NOTIFY=off` | Disable the alert entirely |
| `KDEPS_NOTIFY_MIN=<dur>` | Minimum turn duration to alert (default `10s`; `0` = every turn) |

## Pasting multiple lines

Paste a block of text and the REPL treats it as **one prompt**, not one turn per line. The whole block collapses to a single `▧` marker on the input line (its full content is kept off-screen so a large paste never redraws the terminal); press Enter once to submit and the marker is replaced by the full pasted text, with embedded newlines preserved. This uses the terminal's bracketed-paste mode, so it works in any modern terminal, tmux, and screen.

Because the paste is a single character on the edit line, you can **edit around it**: use the arrow keys (or `Ctrl+A` / `Ctrl+E` for line start/end) to move before or after the `▧` and type text there — for example paste a stack trace and type `why does this happen: ` in front of it, then submit. Everything you type around the marker is preserved and sent with the paste as one prompt.

## Response rendering

The REPL renders the model's markdown responses — headings, bold, lists, tables, and syntax-highlighted code blocks — in color. It **auto-detects the terminal's color depth** (truecolor, 256-color, or none) and downsamples the palette to match, so colors render correctly on terminals without 24-bit color (e.g. macOS Terminal.app) instead of collapsing to gray. Output piped to a file is left uncolored.

When extended reasoning is enabled (`/thinking`), the streamed reasoning is rendered as **live markdown**, updating in place as tokens arrive, shown in muted gray beneath a `* thinking` header and behind a dim left gutter (`│`) so the whole block reads as a distinct aside from the final answer. Inline code renders styled (by color, not literal backticks) in both the reasoning and the response.

## Built-in tools

The agent has access to a set of built-in tools that the LLM can call without any YAML configuration. Tools that require credentials are only registered when the relevant environment variable is set.

### Tool name aliases

Models trained on other agent frameworks or shell habits often call tools by familiar names. Those names are aliased to the real built-in tool, so a call to `grep` runs `search_local`, `cat` runs `read_file`, `bash` runs `bash_exec`, and so on. Aliases are resolved on dispatch and do **not** appear in the advertised tool list (no duplicates for the model to choose between). Common synonym parameter keys are normalized too - `grep`'s `pattern` maps to `search_local`'s `query`, `cat`'s `path` maps to `read_file`'s `file_path`.

| Canonical tool | Example aliases |
|----------------|-----------------|
| `search_local` | `grep`, `rg`, `ripgrep`, `ag`, `search`, `search_file`, `find_in_files` |
| `read_file` | `cat`, `read`, `open`, `view`, `head`, `tail` |
| `write_file` | `write`, `create`, `create_file`, `save`, `touch` |
| `edit_file` | `edit`, `str_replace`, `replace`, `apply_patch`, `sed` |
| `list_files` | `ls`, `dir`, `list`, `tree`, `find`, `glob` |
| `bash_exec` | `bash`, `sh`, `shell`, `exec`, `run`, `cmd`, `terminal` |
| `web_search` | `google`, `web`, `search_web`, `duckduckgo` |
| `web_scraper` | `scrape`, `fetch`, `curl`, `wget`, `browse`, `read_url` |
| `http_request` | `http`, `request`, `api`, `rest` |
| `calculator` | `calc`, `compute`, `eval`, `math` |
| `code_definition` / `code_references` | `go_to_definition`, `find_references`, `usages` |
| `sql_query` / `sql_list_tables` | `sql`, `select`, `list_tables`, `describe_table` |

Aliases whose target tool is not registered (e.g. a credential-gated search) are simply not created.

### Memory tools

Always available. No environment variables required.

| Tool | Description |
|------|-------------|
| `memory_save` | Save a fact to persistent memory. Injected into every LLM call automatically. |
| `memory_search` | Search memory entries by key or value (case-insensitive substring). |
| `memory_delete` | Remove a memory entry by key. |
| `memory_list` | List all stored memory keys. |

Memory is stored per-project at `~/.kdeps/memory/<encoded-cwd>/memory.jsonl`. Facts persist across sessions and are auto-extracted from every turn — the agent can write `[MEMORY: key] value` on its own line to persist a fact without calling `memory_save`. See [Persistent Memory](/concepts/memory) for details.

### Shell execution

`bash_exec` runs any shell command and streams output to the terminal. Two keyboard shortcuts change its behavior mid-run:

| Key | Effect |
|-----|--------|
| `Ctrl+C` | Cancel the running tool. Partial output is returned to the LLM as a result so it can decide what to do next. Works for any built-in tool, not only `bash_exec`. |
| `Ctrl+Z` | Detach the process as a background job. `bash_exec` immediately returns `{"status":"backgrounded","job_id":N}` to the LLM. |

Ctrl+C is read directly from the terminal while a tool runs, so it cancels even long-running tools (e.g. a slow `search_local` or `web_scraper`) - the REPL does not rely on the terminal delivering a signal.

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

`write_file` and `edit_file` print a **colored diff** of what changed under the tool call - removed lines in red, added lines in green, with a couple of context lines - so you can see every change the agent makes at a glance. Large diffs (e.g. writing a whole new file) are capped. The diff is shown in the terminal only; the model receives a concise result, not the ANSI-colored text.

### Web and search

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `web_search` | (none -- uses DuckDuckGo) | Search the web (30s timeout, cached) |
| `wikipedia` | (none) | Fetch a Wikipedia article (30s timeout, cached) |
| `web_scraper` | (none) | Fetch and extract text from any URL (60s timeout, cached) |
| `serpapi_search` | `SERPAPI_API_KEY` | Google search via SerpAPI (30s timeout, cached) |
| `exa_search` | `EXA_API_KEY` or `METAPHOR_API_KEY` | Neural search via Exa (cached) |
| `perplexity_search` | `PERPLEXITY_API_KEY` | Search via Perplexity (30s timeout, cached) |

Web and search tools carry a hard timeout so a hung remote endpoint cannot stall the turn. Ctrl+C during any tool call cancels the in-flight request immediately and skips the round's remaining tools.

While a tool runs, the REPL shows a live monitor line - `⠴ bash_exec running (12m34s) · <latest output line>` - refreshed every second, so a long command (a full test suite, a large download) is visibly alive instead of silent. The line is replaced by the usual `... done (elapsed)` summary when the tool finishes.

Every tool gets a meaningful monitor line, not just `bash_exec`: the line is seeded with what the tool is acting on, derived from its arguments - the URL for `web_scraper`/`http_request`, the query for `web_search`/`sql_query`, the path for `search_local`, and so on (`⠴ web_scraper running (3s) · https://example.com`). Tools that stream output (like `bash_exec`) then replace the seed with their latest output line as it flows.

The same status line covers `! <cmd>` / `!! <cmd>` shell commands and `@file` ref expansion: while a bang command is silent the line shows `⠴ ! make lint running (57s)`, and any real output erases the status line first so the two never collide.

The monitor also detects hung tools. Staleness is measured by *silence*, not wall-clock time - a long build that keeps printing never trips it. After 2 minutes without output the line warns (`no output for 3m20s`); after the stall timeout (default 10 minutes of silence, tune with `/model tool set stall-timeout 5m`, `0` disables) the tool is killed and the model receives a structured error explaining the hang so it can retry with a narrower or more verbose command, or run it in the background.

When a tool stalls, the REPL presents interactive options: `(i)ncrease` the stall timeout (adds the auto-allocation increment, default 5m), `(c)hange` to a specific timeout in minutes, or `(g)nore` to kill the tool. When `AutoStallAllocation` is enabled in config, the timeout increases automatically without prompting.

Tools marked "cached" memoize successful results for the lifetime of the agent process: repeating the same query or URL returns the cached copy instantly instead of refetching. Failed and empty lookups are not cached, so they are retried on the next call. `wolfram_alpha` results are cached the same way.

### Permission modes

Set `KDEPS_PERMISSION_MODE` to restrict which tools the agent may call:

```bash
KDEPS_PERMISSION_MODE=read-only ./kdeps          # reads, searches, lookups only
KDEPS_PERMISSION_MODE=workspace-write ./kdeps    # adds file writes and bash_exec
KDEPS_PERMISSION_MODE=danger-full-access ./kdeps # no restrictions (default)
```

Blocked calls return a `permission denied` tool error to the model, which explains the restriction instead of executing. Tools not in the built-in policy - including workflow, component, and agency tools - require `workspace-write`, so `read-only` blocks anything that could mutate state.

### Git commit attribution

Commits the agent creates carry a co-author trailer naming kdeps:

```text
Co-Authored-By: kdeps <noreply@kdeps.com>
```

### Lean mode

`KDEPS_LEAN_MODE` further restricts the tool surface for CI/automation. When enabled, the agent has no `bash_exec`, `web_search`, `web_scraper`, `wikipedia`, `http_request`, or any external API tools:

```bash
KDEPS_LEAN_MODE=true ./kdeps
```

Tools available in lean mode: `read_file`, `write_file`, `edit_file`, `list_files`, `code_search`, `code_definition`, `code_references`, `code_symbols`, `code_hover`, `code_diagnostics`, `search_local`, `load_document`, `calculator`, `embedding_vectorize`, `embedding_search`, `transcribe_audio`.

### Agent presets

`KDEPS_AGENT_PRESET` combines lean mode with a permission mode in one flag for common workflows:

```bash
KDEPS_AGENT_PRESET=audit       # read-only, lean tools
KDEPS_AGENT_PRESET=explain     # read-only, lean tools
KDEPS_AGENT_PRESET=implement   # workspace-write, lean tools
```

| Preset | Permission mode | Tool set |
|--------|----------------|----------|
| `audit` | ReadOnly | Lean (no bash, no network) |
| `explain` | ReadOnly | Lean (no bash, no network) |
| `implement` | WorkspaceWrite | Lean + file writes |

`IsLeanOrPreseted()` returns `true` when either `KDEPS_LEAN_MODE` or `KDEPS_AGENT_PRESET` is set.

## Agent registries

The agent loop maintains three in-memory registries for lifecycle management:

### TaskRegistry

Tracks every task created by the agent loop. Each task has a unique ID (`task-N`), status (`created` -> `running` -> `completed`/`failed`/`stopped`), description, prompt, and an append-only output and message transcript. Tasks can be assigned to a team and carry a heartbeat for stall detection.

| Method | Description |
|--------|-------------|
| `Create(prompt, description)` | Create a new task in `created` state |
| `Get(taskID)` | Look up a task by ID |
| `List()` | All tasks, newest first |
| `ListByStatus(status)` | Filter by status |
| `SetStatus(taskID, status)` | Transition to a new status |
| `Stop(taskID)` | Set status to `stopped` |
| `AppendOutput(taskID, text)` | Append to task output |
| `AppendMessage(taskID, msg)` | Append to message transcript |
| `AssignTeam(taskID, teamID)` | Attach a team |
| `UpdateHeartbeat(taskID, alive)` | Record lane aliveness |
| `StalledTasks(stalledAfter)` | Running tasks with stale heartbeats |
| `Delete(taskID)` | Remove a task from the registry |

### TeamRegistry

Groups tasks for multi-agent coordination. Each team has a name, a list of task IDs, and a status (`created` -> `running` -> `completed` -> `deleted`).

| Method | Description |
|--------|-------------|
| `Create(name)` | Create a new team |
| `Get(teamID)` | Look up a team by ID |
| `List()` | All teams |
| `AddTask(teamID, taskID)` | Assign a task to a team |
| `SetStatus(teamID, status)` | Update team status |
| `Delete(teamID)` | Mark as deleted |

### CronRegistry

Schedules recurring task creation from the `kdeps serve` process. Each cron job stores a cron expression, prompt/description templates, and tracks last/next run times. **Cron jobs fire automatically** — the server starts a background goroutine that calls `Tick()` every 60 seconds and creates tasks for any due jobs.

| CLI tool | Description |
|----------|-------------|
| `cron_create` | Create a new cron job with expression, prompt, and description |
| `cron_list` | List all cron jobs with status, last/next run times |
| `cron_pause` / `cron_resume` | Pause or resume a cron job |
| `cron_delete` | Delete a cron job |

No manual polling or goroutine setup needed. Start `kdeps serve path/to/agent/` and cron runs in the background.

## Approval tokens

When a tool call is denied by the permission mode, the agent can request a one-time exception via an approval token. Tokens let you grant scoped overrides for specific tool+action combinations without relaxing the overall permission mode.

### How it works in practice

1. You run with `KDEPS_PERMISSION_MODE=read-only`
2. The agent attempts a write operation (e.g. `bash_exec rm -rf /tmp/cache`)
3. `PermissionEnforcer` blocks the call
4. The agent calls `approval_request(tool=bash_exec, action="rm -rf /tmp/cache")` — creates a `pending` token with scope `{ToolName:"bash_exec", Action:"rm -rf /tmp/cache"}`
5. The agent calls `approval_list` to show you the pending token:
   ```
   Pending approval:
     apt-1: tool=bash_exec action="rm -rf /tmp/cache" status=pending
   ```
6. You run `/run approval_grant token_id=apt-1`
7. The agent retries the tool call — `BeforeToolCall` finds the granted token via `FindMatchingGranted`, consumes it (one-time use), and lets the call proceed

### CLI tools

| Tool | Description |
|------|-------------|
| `approval_request` | Create a pending token for a specific tool+action scope |
| `approval_grant` | Grant a pending token (user approves) |
| `approval_list` | List all tokens with status |
| `approval_revoke` | Revoke a granted or pending token |

### Lifecycle

- **Pending** — created by the agent when a tool call is denied, waiting for your approval
- **Granted** — you approved the exception via `approval_grant`
- **Consumed** — the token was used for one tool call and is now spent
- **Expired** — TTL elapsed without being consumed (default 5 minutes)
- **Revoked** — manually revoked via `approval_revoke`

Scope matching supports wildcards: an empty `Action` matches any action. `FindMatchingGranted(toolName, action, now)` is called automatically in the `BeforeToolCall` hook — you never call it directly.

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

### Tool budget and stall timeout

The agent loop tracks a tool budget (`MaxToolRounds`) that limits how many tool calls the agent can make per turn. When the budget is nearly exhausted, the REPL presents interactive options: `(i)ncrease` the budget (adds 100 rounds), `(c)hange` to a specific number (`0` = unlimited), or `(g)nore` to continue with the current budget. When `AutoToolAllocation` is enabled in config, the budget increases automatically without prompting.

Similarly, when a tool stalls (no output for the stall timeout duration), the REPL presents: `(i)ncrease` the stall timeout (adds the auto-allocation increment, default 5m), `(c)hange` to a specific timeout in minutes, or `(g)nore` to kill the tool. When `AutoStallAllocation` is enabled, the timeout increases automatically.

Both settings can also be tuned with `/model tool set rounds <n>` and `/model tool set stall-timeout <dur>`.

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
