# Built-in tools

The [agent loop](/agent/) has access to a set of built-in tools that the LLM can call without any YAML configuration. Tools that require credentials are only registered when the relevant environment variable is set. Workflow `tools:` blocks are [function calling](/reference/tools), a different page.

| Family | Tools | Page |
|---|---|---|
| Files | `read_file`, `write_file`, `edit_file`, `list_files`, `md5_file`, `tail_file` | [File tools](/agent/tools-files). [Edit file](/agent/tools-edit) |
| Memory and identity | `memory_save`, `memory_search`, `memory_delete`, `memory_query`, `identity_get` | [Memory and identity](/agent/tools-memory) |
| Shell | `bash_exec`, `bash_job_list`, `bash_job_wait` | [Shell execution](/agent/shell) |
| Web and search | `web_search`, `wikipedia`, `web_scraper`, plus keyed search APIs | [Web and search](/agent/tools-web) |
| Data | `calculator`, `sql_*`, embeddings, rerank | [Data tools](/agent/tools-data) |
| Actions | `http_request`, `search_local`, `load_document`, `transcribe_audio`, `ocr_image`, Zapier, Google cache | [Action tools](/agent/tools-actions) |
| Code | `code_search`, `code_definition`, and the other LSP tools | [Code intelligence](/workflow/resources/code-navigation) |

Permission modes and lean mode stay on this page. The calling rules are their own pages: [Fenced tools](/agent/tools-fenced), [Harness reminders](/agent/tools-harness), [Session-integrity handshake](/agent/tools-handshake).

## Tool definitions

Each built-in tool's model-facing text is a YAML definition in the `tools`
asset set, installed by `kdeps update`: the description, category, usage
notes and parameter schema the model sees. The tool's code only supplies its
name and what it runs.

```yaml
# tools/md5_file.yaml (built-in, shown for reference)
version: 1.0.0       # bumped whenever the text the model sees changes
category: file       # section header in the tool list sent to the model
description: Compute the MD5 checksum of a file. Call once before and once after a change to prove whether the file's contents actually changed ...
seeAlso: read_file, tail_file, list_files   # shown as "See also: ..."
parameters:
  file_path:
    type: string
    description: Absolute (or cwd-relative) path to the file to hash. Omit to hash the file most recently accessed this session.
```

Optional fields: `constraints` (limits and gotchas, shown inline), `outputFormat`
(what the tool returns), and `name` (the tool name the model sees, when it
differs from the file name: goal mode's `goal_task_complete.yaml` registers as
`task_complete`).

## Declared HTTP tools

A tool whose definition has an `http:` block needs no Go code: kdeps registers it when the block's `env` resolves and runs it as one HTTP call. `exa_search`, `wolfram_alpha`, `zapier_*`, the three rerankers and `retrieve_context` ship this way, and a new HTTP-backed tool is one YAML file published to the `tools` set (or dropped into `$KDEPS_ASSETS_SRC/tools/` while developing).

```text
model calls tool -> args + parameter defaults -> render url/headers/body -> HTTP call -> render result -> model
```

```yaml
# tools/wolfram_alpha.yaml (shipped)
version: 1.1.0
description: Query Wolfram Alpha for factual computations ... Requires WOLFRAM_APP_ID.
parameters:
  query:
    type: string
    required: true           # a missing required argument fails with "wolfram_alpha: query is required"
http:
  env:
    APP_ID: [WOLFRAM_APP_ID]  # .env.APP_ID; first non-empty variable wins; unset -> tool not offered
  method: GET
  url: "https://api.wolframalpha.com/v1/result?i={{urlquery .args.query}}&appid={{urlquery .env.APP_ID}}"
  cache: true                 # repeat calls with the same arguments reuse the answer for the run
```

```yaml
# tools/cohere_rerank.yaml (shipped, abridged): JSON body and a templated result
parameters:
  documents: {type: string, required: true}   # a JSON array, decoded with fromJSON
  model: {type: string, default: rerank-v3.5} # defaults fill missing arguments before rendering
  top_n: {type: number, default: 5}
http:
  env:
    API_KEY: [COHERE_API_KEY]
  method: POST
  url: https://api.cohere.com/v2/rerank
  headers:
    Authorization: "Bearer {{.env.API_KEY}}"
  body: '{"model": {{json .args.model}}, "query": {{json .args.query}}, "documents": {{json (fromJSON .args.documents)}}, "top_n": {{int .args.top_n}}}'
  result: |-                  # rendered over .response (the parsed JSON body)
    {{range $i, $r := .response.results}}{{$r.index}}: {{$r.relevance_score}}
    {{end}}
```

| Field | Meaning |
|---|---|
| `env` | Names templates read as `.env.NAME`, each mapped to the variables that may hold it. Templates see no other environment variable. |
| `method` | `GET`, `POST`, `PUT`, `PATCH` or `DELETE` |
| `url`, `headers`, `body` | Go templates over `.args` (arguments plus parameter `default`s) and `.env` |
| `result` | Template over `.args`, `.env` and `.response`. Empty, or a body that is not JSON: the body is returned as is. |
| `cache` | Reuse the answer for identical arguments within the run |
| `timeout` | Per-call limit (Go duration, default `30s`) |

Template helpers: `json` (encode), `fromJSON` (decode a string argument), `urlquery`, `urlpath`, `trimSuffix`, `int`, `add`, `min`, and `get` (nested lookup by key or index that yields nothing instead of failing on a missing level). A non-2xx status returns `<tool>: API error <status>: <body>` to the model. An `http:` block whose templates do not parse is rejected by `kdeps update`, so a broken download never replaces a working tool.

## Declared exec tools

A tool whose definition has an `exec:` block runs one program directly (no shell) and needs no Go code. `ocr_image` ships this way.

```yaml
# tools/ocr_image.yaml (shipped)
version: 1.1.0
description: Extract text from an image via tesseract OCR ...
parameters:
  file: {type: string, required: true}
  language: {type: string, default: eng}
exec:
  command: tesseract          # looked up on PATH when the tool runs
  install: tesseract-ocr      # the error names it when tesseract is missing
  paths: [file]               # checked against the workspace boundary before running
  args:                       # templates over .args; an arg that renders empty is dropped
    - "{{.args.file}}"
    - stdout
    - "-l"
    - "{{.args.language}}"
```

| Field | Meaning |
|---|---|
| `command` | Program to run, found on `PATH` |
| `args` | One template per argument; empty results are dropped, so an optional flag is two conditional args |
| `paths` | Parameters holding file paths; each is checked like `read_file`'s path |
| `install` | Package to name when `command` is missing |
| `timeout` | Per-run limit (Go duration, default `60s`) |
| `result` | Template over `.args` and `.stdout`; empty returns stdout trimmed |

A non-zero exit returns `<tool>: <command>: exit status N: <stderr>` to the model. A tool declares `http:` or `exec:`, not both.

## Parameter-name synonyms

A model sometimes names a tool's argument something plausible but not exact - `grep`'s `pattern` instead of `search_local`'s `query`, `cat`'s `path` instead of `read_file`'s `file_path`. These synonym keys are normalized to the key the tool actually expects before dispatch, without clobbering a real key already present. This is scoped to parameter names only: kdeps does not maintain a separate table of alternate *tool* names (e.g. routing a call to `grep` as if it were `search_local`) - the tool list already tells the model the real name to call, and a second, silently-redirecting name for the same tool added maintenance cost without fixing the cases where a model loses track of what tools it has.

## Failed tool calls are fed back to the model

Every tool failure is returned to the model as `{"error": ...}` **plus a
`[TOOL FAILED]` banner** ("nothing changed - fix the cause and retry, or say it
failed"), so a skimmed error is hard to miss. The turn will not end on a "done"
claim made right after a work tool failed: the loop pushes back for a real
success or an honest "it failed" - up to twice, with the second push-back
reading as a repeat, same bound as the other corrective nudges. If the model
still has not acknowledged the failure after both, the turn does not settle on
the bald claim unflagged: the response is followed by a notice naming the
tool and its error, so the claim is kept but clearly marked unresolved. An
honest admission at any point (in whatever words) is accepted immediately,
no nudge needed. With a goal active, `task_complete` on such a task is
refused, and a prose "done" records the task failed, not done.

Loop-generated notices - goal transitions, budget changes, forced task failures,
context-window trims - are injected into the model's context as `[kdeps] ...`
messages, since the human at the REPL cannot act on them but the model can
adjust.

## Permission modes

Set `KDEPS_PERMISSION_MODE` to restrict which tools the agent may call:

```bash
KDEPS_PERMISSION_MODE=read-only ./kdeps          # reads, searches, lookups only
KDEPS_PERMISSION_MODE=workspace-write ./kdeps    # adds file writes and bash_exec
KDEPS_PERMISSION_MODE=danger-full-access ./kdeps # no restrictions (default)
```

Blocked calls return a `permission denied` tool error to the model, which explains the restriction instead of executing. Tools not in the built-in policy - including workflow, component, and agency tools - require `workspace-write`, so `read-only` blocks anything that could mutate state.

## Lean mode

The full tool catalog (~55 tools - `bash_exec`, `web_search`, `web_scraper`, `wikipedia`, `http_request`, external API tools, plus the lean set below) is on by default for every session. Trim it down when you want less prompt weight (each tool costs tokens twice - once as a native tool schema, once as prose in the tool-use guidance) or a restricted surface for CI/automation:

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

## See also

- [Fenced tools](/agent/tools-fenced) - invoke blocks and narration
- [Harness reminders](/agent/tools-harness) - force a guidance section onto every turn
- [Session-integrity handshake](/agent/tools-handshake) - prove the model makes a real tool call
- [File tools](/agent/tools-files) - read, write, list
- [Edit file](/agent/tools-edit) - the six `edit_file` commands
- [Edit rules](/agent/tools-edit-rules) - read-first, revision, syntax check
- [Memory and identity](/agent/tools-memory) - memory_save, memory_search, memory_query, identity_get
- [Shell execution](/agent/shell) - bash_exec, jobs, rtk
- [Web and search](/agent/tools-web) - web_search, wikipedia, web_scraper
- [Data tools](/agent/tools-data) - calculator, SQL, embeddings, rerank
- [Action tools](/agent/tools-actions) - HTTP, local search, documents, Zapier, git trailer
- [Code intelligence](/workflow/resources/code-navigation) - code_search and the other LSP tools
- [REPL slash commands](/agent/commands) - full command reference
- [Tool execution monitoring](/agent/monitoring) - status lines and stall detection
- [Agent registries](/agent/registries) - task_*/team_*/cron_* tools
- [Agent mode](/agent/) - overview and starting the REPL
