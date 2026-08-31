# Built-in tools

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

Memory is stored per-project at `~/.kdeps/memory/<encoded-cwd>/memory.bolt`. Facts persist across sessions and are auto-extracted from every turn - the agent can write `[MEMORY: key] value` on its own line to persist a fact without calling `memory_save`. See [Persistent Memory](/concepts/memory) for details.

The `memory_*` tools are how the *model* reads and writes memory during a turn. To inspect the store yourself from the REPL, use `/memory` (overview), `/memory list` (every entry), and `/memory search <query>` - see [REPL slash commands](/modes/agent-loop-commands).

## Identity tool

Always available. `identity_get` returns the agent's configured name, email, and address - see [Agent Identity](/configuration/advanced#agent-identity) for how to set one. Returns "No identity configured for this agent." when unset. Never returns account credentials, even if configured; a model that can read a password can leak it in its own output.

## Shell execution

`bash_exec` runs any shell command and streams output to the terminal, with Ctrl+C to cancel, Ctrl+Z to background it, and companion `bash_job_list`/`bash_job_wait` tools. If [rtk](https://github.com/rtk-ai/rtk) is installed, output is compressed automatically before it reaches the LLM (up to 90% fewer tokens).

See [Shell Execution](/modes/agent-loop-shell) for the full keyboard-shortcut and rtk reference.

## File operations

Always available. No environment variables required.

| Tool | Description |
|------|-------------|
| `read_file` | Read file contents (plain text, plus PDF/DOCX/EPUB/RTF/ODT extraction) |
| `write_file` | Write or overwrite a file |
| `edit_file` | Apply a unified diff to a file |
| `list_files` | List directory contents |
| `md5_file` | Compute a file's MD5 hash - cheap way to check whether content actually changed |
| `tail_file` | Read the last N lines of a file without loading the whole thing |

`write_file` and `edit_file` print a **colored diff** of what changed under the tool call - removed lines in red, added lines in green, with a couple of context lines - so you can see every change the agent makes at a glance. Large diffs (e.g. writing a whole new file) are capped. The diff is shown in the terminal only; the model receives a concise result, not the ANSI-colored text.

## Web and search

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `web_search` | (none - uses DuckDuckGo) | Search the web (30s timeout, cached) |
| `wikipedia` | (none) | Fetch a Wikipedia article (30s timeout, cached) |
| `web_scraper` | (none) | Fetch and extract text from any URL (60s timeout, cached) |
| `serpapi_search` | `SERPAPI_API_KEY` | Google search via SerpAPI (30s timeout, cached) |
| `exa_search` | `EXA_API_KEY` or `METAPHOR_API_KEY` | Neural search via Exa (cached) |
| `perplexity_search` | `PERPLEXITY_API_KEY` | Search via Perplexity (30s timeout, cached) |

Web and search tools carry a hard timeout so a hung remote endpoint cannot stall the turn. Ctrl+C during any tool call cancels the in-flight request immediately and skips the round's remaining tools. Tools marked "cached" memoize successful results for the process lifetime; failed/empty lookups are retried.

While any tool runs, the REPL shows a live status line and detects hangs via a stall timeout - see [Tool Execution Monitoring](/modes/agent-loop-monitoring) for the full mechanics.

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

Cloud and Ollama models are namespaced by their provider (`provider/model`). Local llamafile and GGUF models already carry their HuggingFace namespace in the name, so the runtime is appended instead - the same repo is often published as both, and the name alone cannot tell them apart.

With no model configured, the trailer falls back to `Co-Authored-By: kdeps <noreply@kdeps.com>`.

A configured [identity](/configuration/advanced#agent-identity) takes priority over all of the above: with `identity.name`/`identity.email` set, the trailer becomes `Co-Authored-By: Sales Bot <sales-bot@example.com>` instead of naming the model.

## Lean mode

The full tool catalog (~55 tools - `bash_exec`, `web_search`, `web_scraper`, `wikipedia`, `http_request`, external API tools, plus the lean set below) is **on by default for every session**. Trim it down when you want less prompt weight (each tool costs tokens twice - once as a native tool schema, once as prose in the tool-use guidance) or a restricted surface for CI/automation:

```
/tools lean     # this session only, switch back any time
/tools full     # switch back
/tools          # show current mode and tool count
```

The choice persists automatically across sessions - no flag needed - the same way `/model tool set` settings do. Lean mode keeps `read_file`, `write_file`, `edit_file`, `list_files`, `code_search`, `code_definition`, `code_references`, `code_symbols`, `code_hover`, `code_diagnostics`, `search_local`, `load_document`, `calculator`, `embedding_vectorize`, `embedding_search`, `transcribe_audio` (~16 tools).

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

## See also

- [Agent loop mode](/modes/agent-loop-mode) - overview and starting the REPL
- [REPL slash commands](/modes/agent-loop-commands) - full command reference
- [Shell execution](/modes/agent-loop-shell) - bash_exec keyboard shortcuts and rtk
- [Tool execution monitoring](/modes/agent-loop-monitoring) - status lines and stall detection
- [Agent registries](/modes/agent-loop-registries) - task_*/team_*/cron_* tools for multi-agent coordination
