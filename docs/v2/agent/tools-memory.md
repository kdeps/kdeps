# Memory and identity

`memory_save`, `memory_search`, `memory_delete`, and `memory_query` read and write the agent store during a turn. `identity_get` returns the configured name, email, and address. They are always registered. The rules for every tool are on [Built-in tools](/agent/tools).

## Memory tools

Always available. No environment variables required. `memory_save`, `memory_search`, and `memory_delete` run through the [kdeps API](/reference/api). A call with no memory store returns `memory_save: memory store is not configured`, and the same sentence for `memory_search` and `memory_delete`. The agent attaches its store before the call. `memory_query` runs through the [kdeps API](/reference/api). An empty query returns `memory_query: query is required`. A call with no agent loop returns `memory_query: agent loop is not configured (memory_query is an agent-mode tool)`. The agent installs the loop before the call.

| Tool | Description |
|------|-------------|
| `memory_save` | Save a fact to persistent memory. Injected into every LLM call automatically. |
| `memory_search` | Search memory entries by key or value. At most 20 matches, best first (key hit, then more query words, then newer); each value cut at 500 characters. |
| `memory_delete` | Remove a memory entry by key. |
| `memory_query` | Run an expr-lang relational query over agent state: `memory` (persistent entries), `tool_calls` (recent tool call history), `tasks` (active goal's task list). Supports `filter()`, `map()`, `join()`, `union()`. String matching is infix: `.value contains "x"`, `.key startsWith "x"`, `.value matches "regex"`; `contains(...)` is a parse error. |

Memory is stored per-project as markdown notes in the knowledge vault, `<vault>/projects/<project>/memory/` (see [Persistent memory](/agent/memory)). Facts persist across sessions and are auto-extracted from every turn - the agent can write `[MEMORY: key] value` on its own line to persist a fact without calling `memory_save`. See [Persistent memory](/agent/memory) for details.

The `memory_*` tools are how the *model* reads and writes memory during a turn. To inspect the store yourself from the REPL, use `/memory` (overview), `/memory list` (every entry), and `/memory search <query>` - see [REPL slash commands](/agent/commands).

## Identity tool

Always available. `identity_get` returns the agent's configured name, email, and address - see [Agent Identity](/reference/advanced-config#agent-identity) for how to set one. Returns "No identity configured for this agent." when unset. Never returns account credentials, even if configured; a model that can read a password can leak it in its own output.

