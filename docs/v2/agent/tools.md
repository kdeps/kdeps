# Built-in tools

The [agent loop](/agent/) has access to a set of built-in tools that the LLM can call without any YAML configuration. Tools that require credentials are only registered when the relevant environment variable is set.

*Applies to agent mode.*

## Fenced tools only, and narration

Two instructions go into the system preamble for **every** model whenever tools are registered:

- **Use the fenced kdeps tools, not a built-in sandbox.** Every capability the model has here is a kdeps tool - including `bash_exec` and the file tools. A model trained with a code-interpreter / "run code" / "analysis" habit will otherwise act against its own empty `/mnt/data`-style environment and conclude it "cannot access" anything. The rule is restated in one line on every turn after the first. M365 Copilot gets an extra, stronger version of this on top (its habit is the most persistent). The guidance also spells out how simple a call is, with worked examples: on a backend with no native tool channel, one matched `<invoke>` block does it -

  ```
  <invoke name="read_file">
  <parameter name="file_path">cmd/serve.go</parameter>
  </invoke>
  ```

  and the runtime hands back the real output. Backends with a native tool channel (Anthropic, OpenAI) just use that; kdeps also recovers an `<invoke>` / `<tool_call>` block written as text if a native model emits one anyway.
- **Simulated sandbox sessions are caught, and escalated.** When a round makes no tool call but the text reads like a failed code-interpreter session - `NO CONTENT AVAILABLE`, `/mnt/data`, an "expired" or "reset" session, "cannot access the filesystem" - the loop treats it as a hallucination (kdeps never emits those strings and nothing ran). A model that repeats the same hallucination a second time within one turn gets a second, sharper nudge instead of the turn silently accepting the repeat as a real answer - each corrective nudge (sandbox, fabricated `<tool_response>`, silent round) fires up to twice per turn, not once. If the model still hasn't made a real call after both nudges, the turn does not settle on the fake transcript unflagged: in goal mode the active task is recorded failed, not done; otherwise the returned text is prefixed with a clear "this describes a sandbox that doesn't exist here, treat it as unverified" banner (the model's words are kept, just flagged). Separately, once a session has produced even one such hallucination, every later turn resends the full fenced-tools guidance (with the worked `<invoke>` examples) instead of the one-line reminder - a model that has shown this failure mode gets more reinforcement, not less.
- **Narrate before each tool call.** The model is asked to say in one present-tense sentence what it is about to do ("Reading config.yaml to check the timeout.") before every call. That line is printed to the terminal - without it the loop is silent between actions, because the streamer writes tool-round output to an internal buffer. Suppressed when `StreamFinalOnly` is set.

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

Memory is stored per-project at `~/.kdeps/memory/<encoded-cwd>/memory.bolt`. Facts persist across sessions and are auto-extracted from every turn - the agent can write `[MEMORY: key] value` on its own line to persist a fact without calling `memory_save`. See [Persistent memory](/agent/memory) for details.

The `memory_*` tools are how the *model* reads and writes memory during a turn. To inspect the store yourself from the REPL, use `/memory` (overview), `/memory list` (every entry), and `/memory search <query>` - see [REPL slash commands](/agent/commands).

## Identity tool

Always available. `identity_get` returns the agent's configured name, email, and address - see [Agent Identity](/reference/advanced-config#agent-identity) for how to set one. Returns "No identity configured for this agent." when unset. Never returns account credentials, even if configured; a model that can read a password can leak it in its own output.

## Shell execution

`bash_exec` runs any shell command and streams output to the terminal, with Ctrl+C to cancel, Ctrl+Z to background it, and companion `bash_job_list`/`bash_job_wait` tools. If [rtk](https://github.com/rtk-ai/rtk) is installed, output is compressed automatically before it reaches the LLM (up to 90% fewer tokens).

See [Shell Execution](/agent/shell) for the full keyboard-shortcut and rtk reference.

## File operations

Always available. No environment variables required.

| Tool | Description |
|------|-------------|
| `read_file` | Read file contents with a 1-based line number on every line (plain text, plus PDF/DOCX/EPUB/RTF/ODT extraction) |
| `write_file` | Write or overwrite a file |
| `edit_file` | `command`-dispatched editor: `view`, `str_replace`, `insert`, `undo_edit` |
| `list_files` | List directory contents |
| `md5_file` | Compute a file's MD5 hash - cheap way to check whether content actually changed |
| `tail_file` | Read the last N lines of a file without loading the whole thing |

Every line `read_file` returns is prefixed with its 1-based line number (`  42⇥code`) - those numbers are what `edit_file`'s `insert` and `view` commands point at.

`read_file`, `tail_file`, and `md5_file` treat `file_path` as optional: omit it and the tool operates on the file most recently read, edited, or written this session. This covers the common slip where the model means "the file I was just looking at" and calls `read_file` with only `offset`/`limit`. `write_file` and `edit_file` always require an explicit path.

`write_file` and `edit_file` print a **colored diff** of what changed under the tool call - removed lines in red, added lines in green, with a couple of context lines - so you can see every change the agent makes at a glance. Large diffs (e.g. writing a whole new file) are capped. The diff is shown in the terminal only; the model receives a concise result, not the ANSI-colored text.

### edit_file - four commands

`edit_file` takes a `command`. There is no line-range mode and no fuzzy
matching: the reliable primitives are an exact string swap and a line insert,
each verified.

**`view`** - `edit_file` with `command: view` and a `file_path` prints the file
with a 1-based line number on every line (same as `read_file`). Pass
`view_range: [start, end]` (1-based inclusive, `end` `-1` = to end of file) for
a slice. A `view` also satisfies the read-before-edit requirement below.

**`str_replace`** - pass `old_str` (the exact current text) and `new_str`.
`old_str` must match the file **byte-for-byte** - indentation and all - and
appear **exactly once**. No looser matching:

- 0 matches -> `old_str did not appear verbatim` (copy it from a `view`).
- 2+ matches -> the error lists every line the text starts on; add surrounding
  lines to make it unique.
- 1 match -> the swap is written and the result is a **numbered snippet of the
  changed region** so you can check the edit landed where you meant.

**`insert`** - pass `insert_line` (0 = before the first line, N = after line N)
and `new_str`. Returns the same numbered snippet.

**`undo_edit`** - reverts the last `str_replace`/`insert` on that file (a
per-file history kept for the session).

**Read before you edit.** `str_replace` and `insert` refuse to touch a file
that was not read this turn (`read_file`, or `edit_file command: view`) -
editing a file you have not looked at is how a change lands in the wrong place.
A file you just wrote with `write_file`, or just edited, counts as read.
The file's line-ending style is preserved on write.

### Failed tool calls are fed back to the model

Every tool failure is returned to the model as `{"error": ...}` **plus a
`[TOOL FAILED]` banner** ("nothing changed - fix the cause and retry, or say it
failed"), so a skimmed error is hard to miss. The turn will not end on a "done"
claim made right after a work tool failed: the loop pushes back for a real
success or an honest "it failed" - up to twice, with the second push-back
reading as a repeat, same bound as the other corrective nudges. If the model
still hasn't acknowledged the failure after both, the turn does not settle on
the bald claim unflagged: the response is followed by a notice naming the
tool and its error, so the claim is kept but clearly marked unresolved. An
honest admission at any point (in whatever words) is accepted immediately,
no nudge needed. With a goal active, `task_complete` on such a task is
refused, and a prose "done" records the task **failed**, not done.

Loop-generated notices - goal transitions, budget changes, forced task failures,
context-window trims - are injected into the model's context as `[kdeps] ...`
messages, since the human at the REPL cannot act on them but the model can
adjust.

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

While any tool runs, the REPL shows a live status line and detects hangs via a stall timeout - see [Tool Execution Monitoring](/agent/monitoring) for the full mechanics.

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

A configured [identity](/reference/advanced-config#agent-identity) takes priority over all of the above: with `identity.name`/`identity.email` set, the trailer becomes `Co-Authored-By: Sales Bot <sales-bot@example.com>` instead of naming the model.

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

- [Agent mode](/agent/) - overview and starting the REPL
- [REPL slash commands](/agent/commands) - full command reference
- [Shell execution](/agent/shell) - bash_exec keyboard shortcuts and rtk
- [Tool execution monitoring](/agent/monitoring) - status lines and stall detection
- [Agent registries](/agent/registries) - task_*/team_*/cron_* tools for multi-agent coordination
