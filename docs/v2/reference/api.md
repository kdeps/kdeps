# kdeps API

The kdeps API is the control listener for agent mode (the REPL and the desktop app) and workflow mode (`kdeps run`). It is not the workflow HTTP API (`apiServer`, port 16395) and it is not `/_kdeps/`.

```text
caller -> bearer check -> handler
```

The process calls the handler in memory. Another process calls the same handler over HTTP. Both send the same bearer token.

## Listen address

The listener binds `127.0.0.1` on a free port. It is not published on `0.0.0.0` when the workflow API is public. Set `KDEPS_API_LISTEN` to choose the address, for example `127.0.0.1:9095`.

`KDEPS_API_TOKEN` is the bearer token. When it is unset, kdeps generates a token and keeps it in the process. The token is not written to `workflow.yaml` and it is not printed.

```bash
export KDEPS_API_TOKEN=dev-token
export KDEPS_API_LISTEN=127.0.0.1:9095
kdeps run workflow.yaml
```

## Loop data sets

Five YAML sets configure the agent loop. They are configuration data, not callable work, so the API does not serve them. Each set is read by one loop code path, and a client that needs the active values reads the config path, not a route:

| Set | Read by | What it configures |
|---|---|---|
| `harness/` | `pkg/agent/harness_yaml.go` | Prompt sections the loop sends to the model |
| `events/` | `assets/events/` (names generated into `pkg/agent/events_gen.go`) | Governor triggers: conditions that start an efficiency action |
| `actions/` | `pkg/agent/actions.go` | The governor's action vocabulary |
| `presets/` | `pkg/agent/presets.go` | Named bundles of harness and efficiency settings (`/efficiency preset`) |
| `themes/` | `pkg/agent/theme_yaml.go` | REPL colors and styles |

Users override a set by name under `~/.kdeps/` (see the agent reference). The loop reads the merged result at start.

## Knowledge and code graph rules

Two more asset sets hold business rules for the graph features. Each file is read once, the first time it is needed.

| Set | Read by | What it configures |
|---|---|---|
| `knowledge/` | `pkg/knowledge/` | `recall.yaml`: the word lengths and stopwords a recall query is matched on, and the task words left out when kdeps decides whether a request continues the conversation thread. `policy.yaml`: the graph's limits, thresholds and recall weights. `vault.yaml`: the vault's folder names, a project card's front matter keys, the front matter keys read from every note (summary, tags, aliases, links, canonical id) and a card's link kinds. `check.yaml`: which notes `kdeps knowledge check` requires a summary of, and the wording of each finding. `notes.yaml`: the templates of the notes kdeps writes (card, session, decision, workflow run log, saved note, `GRAPH_REPORT.md`, `kdeps knowledge explain` lines), a session note's file name time and the compaction summary sections that fill them. `extract.yaml`: the entity kinds and relations a model's extraction may name, the fallback for others, the notes never sent and the header the model reads. `search.yaml`: how the local search indexes split files and queries into terms (minimum length, stopwords, stem suffixes; rebuild an index after changing them), which folders and file types they read, which key and credential files they never read or return, and the share of a request's words a search-index file needs to join recall (`limits.recall_match_share`). `prompts.yaml`: what the model reads about the vault, the project status overview, the recalled notes and the search-index files, in both modes |
| `codegraph/` | `pkg/codegraph/codegraph.go` | The code graph's ranking weights, stopwords, result limits and path bounds, handed to Kartographer |

If a set's file is missing or broken, the knowledge recall keeps every word and prints a warning. Install rejects a `vault.yaml` that leaves any name empty, so a broken override never replaces a working one. The code graph reports that it has no rules until the file loads.
## Traceability: API to component to YAML

Every YAML file that runs reaches the kdeps API. A workflow reaches a tool through a component, a slash command reaches the REPL through the commands group, and a loop data set is read directly by the agent loop:

```d2
direction: right

workflow: "workflow.yaml\nresource: component" {shape: oval}
component: "components/X/component.yaml\nrequires: tools/X"
tools: "POST /apis/tools.kdeps.io/v1/tools/X/call"
spec: "ToolSpec (Go)" {shape: diamond}
declared: "assets/tools/X.yaml\nhttp: or exec:" {shape: diamond}
command: "/name" {shape: oval}
commands: "POST /apis/commands.kdeps.io/v1/commands/X/call"
cmdasset: "assets/commands/X.yaml"
loop: "agent loop reads\nharness, events, actions,\npresets, themes" {shape: oval}

workflow -> component
component -> tools: "kdeps.api"
tools -> spec: "Go tool"
tools -> declared: "declared tool"
command -> cmdasset
cmdasset -> commands: "builtin: api:"
loop -> cmdasset: "config data, not served"
```

Chains by asset set:

- **Tools**: `tools/` assets with `http:`, `exec:`, `sql:` or `python:` are served by the declared runner. Go-backed tools have a `ToolSpec`. `session_ops` reads and changes the active agent session. Its `tool` operation runs the session-only tools (`judge_verdict`, `knowledge_*`, `load_skill`, `load_tools`, `identity_get`, `session_handshake`), which have no route of their own. The tool list marks `session_ops` `"agentOnly": true`; in workflow mode, with no agent session, a call fails.
- **Components**: each `components/` asset declares `requires:` and calls `kdeps.api`. The component invariant tests enforce this for workflow tools.
- **Commands**: builtin commands name their operation in `api:`. Declared prompt commands are published under their own name.
- **Loop data**: configuration only, see [Loop data sets](#loop-data-sets).

## Routes

Every route requires the bearer token. A missing or wrong token is `401`.

| Method | Path | Body |
| --- | --- | --- |
| `GET` | `/api` | Core versions. `versions` is `["v1"]`. |
| `GET` | `/apis` | Groups `tools.kdeps.io` and `commands.kdeps.io`, each at `v1`. Workflow execution is not an API group: a workflow runs through `kdeps run`. |
| `GET` | `/openapi/v3` | OpenAPI for the routes below. |
| `GET` | `/apis/tools.kdeps.io/v1/tools` | Tool list. `bash_exec`, `bash_job_list`, `bash_job_wait`, `sql_ops`, `memory_save`, `memory_search`, `memory_delete`, `memory_query`, `session_ops` (which also runs the agent-session tools `identity_get`, `judge_verdict`, `knowledge_card_get`, `knowledge_card_update`, `knowledge_note_save`, `knowledge_recall`, `load_skill`, `load_tools` and `session_handshake`), `http_request`, `exec`, `python`, `search_local`, `load_document`, `vector_store`, `embedding`, `embedding_search`, `embedding_vectorize`, the four `approval_*` tools, `instruct`, and the tools whose implementation is declared in their own asset (`calculator`, `clipboard_copy`, `cohere_rerank`, `edit_file`, `exa_search`, `file_ops`, `git_ops`, `google_cache_create`, `google_cache_delete`, `google_cache_list`, `hf_repo_files`, `hf_search`, `jina_rerank`, `list_files`, `md5_file`, `ocr_image`, `read_file`, `retrieve_context`, `sql_describe_table`, `sql_list_tables`, `sql_query`, `tail_file`, `transcribe_audio`, `transcribe_audio_offline`, `voyageai_rerank`, `web_scraper`, `web_search`, `wikipedia`, `wolfram_alpha`, `write_file`, `zapier_list_actions`, `zapier_run_action`). |
| `POST` | `/apis/tools.kdeps.io/v1/tools/{name}/call` | Tool result, `text/plain`. |
| `GET` | `/apis/commands.kdeps.io/v1/commands` | Slash-command list: each command's operation name (`name`), the name you type (`display`), its usage lines, and the first usage line as `description`. |
| `POST` | `/apis/commands.kdeps.io/v1/commands/{name}/call` | Command output, `text/plain`. `name` is the operation id: the builtin command's `commands/<name>.yaml` file stem, which its `api:` field repeats. Declared prompt commands (`explain`, `frugal`, `review`, `thorough`, and any `commands/` asset with `run:`) use their own name and return each model reply, separated by a blank line. Their `command:` steps print to the REPL's output and are not returned. The call runs against the active REPL session. `{"args": ["new", "ship it"]}`, `{"args": "new ship it"}` and `{"arguments": "new ship it"}` are all accepted; args not as an array are split on whitespace. |

```bash
curl -sS -H "Authorization: Bearer $KDEPS_API_TOKEN" \
  http://127.0.0.1:9095/api
```

```json
{"kind":"APIVersions","apiVersion":"v1","versions":["v1"]}
```

```bash
curl -sS -H "Authorization: Bearer $KDEPS_API_TOKEN" \
  http://127.0.0.1:9095/apis
```

```json
{
  "kind": "APIGroupList",
  "apiVersion": "v1",
  "groups": [
    {
      "name": "tools.kdeps.io",
      "versions": [{"groupVersion": "tools.kdeps.io/v1", "version": "v1"}],
      "preferredVersion": {"groupVersion": "tools.kdeps.io/v1", "version": "v1"}
    },
    {
      "name": "commands.kdeps.io",
      "versions": [{"groupVersion": "commands.kdeps.io/v1", "version": "v1"}],
      "preferredVersion": {"groupVersion": "commands.kdeps.io/v1", "version": "v1"}
    }
  ]
}
```

`POST /api` with a valid token is `405`. `GET` on a call route is `405` and `Allow` is `POST`. `GET /openapi/v3` lists `GET /api`, `GET /apis`, `GET /openapi/v3`, `GET /apis/tools.kdeps.io/v1/tools`, `POST /apis/tools.kdeps.io/v1/tools/{name}/call`, `GET /apis/commands.kdeps.io/v1/commands`, and `POST /apis/commands.kdeps.io/v1/commands/{name}/call`.

## Call read_file

`read_file` reads a file and returns numbered lines. The agent tool, a workflow component, and `curl` all call the same handler.

```text
component -> POST /apis/tools.kdeps.io/v1/tools/read_file/call -> handler
```

```yaml
# resources/read.yaml
actionId: readConfig
component:
  name: read_file
  with:
    file_path: workflow.yaml  # resolved from the process working directory
```

```yaml
# resources/respond.yaml
actionId: respond
flow:
  requires:
    - readConfig
apiResponse:
  success: true
  response: "{{ output('readConfig') }}"  # the numbered file text
```

The component POSTs to `http://kdeps.api/apis/tools.kdeps.io/v1/tools/read_file/call`. That host is not dialed. The HTTP executor hands the request to the in-process handler. Do not put the bearer token in the component.

```bash
curl -sS -H "Authorization: Bearer $KDEPS_API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"file_path":"workflow.yaml"}' \
  http://127.0.0.1:9095/apis/tools.kdeps.io/v1/tools/read_file/call
```

A file whose text is `line 1`, `line 2`, and `line 3` comes back as:

```text
1	line 1$
2	line 2$
3	line 3$
```

An unknown tool name is `404` with message `tool not found`. A body without `file_path` is `400`.

## Run a slash command

Every builtin slash command is a `commands/<op>.yaml` asset. Its `api:` field names the operation that runs it, and the REPL publishes each operation on this API. The file stem, `api:`, and the call-route `name` are all the same string; a command whose asset does not match is skipped at load.

```text
/clear -> commands/clear.yaml (api: clear) -> POST /apis/commands.kdeps.io/v1/commands/clear/call -> handler
```

```bash
curl -sS -H "Authorization: Bearer $KDEPS_API_TOKEN" \
  http://127.0.0.1:9095/apis/commands.kdeps.io/v1/commands
```

```json
{
  "kind": "CommandList",
  "apiVersion": "commands.kdeps.io/v1",
  "items": [
    {
      "name": "clear",
      "display": "clear",
      "description": "Clear the conversation history",
      "usage": [{"args": "", "help": "Clear the conversation history"}]
    }
  ]
}
```

`display` is the name you type in the REPL, which is not always the operation id (`/instruct!` is `instruct-now`). Run one:

```bash
curl -sS -H "Authorization: Bearer $KDEPS_API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"args": [""]}' \
  http://127.0.0.1:9095/apis/commands.kdeps.io/v1/commands/help/call
```

The call runs against the REPL that published it, so it acts on that session (`/clear` empties it). The response is the command's output, `text/plain`; a command that prints only to the terminal returns an empty body, and a command that fails is `400` with the error text. Arg decoding accepts a JSON array, a string (split on whitespace), or an `arguments` string. An unknown name is `404` with message `command not found`.

The routes are published by the agent loop (`kdeps`, the desktop app). A `kdeps run` process serves the routes with no commands registered: the list is empty and every call is `404`. `/update` re-registers the commands after it installs assets, so a new display name or usage line shows without a restart.

## Other calls

These tools use the same call route: `bash_exec`, `bash_job_list`, `bash_job_wait`, `sql_ops`, `memory_save`, `memory_search`, `memory_delete`, `memory_query`, `session_ops`, `http_request`, `exec`, `python`, `search_local`, `load_document`, `vector_store`, `embedding`, `embedding_search`, `embedding_vectorize`, the four `approval_*` tools, `instruct`, and the asset-declared tools. `session_ops` also runs the agent-session tools `identity_get`, `judge_verdict`, `knowledge_card_get`, `knowledge_card_update`, `knowledge_note_save`, `knowledge_recall`, `load_skill`, `load_tools` and `session_handshake`. The handler is the implementation; the agent tool and the shipped component only POST the arguments.

The code-graph tools are the agent's `code_` tools: `code_find_symbol`, `code_callers`, `code_callees`, `code_impact`, `code_impact_and_risk` (files a symbol change touches, with the tests that reach it), `code_impact_of_diff` (symbols a unified diff touches and their dependents), `code_tests_for`, `code_rank_files`, `code_query`, `code_file_neighbors`, `code_node`, `code_trace`, `code_shortest_path`, `code_expand`, `code_find_seeds`, `code_search`, `code_render_subgraph`, `code_references`, `code_imports_exports` and `code_stats`. They run on the working directory and build the index on first call. The `codegraph` asset set (`tools.yaml`) lists them; each is the Kartographer tool of the same name with a `code_` prefix.

Asset-declared tools keep their `http:`, `exec:` or `python:` block in the tool asset, and the handler runs it: `calculator`, `clipboard_copy`, `cohere_rerank`, `edit_file`, `exa_search`, `file_ops`, `git_ops`, `google_cache_create`, `google_cache_delete`, `google_cache_list`, `hf_repo_files`, `hf_search`, `jina_rerank`, `list_files`, `md5_file`, `ocr_image`, `retrieve_context`, `sql_describe_table`, `sql_list_tables`, `sql_query`, `tail_file`, `transcribe_audio`, `transcribe_audio_offline`, `voyageai_rerank`, `web_scraper`, `web_search`, `wikipedia`, `wolfram_alpha`, `zapier_list_actions`, `zapier_run_action`.

- A tool stays listed when its API key is unset; a call returns `<tool>: required environment variables are not set`. The agent offers it to the model only when the environment resolves.
- `ocr_image` (`file`, `language`) and `transcribe_audio_offline` (`file`, `language`, `modelPath`) run local programs; a missing `tesseract` or `whisper-cli` is a `400`.

```text
component -> POST /apis/tools.kdeps.io/v1/tools/calculator/call -> handler
```

```bash
curl -sS -H "Authorization: Bearer $KDEPS_API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"expression":"2 + 2"}' \
  http://127.0.0.1:9095/apis/tools.kdeps.io/v1/tools/calculator/call
```

The body is `4`. A missing required argument is a `400`.

| Tool | Arguments | Result and errors |
|---|---|---|
| `bash_exec` | `command` | `{"command":"echo hi"}` returns `hi`. With `KDEPS_ALLOW_BASH=false` it stays listed and returns `bash_exec: disabled (KDEPS_ALLOW_BASH=false)` |
| `exec` | `command`; optional `args`, `env`, `workingDir`, `timeout` | JSON `{success, exitCode, stdout, stderr, command, result, timedOut}`; non-zero exit is a `400` |
| `python` | `script` or `scriptFile`; optional `args`, `env`, `timeout`, `venvName`, `pythonVersion`, `packages` | stdout parsed as JSON when possible, else raw text; non-zero exit is a `400` |
| `embedding` | `operation` (`index`, `search`, `upsert`, `delete`, `vectorize`, `embed_query`, `rerank`) plus its fields: `text`, `collection`, `dbPath`, `limit`, `inputs`, `model`, `backend`, `baseURL`, `rerankQuery`, `rerankDocuments`, `rerankTopN` | the embedding executor's result map; unknown operation is a `400` |
| `embedding_vectorize` | texts | no texts returns `embedding vectorize: no inputs provided` |
| `vector_store` | `operation` (`add_documents`, `similarity_search`), `collection`, then `documents` (`[{content, metadata}]`), `query`, `topK`; store and embedder: `provider`, `url`, `apiKey`, `embedModel`, `embedBackend`, `embedBaseURL` | `added`/`ids` or `results`/`count`/`json` |
| `list_files` | `path` (empty: working directory) | |
| `md5_file` | `file_path` | `<path>  <hex>` |
| `tail_file` | `file_path`, `lines` (default 20) | |
| `write_file` | `file_path`, `content` | |
| `edit_file` | `file_path`, `line` or `match`, `text` or `append` | |
| `web_search`, `wikipedia` | `query` | |
| `web_scraper` | `url` | |
| `sql_query` | `query`, `db_path` (falls back to `KDEPS_SQL_DB_PATH`) | |
| `load_document` | `source` | |
| `search_local` | query or glob | neither: `search_local: give a query (or a glob) to search for`; path `/`: `search_local: operation on root directory '/' is blocked (set KDEPS_ALLOW_ROOT=true to permit)` |
| `memory_save`, `memory_search`, `memory_delete` | | no store: `<tool>: memory store is not configured` (the agent attaches its store before calling) |
| `memory_query` | `query` | empty: `memory_query: query is required`; no agent loop: `memory_query: agent loop is not configured (memory_query is an agent-mode tool)` |
| `session_ops` | `{"operation":"tool","tool":"identity_get"}` | runs that session tool on the active loop; under `kdeps run`: `identity_get: no active agent session (agent-mode tool)` |
