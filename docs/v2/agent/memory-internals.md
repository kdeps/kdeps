# Memory internals

The agent loop decides what to remember, links entries into a graph, and injects them into every turn. For the memory tools and entry types, see [Persistent memory](/agent/memory).

## Auto-extraction

The loop extracts facts from every turn without a `memory_save` call. Four mechanisms run after each turn:

**1. Explicit markers.** The model writes `[MEMORY: key] value` on its own line:

```
[MEMORY: project_name] kdeps - Go module github.com/kdeps/kdeps/v3
```

kdeps saves the marker. Then it removes the marker from the reply before it shows the reply or writes it to the transcript. kdeps also removes repeated `GOAL:` / `ACTIVE TASK` fragments.

**2. Action sentences.** The first action sentence of the reply becomes `last_action`:

```
"Added rate limiting middleware to the HTTP handler"
-> last_action = "rate limiting middleware to the HTTP handler"
```

**3. File references.** Paths in edit/create/read sentences become `last_files`:

```
"Edited src/api/handler.go and src/api/middleware.go"
-> last_files = "src/api/handler.go, src/api/middleware.go"
```

**4. Tool results.** kdeps saves each captured tool result under a key from the tool name and output. Then the model can search past results. Caps are in [Tool result filtering](#tool-result-filtering).

## Memory graph

Entries link into a directed graph by [type](/agent/memory#memory-entry-types):

```
prompt -> purpose -> progress -> tool_result -> result -> status
                                    |
                               action, error, file, decision, fact, note
```

The graph is inline in the `<memory>` block. Parents come before their children, and each entry shows its parent as `<- parent`. Each entry is one line. Newlines in a value become ` / `.

## Prompt injection

Every turn gets one `<memory>` block containing:

- a legend
- a map line: entry counts by type, plus the resume target and its age
- entries in topological order (parents first) with `<- parent` edges
- `<== RESUME` on the newest unfinished `progress`/`result`/`status` entry, plus its downstream dependencies

```
<memory>
Legend: workflow chain, parents before children. "key [type]: value"; "<- P" = derived from P; "(same as K)" = repeats K's fact; "<== RESUME" = continue here.
map: 1 prompt, 1 tool_result, 1 result | resume: result:build (2m ago)
prompt:build [prompt]: Add /users endpoint
tool:write_users [tool_result]: wrote handlers/users.go  <- prompt:build
result:build [result]: compiles; tests pending  <- tool:write_users  <== RESUME
</memory>
```

After a switch, the new model uses the age (`just now`, `Nm`, `Nh`, `Nd ago`). The age tells it if it must verify again before it continues.

## What the budget keeps

The block is cut to a token budget, but not oldest-first. Always kept:

- the active task chain
- entries matching significant prompt words (whole-word match)
- the newest unresolved error

Unrelated older entries drop first. kdeps also removes edges to removed entries. As a result, each arrow has an entry at its end.

## Unresolved errors

The map line gives the newest `error` that is not resolved. As a result, a model that resumes sees a known failure first. kdeps skips an error with a resolved status (`resolved`, `fixed`, `closed`, ...). It shows an error that opened again (`reopened`, `not fixed`, `still failing`, ...), also next to "fixed".

## Duplicate facts

If a later entry repeats an earlier fact (case and whitespace are ignored), kdeps marks it `(same as <key>)`. It does not count as new evidence. The model also gets a standing rule: "Check memory first. Before taking ANY action, use `memory_search` to see what is already known about the task." There is no `memory_list` tool. The graph is the context.

## Compaction integration

On `/compact` (summarize and clear history), `AutoCapture` saves two summary sections:

- `## Key Decisions` as `decision` entries
- `## Critical Context` as `context` entries

## Checkpoint summaries

Every compaction or [fold](/agent/repl-context#fold) saves a `checkpoint:summary` entry with the Goal, Progress, Key Decisions and Critical Context sections. It persists across sessions.

First, kdeps archives the previous checkpoint as `checkpoint:archive:<timestamp>`. It never discards it. Only the newest `FoldContextItems` checkpoints (active + archived, default 5, see `/fold items`) enter the memory block. Older ones stay retrievable with `/memory list` or `/memory show checkpoint:archive:<timestamp>`.

## Leaf-node limits

Each entry's `references` field points at its parent keys. A **leaf** is an entry nothing references. Most `fact:` and `tool_result:` entries are leaves. `purpose:`/`progress:` entries with children are not.

Two settings cap leaves, independent of the checkpoint cap:

```text
/model tool set leaf-nodes 40    # max leaf entries in the memory block; 0 = unlimited
/model tool set leaf-chars 300   # max characters per leaf value, cut with an ellipsis; 0 = unlimited
```

Both exempt the active task chain (the resume node, its ancestors, and entries matching the prompt), like the budget pass. They change only the rendered block: the store keeps every leaf intact for `/memory list` and `/memory show`.

## Session persistence

kdeps saves the LLM configuration (model, backend, base URL) to `session:config` at startup and after each `/model` switch. It restores it on the next run. kdeps saves the working directory on start (`session:started`) and on resume (`session:resumed`).

## Tool result filtering

- Only tools in the rules' `capture_tools` list create entries. Shipped: `write_file`, `edit_file`, `bash_exec`, `read_file`, `search_local`, `sql_query`, `http_request`, `web_search`, `web_scraper`, `wikipedia`.
- Each tool keeps its newest 20 captures (the `memory-tool-results-limit` event). kdeps deletes older captures.
- A result over its store limit is cut and ends with `...`, on a character boundary so no multibyte character splits.
- Low-signal entries (the rules' `low_signal` types, shipped `note` and `fact`) have a combined maximum of 50 (the `memory-low-signal-limit` event). kdeps prunes the oldest entries on write.
- Structural entries (`prompt`, `purpose`, `progress`, `result`, `status`, `decision`, `context`, `tool_result`, ...) are never pruned by that cap, so the chain and resume point survive.

## Memory rules

The `memory/rules` asset sets the type and links of entries, and the tool results that kdeps captures. It also sets where a resumed session continues, and which text means done or resolved. Install a newer `memory/rules` to change them (see [Updating assets](/agent/repl-updating)). kdeps does not install a file that the loader rejects.

```yaml
version: 1.0.0
types:                           # an entry's type from its key; the first match wins
  - {prefix: "tool:", type: tool_result}
  - {equals: last_action, type: action}
  - {contains: [prompt, goal, task], type: prompt}
  - {contains: [error, fail, bug], type: error}
default_type: note               # a key no rule matches
parents:                         # a new entry links to the newest entry of this type (else the newest entry)
  progress: result
  error: tool_result
capture_tools: [write_file, edit_file, bash_exec]   # tool results saved into memory
resume: [progress, status, result]                  # types a resumed session continues from, best first
low_signal: [note, fact]         # pruned oldest-first beyond the memory-low-signal-limit event
status_done:                     # a status with one of markers and none of not is finished
  not: [not done, in progress, blocked]
  markers: [done, complete, merged]
error_resolved:                  # an error that reads as resolved is not flagged in the memory block
  not: [reopened, still failing]
  markers: [fixed, resolved, closed]
reply_capture:                   # what the store reads from the model's reply besides [MEMORY:] markers
  action_verbs: [Added, Fixed, Updated]   # "<verb> the rest" (any case) becomes last_action
  action_min: 5                  # the rest must be 5 to 200 characters long
  action_max: 200
  file_verbs: [Edited, Created, Wrote]    # "<verb> [file] path" (as written) becomes last_files
focus_stopwords: [the, and, please]       # prompt words ignored when the prompt is matched to memory
focus_min_stem: 3                # a word of 3+ letters without -ing/-ed/-s is matched by that stem (billing finds bills)
focus_weights:                   # how an entry ranks against the prompt for the memory block
  key_weight: 2                  # a prompt word in the key scores 2, one only in the value scores 1
  structural_bonus: 1            # a match on a type outside low_signal adds 1
search_min_word: 2               # memory_search drops the extra words of a query shorter than this
```

Focus matching splits the prompt and each entry into stemmed words, like knowledge recall: letters and digits, camelCase and snake_case split. `billing` matches an entry about `bills`. Status and error markers match whole, unstemmed words:

- a marker with a space, hyphen or apostrophe matches as text
- a marker of 3 letters or fewer matches a whole word (`wip` does not match `wiped`)
- a longer marker also matches a word starting with it (`complete` matches `completed`)

## See also

- [Persistent memory](/agent/memory) - the memory tools and entry types
- [Agent mode](/agent/) - how the agent loop works
