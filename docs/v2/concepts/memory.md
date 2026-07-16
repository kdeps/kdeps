# Persistent Memory

Persistent memory lets the agent store and recall facts across sessions. Unlike [session storage](/configuration/session) (which is request-scoped and survives restarts), persistent memory is **project-scoped** — facts saved in one session are available in future sessions for the same project.

## How it works

Memory is stored as a JSONL file at `~/.kdeps/memory/<encoded-cwd>/memory.jsonl`. Each entry has a key, value, type, timestamps, and optional references to other entries for graph-based relationship tracking.

The memory store is injected into every LLM call automatically as a single graph-ordered `<memory>` block in the system prompt: entries appear in causal order (a parent always before the children that reference it), each value is shown inline with its parent edges, and the newest unfinished task is flagged so a model resuming after an orchestrator model switch knows exactly where to continue.

## Built-in memory tools

The agent has four LLM-callable tools for interacting with persistent memory:

| Tool | Description |
|------|-------------|
| `memory_save` | Save a fact with a key and value. Keys should be short and descriptive. |
| `memory_search` | Search entries by key or value (case-insensitive substring match). |
| `memory_delete` | Remove an entry by key. |
| `memory_list` | List all stored keys (use `memory_search` to find content). |

### memory_save

Creates or updates a memory entry. The entry is persisted immediately to disk.

```json
{
  "name": "memory_save",
  "parameters": {
    "key": "project_name",
    "value": "kdeps — Go module github.com/kdeps/kdeps/v2"
  }
}
```

### memory_search

Finds entries where the key or value contains the query string (case-insensitive).

```json
{
  "name": "memory_search",
  "parameters": {
    "query": "project"
  }
}
```

Returns matching entries as formatted text:

```
Found 2 memory entries:
- project_name: kdeps — Go module github.com/kdeps/kdeps/v2
- project_structure: Monorepo layout: cmd/, pkg/ (25 packages), docs/, tests/
```

### memory_delete

Removes a single entry by key.

```json
{
  "name": "memory_delete",
  "parameters": {
    "key": "stale_fact"
  }
}
```

### memory_list

Returns all stored keys (no content). Use `memory_search` to find specific entries.

```json
{
  "name": "memory_list"
}
```

## Auto-extraction

The agent loop automatically extracts facts from every turn without an explicit `memory_save` call. Three extraction mechanisms run after each turn:

### 1. Explicit markers

The agent can write `[MEMORY: key] value` on its own line in any response. The extractor captures these as memory entries:

```
[MEMORY: project_name] kdeps — Go module github.com/kdeps/kdeps/v2
```

### 2. Action sentences

The extractor captures the first action sentence from the assistant response as a `last_action` entry:

```
"Added rate limiting middleware to the HTTP handler"
→ memory entry: last_action = "rate limiting middleware to the HTTP handler"
```

### 3. File references

File paths mentioned in edit/create/read operations are captured as a `last_files` entry:

```
"Edited src/api/handler.go and src/api/middleware.go"
→ memory entry: last_files = "src/api/handler.go, src/api/middleware.go"
```

### 4. Tool results

Every tool call result is automatically saved as a memory entry with a key derived from the tool name and output. This lets the LLM search for past tool results.

## Memory entry types

Entries are auto-classified by key pattern:

| Type | Key patterns | Description |
|------|-------------|-------------|
| `prompt` | `prompt`, `goal`, `task` | User goals and task descriptions |
| `purpose` | `purpose`, `why`, `reason` | Rationale for decisions |
| `progress` | `progress`, `wip`, `in_progress` | Work in progress tracking |
| `result` | `result`, `output`, `done` | Completed work results |
| `status` | `status`, `state` | Current state information |
| `tool_result` | `tool:*` | Tool call outputs |
| `fact` | (default for unknown) | General facts |
| `decision` | `decision`, `decided` | Design decisions |
| `preference` | `preference`, `prefer`, `like` | User preferences |
| `context` | `context`, `env`, `config` | Environment context |
| `file` | `file`, `path`, `dir`, `last_files` | File references |
| `action` | `last_action` | Last action taken |
| `error` | `error`, `fail`, `bug` | Errors and failures |
| `note` | (fallback) | Uncategorized entries |

## Memory graph

Entries are automatically linked into a directed graph based on their types. The graph follows this dependency chain:

```
prompt → purpose → progress → tool_result → result → status
                                 ↓
                            action, error, file, decision, fact, note
```

Rather than a separate diagram, the graph is **inlined into the `<memory>` block**: entries are ordered so each parent comes before the children that reference it, and every entry shows its parent edge with `<- parent`. The LLM reads the workflow as one chain instead of cross-referencing a separate arrow list. Each entry is rendered on exactly one line — a multiline value (tool output, a captured section) has its newlines collapsed to ` / ` so it never breaks the one-entry-per-line reading.

## Prompt injection

On every turn the memory store injects one graph-ordered `<memory>` block: a legend, a one-line orientation map (entry counts by type + the resume target with its **relative age**), the entries in topological order (parents first) with their values and `<- parent` edges inline, the newest unfinished `progress`/`result`/`status` entry marked `<== RESUME`, and that node's downstream (reverse) dependencies:

```
<memory>
Legend: workflow chain, parents before children. "key [type]: value"; "<- P" = derived from P; "(same as K)" = repeats K's fact; "<== RESUME" = continue here.
map: 1 prompt, 1 tool_result, 1 result | resume: result:build (2m ago)
prompt:build [prompt]: Add /users endpoint
tool:write_users [tool_result]: wrote handlers/users.go  <- prompt:build
result:build [result]: compiles; tests pending  <- tool:write_users  <== RESUME
</memory>
```

The `(2m ago)` hint on the resume target is a coarse relative age (`just now` / `Nm` / `Nh` / `Nd ago`). A model resuming after an orchestrator model switch uses it to judge whether the resume point is still fresh or likely stale enough to re-verify before continuing.

The block is truncated to a token budget. Truncation is prioritized, not just oldest-first: the **active task chain** (the resume node and its nearest ancestors), entries **relevant to the current prompt** (matched on significant prompt words at word boundaries — including 3-character technical terms like `api`, `sql`, or `git`, and their plurals, but never a mid-word substring like `api` inside `capital`), and the **newest unresolved error** are always kept, so a large memory never drops where you are, what you just asked about, or a failure you should not repeat — unrelated older entries drop first. Edges to dropped entries are omitted so no arrow dangles.

The orientation map also names the most recent unresolved `error` entry (e.g. `| error: error:migration`) so a resuming model is reminded of a known failure up front. An error whose value reads as handled (`resolved`, `fixed`, `closed`, ...) is not surfaced.

When two entries carry the same fact (compared case- and whitespace-insensitively), the later one is flagged `(same as <first-key>)` instead of being repeated as if it were independent evidence — so a cold model reads the fact once and treats the rest as copies (possibly stale), not corroboration. Nothing is dropped by this flag, so graph edges stay intact; short common values (e.g. `done`) are never flagged.

The agent also receives a rule: "Check memory first. Before taking ANY action, use `memory_search` and `memory_list` to see what is already known about the task."

## Compaction integration

When the agent runs `/compact` (summarizes and clears conversation history), the `AutoCapture` method extracts structured sections from the summary:

- `## Key Decisions` — saved as `decision` type entries
- `## Critical Context` — saved as `context` type entries

This preserves important information across compaction boundaries.

## Checkpoint summaries

After every compaction, a `checkpoint:summary` entry is saved containing the condensed Goal, Progress, Key Decisions, and Critical Context sections. This provides a running project snapshot that persists across sessions.

## Session persistence

The agent's full LLM config (model, backend, base URL) is saved to `session:config` on startup and after every `/model` switch. On the next run, the config is restored automatically — you pick up right where you left off.

Additionally, the working directory is saved on start (`session:started`) and resume (`session:resumed`) so the agent always knows where it is.

## Workflow mode

Memory tools work in both agent mode and workflow mode. In workflow mode, the store is lazy-initialized on first use via `GetOrCreateMemoryStore()`. No Loop required — memory is available to any resource or tool.

## Tool result filtering

To prevent memory bloat, only write/exec/search tools produce memory entries. Read-only lookups (`read_file`, `list_files`, `search_local`) are filtered out. Each tool type is capped at 20 entries — the oldest are auto-deleted when the cap is reached. A tool result (or any captured section) longer than its store limit is cut and marked with a trailing `...`, so a model reads it as a fragment rather than mistaking a mid-text cut for the complete output. The cut is backed off to the nearest character boundary, so a multibyte character (e.g. non-Latin text or an emoji) is never split.

Auto-extracted **low-signal** entries (types `note` and `fact`) are also globally capped at 50 combined — the noisiest, least-structural entries that accumulate fastest over a long-lived project. When the cap is exceeded, the oldest are pruned on write. Structural entries (`prompt`, `purpose`, `progress`, `result`, `status`, `decision`, `context`, `tool_result`, ...) are never pruned by this cap, so the workflow chain and resume point are always preserved.

## Configuration

Memory is enabled by default when the agent loop starts. No YAML configuration is needed. The store is created at `~/.kdeps/memory/<encoded-cwd>/memory.jsonl` where `<encoded-cwd>` is a sanitized version of the current working directory path.

To disable memory, do not pass a `MemoryStore` to the agent config.

## See Also

- [Session Configuration](/configuration/session) — request-scoped persistent storage
- [Agent Loop Mode](/modes/agent-loop-mode) — how the agent loop works
- [Expression Functions](/reference/expression-functions-reference) — `set()` and `get()` for session/memory
