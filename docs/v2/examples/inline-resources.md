# Run extra resources inline

Attach `exec:`, `python:`, and `sql:` actions directly to
one resource's `before:` and `after:` blocks, instead of creating a separate
file for each. The inline actions run as part of the main resource.

`before:` and `after:` usually hold bare expressions. They can also hold whole
resource actions - `chat:`, `httpClient:`, `sql:`, `python:`, `exec:` - as list
items. Use inline actions for one-off setup or teardown that only this resource
needs; use a separate resource when other resources also depend on the result.

## Step 1: create the project

```bash
mkdir inline-demo
cd inline-demo
mkdir resources

sqlite3 results.db "CREATE TABLE runs (data TEXT, at TEXT);"
```

## Step 2: define the route and connection

Create `workflow.yaml`:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: inline-demo
  version: "1.0.0"
  targetActionId: main

settings:
  apiServer:
    portNum: 16396
    routes:
      - path: /api/v1/process
        methods: [POST]
  agentSettings:
    pythonVersion: "3.12"
```

The `results` database is named in the resources and defined machine-locally:

```bash
export KDEPS_SQL_CONNECTIONS_RESULTS_CONNECTION="sqlite://./results.db"   # or sql_connections.results in ~/.kdeps/config.yaml
```

## Step 3: the main resource with inline actions

Create `resources/main.yaml`:

<div v-pre>

```yaml
# resources/main.yaml
actionId: main
name: Process with inline resources
validations:
  methods: [POST]
  routes: [/api/v1/process]
  required:
    - data
  rules:
    - field: data
      type: string
      minLength: 1
      message: "data is required"

before:
  # setup: a shell command that only this resource needs
  - exec:
      command: "echo 'preparing'"
      timeout: 5s

# main action
chat:
  model: llama3.2:1b
  role: user
  prompt: "Rewrite this more formally: {{ get('data') }}"
  timeout: 30s

after:
  # persist the input
  - sql:
      connectionName: results
      query: "INSERT INTO runs (data, at) VALUES ($1, datetime('now'))"
      params:
        - "{{ get('data') }}"
  # post-process
  - python:
      script: |
        import json
        print(json.dumps({"post": "done"}))

onError:
  action: continue
  fallback:
    error: "processing failed"

apiResponse:
  success: true
  response:
    formal: "{{ get('main').message.content }}"
```

</div>

## Step 4: validate and run

```bash
kdeps validate .
export KDEPS_API_AUTH_TOKEN=dev-token
export KDEPS_SQL_CONNECTIONS_RESULTS_CONNECTION="sqlite://./results.db"   # skip if set in config.yaml
kdeps run .
```

```bash
curl -X POST http://localhost:16396/api/v1/process \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"data": "hey can you send me that file"}'
```

The `before:` exec runs, then the chat, then the `after:` SQL insert and Python
script - all within the `main` resource. The model's wording varies:

```json
{"success": true, "data": {"formal": "Could you please send me the file you requested?"}}
```

Confirm the `after:` insert landed:

```bash
sqlite3 results.db "SELECT * FROM runs"
# hey can you send me that file|2026-10-06 03:09:40
```

## Next steps

- [Inline resources](/workflow/inline-resources) - all supported types, ordering
- [Expression blocks](/reference/expr-blocks) - `before:` / `after:` in detail
- [Error handling (onError)](/workflow/error-handling) - fallback for the whole resource
- [SQL resource](/workflow/resources/sql) - standalone SQL resources
