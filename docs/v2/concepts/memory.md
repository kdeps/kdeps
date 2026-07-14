# Persistent Memory

Persistent memory lets the agent store and recall facts across sessions. Unlike [session storage](/configuration/session) (which is request-scoped and survives restarts), persistent memory is **project-scoped** — facts saved in one session are available in future sessions for the same project.

## How it works

Memory is stored as a JSONL file at `~/.kdeps/memory/<encoded-cwd>/memory.jsonl`. Each entry has a key, value, type, timestamps, and optional references to other entries for graph-based relationship tracking.

The memory store is injected into every LLM call automatically. The agent sees a `<memory>` block in its system prompt containing known facts, plus a `<memory-graph>` block showing how entries relate to each other.

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

The graph is injected into the LLM prompt as a `<memory-graph>` block, showing how entries relate:

```
<memory-graph>
project_name -> tool:bash_exec:main
project_name -> tool:bash_exec:main -> tool:bash_exec:fafdc304_fix__sanitize_the_ter
</memory-graph>
```

This lets the LLM trace dependencies between facts — for example, seeing that a `tool_result` was produced by a specific `prompt`, or that a `decision` was informed by a `result`.

## Prompt injection

On every turn, the memory store injects two blocks into the system prompt:

1. **`<memory>`** — the most recent entries, sorted by recency, truncated to ~500 tokens
2. **`<memory-graph>`** — relationship paths between entries, truncated to ~250 tokens

The agent also receives a rule: "Check memory first. Before taking ANY action, use `memory_search` and `memory_list` to see what is already known about the task."

## Compaction integration

When the agent runs `/compact` (summarizes and clears conversation history), the `AutoCapture` method extracts structured sections from the summary:

- `## Key Decisions` — saved as `decision` type entries
- `## Critical Context` — saved as `context` type entries

This preserves important information across compaction boundaries.

## Configuration

Memory is enabled by default when the agent loop starts. No YAML configuration is needed. The store is created at `~/.kdeps/memory/<encoded-cwd>/memory.jsonl` where `<encoded-cwd>` is a sanitized version of the current working directory path.

To disable memory, do not pass a `MemoryStore` to the agent config.

## See Also

- [Session Configuration](/configuration/session) — request-scoped persistent storage
- [Agent Loop Mode](/modes/agent-loop-mode) — how the agent loop works
- [Expression Functions](/reference/expression-functions-reference) — `set()` and `get()` for session/memory
