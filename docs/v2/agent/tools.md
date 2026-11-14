# Built-in tools

The [agent loop](/agent/) has access to a set of built-in tools that the LLM can call without any YAML configuration. kdeps registers tools that need credentials only when you set the related environment variable. Workflow `tools:` blocks are [function calling](/reference/tools), a different page.

| Family | Tools | Page |
|---|---|---|
| Files | `read_file`, `write_file`, `edit_file`, `list_files`, `md5_file`, `tail_file` | [File tools](/agent/tools-files). [Edit file](/agent/tools-edit) |
| Memory and identity | `memory_save`, `memory_search`, `memory_delete`, `memory_query`, `identity_get` | [Memory and identity](/agent/tools-memory) |
| Knowledge vault | `knowledge_recall`, `knowledge_card_get`, `knowledge_card_update`, `knowledge_note_save` (while the vault is on: the default) | [Knowledge vault](/agent/knowledge#tools) |
| Shell | `bash_exec`, `bash_job_list`, `bash_job_wait` | [Shell execution](/agent/shell) |
| Web and search | `web_search`, `wikipedia`, `web_scraper`, plus keyed search APIs | Web and search |
| Data | `calculator`, `sql_*`, embeddings, rerank | [Data tools](/agent/tools-data) |
| Actions | `http_request`, `search_local`, `load_document`, `transcribe_audio`, `ocr_image`, Zapier, Google cache | [Action tools](/agent/tools-actions) |
| Structured operations | `file_ops`, `git_ops`, `sql_ops` - the same tools behind the `file`, `git` and `sql` components | [File](/workflow/resources/file), [Git](/workflow/resources/git), [SQL](/workflow/resources/sql) |
| Harness | `instruct` (alias `help`) | [Efficiency enforcement](/agent/efficiency#harness-prompt) |
| Subagents | `subagent`: a task for a fresh, read-only agent loop | [Subagents](/agent/subagent) |
| Questions | `ask_question`: show the user a question (with numbered options) and end the turn; their next message is the answer | [Asking the user](#asking-the-user) |
| Planning | `plan_update`: tick a job order, or add one, in the progress file the planning council wrote | [Planning council](/agent/plan) |

Permission modes and lean mode stay on this page. The calling rules are their own pages: [Fenced tools](/agent/tools-fenced), [Harness reminders](/agent/tools-harness), [Session-integrity handshake](/agent/tools-handshake).

## Asking the user

`ask_question` puts a choice to you and ends the turn. The question, with its options numbered, is the reply. What you type next is the answer. Skill packs with interactive steps call it by name. A subagent never gets it.

```json
{"question": "Which review mode?", "options": ["Interactive", "Consolidated report"]}
```

```text
Which review mode?

1. Interactive
2. Consolidated report

Reply with a number, or with your own answer.
```

## Tool definitions

Each built-in tool is a YAML definition in the `tools` asset set. `kdeps update`
installs it. The definition has the description, category, usage notes, and
parameter schema that the model sees. Most built-ins also have their
implementation in the same file. The implementation is an `http:` or `python:`
block on one of the four kernel primitives (`http`, `exec`, `python`, `sql`).
Go code serves the other built-ins, because they need kernel state (memory,
embeddings, the agent session).

| Implemented in YAML | Block |
|---|---|
| `web_search`, `web_scraper`, `wikipedia`, `calculator`, `read_file`, `write_file`, `list_files`, `md5_file`, `tail_file`, `git_ops`, `edit_file`, `file_ops`, `sql_list_tables`, `sql_describe_table`, `sql_query` | `python:` (standard library only; runs on the system python) |
| `google_cache_create`, `google_cache_delete`, `google_cache_list`, `exa_search`, `wolfram_alpha`, `zapier_*`, the rerankers, `retrieve_context`, `transcribe_audio` | `http:` |
| `ocr_image`, `transcribe_audio_offline` | `exec:` |
| `clipboard_copy`, `hf_search`, `hf_repo_files` | `python:` |

```yaml
# tools/md5_file.yaml (built-in, shown for reference)
version: 2.0.0       # bumped whenever the file changes
category: file       # section header in the tool list sent to the model
description: Compute the MD5 checksum of a file. Call once before and once after a change to prove whether the file's contents actually changed ...
seeAlso: read_file, tail_file, list_files   # shown as "See also: ..."
parameters:
  file_path:
    type: string
    description: Path to the file to hash (absolute, or relative to the working directory)
    required: true
python:
  paths: [file_path]  # workspace root, plus ~/.kdeps and /tmp; the root directory stays blocked
  script: |
    import hashlib, json, os, sys
    path = os.path.abspath(json.loads(sys.argv[1])["file_path"])
    ...                                   # hashes the file in 1 MiB chunks
    print("%s  %s" % (path, h.hexdigest()))  # the printed text is the answer
```

Copy one of these files to `~/.kdeps/tools/` and edit it to change what the
tool does. See [Your own tools](#your-own-tools).

Optional fields:

- `constraints`: limits and known problems, shown inline.
- `outputFormat`: what the tool returns.
- `name`: the tool name that the model sees, if it is different from the file name.
- `aliases`: see below.

## Tool aliases

A model trained on another agent often calls a tool by the name it knows - `bash`, `cat`, `grep`, `str_replace_editor` - and gets "tool not found". A definition lists those names under `aliases`, and a call to one runs the tool:

```yaml
# tools/bash_exec.yaml
version: 1.1.0
category: shell
aliases: [bash, shell, sh, terminal, run_command, execute_command, run_terminal_cmd, run_shell_command, exec_command]
```

```text
model calls:   bash {"cmd": "go test ./..."}
kdeps runs:    bash_exec {"command": "go test ./..."}   # the alias picks the tool, then the argument synonyms below map cmd -> command
```

The model sees only the name of the tool, so aliases add nothing to the prompt. Matching ignores case. The name of a real tool always has priority over an alias. If two tools have the same alias, kdeps gives the alias to neither tool and shows a warning at load. Permissions, logs, and the result all use the name of the tool. Shipped aliases:

| Tool | Aliases |
|---|---|
| `bash_exec` | `bash`, `shell`, `sh`, `terminal`, `run_command`, `execute_command`, `run_terminal_cmd`, `run_shell_command`, `exec_command` |
| `read_file` | `read`, `cat`, `view_file`, `open_file`, `read_text_file` |
| `write_file` | `write`, `create_file`, `save_file`, `write_to_file` |
| `edit_file` | `edit`, `str_replace_editor`, `str_replace_based_edit_tool`, `replace_in_file`, `edit_text_file` |
| `list_files` | `ls`, `list_dir`, `list_directory`, `dir` |
| `search_local` | `grep`, `rg`, `ripgrep`, `search_files`, `find_in_files`, `codebase_search`, `grep_search` |
| `tail_file` | `tail` |
| `web_search` | `websearch`, `search_web`, `internet_search` |
| `web_scraper` | `fetch`, `web_fetch`, `fetch_url`, `read_url` |
| `memory_save` | `remember`, `save_memory` |

A name that is neither a tool nor an alias still returns `{"error": "tool \"<name>\" not found"}`.

## Declared HTTP tools

A tool whose definition has an `http:` block needs no Go code. kdeps registers it when the block's `env` resolves and runs it as one HTTP call. `exa_search`, `wolfram_alpha`, `zapier_*`, the three rerankers and `retrieve_context` ship this way. A new HTTP tool is one YAML file that you publish to the `tools` set. During development, put it into `$KDEPS_ASSETS_SRC/tools/`.

```text
model calls tool -> args + parameter defaults -> render url/headers/body -> HTTP call -> render result -> model
```

```yaml
# tools/wolfram_alpha.yaml (shipped)
version: 1.2.0
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
```

`wolfram_alpha` is on the `web-call-budget` event's `tools:` list: a repeat call with the same arguments reuses the turn's answer, and each distinct call counts against the budget. Add a declared tool to a budget's list for the same behavior. See [call-budget events](/agent/events-guards#call-budget-events).

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

A block can upload files. `files` maps form fields to path templates. `form` has text fields, and kdeps does not send empty text fields. kdeps sends these fields as `multipart/form-data`, not as `body`.

`paths` lists the parameters that have file paths. kdeps checks them against the workspace boundary, as for `read_file`. `~/.kdeps` and `/tmp` are permitted.

If the variable list of an `env` entry ends with `""`, the entry is optional. It can be empty, and the tool registers without it. kdeps does not send a header that is empty:

```yaml
# tools/transcribe_audio.yaml (shipped, abridged)
http:
  env:
    OPENAI: [OPENAI_API_KEY, ""]     # optional: a local server may take it or no key at all
    GROQ: [GROQ_API_KEY, ""]
  method: POST
  url: '{{if .args.baseURL}}{{.args.baseURL}}{{else if eq .args.backend "groq"}}https://api.groq.com/openai/v1{{else if eq .args.backend "local"}}http://localhost:8080/v1{{else}}https://api.openai.com/v1{{end}}/audio/transcriptions'
  headers:
    Authorization: '{{if eq .args.backend "groq"}}{{with .env.GROQ}}Bearer {{.}}{{end}}{{else}}{{with .env.OPENAI}}Bearer {{.}}{{end}}{{end}}'   # omitted when empty
  files:
    file: "{{.args.file}}"           # uploaded as the "file" form field
  form:
    model: "{{.args.model}}"
    response_format: "{{.args.responseFormat}}"
  paths: [file]
```

Template helpers: `json` (encode), `fromJSON` (decode a string argument), `urlquery`, `urlpath`, `trimSuffix`, `int`, `add`, `min`, and `get` (nested lookup by key or index; a missing level gives nothing). A non-2xx status returns `<tool>: API error <status>: <body>` to the model. `kdeps update` rejects an `http:` block whose templates do not parse, so a broken download never replaces a working tool.

## Declared SQL tools

A tool can run one parameterized SQLite statement. The YAML sets the statement. kdeps binds each argument in `params` to a `?` placeholder, in order. kdeps never puts arguments into the text, so a value cannot change the statement.

```yaml
# assets/tools/note_get.yaml
version: 1.0.0
description: Read one note by id.
parameters:
  id:
    type: string
    required: true
sql:
  dsn: KDEPS_NOTES_DB           # environment variable holding the SQLite file path
  query: SELECT id, title FROM notes WHERE id = ?
  params: [id]                  # bound to the ? placeholders, in order
```

`mode: rows` (the default) returns a JSON array of row objects. `mode: exec` runs a write and returns `N rows changed`. kdeps registers the tool only when you set the environment variable.

## Declared exec tools

A tool whose definition has an `exec:` block runs one program directly (no shell) and needs no Go code. `ocr_image` and `transcribe_audio_offline` ship this way.

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
  paths: [file]               # workspace root, plus ~/.kdeps and /tmp
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
| `env` | As in an `http:` block: names templates read as `.env.NAME` (a trailing `""` makes one optional) |
| `timeout` | Per-run limit (Go duration, default `60s`) |
| `result` | Template over `.args`, `.env` and `.stdout`; empty returns stdout trimmed |

A non-zero exit returns `<tool>: <command>: exit status N: <stderr>` to the model. A tool declares `http:` or `exec:`, not both.

## Declared python tools

A tool whose definition has a `python:` block runs one Python script and needs no Go code. With no `packages:`, the script runs directly on the system `python3` (or `python`), with no venv. With `packages:`, it runs on the kdeps python runner behind the `python` tool, which installs them into a venv with `uv`. The script reads the call's arguments as a JSON object from `sys.argv[1]` and prints its answer. The script and its arguments reach python on stdin, so an argument of any size (a whole file's content) gets through.

```yaml
# tools/word_count.yaml (or ~/.kdeps/tools/word_count.yaml)
version: 1.0.0
description: Count the words in a text.
parameters:
  text:
    type: string
    required: true
python:
  script: |
    import json, sys
    args = json.loads(sys.argv[1])     # {"text": "..."}
    print(len(args["text"].split()))   # printed text is the tool's answer
  packages: []          # pip packages installed into the runner's venv first
  timeout: 30s          # optional; the runner's default when empty
```

The script's output is UTF-8 on every platform (`PYTHONIOENCODING=utf-8`). An `env:` block operates as in an `http:` block. kdeps gives each resolved value to the script as an environment variable with the same name. The shipped `web_search` uses one to make its endpoint replaceable:

```yaml
# tools/web_search.yaml (shipped, abridged)
python:
  timeout: 30s
  env:
    SEARCH_URL: [KDEPS_WEB_SEARCH_URL, ""]  # trailing "" = optional; unset uses lite.duckduckgo.com
  script: |
    ...
```

A tool declares at most one of `http:`, `exec:`, `sql:` or `python:`.

## Your own tools

A tool you write yourself goes in `~/.kdeps/tools/`. A file with an `http:`, `exec:`, `sql:` or `python:` block is a new tool. kdeps registers it for the agent and serves it on `/apis/tools.kdeps.io/v1/tools/<name>/call`, with no registry or lock entry and no Go code. A file named like a shipped tool replaces its definition.

```yaml
# ~/.kdeps/tools/git_version.yaml - the file name is the tool name
version: 1.0.0
description: Print the installed git version.
exec:
  command: git          # run directly, no shell
  args: ["--version"]
```

```bash
curl -s -X POST -H "Authorization: Bearer $KDEPS_API_TOKEN" -H "Content-Type: application/json" \
  -d '{}' http://127.0.0.1:9095/apis/tools.kdeps.io/v1/tools/git_version/call
# git version 2.47.0
```

## The call runs in the kdeps API

The kdeps API serves a declared `http:` or `exec:` tool, as it serves all other tools. The tool asset declares the call. The handler runs it. The agent loop and a workflow component both reach it the same way.

```text
workflow YAML -> component YAML -> POST http://kdeps.api/apis/tools.kdeps.io/v1/tools/{name}/call -> handler -> http: or exec: block
agent loop    -> POST http://kdeps.api/apis/tools.kdeps.io/v1/tools/{name}/call -> handler -> http: or exec: block
```

```yaml
# resources/ocr.yaml
actionId: ocrText
name: Extract text
component:
  name: ocr_image          # assets/components/ocr_image/component.yaml POSTs the tool's route
  with:
    file: /tmp/scan.png
```

A required `env` entry that is unset keeps the tool out of the agent's catalog. A direct API call returns `<tool>: required environment variables are not set`.

## Parameter-name synonyms

Sometimes a model gives an argument a name that is almost correct. Examples are `pattern` (from `grep`) for `query` of `search_local`, or `path` (from `cat`) for `file_path` of `read_file`. Before dispatch, kdeps changes these synonym keys to the key that the tool expects. It does not overwrite a real key that is already there. Alternate *tool* names are [tool aliases](#tool-aliases), declared in each tool's definition.

## Failed tool calls are fed back to the model

kdeps returns each tool failure to the model as `{"error": ...}` **with a
`[TOOL FAILED]` banner** ("nothing changed - fix the cause and retry, or say it
failed"). As a result, the model does not miss the error.

If a work tool fails, the turn does not end on a "done" claim immediately after
the failure. The loop asks for a real success or an honest "it failed". It
asks a maximum of two times, and the second request is a repeat. This is the
same limit as for the other corrective nudges.

If the model does not acknowledge the failure after the two requests, kdeps
adds a notice after the response. The notice gives the tool and its error.
As a result, kdeps keeps the claim, but clearly marks it as not resolved. kdeps accepts
an honest admission (in all words) immediately, with no nudge.

The loop makes notices about budget changes and context-window trims. kdeps
adds these notices to the context of the model as `[kdeps] ...` messages. The
person at the REPL cannot do anything about them, but the model can adjust.

## Permission modes

To limit the tools that the agent can call, set `KDEPS_PERMISSION_MODE`:

```bash
KDEPS_PERMISSION_MODE=read-only ./kdeps          # reads, searches, lookups only
KDEPS_PERMISSION_MODE=workspace-write ./kdeps    # adds file writes and bash_exec
KDEPS_PERMISSION_MODE=danger-full-access ./kdeps # no restrictions (default)
```

A blocked call does not run. It returns a `permission denied` tool error to the model, and the model explains the limit. Tools that are not in the built-in policy need `workspace-write`. These include workflow, component, and agency tools. As a result, `read-only` blocks all tools that can change state.

The tool-to-mode mapping is one YAML file, `policy/permission.yaml` in the policy asset set. Each tool names a tier, and each tier names the mode it needs:

```yaml
default: allow     # an unlisted tool needs workspace-write (the default); deny needs danger-full-access
tiers:
  T0: read-only
  T1: workspace-write
tools:
  read_file: T0
  bash_exec: T1
```

A tier can also name `ask` (T2 in the built-in file, unused by default): a tool on that tier needs a person to approve each call, in every permission mode. With no terminal and no approver, kdeps refuses the call. Set `default: deny` to run tools that are not listed only in `danger-full-access`. These include workflow, component, and agency tools.

To change the policy without a change to the built-in file, put the same YAML at `~/.kdeps/policy/permission.yaml`. That file replaces the built-in policy. If kdeps cannot parse a user file, it uses `default: deny`. kdeps rejects an unknown tier or mode in the file. If a listed tool has no tier, kdeps uses `danger-full-access` for it.

A tier can name `token` (T3 in the built-in file, unused by default). A tool or stage on that tier runs only when a valid capability token exists for its scope:

```bash
kdeps token grant bash_exec --ttl 1h      # scope: a tool name, or stage:<kind> such as stage:exec
kdeps token list                          # each token and whether it is valid
kdeps token revoke <id>                   # delete one
```

Tokens are YAML files in `~/.kdeps/tokens`. Without a valid token, kdeps refuses the call, and the error gives the grant command.

Workflow stages use the same tiers. The file's `resources` table maps each primary resource kind (`exec`, `python`, `httpClient`, `sql`-style kinds, `chat`, `agent`, `component`, ...) to a tier. Each run, also plain `kdeps run`, checks each stage against `KDEPS_PERMISSION_MODE` before the stage runs. If a stage needs a higher mode, kdeps refuses it and the workflow stops. kdeps also refuses an `ask`-tier kind, because a workflow run has no person to ask. The default run mode is `danger-full-access`. As a result, nothing changes until an operator sets a stricter mode.

The kdeps tool calls of a workflow chat (`write_file`, `bash_exec`, ...) go through the `tools` table in the same way. kdeps refuses a call above the run mode, an `ask`-tier tool, or a `token`-tier tool with no token. The model gets the refusal as the tool result.

When an instrument session records, each gate decision is a `permission_decision` event. The event has the tool, the mode that it needs (`required`), and the active mode (`mode`). It also has `decision` (`allow` or `deny`), and the `reason` if kdeps denies the call.

## Lean mode

kdeps registers the full tool set for each session. This set has about 61 tools: `bash_exec`, `web_search`, `web_scraper`, `wikipedia`, `http_request`, external API tools, and the lean set below. A cloud backend sends each tool as a native schema. A local backend (file, gguf, ollama, llamafile) also gets a short text list: name, one-line description, and parameter names. Trim the set when you want less prompt weight or a restricted surface for CI/automation:

```
/tools lean     # this session only, switch back any time
/tools full     # switch back
/tools          # show current mode and tool count
```

The choice persists automatically across sessions - no flag needed - the same way `/model tool set` settings do. Lean mode keeps `read_file`, `write_file`, `edit_file`, `list_files`, `search_local`, `load_document`, `calculator`, `embedding_vectorize`, `embedding_search`, `transcribe_audio` (~16 tools).

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
- [Edit file](/agent/tools-edit) - replace or append on one line
- [Memory and identity](/agent/tools-memory) - memory_save, memory_search, memory_query, identity_get
- [Shell execution](/agent/shell) - bash_exec, jobs, rtk
- Web and search - web_search, wikipedia, web_scraper
- [Data tools](/agent/tools-data) - calculator, SQL, embeddings, rerank
- [Action tools](/agent/tools-actions) - HTTP, local search, documents, Zapier, git trailer
- [REPL slash commands](/agent/commands) - full command reference
- [Tool execution monitoring](/agent/monitoring) - status lines and stall detection
- [Agent mode](/agent/) - overview and starting the REPL
