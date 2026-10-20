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

## Routes

Every route requires the bearer token. A missing or wrong token is `401`.

| Method | Path | Body |
| --- | --- | --- |
| `GET` | `/api` | Core versions. `versions` is `["v1"]`. |
| `GET` | `/apis` | Groups `tools.kdeps.io` and `workflow.kdeps.io`, each at `v1`. |
| `GET` | `/openapi/v3` | OpenAPI for the routes below. |
| `GET` | `/apis/tools.kdeps.io/v1/tools` | Tool list. `bash_exec`, `bash_job_list`, `bash_job_wait`, `read_file`, `list_files`, `md5_file`, `tail_file`, `write_file`, `edit_file`, `web_search`, `wikipedia`, `web_scraper`, `calculator`, the three `sql_*` tools, `serpapi_search`, `perplexity_search`, the three `google_cache_*` tools, `memory_save`, `memory_search`, `memory_delete`, the `code_*` tools, `http_request`, `search_local`, `load_document`, `embedding_search`, and `embedding_vectorize`. |
| `POST` | `/apis/tools.kdeps.io/v1/tools/{name}/call` | Tool result, `text/plain`. |

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
      "name": "workflow.kdeps.io",
      "versions": [{"groupVersion": "workflow.kdeps.io/v1", "version": "v1"}],
      "preferredVersion": {"groupVersion": "workflow.kdeps.io/v1", "version": "v1"}
    }
  ]
}
```

`POST /api` with a valid token is `405`. `GET` on a call route is `405` and `Allow` is `POST`. `GET /openapi/v3` lists `GET /api`, `GET /apis`, `GET /openapi/v3`, `GET /apis/tools.kdeps.io/v1/tools`, and `POST /apis/tools.kdeps.io/v1/tools/{name}/call`.

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

## Other calls

`bash_exec`, `bash_job_list`, `bash_job_wait`, `list_files`, `md5_file`, `tail_file`, `write_file`, `edit_file`, `web_search`, `wikipedia`, `web_scraper`, `calculator`, `sql_list_tables`, `sql_describe_table`, `sql_query`, `serpapi_search`, `perplexity_search`, `google_cache_create`, `google_cache_delete`, `google_cache_list`, `memory_save`, `memory_search`, `memory_delete`, the `code_*` tools, `http_request`, `search_local`, `load_document`, `embedding_search`, and `embedding_vectorize` use the same call route. The handler is the implementation. The agent tool and the shipped component only POST the arguments.

```text
component -> POST /apis/tools.kdeps.io/v1/tools/calculator/call -> handler
```

```bash
curl -sS -H "Authorization: Bearer $KDEPS_API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"expression":"2 + 2"}' \
  http://127.0.0.1:9095/apis/tools.kdeps.io/v1/tools/calculator/call
```

The body is `4`. `bash_exec` takes `command` (`{"command":"echo hi"}` returns `hi`). `list_files` takes `path` (empty lists the working directory). `md5_file` takes `file_path` and returns `<path>  <hex>`. `tail_file` takes `file_path` and `lines` (default 20). `write_file` takes `file_path` and `content`. `edit_file` takes `file_path` and `command`. `web_search` and `wikipedia` take `query`. `web_scraper` takes `url`. `sql_query` takes `query` and `db_path` (`db_path` falls back to `KDEPS_SQL_DB_PATH`). A missing required argument is `400`. `KDEPS_ALLOW_BASH=false` leaves `bash_exec` on this list and a call returns `bash_exec: disabled (KDEPS_ALLOW_BASH=false)`. `serpapi_search`, `perplexity_search`, and the `google_cache_*` tools stay on this list when their API key is unset. A call then returns `<tool>: disabled (<ENV> is unset)`. The agent still registers those tools for the model only when the key is set. A workflow `edit_file` call passes `file_path` and does not need a prior read. The read-before-edit check stays in the agent loop. `memory_save`, `memory_search`, and `memory_delete` stay on this list. A call with no memory store returns `memory_save: memory store is not configured`, and the same sentence for `memory_search` and `memory_delete`. The agent attaches its store before the call. `memory_query` is not on this list. `code_search`, `code_definition`, `code_references`, `code_symbols`, `code_hover`, `code_diagnostics`, `code_index_folder`, `code_graph_file`, `code_graph_topic`, and `code_graph_all` run in the same handler. `code_search` with no query returns `codeIntelligence: query is required for symbolSearch`. `code_index_folder` indexes the working directory. A path of `/` returns `code_index_folder: operation on root directory '/' is blocked (set KDEPS_ALLOW_ROOT=true to permit)`. `search_local` with no query and no glob returns `search_local: give a query (or a glob) to search for`. A path of `/` on that call returns `search_local: operation on root directory '/' is blocked (set KDEPS_ALLOW_ROOT=true to permit)`. `load_document` reads `source`. `embedding_vectorize` with no texts returns `embedding vectorize: no inputs provided`.
