# Persistent memory

Persistent memory lets the agent store and recall facts across sessions. Unlike [session storage](/configuration/session) (which is request-scoped and survives restarts), persistent memory is **project-scoped** - facts saved in one session are available in future sessions for the same project.

Persistent memory is primarily an agent mode concept, but the memory tools also work in workflow mode (see [Workflow mode](#workflow-mode) below). For how the agent decides what to remember and what to show the model each turn, see [Memory internals](/concepts/memory-internals).

## How it works

Memory is stored in a bbolt (embedded key-value) database at `~/.kdeps/memory/<encoded-cwd>/memory.bolt`. Each entry has a key, value, type, timestamps, and optional references to other entries for graph-based relationship tracking.

The memory store is injected into every LLM call automatically as a single graph-ordered `<memory>` block in the system prompt. Entries appear in causal order and the newest unfinished task is flagged, so a model resuming after an orchestrator model switch knows where to continue. See [Memory internals](/concepts/memory-internals#prompt-injection) for the block format.

## Built-in memory tools

The agent has four LLM-callable tools for interacting with persistent memory:

| Tool | Description |
|------|-------------|
| `memory_save` | Save a fact with a key and value. Keys should be short and descriptive. |
| `memory_search` | Search entries by key or value (case-insensitive substring match). |
| `memory_delete` | Remove an entry by key. |
| `memory_list` | List all stored keys (use `memory_search` to find content). |

A fifth tool, `memory_query`, runs relational queries (select/project/join/union) over memory plus tool-call history and task state - see [Relational query](#relational-query-memory-query) below.

### memory_save

Creates or updates a memory entry. The entry is persisted immediately to disk.

```json
{
  "name": "memory_save",
  "parameters": {
    "key": "project_name",
    "value": "kdeps - Go module github.com/kdeps/kdeps/v2"
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
- project_name: kdeps - Go module github.com/kdeps/kdeps/v2
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

## Relational query (memory_query)

For filtering by field, combining facts across sources, or correlating past tool calls with the task that triggered them, `memory_query` runs a relational query - select/project/join/union - over three relations built from agent state:

| Relation | Fields | Source |
|---|---|---|
| `memory` | `key`, `value`, `namespace`, `type`, `references`, `createdAt`, `updatedAt` | Persistent memory entries |
| `tool_calls` | `name`, `args`, `result`, `timestamp` | Recent tool-call history (this session, most recent 200) |
| `tasks` | `id`, `desc`, `status`, `rounds`, `note` | The active goal's task list (empty when no goal is active) |

The query language is [expr-lang](https://expr-lang.org/), the same engine `before:`/`after:` expressions use. `filter()`/`map()` are its own built-ins (select/project); `join()`/`union()` are added by `memory_query`:

| Operation | Function | Example |
|---|---|---|
| Select (WHERE) | `filter(relation, predicate)` | `filter(memory, .type == "error")` |
| Project (columns) | `map(relation, expr)` | `map(memory, {key: .key, value: .value})` |
| Join | `join(left, right, leftField, rightField)` | `join(tool_calls, memory, "name", "key")` |
| Union | `union(a, b)` | `union(filter(memory, .type == "error"), filter(memory, .type == "decision"))` |

`join` is an equi-join, merging rows with `left_`/`right_` prefixed field names so same-named fields never collide; no inequality/range-join or multi-field-key support today.

```json
{"name": "memory_query", "parameters": {"query": "filter(memory, .type == \"error\")", "limit": 20}}
```

The result has `rows` (capped at `limit`, default 50, max 500), `count` (total matches before capping), and `truncated` (bool). `memory_query` is **agent-mode only** - it reads the active `Loop`'s state directly, so workflow mode has no LLM tool-call state to query.

## Memory entry types

Entries are auto-classified by key pattern. The type controls where the entry sits in the [memory graph](/concepts/memory-internals#memory-graph) and whether it can be pruned.

| Type | Key patterns | Description |
|------|-------------|-------------|
| `prompt` | `prompt`, `goal`, `task` | User goals and task descriptions |
| `purpose` | `purpose`, `why`, `reason` | Rationale for decisions |
| `progress` | `progress`, `wip`, `in_progress` | Work in progress tracking |
| `result` | `result`, `output`, `done` | Completed work results |
| `status` | `status`, `state` | Current state information |
| `tool_result` | `tool:*` | Tool call outputs (capped at 20) |
| `thinking` | `thinking:*` | The model's reasoning text for a round, across every thinking-capable backend - searchable via `memory_search`. Capped at 20. |
| `decision` | `decision`, `decided` | Design decisions |
| `preference` | `preference`, `prefer`, `like` | User preferences |
| `context` | `context`, `env`, `config` | Environment context |
| `file` | `file`, `path`, `dir`, `last_files` | File references |
| `action` | `last_action` | Last action taken |
| `error` | `error`, `fail`, `bug` | Errors and failures |
| `note` | (default for unknown) | Uncategorized entries |
| `fact` | (not auto-assigned) | General facts; grouped with `note` as low-signal (both capped at 50 combined) |

## Workflow mode

Memory tools work in both agent mode and workflow mode. In workflow mode, the store is lazy-initialized on first use via `GetOrCreateMemoryStore()`. No Loop required - memory is available to any resource or tool. `memory_query` is the exception: it needs the agent loop's live state and is agent-mode only.

## Configuration

Memory is enabled by default when the agent loop starts. No YAML configuration is needed. The store is created at `~/.kdeps/memory/<encoded-cwd>/memory.bolt` where `<encoded-cwd>` is a sanitized version of the current working directory path. To disable memory, do not pass a `MemoryStore` to the agent config.

## See also

- [Memory internals](/concepts/memory-internals) - auto-extraction, the memory graph, prompt injection, compaction
- [Session configuration](/configuration/session) - request-scoped persistent storage
- [Agent loop mode](/modes/agent-loop-mode) - how the agent loop works
