# Memory internals

How the agent loop decides what to remember, how entries are linked, and what the model sees each turn. This is agent mode only. For the memory tools and entry types, see [Persistent memory](/concepts/memory).

## Auto-extraction

The agent loop extracts facts from every turn without an explicit `memory_save` call. Four mechanisms run after each turn:

**1. Explicit markers.** The agent can write `[MEMORY: key] value` on its own line in any response:

```
[MEMORY: project_name] kdeps - Go module github.com/kdeps/kdeps/v2
```

The marker is extracted into the store and then **removed from the reply** before it is shown or written to the transcript - the model's `[MEMORY: ...]` line and any echoed `GOAL:` / `ACTIVE TASK` directive fragment never reach the terminal.

**2. Action sentences.** The first action sentence of the assistant response is captured as a `last_action` entry:

```
"Added rate limiting middleware to the HTTP handler"
-> last_action = "rate limiting middleware to the HTTP handler"
```

**3. File references.** File paths in edit/create/read operations are captured as a `last_files` entry:

```
"Edited src/api/handler.go and src/api/middleware.go"
-> last_files = "src/api/handler.go, src/api/middleware.go"
```

**4. Tool results.** Every tool-call result is saved with a key derived from the tool name and output, so the LLM can search past results. See [Tool result filtering](#tool-result-filtering) for the caps.

## Memory graph

Entries are linked into a directed graph based on their [types](/concepts/memory#memory-entry-types). The dependency chain:

```
prompt -> purpose -> progress -> tool_result -> result -> status
                                    |
                               action, error, file, decision, fact, note
```

The graph is **inlined into the `<memory>` block** rather than drawn as a separate diagram: entries are ordered so each parent comes before the children that reference it, and every entry shows its parent edge with `<- parent`. Each entry renders on exactly one line - a multiline value has its newlines collapsed to ` / ` so it never breaks the one-entry-per-line reading.

## Prompt injection

On every turn the memory store injects one graph-ordered `<memory>` block: a legend, a one-line orientation map (entry counts by type plus the resume target with its **relative age**), the entries in topological order (parents first) with `<- parent` edges inline, the newest unfinished `progress`/`result`/`status` entry marked `<== RESUME`, and that node's downstream dependencies:

```
<memory>
Legend: workflow chain, parents before children. "key [type]: value"; "<- P" = derived from P; "(same as K)" = repeats K's fact; "<== RESUME" = continue here.
map: 1 prompt, 1 tool_result, 1 result | resume: result:build (2m ago)
prompt:build [prompt]: Add /users endpoint
tool:write_users [tool_result]: wrote handlers/users.go  <- prompt:build
result:build [result]: compiles; tests pending  <- tool:write_users  <== RESUME
</memory>
```

The `(2m ago)` hint is a coarse relative age (`just now`/`Nm`/`Nh`/`Nd ago`) a model uses after an orchestrator model switch to judge whether to re-verify before continuing.

The block is truncated to a token budget, but not oldest-first: the **active task chain**, entries **relevant to the current prompt** (matched on significant prompt words at word boundaries), and the **newest unresolved error** are always kept - unrelated older entries drop first, and edges to dropped entries are omitted so no arrow dangles.

The orientation map also names the most recent unresolved `error` entry so a resuming model is reminded of a known failure up front - one that reads as handled (`resolved`, `fixed`, `closed`, ...) is not surfaced, but a re-opened one (`reopened`, `not fixed`, `still failing`, ...) is, even alongside the word "fixed".

Duplicate facts (case/whitespace-insensitive) are flagged `(same as <key>)` on the later entry instead of repeated as independent evidence. The agent also receives a standing rule: "Check memory first. Before taking ANY action, use `memory_search` and `memory_list` to see what is already known about the task."

## Compaction integration

When the agent runs `/compact` (summarizes and clears conversation history), the `AutoCapture` method extracts structured sections from the summary:

- `## Key Decisions` - saved as `decision` type entries
- `## Critical Context` - saved as `context` type entries

This preserves important information across compaction boundaries.

## Checkpoint summaries

After every compaction, a `checkpoint:summary` entry is saved containing the condensed Goal, Progress, Key Decisions, and Critical Context sections. This provides a running project snapshot that persists across sessions.

## Session persistence

The agent's full LLM config (model, backend, base URL) is saved to `session:config` on startup and after every `/model` switch. On the next run, the config is restored automatically. The working directory is also saved on start (`session:started`) and resume (`session:resumed`) so the agent always knows where it is.

## Tool result filtering

To prevent memory bloat, only write/exec/search tools produce memory entries. Read-only lookups (`read_file`, `list_files`, `search_local`) are filtered out. Each tool type is capped at 20 entries - the oldest are auto-deleted when the cap is reached. A tool result longer than its store limit is cut and marked with a trailing `...`, backed off to the nearest character boundary so a multibyte character is never split.

Auto-extracted **low-signal** entries (types `note` and `fact`) are globally capped at 50 combined - when the cap is exceeded, the oldest are pruned on write. Structural entries (`prompt`, `purpose`, `progress`, `result`, `status`, `decision`, `context`, `tool_result`, ...) are never pruned by this cap, so the workflow chain and resume point are always preserved.

## See also

- [Persistent memory](/concepts/memory) - the memory tools and entry types
- [Agent loop mode](/modes/agent-loop-mode) - how the agent loop works
- [Goal-directed execution](/modes/agent-loop-goals) - task state that `memory_query` can read
