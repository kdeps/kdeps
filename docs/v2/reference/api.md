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
| `events/` | `pkg/agent/events.go` | Governor triggers: conditions that start an efficiency action |
| `actions/` | `pkg/agent/actions.go` | The governor's action vocabulary |
| `presets/` | `pkg/agent/presets.go` | Named bundles of harness and efficiency settings (`/efficiency preset`) |
| `themes/` | `pkg/agent/theme_yaml.go` | REPL colors and styles |

Users override a set by name under `~/.kdeps/` (see the agent reference). The loop reads the merged result at start.
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

- **Tools**: `tools/` assets with `http:` or `exec:` are served by the declared runner. Go-backed tools have a `ToolSpec`. Agent-session tools run the active loop's closure.
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
| `GET` | `/apis/tools.kdeps.io/v1/tools` | Tool list. `bash_exec`, `bash_job_list`, `bash_job_wait`, `read_file`, `list_files`, `md5_file`, `tail_file`, `write_file`, `edit_file`, `web_search`, `wikipedia`, `web_scraper`, `calculator`, the four `sql_*` tools, the three `google_cache_*` tools, `memory_save`, `memory_search`, `memory_delete`, `memory_query`, the agent-session tools (`identity_get`, `judge_verdict`, `knowledge_card_get`, `knowledge_card_update`, `knowledge_note_save`, `knowledge_recall`, `load_skill`, `session_handshake`), `http_request`, `exec`, `python`, `file_ops`, `git_ops`, `sql_ops`, `search_local`, `load_document`, `vector_store`, `embedding`, `embedding_search`, `embedding_vectorize`, the four `approval_*` tools, `instruct`, and the tools whose implementation is declared in their own asset (`cohere_rerank`, `exa_search`, `jina_rerank`, `ocr_image`, `retrieve_context`, `transcribe_audio`, `transcribe_audio_offline`, `voyageai_rerank`, `wolfram_alpha`, `zapier_list_actions`, `zapier_run_action`). |
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

`bash_exec`, `bash_job_list`, `bash_job_wait`, `list_files`, `md5_file`, `tail_file`, `write_file`, `edit_file`, `web_search`, `wikipedia`, `web_scraper`, `calculator`, `sql_list_tables`, `sql_describe_table`, `sql_query`, `sql_ops`, `google_cache_create`, `google_cache_delete`, `google_cache_list`, `memory_save`, `memory_search`, `memory_delete`, `memory_query`, the agent-session tools (`identity_get`, `judge_verdict`, `knowledge_card_get`, `knowledge_card_update`, `knowledge_note_save`, `knowledge_recall`, `load_skill`, `session_handshake`), `http_request`, `exec`, `python`, `file_ops`, `git_ops`, `search_local`, `load_document`, `vector_store`, `embedding`, `embedding_search`, `embedding_vectorize`, the four `approval_*` tools, `instruct`, and the asset-declared tools use the same call route. The handler is the implementation. The agent tool and the shipped component only POST the arguments.

An asset-declared tool (`cohere_rerank`, `exa_search`, `jina_rerank`, `ocr_image`, `retrieve_context`, `transcribe_audio`, `transcribe_audio_offline`, `voyageai_rerank`, `wolfram_alpha`, `zapier_list_actions`, `zapier_run_action`) keeps its `http:` or `exec:` block in its tool asset; the handler runs that block. The tool stays on this list when its API key is unset, like the keyed search tools above. A call then returns `<tool>: required environment variables are not set`. The agent registers it for the model only when the environment resolves. `ocr_image` takes `file` and `language`, and `transcribe_audio_offline` takes `file`, `language`, and `modelPath`; both run a local program, so a missing `tesseract` or `whisper-cli` is a `400`.

```text
component -> POST /apis/tools.kdeps.io/v1/tools/calculator/call -> handler
```

```bash
curl -sS -H "Authorization: Bearer $KDEPS_API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"expression":"2 + 2"}' \
  http://127.0.0.1:9095/apis/tools.kdeps.io/v1/tools/calculator/call
```

The body is `4`. `bash_exec` takes `command` (`{"command":"echo hi"}` returns `hi`). `exec` takes `command` plus optional `args`, `env`, `workingDir`, and `timeout`, and returns the exec executor's structured result (`{success, exitCode, stdout, stderr, command, result, timedOut}`) as a JSON body; a non-zero exit is a `400`. `python` takes `script` or `scriptFile` plus optional `args`, `env`, `timeout`, `venvName`, `pythonVersion`, and `packages`, and returns the script's stdout parsed as JSON when possible, otherwise the raw text; a non-zero exit is a `400`. `embedding` takes `operation` (`index`, `search`, `upsert`, `delete`, `vectorize`, `embed_query`, or `rerank`) plus the fields for that operation: `text`, `collection`, `dbPath`, `limit`, `inputs`, `model`, `backend`, `baseURL`, `rerankQuery`, `rerankDocuments`, and `rerankTopN`, and returns the embedding executor's result map; an unknown operation is a `400`. `list_files` takes `path` (empty lists the working directory). `md5_file` takes `file_path` and returns `<path>  <hex>`. `tail_file` takes `file_path` and `lines` (default 20). `write_file` takes `file_path` and `content`. `edit_file` takes `file_path` and `command`. `web_search` and `wikipedia` take `query`. `web_scraper` takes `url`. `sql_query` takes `query` and `db_path` (`db_path` falls back to `KDEPS_SQL_DB_PATH`). A missing required argument is `400`. `KDEPS_ALLOW_BASH=false` leaves `bash_exec` on this list and a call returns `bash_exec: disabled (KDEPS_ALLOW_BASH=false)`. The `google_cache_*` tools stay on this list when their API key is unset. A call then returns `<tool>: disabled (<ENV> is unset)`. The agent still registers those tools for the model only when the key is set. A workflow `edit_file` call passes `file_path` and does not need a prior read. The read-before-edit check stays in the agent loop. `memory_save`, `memory_search`, and `memory_delete` stay on this list. A call with no memory store returns `memory_save: memory store is not configured`, and the same sentence for `memory_search` and `memory_delete`. The agent attaches its store before the call. `memory_query` stays on this list. An empty query returns `memory_query: query is required`. A call with no agent loop returns `memory_query: agent loop is not configured (memory_query is an agent-mode tool)`. The agent-session tools stay on this list and run against the active loop; in a `kdeps run` process a call returns `<tool>: no active agent session (agent-mode tool)`. `search_local` with no query and no glob returns `search_local: give a query (or a glob) to search for`. A path of `/` on that call returns `search_local: operation on root directory '/' is blocked (set KDEPS_ALLOW_ROOT=true to permit)`. `load_document` reads `source`. `vector_store` takes `operation` (`add_documents` or `similarity_search`) plus `collection`, and for the operation: `documents` (`[{content, metadata}]`), `query`, and `topK`, with `provider`, `url`, `apiKey`, `embedModel`, `embedBackend`, and `embedBaseURL` describing the store and the embedding backend; the result is the executor's map (`added`/`ids` or `results`/`count`/`json`). `embedding_vectorize` with no texts returns `embedding vectorize: no inputs provided`.
