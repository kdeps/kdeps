# Persistent memory

Persistent memory stores and recalls facts across agent sessions for one project. [Session storage](/workflow/sessions) is different: it persists across HTTP requests from one caller.

Persistent memory is primarily an agent mode concept, but the memory tools also work in workflow mode (see [Workflow mode](#workflow-mode) below). For how the agent decides what to remember and what to show the model each turn, see [Memory internals](/agent/memory-internals). Memory holds the current work's short facts. The long-term story (sessions, decisions, gotchas across projects) belongs in the [knowledge vault](/agent/knowledge).

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

Because entries are vault notes, the knowledge graph indexes them and recall can surface them. Older memory is in `~/.kdeps/memory/`. The first time that the memory of a folder loads, kdeps moves the older memory into the vault. It renames the old database to `memory.bolt.moved`.

Memory lives in the [knowledge vault](/agent/knowledge) as one markdown note per entry, in `<vault>/projects/<project>/memory/`. The project is the one whose card covers the folder, else one kdeps creates, named after the git repository root or folder. Thinking and tool-result captures go in a hidden `memory/.auto/` folder, outside the link graph. A note you write there by hand is an entry too, keyed by its title. Each entry has a key, value, type, timestamps and optional references to other entries.

`KDEPS_MEMORY_GLOBAL=1` shares one store across every project, in `<vault>/shared/memory/`. Session storage (conversation history) stays per-directory either way.

Every turn gets one graph-ordered `<memory>` block, rebuilt and focused on that turn's request. It is not in the cached system prompt, so memory saved mid-session reaches the next turn. Entries appear in causal order with the newest unfinished task flagged, so a model taking over after a switch knows where to continue. Format: [Memory internals](/agent/memory-internals#prompt-injection).

## Built-in memory tools

The agent has three memory tools. There is no `memory_list`, because the graph and recent keys are already in context. To find a fact, use `memory_query` (preferred) or `memory_search` (fuzzy text). The prompt tells the model to look in memory in these conditions: the task refers to earlier work, the prompt does not have a fact, or it is the first turn after a model switch, resume, or compaction. The model does not look in memory before each action.

| Tool | Description |
|------|-------------|
| `memory_save` | Save a fact with a key and value. Keys should be short and descriptive. |
| `memory_search` | Fuzzy text search of entries by key or value. Returns at most 20 matches, best first; each value is cut at 500 characters. |
| `memory_delete` | Remove an entry by key. |

Another tool, `memory_query`, runs relational queries (select/project/join/union) over memory plus tool-call history and task state - see [Relational query](#relational-query-memory-query) below.

## memory_save

Creates or updates a memory entry. kdeps writes the entry to disk immediately.

```json
{
  "name": "memory_save",
  "parameters": {
    "key": "project_name",
    "value": "kdeps - Go module github.com/kdeps/kdeps/v3"
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

Returns the entries that match, as formatted text. It prints a maximum of 20 matches, with the best match first. It cuts each value at 500 characters (a cut value ends in `...`). If there are more matches, the header gives the number found and the number shown. The last line gives the number that it did not show:

```
Found 2 memory entries:
- project_name: kdeps - Go module github.com/kdeps/kdeps/v3
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

`memory_query` runs a relational query (select, project, join, union) on two relations that kdeps makes from the agent state. Use it to filter by field, combine facts from different sources, or find relations between past tool calls:

| Relation | Fields | Source |
|---|---|---|
| `memory` | `key`, `value`, `namespace`, `type`, `references`, `createdAt`, `updatedAt` | Persistent memory entries |
| `tool_calls` | `name`, `args` (an object when the arguments are JSON, so `.args.command` works), `args_json` (the raw text), `result`, `timestamp` | Recent tool-call history (this session, most recent 200) |

The query language is [expr-lang](https://expr-lang.org/), the same engine `before:`/`after:` expressions use. `filter()`/`map()` are its own built-ins (select/project). `memory_query` adds `join()`/`union()`:

| Operation | Function | Example |
|---|---|---|
| Select (WHERE) | `filter(relation, predicate)` | `filter(memory, .type == "error")` |
| Project (columns) | `map(relation, expr)` | `map(memory, {key: .key, value: .value})` |
| Join | `join(left, right, leftField, rightField)` | `join(tool_calls, memory, "name", "key")` |
| Union | `union(a, b)` | `union(filter(memory, .type == "error"), filter(memory, .type == "decision"))` |

String matching uses infix operators, not function calls. `contains(...)` is a parse error. Put the field on the left:

| Match | Operator | Example |
|---|---|---|
| Substring | `contains` | `filter(memory, .value contains "latency")` |
| Prefix | `startsWith` | `filter(memory, .key startsWith "tool:read_file")` |
| Suffix | `endsWith` | `filter(memory, .key endsWith ":summary")` |
| Regex | `matches` | `filter(memory, .key matches "^tool:")` |
| Case-insensitive substring | `lower()` + `contains` | `filter(memory, lower(.value) contains "latency")` |
| Membership | `in` | `filter(memory, .type in ["error", "decision"])` |

`join` is an equi-join, merging rows with `left_`/`right_` prefixed field names so same-named fields never collide. No inequality/range-join or multi-field-key support today.

```json
{"name": "memory_query", "parameters": {"query": "filter(memory, .type == \"error\")", "limit": 20}}
```

The result has `rows` (capped at `limit`: the tool default is 50 and the `memory-query-rows-max` event caps it at 500), `count` (total matches before capping), and `truncated` (bool). `memory_query` runs through the kdeps API and reads the in-process agent loop. An empty query returns `memory_query: query is required`. A call with no loop returns `memory_query: agent loop is not configured (memory_query is an agent-mode tool)`.

## Memory entry types

kdeps classifies entries automatically by key pattern, with the rules in the `memory/rules` asset (see [Memory rules](/agent/memory-internals#memory-rules)). The type sets the position of the entry in the [memory graph](/agent/memory-internals#memory-graph). It also sets if kdeps can prune the entry. The table lists the shipped rules.

| Type | Key patterns | Description |
|------|-------------|-------------|
| `prompt` | `prompt`, `goal`, `task` | User goals and task descriptions |
| `purpose` | `purpose`, `why`, `reason` | Rationale for decisions |
| `progress` | `progress`, `wip`, `in_progress` | Work in progress tracking |
| `result` | `result`, `output`, `done` | Completed work results |
| `status` | `status`, `state` | Current state information |
| `tool_result` | `tool:*` | Tool call outputs (20 per tool, the `memory-tool-results-limit` event) |
| `thinking` | `thinking:*` | The model's reasoning text for a round, across every thinking-capable backend - searchable via `memory_search`. Capped at 20 (the `memory-thinking-limit` event). |
| `decision` | `decision`, `decided` | Design decisions |
| `preference` | `preference`, `prefer`, `like` | User preferences |
| `context` | `context`, `env`, `config` | Environment context |
| `file` | `file`, `path`, `dir`, `last_files` | File references |
| `action` | `last_action` | Last action taken |
| `error` | `error`, `fail`, `bug` | Errors and failures |
| `note` | (default for unknown) | Uncategorized entries |
| `fact` | (not auto-assigned) | General facts; grouped with `note` as low-signal (both capped at 50 combined) |

## Workflow mode

Memory tools operate in agent mode and in workflow mode. In workflow mode, `GetOrCreateMemoryStore()` initializes the store on first use. You do not need a Loop. Memory is available to all resources and tools.

Workflow resources can call `memory_list()` to list the keys. The agent loop does not register that as a tool. `memory_query` uses that same call route and still needs the agent loop's live state. A workflow call with no loop returns `memory_query: agent loop is not configured (memory_query is an agent-mode tool)`.

## Configuration

Memory is on by default when the agent loop starts. You do not need YAML configuration. The store lives in `<vault>/projects/<project>/memory/` (default vault: `~/.kdeps/notes`).

## See also

- [Memory internals](/agent/memory-internals) - auto-extraction, the memory graph, prompt injection, compaction
- [Session configuration](/workflow/sessions) - HTTP session storage across requests from the same caller
- [Agent mode](/agent/) - how the agent loop works
