# Built-in Tools

The [agent loop](/modes/agent-loop-mode) has access to a set of built-in tools that the LLM can call without any YAML configuration. Tools that require credentials are only registered when the relevant environment variable is set.

## Tool name aliases

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

## Memory tools

Always available. No environment variables required.

| Tool | Description |
|------|-------------|
| `memory_save` | Save a fact to persistent memory. Injected into every LLM call automatically. |
| `memory_search` | Search memory entries by key or value (case-insensitive substring). |
| `memory_delete` | Remove a memory entry by key. |
| `memory_list` | List all stored memory keys. |
| `memory_query` | Run an expr-lang relational query over agent state: `memory` (persistent entries), `tool_calls` (recent tool call history), `tasks` (active goal's task list). Supports `filter()`, `map()`, `join()`, `union()`. |

Memory is stored per-project at `~/.kdeps/memory/<encoded-cwd>/memory.bolt`. Facts persist across sessions and are auto-extracted from every turn — the agent can write `[MEMORY: key] value` on its own line to persist a fact without calling `memory_save`. See [Persistent Memory](/concepts/memory) for details.

The `memory_*` tools are how the *model* reads and writes memory during a turn. To inspect the store yourself from the REPL, use `/memory` (overview), `/memory list` (every entry), and `/memory search <query>` — see [REPL slash commands](/modes/agent-loop-commands).

## Identity tool

Always available. `identity_get` returns the agent's configured name, email, and address — see [Agent Identity](/configuration/advanced#agent-identity) for how to set one. Returns "No identity configured for this agent." when unset. Never returns account credentials, even if configured; a model that can read a password can leak it in its own output.

## Shell execution

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

### Token savings with rtk (optional)

[rtk](https://github.com/rtk-ai/rtk) is a CLI proxy that compresses command output before it reaches the LLM. `git status` costs ~300 tokens; `rtk git status` costs ~60 for the same information. If rtk is installed, `bash_exec` uses it automatically — nothing to configure.

```text
LLM calls bash_exec("go test ./...")
  -> kdeps asks: rtk rewrite "go test ./..."
  -> rtk answers: rtk go test ./...
  -> kdeps runs the rewritten command
  -> LLM sees filtered output (up to 90% fewer tokens)
```

Install it with `brew install rtk`, or skip it — kdeps runs your commands unchanged when rtk is absent.

| Env var | Effect |
|---------|--------|
| _(none)_ | Auto-detect. rtk is used when it is on `PATH` and passes verification. |
| `KDEPS_RTK=off` | Never use rtk, even if installed. |
| `RTK_DISABLED=1` | Also honored. rtk's own escape hatch, so one variable turns it off everywhere. |

What this does **not** change:

- **Your commands still run.** If rtk is missing, too old, wedged, or has no compression for a command, kdeps runs the original. rtk can never block execution.
- **Permissions are unaffected.** kdeps gates shell commands itself. rtk is only a compressor here, so its own permission verdicts are ignored rather than double-gating you.
- **Workflow mode is untouched.** Only agent loop `bash_exec` uses rtk. Workflow `exec` resources keep raw output, because pipelines parse it downstream.

::: tip Verifying the right rtk
An unrelated crate on crates.io is also named `rtk`. kdeps does not trust the name — it verifies the binary by behavior, so an impostor on your `PATH` is ignored rather than producing broken commands. Check yours with `rtk gain`: it works on the real one.
:::

## File operations

Always available. No environment variables required.

| Tool | Description |
|------|-------------|
| `read_file` | Read file contents (plain text, plus PDF/DOCX/EPUB/RTF/ODT extraction) |
| `write_file` | Write or overwrite a file |
| `edit_file` | Apply a unified diff to a file |
| `list_files` | List directory contents |
| `md5_file` | Compute a file's MD5 hash -- cheap way to check whether content actually changed |
| `tail_file` | Read the last N lines of a file without loading the whole thing |

`write_file` and `edit_file` print a **colored diff** of what changed under the tool call - removed lines in red, added lines in green, with a couple of context lines - so you can see every change the agent makes at a glance. Large diffs (e.g. writing a whole new file) are capped. The diff is shown in the terminal only; the model receives a concise result, not the ANSI-colored text.

## Web and search

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

When a tool stalls, the default is **auto-increase**: the stall timeout is bumped by the increment (default 5m) and the bump is announced (`[Auto-stall allocation: stall timeout increased by 5m. New timeout: 15m.]`), so a long silent-but-alive command keeps running without a prompt. This is on by default.

Two other modes are available via `/model tool set autokill <on|off>` (autokill and auto-increase are mutually exclusive — enabling one disables the other):

- `autokill on` — a stalled tool is **killed** at the stall timeout (no increase, no prompt), and the model gets a structured error so it can retry differently.
- `autokill off` — the default auto-increase-and-announce behavior.

To be prompted interactively instead, turn both off in config; the REPL then offers `(i)ncrease` / `(k)ill` when a tool stalls.

Tools marked "cached" memoize successful results for the lifetime of the agent process: repeating the same query or URL returns the cached copy instantly instead of refetching. Failed and empty lookups are not cached, so they are retried on the next call. `wolfram_alpha` results are cached the same way.

## Permission modes

Set `KDEPS_PERMISSION_MODE` to restrict which tools the agent may call:

```bash
KDEPS_PERMISSION_MODE=read-only ./kdeps          # reads, searches, lookups only
KDEPS_PERMISSION_MODE=workspace-write ./kdeps    # adds file writes and bash_exec
KDEPS_PERMISSION_MODE=danger-full-access ./kdeps # no restrictions (default)
```

Blocked calls return a `permission denied` tool error to the model, which explains the restriction instead of executing. Tools not in the built-in policy - including workflow, component, and agency tools - require `workspace-write`, so `read-only` blocks anything that could mutate state.

## Git commit attribution

Commits the agent creates carry a co-author trailer naming kdeps and the model that wrote them. Switching models mid-session with `/model` is normal, so the trailer records which one was actually driving:

```text
Co-Authored-By: kdeps (deepseek/deepseek-reasoner) <noreply@kdeps.com>
```

How the model is named depends on where it runs:

| Model | Trailer |
|-------|---------|
| Cloud provider | `kdeps (deepseek/deepseek-reasoner) <noreply@kdeps.com>` |
| Cloud provider | `kdeps (openai/gpt-4o-mini) <noreply@kdeps.com>` |
| Ollama | `kdeps (ollama/llama3.2) <noreply@kdeps.com>` |
| llamafile | `kdeps (hfuser/gemma4-2-9b llamafile) <noreply@kdeps.com>` |
| GGUF | `kdeps (hfuser/gemma4-2-9b gguf) <noreply@kdeps.com>` |

Cloud and Ollama models are namespaced by their provider (`provider/model`). Local llamafile and GGUF models already carry their HuggingFace namespace in the name, so the runtime is appended instead — the same repo is often published as both, and the name alone cannot tell them apart.

With no model configured, the trailer falls back to `Co-Authored-By: kdeps <noreply@kdeps.com>`.

A configured [identity](/configuration/advanced#agent-identity) takes priority over all of the above: with `identity.name`/`identity.email` set, the trailer becomes `Co-Authored-By: Sales Bot <sales-bot@example.com>` instead of naming the model.

## Lean mode

The full tool catalog (~55 tools — `bash_exec`, `web_search`, `web_scraper`, `wikipedia`, `http_request`, external API tools, plus the lean set below) is **on by default for every session**. Trim it down when you want less prompt weight (each tool costs tokens twice — once as a native tool schema, once as prose in the tool-use guidance) or a restricted surface for CI/automation:

```
/tools lean     # this session only, switch back any time
/tools full     # switch back
/tools          # show current mode and tool count
```

The choice persists automatically across sessions — no flag needed — the same way `/model tool set` settings do. Lean mode keeps `read_file`, `write_file`, `edit_file`, `list_files`, `code_search`, `code_definition`, `code_references`, `code_symbols`, `code_hover`, `code_diagnostics`, `search_local`, `load_document`, `calculator`, `embedding_vectorize`, `embedding_search`, `transcribe_audio` (~16 tools).

`KDEPS_LEAN_MODE`/`KDEPS_AGENT_PRESET` (below) start a session already in lean mode and take priority over the persisted `/tools` choice.

## Agent presets

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

## Computation

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `calculator` | (none) | Evaluate math expressions |
| `wolfram_alpha` | `WOLFRAM_APP_ID` | Wolfram Alpha queries |

## Data and SQL

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `sql_list_tables` | `KDEPS_SQL_DB_PATH` or connection config | List tables in a database |
| `sql_describe_table` | same | Describe a table's columns and types |
| `sql_query` | same | Execute a SELECT query |

## Embeddings and reranking

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `embedding_vectorize` | (none) | Convert text to embeddings and index it in the local embedding DB |
| `embedding_search` | (none) | Semantic search over the local embedding DB |
| `retrieve_context` | `KDEPS_RAG_BASE_URL` | Retrieve chunks from a remote RAG endpoint (only registered when the URL is set) |
| `cohere_rerank` | `COHERE_API_KEY` | Rerank results using Cohere |
| `voyageai_rerank` | `VOYAGEAI_API_KEY` | Rerank results using VoyageAI |
| `jina_rerank` | `JINA_API_KEY` | Rerank results using Jina |

## Actions and integrations

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `zapier_list_actions` | `ZAPIER_NLA_API_KEY` | List available Zapier NLA actions |
| `zapier_run_action` | `ZAPIER_NLA_API_KEY` | Execute a Zapier NLA action |
| `google_cache_create` | (Google credentials) | Create a Google AI cached content object |
| `google_cache_list` | (Google credentials) | List Google AI cached content objects |
| `google_cache_delete` | (Google credentials) | Delete a Google AI cached content object |

## Resource-backed tools

These always-on tools invoke the corresponding kdeps executor directly:

| Tool | Description |
|------|-------------|
| `http_request` | Make an HTTP request (GET/POST/PUT/DELETE/PATCH) |
| `search_local` | Search the local document index |
| `transcribe_audio` | Transcribe an audio file (OpenAI, Groq, a local HTTP server, or offline via whisper-cpp) |
| `ocr_image` | Extract text from an image via tesseract (local, no API key) |
| `load_document` | Load and extract text from a document |

## See Also

- [Agent Loop Mode](/modes/agent-loop-mode) -- overview and starting the REPL
- [REPL Slash Commands](/modes/agent-loop-commands) -- full command reference
- [Agent Registries](/modes/agent-loop-registries) -- task_*/team_*/cron_* tools for multi-agent coordination
