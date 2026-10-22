# Persistent memory

Persistent memory lets the agent store and recall facts across sessions. Unlike [session storage](/workflow/sessions) (which persists across HTTP requests from the same caller), persistent memory is project-scoped - facts saved in one REPL session are available in later sessions for the same project.

Persistent memory is primarily an agent mode concept, but the memory tools also work in workflow mode (see [Workflow mode](#workflow-mode) below). For how the agent decides what to remember and what to show the model each turn, see [Memory internals](/agent/memory-internals). Memory holds the current work's short facts; the long-term story (sessions, decisions, gotchas across projects) belongs in the [knowledge vault](/agent/knowledge).

## How it works

```markdown
<!-- ~/.kdeps/notes/projects/kdeps/memory/decision-louvain.md -->
---
key: decision:louvain
type: decision
created: 2026-10-15T09:12:03Z
updated: 2026-10-15T09:12:03Z
summary: Louvain first, Leiden later.
references: ["fact:vault"]       # memory keys this entry depends on
links: ["[[fact-vault]]"]          # the same, as links the knowledge graph follows
---
Louvain first, Leiden later.
```

Because entries are vault notes, the knowledge graph indexes them and recall can surface them. Memory written before this layout (in `~/.kdeps/memory/`) is moved into the vault the first time each folder's memory loads; the old database is renamed `memory.bolt.moved`.

Memory is stored in the [knowledge vault](/agent/knowledge) as markdown notes, one per entry, in the project's `memory/` folder: `<vault>/projects/<project>/memory/` (the project whose card covers the folder, else the one kdeps creates for it, named after the git repository root or the folder). Thinking and tool-result captures go in a hidden `memory/.auto/` folder, so they stay out of the link graph. A note you write in that folder by hand is a memory entry too, keyed by its title. Each entry has a key, value, type, timestamps, and optional references to other entries for graph-based relationship tracking.

Set `KDEPS_MEMORY_GLOBAL=1` to share one memory store across every project instead - it is stored in `<vault>/shared/memory/` and every session on the machine reads and writes the same facts, regardless of working directory. Session storage (conversation history) always stays per-directory either way.

The memory store is injected into every turn automatically as a single graph-ordered `<memory>` block, rebuilt each turn and focused on that turn's request (it is not part of the cached system prompt, so memory saved mid-session reaches the next turn). Entries appear in causal order and the newest unfinished task is flagged, so a model resuming after an orchestrator model switch knows where to continue. See [Memory internals](/agent/memory-internals#prompt-injection) for the block format.

## Built-in memory tools

The agent has three LLM-callable tools for interacting with persistent memory. There is no `memory_list` tool. The graph and the recent keys are already in each turn's context; look a fact up with `memory_query` (preferred) or `memory_search` (fuzzy text match). The prompt tells the model to look memory up when the task refers to earlier work, when a fact is not already in the prompt, or on its first turn after a model switch, resume or compaction - not before every action.

| Tool | Description |
|------|-------------|
| `memory_save` | Save a fact with a key and value. Keys should be short and descriptive. |
| `memory_search` | Fuzzy text search of entries by key or value. Returns at most 20 matches, best first; each value is cut at 500 characters. |
| `memory_delete` | Remove an entry by key. |

Another tool, `memory_query`, runs relational queries (select/project/join/union) over memory plus tool-call history and task state - see [Relational query](#relational-query-memory-query) below.

## memory_save

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

## memory_search

Finds entries where a query word appears in the key or value (case-insensitive). A key hit ranks above a value-only hit. More words matched ranks above fewer. A newer update breaks a tie.

```json
{
  "name": "memory_search",
  "parameters": {
    "query": "project"
  }
}
```

Returns matching entries as formatted text. At most 20 matches are printed, best match first, and each value is cut at 500 characters (a cut value ends in `...`). When more matches exist, the header says how many were found and how many are shown, and the last line says how many were left out:

```
Found 2 memory entries:
- project_name: kdeps - Go module github.com/kdeps/kdeps/v2
- project_structure: Monorepo layout: cmd/, pkg/ (25 packages), docs/, tests/
```

```
Found 25 memory entries (showing 20):
- match-00: shared fact
... and 5 more (narrow the query to see them)
```

No memory hits falls through to a local file search of the working directory, also capped at 20 matches. The file-search JSON is cut if it exceeds the tool-result byte cap.

## memory_delete

Removes a single entry by key.

```json
{
  "name": "memory_delete",
  "parameters": {
    "key": "stale_fact"
  }
}
```

## Relational query (memory_query)

For filtering by field, combining facts across sources, or correlating past tool calls, `memory_query` runs a relational query - select/project/join/union - over two relations built from agent state:

| Relation | Fields | Source |
|---|---|---|
| `memory` | `key`, `value`, `namespace`, `type`, `references`, `createdAt`, `updatedAt` | Persistent memory entries |
| `tool_calls` | `name`, `args`, `result`, `timestamp` | Recent tool-call history (this session, most recent 200) |

The query language is [expr-lang](https://expr-lang.org/), the same engine `before:`/`after:` expressions use. `filter()`/`map()` are its own built-ins (select/project); `join()`/`union()` are added by `memory_query`:

| Operation | Function | Example |
|---|---|---|
| Select (WHERE) | `filter(relation, predicate)` | `filter(memory, .type == "error")` |
| Project (columns) | `map(relation, expr)` | `map(memory, {key: .key, value: .value})` |
| Join | `join(left, right, leftField, rightField)` | `join(tool_calls, memory, "name", "key")` |
| Union | `union(a, b)` | `union(filter(memory, .type == "error"), filter(memory, .type == "decision"))` |

String matching uses infix operators, not function calls. `contains(...)` is a parse error; put the field on the left:

| Match | Operator | Example |
|---|---|---|
| Substring | `contains` | `filter(memory, .value contains "latency")` |
| Prefix | `startsWith` | `filter(memory, .key startsWith "tool:read_file")` |
| Suffix | `endsWith` | `filter(memory, .key endsWith ":summary")` |
| Regex | `matches` | `filter(memory, .key matches "^tool:")` |
| Case-insensitive substring | `lower()` + `contains` | `filter(memory, lower(.value) contains "latency")` |
| Membership | `in` | `filter(memory, .type in ["error", "decision"])` |

`join` is an equi-join, merging rows with `left_`/`right_` prefixed field names so same-named fields never collide; no inequality/range-join or multi-field-key support today.

```json
{"name": "memory_query", "parameters": {"query": "filter(memory, .type == \"error\")", "limit": 20}}
```

The result has `rows` (capped at `limit`, default 50, max 500), `count` (total matches before capping), and `truncated` (bool). `memory_query` runs through the kdeps API and reads the in-process agent loop. An empty query returns `memory_query: query is required`. A call with no loop returns `memory_query: agent loop is not configured (memory_query is an agent-mode tool)`.

## Memory entry types

Entries are auto-classified by key pattern. The type controls where the entry sits in the [memory graph](/agent/memory-internals#memory-graph) and whether it can be pruned.

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

Memory tools work in both agent mode and workflow mode. In workflow mode, the store is lazy-initialized on first use via `GetOrCreateMemoryStore()`. No Loop required - memory is available to any resource or tool. Workflow resources can still call `memory_list()` to enumerate keys. The agent loop does not register that as a tool. `memory_query` uses that same call route and still needs the agent loop's live state. A workflow call with no loop returns `memory_query: agent loop is not configured (memory_query is an agent-mode tool)`.

## Configuration

Memory is enabled by default when the agent loop starts. No YAML configuration is needed. The store lives in `<vault>/projects/<project>/memory/` (default vault: `~/.kdeps/notes`).

## See also

- [Memory internals](/agent/memory-internals) - auto-extraction, the memory graph, prompt injection, compaction
- [Session configuration](/workflow/sessions) - HTTP session storage across requests from the same caller
- [Agent mode](/agent/) - how the agent loop works
