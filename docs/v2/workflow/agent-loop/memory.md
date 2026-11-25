# Persistent memory

Persistent memory stores and recalls facts across agent sessions for one project. [Session storage](/workflow/sessions) is different: it persists across HTTP requests from one caller.

The agent loop gives the model memory tools:

| Tool | Does |
|------|------|
| `memory_save` | save a fact under a short key |
| `memory_query` | relational query over memory, tool-call history and task state |
| `memory_search` | fuzzy text search by key or value |
| `memory_delete` | remove an entry by key |

## Read next

- [Persistent memory reference](/reference/agent-loop/memory) - how it works, built-in memory tools, memory_save, memory_search, memory_delete, relational query (memory_query), memory entry types, memory in a workflow
- [Memory internals](/reference/agent-loop/memory-internals) - auto-extraction, the memory graph, prompt injection, compaction
- [Session configuration](/workflow/sessions) - HTTP session storage across requests from the same caller
- [Agent loop](/workflow/agent-loop/) - how the loop works
